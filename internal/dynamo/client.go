package dynamo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/kaysush-twilio/identity-explorer/internal/models"
)

// Client wraps the DynamoDB client with identity-specific operations
type Client struct {
	db             *dynamodb.Client
	mappingsTable  string
	mergesTable    string
}

// NewClient creates a new DynamoDB client for identity operations
func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	mappingsTable := os.Getenv("MAPPINGS_TABLE")
	if mappingsTable == "" {
		mappingsTable = "local.IdentityMappings.v1"
	}

	mergesTable := os.Getenv("MERGES_TABLE")
	if mergesTable == "" {
		mergesTable = "local.IdentityMerges.v1"
	}

	client := dynamodb.NewFromConfig(cfg)

	return &Client{
		db:            client,
		mappingsTable: mappingsTable,
		mergesTable:   mergesTable,
	}, nil
}

// GetMappingsTableName returns the configured mappings table name
func (c *Client) GetMappingsTableName() string {
	return c.mappingsTable
}

// GetMergesTableName returns the configured merges table name
func (c *Client) GetMergesTableName() string {
	return c.mergesTable
}

// QueryMappingsByProfileID fetches all mappings for a given profile
func (c *Client) QueryMappingsByProfileID(ctx context.Context, storeID, profileID string) ([]models.Mapping, error) {
	pk := fmt.Sprintf("%s#%s", profileID, storeID)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(c.mappingsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
	}

	result, err := c.db.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query mappings table '%s' with PK='%s': %w", c.mappingsTable, pk, err)
	}

	var mappings []models.Mapping
	for _, item := range result.Items {
		mapping := models.Mapping{
			StoreID:   storeID,
			ProfileID: profileID,
		}

		if v, ok := item["SK"].(*types.AttributeValueMemberS); ok {
			mapping.MappingID = v.Value
			mapping.IsUnique = strings.HasPrefix(v.Value, "UNIQUE#")
		}
		if v, ok := item["Type"].(*types.AttributeValueMemberS); ok {
			mapping.IDType = v.Value
		}
		if v, ok := item["Value"].(*types.AttributeValueMemberS); ok {
			mapping.IDValue = v.Value
		}
		if v, ok := item["MID"].(*types.AttributeValueMemberS); ok {
			mapping.MessageID = v.Value
		}
		if v, ok := item["CreatedAt"].(*types.AttributeValueMemberS); ok {
			if t, err := time.Parse(time.RFC3339, v.Value); err == nil {
				mapping.CreatedAt = t
			}
		}
		if v, ok := item["EventAt"].(*types.AttributeValueMemberS); ok {
			if t, err := time.Parse(time.RFC3339, v.Value); err == nil {
				mapping.EventAt = t
			}
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// QueryProfileIDsByIdentifier finds profile IDs for a given identifier
func (c *Client) QueryProfileIDsByIdentifier(ctx context.Context, storeID, idType, idValue string) ([]string, error) {
	pk := fmt.Sprintf("%s#%s#%s", idValue, idType, storeID)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(c.mappingsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
	}

	result, err := c.db.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query profiles by identifier from table '%s' with PK='%s': %w", c.mappingsTable, pk, err)
	}

	profileIDs := make(map[string]struct{})
	for _, item := range result.Items {
		if v, ok := item["PID"].(*types.AttributeValueMemberS); ok {
			profileIDs[v.Value] = struct{}{}
		}
	}

	var ids []string
	for id := range profileIDs {
		ids = append(ids, id)
	}

	return ids, nil
}

// QueryMergesByProfileID fetches merge records for a profile (non-canonical lookup)
func (c *Client) QueryMergesByProfileID(ctx context.Context, accountSID, storeID, profileID string) ([]models.Merge, error) {
	pk := fmt.Sprintf("NC#%s#%s#%s", accountSID, storeID, profileID)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(c.mergesTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
	}

	result, err := c.db.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query merges from table '%s' with PK='%s': %w", c.mergesTable, pk, err)
	}

	var merges []models.Merge
	for _, item := range result.Items {
		merge := models.Merge{
			MergeFrom: profileID,
		}

		if v, ok := item["SK"].(*types.AttributeValueMemberS); ok {
			merge.MergeID = v.Value
		}
		if v, ok := item["To"].(*types.AttributeValueMemberS); ok {
			merge.MergeTo = v.Value
		}
		if v, ok := item["CPID"].(*types.AttributeValueMemberS); ok {
			merge.CanonicalProfileID = v.Value
		}
		if v, ok := item["MID"].(*types.AttributeValueMemberS); ok {
			merge.MessageID = v.Value
		}
		if v, ok := item["Reason"].(*types.AttributeValueMemberS); ok {
			merge.Reason = v.Value
		}
		if v, ok := item["CreatedAt"].(*types.AttributeValueMemberS); ok {
			if t, err := time.Parse(time.RFC3339, v.Value); err == nil {
				merge.CreatedAt = t
			}
		}
		if v, ok := item["UpdatedAt"].(*types.AttributeValueMemberS); ok {
			if t, err := time.Parse(time.RFC3339, v.Value); err == nil {
				merge.UpdatedAt = t
			}
		}

		merges = append(merges, merge)
	}

	return merges, nil
}

