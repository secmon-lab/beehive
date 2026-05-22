package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/id"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// FetchSource runs one Source's pipeline (locking, fetching, extracting,
// upserting IoCs / refs / state). Returns the RunSource summary so the
// caller can append it to the parent Run.
//
// The provider is resolved via the registry. Blog providers go through
// the LLM extractor; feed providers return seeds directly.
func FetchSource(ctx context.Context, deps Deps, src *model.Source, runID types.RunID, force bool) (*model.RunSource, error) {
	rs := &model.RunSource{
		SourceID:   src.ID,
		SourceKind: src.Kind,
		StartedAt:  time.Now().UTC(),
	}

	// Acquire per-source lock.
	lock, err := deps.Lock.Acquire(ctx, model.LockKindFetch, string(src.ID))
	if err != nil {
		if errors.Is(err, interfaces.ErrLockBusy) {
			rs.Status = types.RunStatusSkipped
			rs.SkipReason = types.SkipReasonLocked
			rs.FinishedAt = time.Now().UTC()
			return rs, nil
		}
		return failedRunSource(rs, err), nil
	}
	defer func() {
		if rerr := lock.Release(ctx); rerr != nil {
			errutil.Handle(ctx, goerr.Wrap(rerr, "release fetch lock",
				goerr.V("source_id", src.ID)))
		}
	}()

	if deps.Registry == nil {
		return failedRunSource(rs, goerr.New("provider registry not configured",
			goerr.T(errutil.TagInvalidInput))), nil
	}
	prov, err := deps.Registry.Resolve(string(src.Type))
	if err != nil {
		return failedRunSource(rs, err), nil
	}

	switch p := prov.(type) {
	case interfaces.BlogProvider:
		err = fetchBlog(ctx, deps, src, runID, p, rs)
	case interfaces.FeedProvider:
		err = fetchFeed(ctx, deps, src, runID, p, rs)
	default:
		err = goerr.New("provider implements neither BlogProvider nor FeedProvider",
			goerr.V("type", src.Type),
			goerr.T(errutil.TagInvalidInput))
	}
	if err != nil {
		return failedRunSource(rs, err), nil
	}
	rs.Status = types.RunStatusSuccess
	rs.FinishedAt = time.Now().UTC()

	if deps.Repo != nil {
		state := &model.SourceState{
			ID:            src.ID,
			LastFetchedAt: rs.FinishedAt,
			LastStatus:    rs.Status,
			LastRunID:     runID,
			TotalIoCCount: rs.IoCCount,
			UpdatedAt:     rs.FinishedAt,
		}
		if cur, err := deps.Repo.GetSourceState(ctx, src.ID); err == nil && cur != nil {
			state.EnabledOverride = cur.EnabledOverride
			state.TotalIoCCount = cur.TotalIoCCount + rs.IoCCount
		}
		if err := deps.Repo.UpdateSourceState(ctx, state); err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "update source state",
				goerr.V("source_id", src.ID)))
		}
	}
	_ = force // currently unused; reserved for future "force re-extract" paths
	return rs, nil
}

func failedRunSource(rs *model.RunSource, err error) *model.RunSource {
	rs.Status = types.RunStatusFailed
	rs.ErrorMessage = err.Error()
	rs.FinishedAt = time.Now().UTC()
	return rs
}

