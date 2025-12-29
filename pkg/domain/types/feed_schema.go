package types

import "github.com/m-mizutani/goerr/v2"

// FeedSchema represents a threat intelligence feed schema
// A schema defines:
// - Data format/structure for parsing
// - Parser implementation to use
// - Default fetch URL for the feed
type FeedSchema string

const (
	FeedSchemaAbuseCHURLhaus   FeedSchema = "abuse_ch_urlhaus"
	FeedSchemaAbuseCHThreatFox FeedSchema = "abuse_ch_threatfox"
)

// AllFeedSchemas returns all valid feed schemas
func AllFeedSchemas() []FeedSchema {
	return []FeedSchema{
		FeedSchemaAbuseCHURLhaus,
		FeedSchemaAbuseCHThreatFox,
	}
}

// NewFeedSchema creates a validated feed schema
func NewFeedSchema(s string) (FeedSchema, error) {
	fs := FeedSchema(s)
	for _, valid := range AllFeedSchemas() {
		if fs == valid {
			return fs, nil
		}
	}
	return "", goerr.New("invalid feed schema",
		goerr.V("schema", s),
		goerr.V("valid_schemas", AllFeedSchemas()))
}

// DefaultURL returns the default fetch URL for this schema
func (fs FeedSchema) DefaultURL() string {
	switch fs {
	case FeedSchemaAbuseCHURLhaus:
		return "https://urlhaus.abuse.ch/downloads/csv_recent/"
	case FeedSchemaAbuseCHThreatFox:
		return "https://threatfox.abuse.ch/export/csv/recent/"
	default:
		return ""
	}
}

func (fs FeedSchema) String() string {
	return string(fs)
}
