package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kaysush-twilio/identity-explorer/internal/ui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// AWS/DynamoDB flags
	awsProfile := flag.String("profile", "", "AWS profile to use (overrides AWS_PROFILE env var)")
	region := flag.String("region", "", "AWS region to use (overrides AWS_REGION env var)")

	// Table configuration - can use explicit table names or env+cell to auto-construct
	env := flag.String("env", "", "Environment (e.g., dev, stage, prod) - used to construct table names")
	cell := flag.String("cell", "", "Cell identifier (e.g., cell-1, cell-2) - used to construct table names")
	mappingsTable := flag.String("mappings-table", "", "DynamoDB mappings table name (overrides env/cell-based naming)")
	mergesTable := flag.String("merges-table", "", "DynamoDB merges table name (overrides env/cell-based naming)")

	// Query mode and input prefill flags
	mode := flag.String("mode", "", "Query mode: 'profile' or 'identifier' (skips mode selection screen)")
	accountID := flag.String("account-id", "", "Account ID to prefill")
	storeID := flag.String("store-id", "", "Store ID to prefill")
	profileID := flag.String("profile-id", "", "Profile ID to prefill (for profile mode)")
	idType := flag.String("id-type", "", "Identifier type to prefill (for identifier mode, e.g., email, phone)")
	idValue := flag.String("id-value", "", "Identifier value to prefill (for identifier mode)")

	showVersion := flag.Bool("version", false, "Show version information")
	flag.BoolVar(showVersion, "v", false, "Show version information (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Identity Explorer - TUI tool for exploring Identity Service data\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  AWS_PROFILE       AWS profile to use\n")
		fmt.Fprintf(os.Stderr, "  AWS_REGION        AWS region\n")
		fmt.Fprintf(os.Stderr, "  IDENTITY_ENV      Environment (dev, stage, prod)\n")
		fmt.Fprintf(os.Stderr, "  IDENTITY_CELL     Cell identifier (cell-1, cell-2)\n")
		fmt.Fprintf(os.Stderr, "  MAPPINGS_TABLE    DynamoDB mappings table name (explicit override)\n")
		fmt.Fprintf(os.Stderr, "  MERGES_TABLE      DynamoDB merges table name (explicit override)\n")
		fmt.Fprintf(os.Stderr, "\nTable Naming:\n")
		fmt.Fprintf(os.Stderr, "  Tables are constructed as: {env}-{region}-{cell}.IdentityMappings.v1\n")
		fmt.Fprintf(os.Stderr, "  Example: dev-us-east-1-cell-1.IdentityMappings.v1\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Query using env/cell (recommended)\n")
		fmt.Fprintf(os.Stderr, "  %s --profile my-aws-profile --env dev --region us-east-1 --cell cell-1 --mode profile --account-id AC123 --store-id store1 --profile-id prof-uuid\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Query with explicit table names\n")
		fmt.Fprintf(os.Stderr, "  %s --mappings-table prod-us-east-1-cell-1.IdentityMappings.v1 --merges-table prod-us-east-1-cell-1.IdentityMerges.v1\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Query by identifier\n")
		fmt.Fprintf(os.Stderr, "  %s --env dev --region us-east-1 --cell cell-1 --mode identifier --account-id AC123 --store-id store1 --id-type email --id-value user@example.com\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("identity-explorer %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Set AWS environment variables from flags (flags take precedence)
	if *awsProfile != "" {
		os.Setenv("AWS_PROFILE", *awsProfile)
	}
	if *region != "" {
		os.Setenv("AWS_REGION", *region)
	}

	// Resolve environment and cell (flags > env vars)
	resolvedEnv := *env
	if resolvedEnv == "" {
		resolvedEnv = os.Getenv("IDENTITY_ENV")
	}
	resolvedCell := *cell
	if resolvedCell == "" {
		resolvedCell = os.Getenv("IDENTITY_CELL")
	}
	resolvedRegion := *region
	if resolvedRegion == "" {
		resolvedRegion = os.Getenv("AWS_REGION")
	}

	// Construct table names from env/region/cell if not explicitly provided
	if *mappingsTable != "" {
		os.Setenv("MAPPINGS_TABLE", *mappingsTable)
	} else if resolvedEnv != "" && resolvedRegion != "" && resolvedCell != "" {
		tableName := fmt.Sprintf("%s-%s-%s.IdentityMappings.v1", resolvedEnv, resolvedRegion, resolvedCell)
		os.Setenv("MAPPINGS_TABLE", tableName)
	}

	if *mergesTable != "" {
		os.Setenv("MERGES_TABLE", *mergesTable)
	} else if resolvedEnv != "" && resolvedRegion != "" && resolvedCell != "" {
		tableName := fmt.Sprintf("%s-%s-%s.IdentityMerges.v1", resolvedEnv, resolvedRegion, resolvedCell)
		os.Setenv("MERGES_TABLE", tableName)
	}

	// Build UI config from flags
	cfg := ui.Config{
		Mode:      *mode,
		AccountID: *accountID,
		StoreID:   *storeID,
		ProfileID: *profileID,
		IDType:    *idType,
		IDValue:   *idValue,
	}

	p := tea.NewProgram(ui.NewModelWithConfig(cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
