package config

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Config represents the entire application configuration
type Config struct {
	RSS  map[string]RSSSource  `toml:"rss"`
	Feed map[string]FeedSource `toml:"feed"`
}

// RSSSource represents RSS-specific configuration
type RSSSource struct {
	URL         string     `toml:"url"`
	Tags        types.Tags `toml:"-"` // Not directly unmarshaled
	RawTags     []string   `toml:"tags,omitempty"`
	Disabled    bool       `toml:"disabled,omitempty"`
	MaxArticles int        `toml:"max_articles,omitempty"`
}

// FeedSource represents feed-specific configuration
type FeedSource struct {
	Schema    types.FeedSchema `toml:"-"` // Not directly unmarshaled
	RawSchema string           `toml:"schema"`
	URL       string           `toml:"url,omitempty"` // Optional, defaults to schema's default URL
	Tags      types.Tags       `toml:"-"`             // Not directly unmarshaled
	RawTags   []string         `toml:"tags,omitempty"`
	Disabled  bool             `toml:"disabled,omitempty"`
	MaxItems  int              `toml:"max_items,omitempty"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Check for duplicate source IDs across RSS and Feed
	seenIDs := make(map[string]bool)

	for id, src := range c.RSS {
		if seenIDs[id] {
			return goerr.New("duplicate source ID", goerr.V("id", id))
		}
		seenIDs[id] = true

		if err := src.Validate(); err != nil {
			return goerr.Wrap(err, "invalid RSS source", goerr.V("id", id))
		}
		// Update map with validated values (range gives us a copy, not a reference)
		c.RSS[id] = src
	}

	for id, src := range c.Feed {
		if seenIDs[id] {
			return goerr.New("duplicate source ID", goerr.V("id", id))
		}
		seenIDs[id] = true

		if err := src.Validate(); err != nil {
			return goerr.Wrap(err, "invalid Feed source", goerr.V("id", id))
		}
		// Update map with validated values (range gives us a copy, not a reference)
		c.Feed[id] = src
	}

	return nil
}

// Validate validates RSS source configuration and converts raw values to typed values
func (r *RSSSource) Validate() error {
	// URL required
	if r.URL == "" {
		return goerr.New("url is required")
	}

	// URL must be valid
	if _, err := url.Parse(r.URL); err != nil {
		return goerr.Wrap(err, "invalid url", goerr.V("url", r.URL))
	}

	// Tags validation and conversion
	tags, err := types.NewTags(r.RawTags)
	if err != nil {
		return goerr.Wrap(err, "invalid tags")
	}
	r.Tags = tags

	// MaxArticles must be non-negative
	if r.MaxArticles < 0 {
		return goerr.New("max_articles must be >= 0", goerr.V("max_articles", r.MaxArticles))
	}

	return nil
}

// Validate validates feed source configuration and converts raw values to typed values
func (f *FeedSource) Validate() error {
	// Schema required
	if f.RawSchema == "" {
		return goerr.New("schema is required")
	}

	// Schema must be valid and convert
	feedSchema, err := types.NewFeedSchema(f.RawSchema)
	if err != nil {
		return err
	}
	f.Schema = feedSchema

	// URL validation (if specified)
	if f.URL != "" {
		if _, err := url.Parse(f.URL); err != nil {
			return goerr.Wrap(err, "invalid url", goerr.V("url", f.URL))
		}
	}
	// URL is optional - default URLs are defined in feed service

	// Tags validation and conversion
	tags, err := types.NewTags(f.RawTags)
	if err != nil {
		return goerr.Wrap(err, "invalid tags")
	}
	f.Tags = tags

	// MaxItems must be non-negative
	if f.MaxItems < 0 {
		return goerr.New("max_items must be >= 0", goerr.V("max_items", f.MaxItems))
	}

	return nil
}

// GetURL returns the effective URL (explicit or default)
// This method assumes the config has been validated.
func (f *FeedSource) GetURL() string {
	if f.URL != "" {
		return f.URL
	}

	return f.Schema.DefaultURL()
}

// LoadConfig loads configuration from a TOML file
func LoadConfig(path string) (*Config, error) {
	// Clean the path to prevent directory traversal attacks
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read config file", goerr.V("path", path))
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, goerr.Wrap(err, "failed to parse TOML config", goerr.V("path", path))
	}

	// Validate after loading
	if err := cfg.Validate(); err != nil {
		return nil, goerr.Wrap(err, "config validation failed", goerr.V("path", path))
	}

	return &cfg, nil
}
