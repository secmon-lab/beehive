package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/normalize"
	"golang.org/x/sync/errgroup"
)

// LookupIoC normalises (type, value) and returns the matching IoC if
// any. Returns a tagged not-found error when nothing matches, so the
// HTTP layer can map it to 404.
func LookupIoC(ctx context.Context, repo interfaces.IoCRepository, t types.IoCType, value string) (*model.IoC, error) {
	v, err := normalize.IoC(t, value)
	if err != nil {
		return nil, err
	}
	id := model.ComputeIoCID(t, v)
	return repo.GetIoC(ctx, id)
}

// IoCListLimits caps how many IoCs the UI list endpoint can request.
const (
	IoCListDefaultLimit = 100
	IoCListMaxLimit     = 500
)

// ListRecentIoCs is a thin wrapper around the repository call so the
// HTTP handler depends on the usecase layer rather than reaching into
// the repository directly. The limit is clamped to [1, IoCListMaxLimit]
// — bulk analytics still belong in BigQuery (CLAUDE.md §0).
func ListRecentIoCs(ctx context.Context, repo interfaces.IoCRepository, limit int) ([]*model.IoC, error) {
	return ListRecentIoCsAfter(ctx, repo, limit, nil)
}

// ListRecentIoCsAfter pages over the recent-IoCs list using a
// LastSeenAt cursor. Pass nil to start at the head.
func ListRecentIoCsAfter(ctx context.Context, repo interfaces.IoCRepository, limit int, after *time.Time) ([]*model.IoC, error) {
	if limit <= 0 {
		limit = IoCListDefaultLimit
	}
	if limit > IoCListMaxLimit {
		limit = IoCListMaxLimit
	}
	return repo.ListRecentIoCsAfter(ctx, limit, after)
}

// GetIoCCounts returns the precomputed counter document. The Firestore
// backend keeps this in metrics/ioc_counts, written by
// RefreshIoCCounts at the end of every fetch run — so this call is
// O(1) (one doc read) regardless of collection size.
func GetIoCCounts(ctx context.Context, repo interfaces.IoCRepository) (*model.IoCCounts, error) {
	return repo.GetIoCCounts(ctx)
}

// RefreshIoCCounts fans out one aggregation count() query per IoC type
// (concurrently via errgroup), assembles the result into an
// IoCCounts, and persists it via SaveIoCCounts. The repository
// contract intentionally keeps fan-out OUT of the repo layer so the
// implementation strategy (sequential, concurrent, future cache) lives
// in one place.
//
// Called from the tail of every fetch run (best-effort: a refresh
// failure does not roll back the fetch). Readers go through
// GetIoCCounts and never trigger this path.
func RefreshIoCCounts(ctx context.Context, repo interfaces.IoCRepository) (*model.IoCCounts, error) {
	out := &model.IoCCounts{
		ByType:    make(map[types.IoCType]int64, len(types.AllIoCTypes())),
		UpdatedAt: time.Now().UTC(),
	}
	for _, t := range types.AllIoCTypes() {
		out.ByType[t] = 0
	}

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	for _, t := range types.AllIoCTypes() {
		t := t
		g.Go(func() error {
			n, err := repo.CountIoCsOfType(gctx, t)
			if err != nil {
				return goerr.Wrap(err, "count ioc of type", goerr.V("type", t))
			}
			mu.Lock()
			out.ByType[t] = n
			out.Total += n
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	if err := repo.SaveIoCCounts(ctx, out); err != nil {
		return nil, goerr.Wrap(err, "save ioc counts")
	}
	return out, nil
}
