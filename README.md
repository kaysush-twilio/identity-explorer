# Identity Explorer

A TUI (Terminal User Interface) application for exploring Identity Service data in DynamoDB. This tool allows you to query profile mappings and merge records interactively.

## Features

- **Query by Profile**: Look up all identifiers and merge records for a specific profile ID
- **Query by Identifier**: Find profiles associated with a specific identifier (email, phone, etc.)
- Interactive profile selection when multiple matches are found
- Formatted table display for mappings and merges
- Cross-platform support (Linux, macOS, Windows)

## Installation

### Using Homebrew (macOS/Linux)

```bash
brew tap kaysush-twilio/tap
brew install identity-explorer
```

### Download Binary

Download the latest release from the [Releases page](https://github.com/kaysush-twilio/identity-explorer/releases).

### Build from Source

Requires Go 1.22+

```bash
git clone https://github.com/kaysush-twilio/identity-explorer.git
cd identity-explorer
make build
```

## Usage

Run the application:

```bash
identity-explorer
```

### Environment Variables

Configure the DynamoDB tables using environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `MAPPINGS_TABLE` | `local.IdentityMappings.v1` | DynamoDB table for identity mappings |
| `MERGES_TABLE` | `local.IdentityMerges.v1` | DynamoDB table for profile merges |

### AWS Configuration

The tool uses the AWS SDK's default credential chain. Configure your AWS credentials using:

- Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
- AWS credentials file (`~/.aws/credentials`)
- IAM role (when running on AWS infrastructure)

Set the AWS region:

```bash
export AWS_REGION=us-east-1
```

## Query Modes

### 1. Query Profile

Look up all data for a specific profile:

**Inputs:**
- Account ID (e.g., `AC1234567890`)
- Store ID
- Profile ID

**Output:**
- All identifier mappings (email, phone, etc.)
- Merge records (if the profile has been merged)
- Canonical profile information

### 2. Query Identifier

Find profiles by a specific identifier:

**Inputs:**
- Account ID
- Store ID
- ID Type (e.g., `email`, `phone`, `external_id`)
- ID Value (e.g., `user@example.com`)

**Output:**
- List of matching profile IDs (interactive selection if multiple)
- All mappings and merges for the selected profile

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `Down` | Move selection down |
| `k` / `Up` | Move selection up |
| `Tab` | Next input field |
| `Shift+Tab` | Previous input field |
| `Enter` | Select / Submit |
| `Esc` | Go back |
| `q` | Quit (from main menu) |
| `Ctrl+C` | Force quit |

## Development

### Prerequisites

- Go 1.22+
- golangci-lint (for linting)
- goreleaser (for releases)

### Commands

```bash
# Build
make build

# Run
make run

# Test
make test

# Lint
make lint

# Create snapshot release
make release-snapshot
```

### Project Structure

```
identity-explorer/
├── cmd/
│   └── identity-explorer/
│       └── main.go           # Application entry point
├── internal/
│   ├── dynamo/
│   │   └── client.go         # DynamoDB client and queries
│   ├── models/
│   │   └── models.go         # Data models
│   └── ui/
│       ├── app.go            # TUI application logic
│       └── styles.go         # UI styles
├── .github/
│   └── workflows/
│       ├── ci.yml            # CI workflow
│       └── release.yml       # Release workflow
├── .goreleaser.yaml          # GoReleaser configuration
├── Makefile
└── README.md
```

## License

MIT License - see [LICENSE](LICENSE) for details.
