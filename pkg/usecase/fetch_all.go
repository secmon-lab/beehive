package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/id"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"golang.org/x/sync/errgroup"
)

// FetchAllConcurrency is the default `errgroup.SetLimit` value. Kept as
// a package var so tests can shrink it.
var FetchAllConcurrency = 8

// FetchAll walks the catalog, runs `isDue` to decide which sources to
// touch, and processes the due ones concurrently. Skipped sources are
// recorded with their reason so the operator can audit a quiet run.
//
// trigger is one of "api" / "manual" / "cli" and is stored on the Run.
func FetchAll(ctx context.Context, deps Deps, trigger string, forceSource types.SourceID) (*model.Run, error) {
	if deps.Catalog == nil {
		return nil, goerr.New("catalog not initialised")
	}

	run := &model.Run{
		ID:        types.RunID("run-" + id.NewULID()),
		Trigger:   trigger,
		Status:    types.RunStatusRunning,
		StartedAt: time.Now().UTC(),
	}
	if deps.Repo != nil {
		if err := deps.Repo.CreateRun(ctx, run); err != nil {
			return nil, goerr.Wrap(err, "create run")
		}
	}

	now := time.Now().UTC()
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(FetchAllConcurrency)

	for _, src := range deps.Catalog.Sources {
		run.Total++

		var state *model.SourceState
		if deps.Repo != nil {
			s, err := deps.Repo.GetSourceState(gctx, src.ID)
			switch {
			case err == nil:
				state = s
			case errutil.IsNotFound(err):
				// First-time fetch — no state row yet.
			default:
				// A real backend failure (Firestore unavailable, RPC
				// errors, …) MUST abort the whole run, not be quietly
				// treated as "no state". Returning the error here lets
				// the HTTP layer surface it and log via errutil.Handle.
				return nil, goerr.Wrap(err, "load source state for due check",
					goerr.V("source_id", src.ID),
					goerr.V("run_id", run.ID))
			}
		}

		force := forceSource != "" && forceSource == src.ID
		if !force && !isDue(src, state, now) {
			rs := &model.RunSource{
				SourceID:   src.ID,
				SourceKind: src.Kind,
				Status:     types.RunStatusSkipped,
				SkipReason: skipReason(src, state),
			}
			// Skipped sources are accumulated in memory and flushed via
			// a single UpdateRun at the end. Calling AppendRunSource
			// per skip would be one Firestore transaction per skip —
			// expensive and contention-heavy on the single Run doc (see
			// CLAUDE.md §3 on write-cost discipline).
			mu.Lock()
			run.Sources = append(run.Sources, *rs)
			if !containsSourceID(run.SourceIDs, src.ID) {
				run.SourceIDs = append(run.SourceIDs, src.ID)
			}
			run.Skipped++
			mu.Unlock()
			continue
		}

		g.Go(func() error {
			logger := logging.From(gctx)
			logger.LogAttrs(gctx, slog.LevelInfo, "source: start",
				slog.String("source_id", string(src.ID)),
				slog.String("kind", string(src.Kind)),
				slog.String("type", src.Type),
				slog.String("run_id", string(run.ID)),
			)
			start := time.Now()
			rs, _ := FetchSource(gctx, deps, src, run.ID, force)
			if rs == nil {
				return nil
			}
			logger.LogAttrs(gctx, slog.LevelInfo, "source: end",
				slog.String("source_id", string(src.ID)),
				slog.String("status", string(rs.Status)),
				slog.Int("ioc_count", rs.IoCCount),
				slog.Int("article_count", rs.ArticleCount),
				slog.Duration("elapsed", time.Since(start)),
			)
			mu.Lock()
			defer mu.Unlock()
			switch rs.Status {
			case types.RunStatusSkipped:
				run.Skipped++
			case types.RunStatusFailed:
				run.Failed++
			default:
				run.Triggered++
			}
			// Same coalescing rule as the skip branch — keep everything
			// in memory and persist once at the end.
			run.Sources = append(run.Sources, *rs)
			if !containsSourceID(run.SourceIDs, rs.SourceID) {
				run.SourceIDs = append(run.SourceIDs, rs.SourceID)
			}
			return nil
		})
	}
	_ = g.Wait()

	run.FinishedAt = time.Now().UTC()
	run.DurationMs = run.FinishedAt.Sub(run.StartedAt).Milliseconds()
	run.Status = overallStatus(run)
	if deps.Repo != nil {
		if err := deps.Repo.UpdateRun(ctx, run); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "update run",
				goerr.V("run_id", run.ID)))
		}
	}
	return run, nil
}

func containsSourceID(ids []types.SourceID, id types.SourceID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func isDue(src *model.Source, state *model.SourceState, now time.Time) bool {
	if !src.EffectiveEnabled(state) {
		return false
	}
	if state == nil || state.LastFetchedAt.IsZero() {
		return true
	}
	return state.LastFetchedAt.Add(src.Interval).Before(now)
}

func skipReason(src *model.Source, state *model.SourceState) types.SkipReason {
	if !src.EffectiveEnabled(state) {
		return types.SkipReasonDisabled
	}
	return types.SkipReasonNotDue
}

func overallStatus(run *model.Run) types.RunStatus {
	if run.Failed == 0 {
		return types.RunStatusSuccess
	}
	if run.Triggered > 0 {
		return types.RunStatusPartial
	}
	return types.RunStatusFailed
}
