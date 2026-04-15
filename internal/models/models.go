package models

import "time"

// Mapping represents an identifier-to-profile association
type Mapping struct {
	MappingID  string
	StoreID    string
	MessageID  string
	ProfileID  string
	IDType     string
	IDValue    string
	CreatedAt  time.Time
	EventAt    time.Time
	IsUnique   bool
}

// Merge represents a profile merge record
type Merge struct {
	MergeID            string
	MergeFrom          string
	MergeTo            string
	CanonicalProfileID string
	MessageID          string
	Reason             string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CanonicalLink represents the canonical profile with its merged profiles
type CanonicalLink struct {
	CanonicalProfileID string
	MergedProfileIDs   []string
	StoreID            string
	AccountSID         string
}

// QueryResult holds the combined result of mappings and merges queries
type QueryResult struct {
	ProfileID      string
	Mappings       []Mapping
	Merges         []Merge
	CanonicalLink  *CanonicalLink
	Error          error
}
