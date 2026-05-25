package usecase

import (
	"context"
	"log/slog"
	"slices"
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
// FetchAllOption tunes the behaviour of FetchAll. The async-mode HTTP
// handler uses WithRunID so the caller can pre-create the Run document
// and return its id to the client before the work actually starts.
type FetchAllOption func(*fetchAllOpts)

type fetchAllOpts struct {
	runID types.RunID
}

// WithRunID skips the automatic Run creation inside FetchAll and uses
// the supplied id instead. The caller MUST have already created the
// Run document in the repository (status=running, StartedAt set) so
// that the rest of the pipeline can Update it.
func WithRunID(runID types.RunID) FetchAllOption {
	return func(o *fetchAllOpts) { o.runID = runID }
}

func FetchAll(ctx context.Context, deps Deps, trigger string, forceSource types.SourceID, opts ...FetchAllOption) (*model.Run, error) {
	if deps.Catalog == nil {
		return nil, goerr.New("catalog not initialised")
	}

	o := &fetchAllOpts{}
	for _, f := range opts {
		f(o)
	}

	var run *model.Run
	if o.runID != "" {
		// Async-mode: the handler already created the Run document and
		// returned its id to the client. Load it so subsequent updates
		// reuse the same record (the alternative — a second CreateRun
		// — would fail with AlreadyExists on memory and overwrite on
		// Firestore).
		if deps.Repo == nil {
			return nil, goerr.New("WithRunID requires a repository")
		}
		existing, err := deps.Repo.GetRun(ctx, o.runID)
		if err != nil {
			return nil, goerr.Wrap(err, "load preallocated run",
				goerr.V("run_id", o.runID))
		}
		run = existing
	} else {
		run = &model.Run{
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
	}

	logger := logging.From(ctx)
	logger.LogAttrs(ctx, slog.LevelInfo, "fetch all: start",
		slog.String("run_id", string(run.ID)),
		slog.String("trigger", trigger),
		slog.Int("catalog_size", len(deps.Catalog.Sources)),
	)

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
			if !slices.Contains(run.SourceIDs, src.ID) {
				run.SourceIDs = append(run.SourceIDs, src.ID)
			}
			run.Skipped++
			mu.Unlock()
			continue
		}

		g.Go(func() error {
			logger := logging.From(gctx)
			logger.LogAttrs(gctx, slog.LevelDebug, "source: start",
				slog.String("source_id", string(src.ID)),
				slog.String("kind", string(src.Kind)),
				slog.String("type", src.Type),
				slog.String("run_id", string(run.ID)),
			)
			start := time.Now()
			rs, ferr := FetchSource(gctx, deps, src, run.ID, force)
			if ferr != nil {
				// FetchSource normally tucks failures into rs.ErrorMessage
				// and returns (rs, nil). A non-nil ferr means the failure
				// happened before rs was populated — surface it so the
				// operator does not have to read the Run doc to find out.
				errutil.Handle(gctx, goerr.Wrap(ferr, "fetch source",
					goerr.V("source_id", src.ID),
					goerr.V("run_id", run.ID)))
			}
			if rs == nil {
				return nil
			}
			logger.LogAttrs(gctx, slog.LevelDebug, "source: end",
				slog.String("source_id", string(src.ID)),
				slog.String("status", string(rs.Status)),
				slog.Int("ioc_count", rs.IoCCount),
				slog.Int("article_count", rs.ArticleCount),
				slog.String("error_message", rs.ErrorMessage),
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
			if !slices.Contains(run.SourceIDs, rs.SourceID) {
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
		// Refresh the per-type IoC counters so the operator UI sees an
		// up-to-date stats strip on the next page view. Best-effort: a
		// counter refresh failure must not roll back the fetch (the
		// counts will be reconciled on the next successful run).
		if _, err := RefreshIoCCounts(ctx, deps.Repo); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "refresh ioc counts",
				goerr.V("run_id", run.ID)))
		}
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "fetch all: end",
		slog.String("run_id", string(run.ID)),
		slog.String("status", string(run.Status)),
		slog.Int("total", run.Total),
		slog.Int("triggered", run.Triggered),
		slog.Int("skipped", run.Skipped),
		slog.Int("failed", run.Failed),
		slog.Duration("elapsed", run.FinishedAt.Sub(run.StartedAt)),
	)
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
