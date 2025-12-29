package usecase

import (
	"context"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// FetchAllSources fetches IoCs from all provided sources
func FetchAllSources(ctx context.Context, sources []interfaces.Source) ([]*interfaces.FetchStats, error) {
	logger := logging.From(ctx)
	var allStats []*interfaces.FetchStats

	for _, src := range sources {
		if !src.Enabled() {
			logger.Info("skipping disabled source", "id", src.ID())
			continue
		}

		logger.Info("fetching from source", "id", src.ID(), "type", src.Type())

		stats, err := src.Fetch(ctx)
		if err != nil {
			logger.Error("failed to fetch from source",
				"id", src.ID(),
				"type", src.Type(),
				"error", err)
			// Create error stats
			stats = &interfaces.FetchStats{
				SourceID:   src.ID(),
				SourceType: src.Type(),
				Errors:     1,
			}
		}

		allStats = append(allStats, stats)
	}

	return allStats, nil
}
