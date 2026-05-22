package source_catalog

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// DetectOrphanStates compares the in-memory catalog against persisted
// state documents and reports any SourceID that has state in Firestore
// but no longer appears in TOML. We do NOT delete the state — it stays
// around as historical evidence and shows up in the UI with an
// `archived` badge. Use a one-off `prune-state` CLI to clean up.
//
// Returns the list of orphan SourceIDs so callers can render them in
// the UI; it also writes a warn-level log entry per orphan via the
// context-scoped logger.
func DetectOrphanStates(ctx context.Context, repo interfaces.StateRepository, cat *Catalog) ([]types.SourceID, error) {
	states, err := repo.ListSourceStates(ctx)
	if err != nil {
		return nil, err
	}
	known := map[types.SourceID]struct{}{}
	for _, s := range cat.Sources {
		known[s.ID] = struct{}{}
	}
	logger := logging.From(ctx)

	var orphans []types.SourceID
	for _, st := range states {
		if _, ok := known[st.ID]; ok {
			continue
		}
		orphans = append(orphans, st.ID)
		logger.LogAttrs(ctx, slog.LevelWarn, "orphan source state",
			slog.String("source_id", string(st.ID)),
			slog.Time("last_fetched_at", st.LastFetchedAt),
		)
	}
	return orphans, nil
}
