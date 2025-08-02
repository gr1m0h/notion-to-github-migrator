# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build
```bash
# Build the main binary
go build -o notion-to-github-migrator cmd/notion-to-github-migrator/main.go

# Install from local directory
go install ./cmd/notion-to-github-migrator
```

### Testing
```bash
# Run all tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./cmd/notion-to-github-migrator
```

### Code Quality
```bash
# Check code with go vet
go vet ./...

# Format code
go fmt ./...

# Download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

## Architecture Overview

This is a single-purpose CLI tool that migrates CSV exports from Notion databases to GitHub issues. The entire application is contained in a single Go package at `cmd/notion-to-github-migrator/`.

### Core Components

- **main.go**: Contains the complete application logic including:
  - `Config` struct: Configuration management with support for JSON config files and environment variables
  - `FieldMap` struct: Mapping between Notion CSV columns and GitHub issue fields
  - CSV parsing and processing with configurable field mappings
  - GitHub API integration using `github.com/google/go-github/v57`
  - Retry mechanism with configurable attempts and delays
  - Automatic label creation with MD5-based color generation

### Key Functions
- `loadConfig()`: Loads configuration from file or environment variables
- `migrateNotionToGitHub()`: Main migration orchestrator
- `processRecord()`: Converts CSV records to GitHub issue fields
- `ensureLabel()`: Creates GitHub labels if they don't exist
- `createIssueWithRetry()`: Creates issues with retry logic

### Configuration
The tool supports flexible configuration via JSON files with these key sections:
- `github`: Repository details and authentication token
- `fieldMapping`: Maps Notion columns to GitHub fields (title, label, body)
- `retry`: Configures retry behavior for API failures

Authentication can be provided via:
- `GITHUB_TOKEN` environment variable
- Token in JSON config file

### Dependencies
- `github.com/google/go-github/v57`: GitHub API client
- `golang.org/x/oauth2`: OAuth2 authentication
- Standard library only for CSV parsing, JSON handling, and HTTP functionality

### Testing
Comprehensive test suite covers:
- Configuration loading and validation
- CSV record processing with various field mappings
- GitHub API interactions with mock HTTP servers
- Retry mechanisms and error handling
- Label color generation consistency