func fetchBlog(ctx context.Context, deps Deps, src *model.Source, runID types.RunID, prov interfaces.BlogProvider, rs *model.RunSource) error {
	logger := logging.From(ctx)

	if deps.Extractor == nil {
		// Blog sources fundamentally need the LLM-backed Extractor —
		// the CLI validates BEEHIVE_LLM_* at startup so this branch is
		// only hit when someone wires Deps by hand (e.g. tests). Keep
		// the guard so we never panic with a nil-pointer deref deep in
		// the article loop.
		return goerr.New("extractor not configured; blog-kind sources require BEEHIVE_LLM_*",
			goerr.V("source_id", src.ID),
			goerr.T(errutil.TagInvalidInput))
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "blog: fetching feed",
		slog.String("source_id", string(src.ID)),
		slog.String("url", src.URL),
	)
	feedStart := time.Now()
	articles, err := prov.Fetch(ctx, src)
	if err != nil {
		return err
	}
	rs.ArticleCount = len(articles)
	logger.LogAttrs(ctx, slog.LevelInfo, "blog: feed fetched",
		slog.String("source_id", string(src.ID)),
		slog.Int("articles", len(articles)),
		slog.Duration("elapsed", time.Since(feedStart)),
	)
	now := time.Now().UTC()
	for i, a := range articles {
		hash := hashBody(a.BodyText)
		existing, gErr := deps.Repo.GetArticleByURL(ctx, a.URL)
		if gErr != nil && !errutil.IsNotFound(gErr) {
			return gErr
		}
		article := &model.Article{
			ID:               articleIDOrNew(existing),
			SourceID:         src.ID,
			URL:              a.URL,
			Title:            a.Title,
			PublishedAt:      a.PublishedAt,
			FetchedAt:        now,
			ContentHash:      hash,
			BodyText:         a.BodyText,
			Summary:          a.Summary,
			ExtractionStatus: types.ExtractionPending,
			RunID:            runID,
		}
		if existing != nil && existing.ContentHash == hash {
			// Body unchanged — no write, no extraction.
			continue
		}
		if existing == nil {
			rs.NewArticleCount++
			if err := deps.Repo.CreateArticle(ctx, article); err != nil {
				return goerr.Wrap(err, "create article", goerr.V("url", a.URL))
			}
		} else {
			if err := deps.Repo.UpdateArticle(ctx, article); err != nil {
				return goerr.Wrap(err, "update article", goerr.V("url", a.URL))
			}
		}
		logger.LogAttrs(ctx, slog.LevelInfo, "blog: extracting article",
			slog.String("source_id", string(src.ID)),
			slog.Int("index", i+1),
			slog.Int("total", len(articles)),
			slog.String("url", a.URL),
			slog.Int("body_bytes", len(a.BodyText)),
		)
		extractStart := time.Now()
		seeds, err := deps.Extractor.Extract(ctx, a.BodyText)
		if err != nil {
			article.ExtractionStatus = types.ExtractionFailed
			if uerr := deps.Repo.UpdateArticle(ctx, article); uerr != nil {
				errutil.Handle(ctx, goerr.Wrap(uerr, "mark article failed",
					goerr.V("url", a.URL)))
			}
			logger.LogAttrs(ctx, slog.LevelWarn, "blog: extract failed",
				slog.String("url", a.URL),
				slog.Duration("elapsed", time.Since(extractStart)),
				slog.String("error", err.Error()),
			)
			return goerr.Wrap(err, "llm extract", goerr.V("url", a.URL))
		}
		logger.LogAttrs(ctx, slog.LevelInfo, "blog: article extracted",
			slog.String("url", a.URL),
			slog.Int("seeds", len(seeds)),
			slog.Duration("elapsed", time.Since(extractStart)),
		)
		if err := persistSeeds(ctx, deps, src, runID, article.ID, "", seeds); err != nil {
			return err
		}
		rs.IoCCount += len(seeds)
		article.ExtractionStatus = types.ExtractionCompleted
		article.ExtractedAt = time.Now().UTC()
		article.IoCCount = len(seeds)
		if uerr := deps.Repo.UpdateArticle(ctx, article); uerr != nil {
			errutil.Handle(ctx, goerr.Wrap(uerr, "mark article completed",
				goerr.V("url", a.URL)))
		}
	}
	return nil
}

func fetchFeed(ctx context.Context, deps Deps, src *model.Source, runID types.RunID, prov interfaces.FeedProvider, rs *model.RunSource) error {
	logger := logging.From(ctx)
	logger.LogAttrs(ctx, slog.LevelInfo, "feed: fetching",
		slog.String("source_id", string(src.ID)),
	)
	feedStart := time.Now()
	seeds, err := prov.Fetch(ctx, src)
	if err != nil {
		return err
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "feed: fetched",
		slog.String("source_id", string(src.ID)),
		slog.Int("seeds", len(seeds)),
		slog.Duration("elapsed", time.Since(feedStart)),
	)
	// Provider-supplied seeds skip the LLM but still go through the
	// normalization post-processor so the IoC.Value stays canonical.
	asArr := make([]interfaces.IoCSeed, 0, len(seeds))
	for _, s := range seeds {
		if s == nil {
			continue
		}
		asArr = append(asArr, *s)
	}
	postSeeds := postNormalize(asArr)
	rs.IoCCount = len(postSeeds)
	return persistSeeds(ctx, deps, src, runID, "", runID, postSeeds)
}

func persistSeeds(ctx context.Context, deps Deps, src *model.Source, runID types.RunID, articleID types.ArticleID, feedRun types.RunID, seeds []*interfaces.IoCSeed) error {
	now := time.Now().UTC()
	for _, s := range seeds {
		if s == nil {
			continue
		}
		ioc := &model.IoC{
			ID:          model.ComputeIoCID(s.Type, s.Value),
			Type:        s.Type,
			Value:       s.Value,
			Raw:         s.Raw,
			FirstSeenAt: now,
			LastSeenAt:  now,
		}
		ref := &model.IoCRef{
			SourceID:   src.ID,
			SourceKind: src.Kind,
			ArticleID:  articleID,
			RunID:      feedRun,
			Raw:        s.Raw,
			Confidence: s.Confidence,
			SeenAt:     now,
		}
		if err := deps.Repo.UpsertIoCWithRef(ctx, ioc, ref); err != nil {
			return goerr.Wrap(err, "upsert ioc",
				goerr.V("source_id", src.ID),
				goerr.V("type", s.Type))
		}
	}
	return nil
}

// postNormalize re-runs the extractor postprocessor on raw seeds. We
// avoid importing the extractor package here to keep usecase layer dep-
// clean; instead we replicate the small filter inline.
func postNormalize(seeds []interfaces.IoCSeed) []*interfaces.IoCSeed {
	return importExtractorPostprocess(seeds)
}

func hashBody(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func articleIDOrNew(existing *model.Article) types.ArticleID {
	if existing != nil {
		return existing.ID
	}
	return types.ArticleID("art-" + id.NewULID())
}
