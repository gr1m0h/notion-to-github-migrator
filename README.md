# notion-to-github-migrator

A Go tool to migrate CSV exports from Notion databases to GitHub issues.

## Features

- 🔧 Flexible Configuration: Customize migration mappings via JSON configuration
- 🏷️ Auto Label Creation: Automatically creates labels that don't exist in GitHub
- 🔄 Retry Mechanism: Automatic retry for temporary errors
- 🎨 Auto Color Generation: Automatically generates label colors
- ⚡ Fast Execution: High-performance processing with Go

## Requirements

Go 1.21+
GitHub Personal Access Token (Fine-grained)

## Installation

```bash
# Clone repository
git clone https://github.com/gr1m0h/notion-to-github-migrator.git
cd notion-to-github-migrator

# Download dependencies
go mod download

# Build
go build -o notion-to-github-migrator cmd/notion-to-github-migrator/main.go

```

### Using go install

```bash
go install github.com/gr1m0h/notion-to-github-migrator@latest

# Or install from local directory
go install ./cmd/notion-to-github-migrator
```

## Usage

### 1. Create GitHub Personal Access Token

1. Go to GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens
2. Click "Generate new token"
3. Set required permissions:
   - Repository access: Select target repository
   - Permissions:
     - Issues: Read & Write
     - Metadata: Read

### 2. Export CSV from Notion

1. Select Export from your Notion database menu
2. Choose "CSV" as Export format
3. Save the exported CSV file

### 3. Create Configuration File (Optional)

Create `config.json` with the following content:

```json
{
  "github": {
    "token": "github_pat_xxxxxxxxxxxxx",
    "owner": "your-org-or-username",
    "repo": "your-repository-name"
  },
  "fieldMapping": {
    "Name": {
      "githubField": "title"
    },
    "Tag": {
      "githubField": "label",
      "delimiter": ", "
    },
    "Priority": {
      "githubField": "label",
      "delimiter": ", "
    }
  },
  "retry": {
    "maxAttempts": 3,
    "delayMs": 1000
  }
}
```

### 4. Run Migration

```bash
# Using config file
./notion-to-github-migrator -csv notion-export.csv -config config.json

# Using environment variable for token (without config file)
export GITHUB_TOKEN=github_pat_xxxxxxxxxxxxx
./notion-to-github-migrator -csv notion-export.csv

# Show help
./notion-to-github-migrator -h
```

## Configuration Options

### github

- `token`: GitHub Personal Access Token (can also be set via GITHUB_TOKEN environment variable)
- `owner`: Repository owner (organization or username)
- `repo`: Repository name

### fieldMapping

Defines mapping between Notion fields and GitHub fields.

- Key: Notion CSV column name
- Value:
  - `githubField`: Target field ("title", "label", or "body")
  - `delimiter`: Delimiter for multiple values (only for labels, default: ",")

### retry

- `maxAttempts`: Maximum retry attempts (default: 3)
- `delayMs`: Delay between retries in milliseconds (default: 1000)

## Customization Examples

### Additional Field Mapping

```json
{
  "fieldMapping": {
    "Name": { "githubField": "title" },
    "Description": { "githubField": "body" },
    "Status": { "githubField": "label" },
    "Assignee": { "githubField": "label", "delimiter": ";" }
  }
}
```

### Minimal Configuration

Set token via environment variable and specify minimal info in config:

```json
{
  "github": {
    "owner": "your-org-or-username",
    "repo": "your-repository-name"
  }
}
```

## Note

- CSV file must be UTF-8 encoded
- Be aware of GitHub API rate limits when creating many issues
- Label colors are auto-generated but can be changed later in GitHub UI
- The program continues on errors to create as many issues as possible
