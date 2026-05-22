// Package providers is the wiring layer that turns concrete provider
// constructors into a populated *fetcher.Registry.
//
// It exists so the rest of the codebase has no need for blank imports
// or init()-time side effects: callers ask for `providers.All(...)` and
// receive a ready-to-use Registry. Adding a new provider means editing
// exactly one file — this one.
package providers

import (
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/blog"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

// All builds the canonical *fetcher.Registry containing every provider
// shipped with beehive. The HTTP client is shared so SSRF protection
// and timeout settings stay consistent across providers.
//
// Returns an error rather than panicking so the caller can surface
// duplicate-typeID problems through normal error paths (the registry
// itself flags duplicates).
func All(http *fetcher.HTTPClient) (*fetcher.Registry, error) {
	if http == nil {
		http = fetcher.NewHTTPClient()
	}
	reg := fetcher.NewRegistry()

	entries := []struct {
		typeID string
		prov   interfaces.Provider
	}{
		{blog.RSSTypeID, blog.New(http)},
		{feed.AbuseIPDBTypeID, feed.NewAbuseIPDB(http, os.Getenv(feed.AbuseIPDBAPIKeyEnv))},
		{feed.CINSScoreTypeID, feed.NewCINSScore(http)},
		{feed.FeodoTrackerTypeID, feed.NewFeodoTracker(http)},
		{feed.URLhausTypeID, feed.NewURLhaus(http)},
		{feed.ThreatFoxTypeID, feed.NewThreatFox(http)},
	}

	for _, e := range entries {
		if err := reg.Register(e.typeID, e.prov); err != nil {
			return nil, goerr.Wrap(err, "register provider",
				goerr.V("type", e.typeID))
		}
	}

	return reg, nil
}
