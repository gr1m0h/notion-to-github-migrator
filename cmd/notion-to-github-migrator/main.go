package main

import (
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Config represents the configuration structure
type Config struct {
	GitHub struct {
		Token string `json:"token"`
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	} `json:"github"`
	FieldMapping map[string]FieldMap `json:"fieldMapping"`
	Retry        struct {
		MaxAttempts int `json:"maxAttempts"`
		DelayMs     int `json:"delayMs"`
	} `json:"retry"`
}

// FieldMap represents field mapping configuration
type FieldMap struct {
	GitHubField string `json:"githubField"`
	Delimiter   string `json:"delimiter,omitempty"`
}

// Default configuration
func getDefaultConfig() Config {
	config := Config{}
	config.GitHub.Token = os.Getenv("GITHUB_TOKEN")
	config.FieldMapping = map[string]FieldMap{
		"Name":          {GitHubField: "title"},
		"Tag":           {GitHubField: "label", Delimiter: ", "},
		"SRE Goals":     {GitHubField: "label", Delimiter: ", "},
		"Priority":      {GitHubField: "label", Delimiter: ", "},
		"T-shirt sizes": {GitHubField: "label", Delimiter: ", "},
	}
	config.Retry.MaxAttempts = 3
	config.Retry.DelayMs = 1000
	return config
}

// Load configuration from file
func loadConfig(configPath string) (*Config, error) {
	config := getDefaultConfig()

	if configPath != "" {
		file, err := os.Open(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(&config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Validate configuration
	if config.GitHub.Token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}
	if config.GitHub.Owner == "" {
		return nil, fmt.Errorf("GitHub owner is required")
	}
	if config.GitHub.Repo == "" {
		return nil, fmt.Errorf("GitHub repo is required")
	}

	return &config, nil
}

// Generate label color based on label name
func generateLabelColor(labelName string) string {
	hash := md5.Sum([]byte(labelName))
	return fmt.Sprintf("%x", hash[:3])
}

// Ensure label exists in GitHub
func ensureLabel(ctx context.Context, client *github.Client, owner, repo, labelName string) error {
	// Check if label exists
	_, _, err := client.Issues.GetLabel(ctx, owner, repo, labelName)
	if err == nil {
		return nil // Label already exists
	}

	// Create label if it doesn't exist
	if err != nil {
		color := generateLabelColor(labelName)
		label := &github.Label{
			Name:  &labelName,
			Color: &color,
		}

		_, _, err = client.Issues.CreateLabel(ctx, owner, repo, label)
		if err != nil {
			return fmt.Errorf("failed to create label %s: %w", labelName, err)
		}
		log.Printf("Created label: %s with color #%s\n", labelName, color)
	}

	return nil
}

// Create issue with retry
func createIssueWithRetry(ctx context.Context, client *github.Client, config *Config, issue *github.IssueRequest) error {
	var lastErr error

	for attempt := 1; attempt <= config.Retry.MaxAttempts; attempt++ {
		_, _, err := client.Issues.Create(ctx, config.GitHub.Owner, config.GitHub.Repo, issue)
		if err == nil {
			log.Printf("Created issue: %s\n", *issue.Title)
			return nil
		}

		lastErr = err
		log.Printf("Failed to create issue \"%s\" (attempt %d/%d): %v\n",
			*issue.Title, attempt, config.Retry.MaxAttempts, err)

		if attempt < config.Retry.MaxAttempts {
			time.Sleep(time.Duration(config.Retry.DelayMs) * time.Millisecond)
		}
	}

	log.Printf("Failed to create issue \"%s\" after %d attempts. Continuing...\n",
		*issue.Title, config.Retry.MaxAttempts)
	return lastErr
}

// Process CSV record
func processRecord(record map[string]string, config *Config) (string, []string, string) {
	var title string
	var body string
	labelsMap := make(map[string]bool)

	for notionField, mapping := range config.FieldMapping {
		value, exists := record[notionField]
		if !exists || value == "" {
			continue
		}

		switch mapping.GitHubField {
		case "title":
			title = value

		case "label":
			delimiter := mapping.Delimiter
			if delimiter == "" {
				delimiter = ","
			}

			labelValues := strings.Split(value, delimiter)
			for _, labelValue := range labelValues {
				trimmed := strings.TrimSpace(labelValue)
				if trimmed != "" {
					labelsMap[trimmed] = true
				}
			}

		case "body":
			if body != "" {
				body += "\n\n"
			}
			body += value
		}
	}

	// Convert labels map to slice
	var labels []string
	for label := range labelsMap {
		labels = append(labels, label)
	}

	return title, labels, body
}

// Migrate Notion to GitHub
func migrateNotionToGitHub(csvPath string, configPath string) error {
	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create GitHub client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GitHub.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	// Read CSV
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Process records
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error reading CSV record: %v, skipping...\n", err)
			continue
		}

		// Create map from record
		recordMap := make(map[string]string)
		for i, value := range record {
			if i < len(header) {
				recordMap[header[i]] = value
			}
		}

		// Process record
		title, labels, body := processRecord(recordMap, config)

		// Skip if no title
		if title == "" {
			log.Println("Skipping record with empty title")
			continue
		}

		// Ensure all labels exist
		for _, label := range labels {
			if err := ensureLabel(ctx, client, config.GitHub.Owner, config.GitHub.Repo, label); err != nil {
				log.Printf("Warning: %v\n", err)
			}
		}

		// Create issue
		issue := &github.IssueRequest{
			Title:  &title,
			Body:   &body,
			Labels: &labels,
		}

		if err := createIssueWithRetry(ctx, client, config, issue); err != nil {
			log.Printf("Error creating issue: %v\n", err)
		}
	}

	log.Println("Migration completed!")
	return nil
}

func main() {
	var csvPath string
	var configPath string

	flag.StringVar(&csvPath, "csv", "", "Path to the CSV file containing Notion data")
	flag.StringVar(&configPath, "config", "", "Path to the configuration file")
	flag.Parse()

	if csvPath == "" {
		log.Fatal("CSV file path is required. Use -csv flag")
	}

	if err := migrateNotionToGitHub(csvPath, configPath); err != nil {
		log.Fatal(err)
	}
}
