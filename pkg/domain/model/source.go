package model

import "time"

// SourceType represents the type of source
type SourceType string

const (
	SourceTypeRSS  SourceType = "rss"
	SourceTypeFeed SourceType = "feed"
)

// SourcesConfig represents the entire sources configuration
type SourcesConfig struct {
	Sources map[string]Source // key = source ID (from TOML section name)
}

// Source represents a single source configuration
type Source struct {
	Type       SourceType  `toml:"type"`
	URL        string      `toml:"url"`
	Tags       []string    `toml:"tags"`
	Enabled    bool        `toml:"enabled"`
	RSSConfig  *RSSConfig  `toml:"rss_config,omitempty"`  // Only for type="rss"
	FeedConfig *FeedConfig `toml:"feed_config,omitempty"` // Only for type="feed"
}

// RSSConfig contains RSS-specific configuration
type RSSConfig struct {
	MaxArticles int `toml:"max_articles"` // Maximum articles to fetch per run
}

// FeedConfig contains feed-specific configuration
type FeedConfig struct {
	Schema   string `toml:"schema"`    // Schema name that identifies the parser implementation
	MaxItems int    `toml:"max_items"` // Maximum items to fetch per run (0 = unlimited)
}

// SourceState represents the state of a source
type SourceState struct {
	SourceID      string    `firestore:"source_id"`
	LastFetchedAt time.Time `firestore:"last_fetched_at"`
	LastItemID    string    `firestore:"last_item_id"`   // For RSS: last GUID, for feeds: last entry ID
	LastItemDate  time.Time `firestore:"last_item_date"` // Last item's published date
	ItemCount     int64     `firestore:"item_count"`     // Total items processed
	ErrorCount    int64     `firestore:"error_count"`    // Error count
	LastError     string    `firestore:"last_error"`     // Last error message
	UpdatedAt     time.Time `firestore:"updated_at"`
}
