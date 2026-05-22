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
			if err := appendRunSource(ctx, deps, run, rs, &mu); err != nil {
				errutil.Handle(ctx, goerr.Wrap(err, "append run source (skipped)",
					goerr.V("source_id", src.ID),
					goerr.V("run_id", run.ID)))
			}
			run.Skipped++
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
			return appendRunSource(ctx, deps, run, rs, nil)
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

func appendRunSource(ctx context.Context, deps Deps, run *model.Run, rs *model.RunSource, mu *sync.Mutex) error {
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if deps.Repo != nil {
		return deps.Repo.AppendRunSource(ctx, run.ID, rs)
	}
	run.Sources = append(run.Sources, *rs)
	return nil
}