// QueryCanonicalLink fetches the canonical link for a profile
func (c *Client) QueryCanonicalLink(ctx context.Context, accountSID, storeID, canonicalProfileID string) (*models.CanonicalLink, error) {
	pk := fmt.Sprintf("C#%s#%s#%s", accountSID, storeID, canonicalProfileID)

	input := &dynamodb.QueryInput{
		TableName:              aws.String(c.mergesTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk": &types.AttributeValueMemberS{Value: pk},
		},
	}

	result, err := c.db.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query canonical link from table '%s' with PK='%s': %w", c.mergesTable, pk, err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	item := result.Items[0]
	link := &models.CanonicalLink{
		CanonicalProfileID: canonicalProfileID,
		StoreID:            storeID,
		AccountSID:         accountSID,
	}

	if v, ok := item["Merges"].(*types.AttributeValueMemberSS); ok {
		link.MergedProfileIDs = v.Value
	}

	return link, nil
}

// QueryAllMergesForProfile fetches all merge-related data for a profile
// This includes both NC# (non-canonical) and C# (canonical) queries
func (c *Client) QueryAllMergesForProfile(ctx context.Context, accountSID, storeID, profileID string) ([]models.Merge, *models.CanonicalLink, error) {
	var allMerges []models.Merge

	// First, check if this profile has been merged into another (NC# lookup for this profile)
	merges, err := c.QueryMergesByProfileID(ctx, accountSID, storeID, profileID)
	if err != nil {
		return nil, nil, err
	}
	allMerges = append(allMerges, merges...)

	// Check if this profile is a canonical profile (C# lookup)
	canonicalLink, err := c.QueryCanonicalLink(ctx, accountSID, storeID, profileID)
	if err != nil {
		return nil, nil, err
	}

	// If we found a merge record pointing to a different canonical, look up that canonical link
	if len(merges) > 0 && merges[0].CanonicalProfileID != "" && merges[0].CanonicalProfileID != profileID {
		targetLink, err := c.QueryCanonicalLink(ctx, accountSID, storeID, merges[0].CanonicalProfileID)
		if err == nil && targetLink != nil {
			canonicalLink = targetLink
		}
	}

	// If we have a canonical link, query the merge item (NC#) for each merged profile ID
	// This gives us the detailed merge records (MergeFrom, MergeTo, Reason, etc.)
	if canonicalLink != nil && len(canonicalLink.MergedProfileIDs) > 0 {
		for _, mergedProfileID := range canonicalLink.MergedProfileIDs {
			// Query the NC# item for this merged profile
			mergeItems, err := c.QueryMergesByProfileID(ctx, accountSID, storeID, mergedProfileID)
			if err != nil {
				continue // Don't fail on individual lookups
			}
			allMerges = append(allMerges, mergeItems...)
		}
	}

	// Deduplicate merges by MergeID
	seen := make(map[string]bool)
	var uniqueMerges []models.Merge
	for _, m := range allMerges {
		if !seen[m.MergeID] {
			seen[m.MergeID] = true
			uniqueMerges = append(uniqueMerges, m)
		}
	}

	return uniqueMerges, canonicalLink, nil
}
