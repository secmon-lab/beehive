package cli

import (
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/source/feed"
	"github.com/secmon-lab/beehive/pkg/domain/source/rss"
)

// createSources creates Source instances from configuration
func createSources(
	cfg *config.Config,
	iocRepo interfaces.IoCRepository,
	rssStateRepo rss.RSSStateRepository,
	feedStateRepo feed.FeedStateRepository,
	llmClient gollem.LLMClient,
) []interfaces.Source {
	var sources []interfaces.Source

	// Create RSS sources
	for id, rssCfg := range cfg.RSS {
		if rssCfg.Disabled {
			continue
		}
		src, err := rss.New(id, rssCfg, iocRepo, rssStateRepo, llmClient)
		if err != nil {
			// Log error and skip this source
			// In production, you might want to collect and report errors
			continue
		}
		sources = append(sources, src)
	}

	// Create Feed sources
	for id, feedCfg := range cfg.Feed {
		if feedCfg.Disabled {
			continue
		}
		src, err := feed.New(id, feedCfg, iocRepo, feedStateRepo, llmClient)
		if err != nil {
			// Log error and skip this source
			continue
		}
		sources = append(sources, src)
	}

	return sources
}

// filterByTags filters sources by tags
func filterByTags(sources []interfaces.Source, tags []string) []interfaces.Source {
	if len(tags) == 0 {
		return sources
	}

	var filtered []interfaces.Source
	for _, src := range sources {
		if hasAnyTag(src.Tags(), tags) {
			filtered = append(filtered, src)
		}
	}
	return filtered
}

// hasAnyTag checks if slice a contains any element from slice b
func hasAnyTag(sourceTags, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, sourceTag := range sourceTags {
			if sourceTag == filterTag {
				return true
			}
		}
	}
	return false
}
