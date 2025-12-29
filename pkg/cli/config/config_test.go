package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/cli/config"
)

func TestRSSSourceValidate(t *testing.T) {
	tests := []struct {
		name    string
		src     config.RSSSource
		wantErr bool
	}{
		{
			name: "valid minimal",
			src: config.RSSSource{
				URL: "https://example.com/feed",
			},
			wantErr: false,
		},
		{
			name: "valid with tags and max_articles",
			src: config.RSSSource{
				URL:         "https://example.com/feed",
				RawTags:     []string{"vendor", "google"},
				MaxArticles: 10,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			src: config.RSSSource{
				RawTags: []string{"vendor"},
			},
			wantErr: true,
		},
		{
			name: "invalid tag",
			src: config.RSSSource{
				URL:     "https://example.com/feed",
				RawTags: []string{"-Invalid"},
			},
			wantErr: true,
		},
		{
			name: "negative max_articles",
			src: config.RSSSource{
				URL:         "https://example.com/feed",
				MaxArticles: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.src.Validate()
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}

func TestFeedSourceValidate(t *testing.T) {
	tests := []struct {
		name    string
		src     config.FeedSource
		wantErr bool
	}{
		{
			name: "valid with schema only",
			src: config.FeedSource{
				RawSchema: "abuse_ch_urlhaus",
			},
			wantErr: false,
		},
		{
			name: "valid with custom URL",
			src: config.FeedSource{
				RawSchema: "abuse_ch_urlhaus",
				URL:       "https://mirror.example.com/urlhaus.csv",
			},
			wantErr: false,
		},
		{
			name: "valid with tags and max_items",
			src: config.FeedSource{
				RawSchema: "abuse_ch_threatfox",
				RawTags:   []string{"threat-intel", "hash"},
				MaxItems:  1000,
			},
			wantErr: false,
		},
		{
			name:    "missing schema",
			src:     config.FeedSource{},
			wantErr: true,
		},
		{
			name: "invalid schema",
			src: config.FeedSource{
				RawSchema: "unknown_schema",
			},
			wantErr: true,
		},
		{
			name: "invalid tag",
			src: config.FeedSource{
				RawSchema: "abuse_ch_urlhaus",
				RawTags:   []string{"_Invalid_Tag"},
			},
			wantErr: true,
		},
		{
			name: "negative max_items",
			src: config.FeedSource{
				RawSchema: "abuse_ch_urlhaus",
				MaxItems:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.src.Validate()
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}

func TestFeedSourceGetURL(t *testing.T) {
	t.Run("explicit URL", func(t *testing.T) {
		src := config.FeedSource{
			RawSchema: "abuse_ch_urlhaus",
			URL:       "https://custom.example.com/feed.csv",
		}
		gt.NoError(t, src.Validate())
		gt.S(t, src.GetURL()).Equal("https://custom.example.com/feed.csv")
	})

	t.Run("default URL from schema", func(t *testing.T) {
		src := config.FeedSource{
			RawSchema: "abuse_ch_urlhaus",
		}
		gt.NoError(t, src.Validate())
		// GetURL returns empty string when URL not specified
		// Actual default URLs are defined in feed service
		gt.S(t, src.GetURL()).Equal("")
	})

	t.Run("threatfox default URL", func(t *testing.T) {
		src := config.FeedSource{
			RawSchema: "abuse_ch_threatfox",
		}
		gt.NoError(t, src.Validate())
		// GetURL returns empty string when URL not specified
		// Actual default URLs are defined in feed service
		gt.S(t, src.GetURL()).Equal("")
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("valid config with both RSS and Feed", func(t *testing.T) {
		cfg := &config.Config{
			RSS: map[string]config.RSSSource{
				"google-blog": {
					URL:     "https://security.googleblog.com/feeds/posts/default",
					RawTags: []string{"vendor", "google"},
				},
			},
			Feed: map[string]config.FeedSource{
				"urlhaus": {
					RawSchema: "abuse_ch_urlhaus",
					RawTags:   []string{"threat-intel"},
				},
			},
		}
		gt.NoError(t, cfg.Validate())
	})

	t.Run("duplicate source ID", func(t *testing.T) {
		cfg := &config.Config{
			RSS: map[string]config.RSSSource{
				"test-source": {
					URL: "https://example.com/rss",
				},
			},
			Feed: map[string]config.FeedSource{
				"test-source": {
					RawSchema: "abuse_ch_urlhaus",
				},
			},
		}
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid RSS source", func(t *testing.T) {
		cfg := &config.Config{
			RSS: map[string]config.RSSSource{
				"bad-rss": {
					// Missing URL
					RawTags: []string{"vendor"},
				},
			},
		}
		gt.Error(t, cfg.Validate())
	})

	t.Run("invalid Feed source", func(t *testing.T) {
		cfg := &config.Config{
			Feed: map[string]config.FeedSource{
				"bad-feed": {
					// Missing schema
					RawTags: []string{"threat-intel"},
				},
			},
		}
		gt.Error(t, cfg.Validate())
	})

	t.Run("validated values are persisted in map", func(t *testing.T) {
		cfg := &config.Config{
			RSS: map[string]config.RSSSource{
				"blog": {
					URL:     "https://example.com/feed",
					RawTags: []string{"test", "vendor"},
				},
			},
			Feed: map[string]config.FeedSource{
				"urlhaus": {
					RawSchema: "abuse_ch_urlhaus",
					RawTags:   []string{"threat-intel"},
				},
			},
		}

		gt.NoError(t, cfg.Validate())

		// Check RSS source has typed values
		rssSrc := cfg.RSS["blog"]
		gt.A(t, rssSrc.Tags).Length(2).Describe("RSS tags should be populated")
		gt.S(t, rssSrc.Tags[0].String()).Equal("test").Describe("first RSS tag")
		gt.S(t, rssSrc.Tags[1].String()).Equal("vendor").Describe("second RSS tag")

		// Check Feed source has typed values
		feedSrc := cfg.Feed["urlhaus"]
		gt.S(t, feedSrc.Schema.String()).Equal("abuse_ch_urlhaus").Describe("Feed schema should be populated")
		gt.A(t, feedSrc.Tags).Length(1).Describe("Feed tags should be populated")
		gt.S(t, feedSrc.Tags[0].String()).Equal("threat-intel").Describe("Feed tag")
	})
}
