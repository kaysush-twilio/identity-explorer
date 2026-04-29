package dynamo

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
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

// QueryMappingsForMultipleProfiles fetches mappings for multiple profiles in parallel
func (c *Client) QueryMappingsForMultipleProfiles(ctx context.Context, storeID string, profileIDs []string) ([]models.Mapping, error) {
	if len(profileIDs) == 0 {
		return nil, nil
	}

	var (
		mu          sync.Mutex
		wg          sync.WaitGroup
		allMappings []models.Mapping
		errors      []error
	)

	for _, pid := range profileIDs {
		wg.Add(1)
		go func(profileID string) {
			defer wg.Done()
			mappings, err := c.QueryMappingsByProfileID(ctx, storeID, profileID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors = append(errors, err)
				return
			}
			allMappings = append(allMappings, mappings...)
		}(pid)
	}

	wg.Wait()

	if len(errors) > 0 {
		return allMappings, fmt.Errorf("some mapping queries failed: %v", errors[0])
	}

	return allMappings, nil
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

// ResolveCanonicalProfileID resolves the canonical profile ID for a given profile.
// It follows the merge chain: if the profile has an NC# item, it follows CPID until it finds a C# item.
func (c *Client) ResolveCanonicalProfileID(ctx context.Context, accountSID, storeID, profileID string) (string, error) {
	currentID := profileID

	for {
		// Check if this is a canonical profile (C# lookup)
		canonicalLink, err := c.QueryCanonicalLink(ctx, accountSID, storeID, currentID)
		if err != nil {
			return "", err
		}
		if canonicalLink != nil {
			// Found the canonical profile
			return currentID, nil
		}

		// Not a canonical, check if it was merged (NC# lookup)
		merges, err := c.QueryMergesByProfileID(ctx, accountSID, storeID, currentID)
		if err != nil {
			return "", err
		}
		if len(merges) == 0 {
			// No merge record found - this profile doesn't exist or is standalone
			// Return the original profileID as canonical
			return profileID, nil
		}

		// Follow the merge chain to the canonical
		if merges[0].CanonicalProfileID == "" {
			return profileID, nil
		}
		currentID = merges[0].CanonicalProfileID
	}
}

// ShardCountResult represents the result of counting profiles for one shard
type ShardCountResult struct {
	Shard string
	Count int
	Error error
}

// CountProfilesInStoreByShard queries the GSI on Merges table for a specific shard value
// and returns the count of profiles. The storeIDValue should be either "storeId" or "storeId#N"
func (c *Client) CountProfilesInStoreByShard(ctx context.Context, storeIDValue string) (int, error) {
	count := 0
	var lastEvaluatedKey map[string]types.AttributeValue

	for {
		input := &dynamodb.QueryInput{
			TableName:              aws.String(c.mergesTable),
			IndexName:              aws.String("StoreID-SK-Index"),
			KeyConditionExpression: aws.String("StoreID = :sid"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":sid": &types.AttributeValueMemberS{Value: storeIDValue},
			},
			Select:            types.SelectCount,
			ExclusiveStartKey: lastEvaluatedKey,
		}

		result, err := c.db.Query(ctx, input)
		if err != nil {
			return count, fmt.Errorf("failed to query GSI for StoreID='%s': %w", storeIDValue, err)
		}

		count += int(result.Count)
		lastEvaluatedKey = result.LastEvaluatedKey

		if lastEvaluatedKey == nil {
			break
		}

		// Small delay between pagination requests to avoid throttling
		time.Sleep(50 * time.Millisecond)
	}

	return count, nil
}

// CountProfilesInStoreWithCallback queries all shards in parallel and calls the callback for each result
// The callback receives intermediate results as they complete
func (c *Client) CountProfilesInStoreWithCallback(ctx context.Context, storeID string, callback func(ShardCountResult)) int {
	// Generate all shard values: storeId, storeId#0, storeId#1, ..., storeId#9
	shards := make([]string, 11)
	shards[0] = storeID
	for i := 0; i < 10; i++ {
		shards[i+1] = fmt.Sprintf("%s#%d", storeID, i)
	}

	var wg sync.WaitGroup
	resultChan := make(chan ShardCountResult, 11)

	// Limit concurrency to avoid hitting DynamoDB too hard
	semaphore := make(chan struct{}, 3) // Max 3 concurrent queries

	for _, shard := range shards {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			count, err := c.CountProfilesInStoreByShard(ctx, s)
			resultChan <- ShardCountResult{
				Shard: s,
				Count: count,
				Error: err,
			}
		}(shard)
	}

	// Close channel when all queries complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results and call callback
	totalCount := 0
	for result := range resultChan {
		if callback != nil {
			callback(result)
		}
		if result.Error == nil {
			totalCount += result.Count
		}
	}

	return totalCount
}

// QueryAllMergesForProfile fetches all merge-related data for a profile
// Flow:
// 1. Resolve the canonical profile ID for the input profile
// 2. Query C# item to get list of all merged profile IDs
// 3. For each merged profile ID, query NC# item to get merge details
func (c *Client) QueryAllMergesForProfile(ctx context.Context, accountSID, storeID, profileID string) ([]models.Merge, *models.CanonicalLink, error) {
	// Step 1: Resolve to canonical profile ID
	canonicalID, err := c.ResolveCanonicalProfileID(ctx, accountSID, storeID, profileID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve canonical profile ID: %w", err)
	}

	// Step 2: Query C# item to get canonical link with all merged profiles
	canonicalLink, err := c.QueryCanonicalLink(ctx, accountSID, storeID, canonicalID)
	if err != nil {
		return nil, nil, err
	}

	if canonicalLink == nil {
		// No canonical link exists - profile has no merges
		return nil, nil, nil
	}

	// Step 3: For each merged profile ID, query NC# item to get merge details
	var allMerges []models.Merge
	for _, mergedProfileID := range canonicalLink.MergedProfileIDs {
		if mergedProfileID == "" {
			continue
		}
		mergeItems, err := c.QueryMergesByProfileID(ctx, accountSID, storeID, mergedProfileID)
		if err != nil {
			continue // Don't fail on individual lookups
		}
		allMerges = append(allMerges, mergeItems...)
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
