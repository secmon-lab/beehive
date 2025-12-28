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
	URL         string   `toml:"url"`
	Tags        []string `toml:"tags,omitempty"`
	Disabled    bool     `toml:"disabled,omitempty"`
	MaxArticles int      `toml:"max_articles,omitempty"`
}

// FeedSource represents feed-specific configuration
type FeedSource struct {
	Schema   string   `toml:"schema"`
	URL      string   `toml:"url,omitempty"` // Optional, defaults to schema's default URL
	Tags     []string `toml:"tags,omitempty"`
	Disabled bool     `toml:"disabled,omitempty"`
	MaxItems int      `toml:"max_items,omitempty"`
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
	}

	for id, src := range c.Feed {
		if seenIDs[id] {
			return goerr.New("duplicate source ID", goerr.V("id", id))
		}
		seenIDs[id] = true

		if err := src.Validate(); err != nil {
			return goerr.Wrap(err, "invalid Feed source", goerr.V("id", id))
		}
	}

	return nil
}

// Validate validates RSS source configuration
func (r *RSSSource) Validate() error {
	// URL required
	if r.URL == "" {
		return goerr.New("url is required")
	}

	// URL must be valid
	if _, err := url.Parse(r.URL); err != nil {
		return goerr.Wrap(err, "invalid url", goerr.V("url", r.URL))
	}

	// Tags validation
	if _, err := types.NewTags(r.Tags); err != nil {
		return goerr.Wrap(err, "invalid tags")
	}

	// MaxArticles must be non-negative
	if r.MaxArticles < 0 {
		return goerr.New("max_articles must be >= 0", goerr.V("max_articles", r.MaxArticles))
	}

	return nil
}

// Validate validates feed source configuration
func (f *FeedSource) Validate() error {
	// Schema required
	if f.Schema == "" {
		return goerr.New("schema is required")
	}

	// Schema must be valid
	feedSchema, err := types.NewFeedSchema(f.Schema)
	if err != nil {
		return err
	}

	// URL validation (if specified)
	if f.URL != "" {
		if _, err := url.Parse(f.URL); err != nil {
			return goerr.Wrap(err, "invalid url", goerr.V("url", f.URL))
		}
	}

	// If URL not specified, ensure schema has a default URL
	if f.URL == "" {
		defaultURL := feedSchema.DefaultURL()
		if defaultURL == "" {
			return goerr.New("no default URL for feed schema", goerr.V("schema", f.Schema))
		}
	}

	// Tags validation
	if _, err := types.NewTags(f.Tags); err != nil {
		return goerr.Wrap(err, "invalid tags")
	}

	// MaxItems must be non-negative
	if f.MaxItems < 0 {
		return goerr.New("max_items must be >= 0", goerr.V("max_items", f.MaxItems))
	}

	return nil
}

// GetURL returns the effective URL (explicit or default)
func (f *FeedSource) GetURL() string {
	if f.URL != "" {
		return f.URL
	}

	feedSchema, _ := types.NewFeedSchema(f.Schema)
	return feedSchema.DefaultURL()
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
