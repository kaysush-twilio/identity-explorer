# Identity Explorer

A TUI (Terminal User Interface) application for exploring Identity Service data in DynamoDB. This tool allows you to query profile mappings and merge records interactively.

## Features

- **Query by Profile**: Look up all identifiers and merge records for a specific profile ID
- **Query by Identifier**: Find profiles associated with a specific identifier (email, phone, etc.)
- Interactive profile selection when multiple matches are found
- Formatted table display for mappings and merges
- Command-line prefill for quick queries
- Auto-constructed table names from environment/region/cell
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

## Quick Start

```bash
# Basic usage - launches interactive TUI (uses defaults: dev/us-east-1/cell-1)
identity-explorer --profile my-aws-profile

# Query a profile directly (skips mode selection)
identity-explorer --profile my-aws-profile \
  --mode profile \
  --account-id AC123456 --store-id my-store --profile-id prof-uuid

# Query by identifier
identity-explorer --profile my-aws-profile \
  --mode identifier \
  --account-id AC123456 --store-id my-store \
  --id-type email --id-value user@example.com

# Query in a different environment
identity-explorer --profile my-aws-profile \
  --env prod --region us-west-2 --cell cell-2 \
  --mode profile \
  --account-id AC123456 --store-id my-store --profile-id prof-uuid
```

## Command-Line Options

### AWS Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--profile` | `AWS_PROFILE` | - | AWS profile to use |
| `--region` | `AWS_REGION` | `us-east-1` | AWS region |

### Table Configuration

Tables can be configured in two ways:

**Option 1: Environment/Cell-based naming (recommended)**

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--env` | `IDENTITY_ENV` | `dev` | Environment (dev, stage, prod) |
| `--cell` | `IDENTITY_CELL` | `cell-1` | Cell identifier (cell-1, cell-2) |

Tables are auto-constructed as: `{env}-{region}-{cell}.IdentityMappings.v1`

Example: `dev-us-east-1-cell-1.IdentityMappings.v1`

**Option 2: Explicit table names**

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--mappings-table` | `MAPPINGS_TABLE` | - | Full DynamoDB mappings table name (overrides env/cell) |
| `--merges-table` | `MERGES_TABLE` | - | Full DynamoDB merges table name (overrides env/cell) |

### Query Prefill Options

| Flag | Description |
|------|-------------|
| `--mode` | Query mode: `profile` or `identifier` (skips mode selection screen) |
| `--account-id` | Account ID to prefill |
| `--store-id` | Store ID to prefill |
| `--profile-id` | Profile ID to prefill (for profile mode) |
| `--id-type` | Identifier type to prefill (e.g., email, phone) |
| `--id-value` | Identifier value to prefill |

### Other Options

| Flag | Description |
|------|-------------|
| `--version`, `-v` | Show version information |
| `-h`, `--help` | Show help |

## Usage Examples

### Interactive Mode

```bash
# Just launch the TUI with table configuration
identity-explorer --env dev --region us-east-1 --cell cell-1

# Prefill common fields, but still use interactive mode selection
identity-explorer --env dev --region us-east-1 --cell cell-1 \
  --account-id AC123456 --store-id my-store
```

### Direct Query Mode

```bash
# Query profile - all fields prefilled, just press Enter
identity-explorer --env dev --region us-east-1 --cell cell-1 \
  --mode profile \
  --account-id AC123456 \
  --store-id my-store \
  --profile-id mem_profile_01abc123

# Query identifier - find profiles by email
identity-explorer --env dev --region us-east-1 --cell cell-1 \
  --mode identifier \
  --account-id AC123456 \
  --store-id my-store \
  --id-type email \
  --id-value user@example.com
```

### Using Environment Variables

```bash
# Set common configuration once
export AWS_PROFILE=my-aws-profile
export AWS_REGION=us-east-1
export IDENTITY_ENV=dev
export IDENTITY_CELL=cell-1

# Then run queries without repeating flags
identity-explorer --mode profile --account-id AC123 --store-id store1 --profile-id prof-uuid
```

### Using Explicit Table Names

```bash
# For non-standard table naming or cross-account access
identity-explorer \
  --mappings-table custom-IdentityMappings \
  --merges-table custom-IdentityMerges \
  --mode profile \
  --account-id AC123 --store-id store1 --profile-id prof-uuid
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
| `q` | Go back / Quit (from main menu) |
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

# Install locally
make install

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

## Troubleshooting

### Common Errors

**ResourceNotFoundException: Requested resource not found**
- Check that `--env`, `--region`, and `--cell` are correct
- Verify the table exists: `aws dynamodb describe-table --table-name {env}-{region}-{cell}.IdentityMappings.v1`

**AccessDeniedException**
- Verify your AWS profile has DynamoDB read permissions
- Check you're using the correct AWS profile: `--profile your-profile`

**No profiles found for the given identifier**
- The identifier may not exist in the system
- Check the ID type matches exactly (e.g., `email` not `Email`)

### Debug Tips

- Use `--help` to see all available options
- Check which tables are being queried - errors show the full table name and PK
- Verify AWS credentials: `aws sts get-caller-identity --profile your-profile`

## License

MIT License - see [LICENSE](LICENSE) for details.
