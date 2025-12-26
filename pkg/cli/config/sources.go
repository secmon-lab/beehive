package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/goerr/v2"
)

// Sources represents the complete source configuration
type Sources struct {
	Sources map[string]SourceConfig `toml:"sources"`
}

// SourceConfig represents a single source configuration
type SourceConfig struct {
	Type       string      `toml:"type"`
	URL        string      `toml:"url"`
	Tags       []string    `toml:"tags"`
	Enabled    bool        `toml:"enabled"`
	RSSConfig  *RSSConfig  `toml:"rss_config,omitempty"`
	FeedConfig *FeedConfig `toml:"feed_config,omitempty"`
}

// RSSConfig represents RSS-specific configuration
type RSSConfig struct {
	MaxArticles int `toml:"max_articles"`
}

// FeedConfig represents feed-specific configuration
type FeedConfig struct {
	Schema   string `toml:"schema"`
	MaxItems int    `toml:"max_items"`
}

// LoadSources loads source configuration from a TOML file
func LoadSources(path string) (*Sources, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read config file", goerr.V("path", path))
	}

	var sources Sources
	if err := toml.Unmarshal(data, &sources); err != nil {
		return nil, goerr.Wrap(err, "failed to parse TOML config", goerr.V("path", path))
	}

	return &sources, nil
}
