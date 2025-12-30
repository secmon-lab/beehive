package rss

import (
	"context"
	"errors"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/domain/extractor"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/rss"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// RSSSource implements Source interface for RSS feeds
type RSSSource struct {
	id          string
	url         string
	tags        types.Tags
	maxArticles int

	// Dependencies (RSS-specific dependencies encapsulated)
	iocRepo    interfaces.IoCRepository
	stateRepo  RSSStateRepository
	rssService *rss.Service
	llmClient  gollem.LLMClient
	extractor  *extractor.Extractor
}

// New creates a new RSSSource instance
func New(
	id string,
	cfg config.RSSSource,
	iocRepo interfaces.IoCRepository,
	stateRepo RSSStateRepository,
	llmClient gollem.LLMClient,
) (*RSSSource, error) {
	// Config should already be validated, so Tags should be populated
	return &RSSSource{
		id:          id,
		url:         cfg.URL,
		tags:        cfg.Tags,
		maxArticles: cfg.MaxArticles,
		iocRepo:     iocRepo,
		stateRepo:   stateRepo,
		rssService:  rss.New(),
		llmClient:   llmClient,
		extractor:   extractor.New(llmClient),
	}, nil
}

// ID returns the source ID
func (s *RSSSource) ID() string {
	return s.id
}

// Type returns the source type
func (s *RSSSource) Type() model.SourceType {
	return model.SourceTypeRSS
}

// Tags returns the source tags as string slice
func (s *RSSSource) Tags() []string {
	return s.tags.Strings()
}

// Enabled returns whether the source is enabled
func (s *RSSSource) Enabled() bool {
	return true
}

// Fetch fetches and processes IoCs from the RSS source
func (s *RSSSource) Fetch(ctx context.Context) (*interfaces.FetchStats, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &interfaces.FetchStats{
		SourceID:   s.id,
		SourceType: model.SourceTypeRSS,
	}

	// Get previous state
	state, err := s.stateRepo.GetRSSState(ctx, s.id)
	if err != nil {
		if errors.Is(err, ErrRSSStateNotFound) {
			// No previous state found - this is expected for first run
			logger.Info("no previous state found, starting fresh", "source_id", s.id)
			state = &RSSState{
				SourceID: s.id,
			}
		} else {
			// Unexpected error - fail fast to avoid re-processing
			return stats, goerr.Wrap(err, "failed to get RSS state")
		}
	}

	// Fetch RSS feed
	articles, err := s.rssService.FetchFeed(ctx, s.url)
	if err != nil {
		return stats, goerr.Wrap(err, "failed to fetch RSS feed",
			goerr.V("source_id", s.id),
			goerr.V("url", s.url))
	}

	logger.Info("fetched articles from RSS",
		"source_id", s.id,
		"total_articles", len(articles))

	// Filter for new articles only
	newArticles := rss.FilterNewArticles(articles, state.LastArticleID, state.LastItemDate)
	stats.ItemsFetched = len(newArticles)

	// Apply max articles limit if configured
	if s.maxArticles > 0 {
		if len(newArticles) > s.maxArticles {
			newArticles = newArticles[:s.maxArticles]
		}
	}

	logger.Info("processing new articles",
		"source_id", s.id,
		"new_articles", len(newArticles))

	// Accumulate IoCs for batch writing
	var iocsToSave []*model.IoC

	// Process each article
	for _, article := range newArticles {
		// Fetch article content
		content, err := s.rssService.FetchArticleContent(ctx, article.Link)
		if err != nil {
			logger.Warn("failed to fetch article content",
				"source_id", s.id,
				"url", article.Link,
				"error", err)
			stats.ErrorCount++
			continue
		}

		// Extract IoCs from article using LLM
		extracted, err := s.extractor.ExtractFromArticle(ctx, article.Title, content)
		if err != nil {
			logger.Warn("failed to extract IoCs from article",
				"source_id", s.id,
				"url", article.Link,
				"error", err)
			stats.ErrorCount++
			continue
		}

		stats.IoCsExtracted += len(extracted)

		// Track IoCs for this article
		var articleIoCs []*model.IoC

		// Convert and accumulate IoCs
		for _, ext := range extracted {
			// Prepare context parameters for RSS feeds
			// Use article GUID as primary context for deduplication
			contextParams := map[string]string{
				"article_guid": article.GUID,
				"article_url":  article.Link,
			}

			ioc, err := extractor.ConvertToIoC(s.id, string(model.SourceTypeRSS), article.Link, ext, contextParams)
			if err != nil {
				logger.Warn("failed to convert extracted IoC",
					"source_id", s.id,
					"error", err)
				stats.ErrorCount++
				continue
			}

			// Generate embedding
			embedText := ioc.Value + " " + ioc.Description
			embedding, err := s.extractor.GenerateEmbedding(ctx, embedText)
			if err != nil {
				logger.Warn("failed to generate embedding",
					"source_id", s.id,
					"ioc_id", ioc.ID,
					"error", err)
				// Continue without embedding
			} else {
				copy(ioc.Embedding, embedding)
			}

			articleIoCs = append(articleIoCs, ioc)
			iocsToSave = append(iocsToSave, ioc)
		}

		// Log extracted IoCs for this article at Debug level
		if len(articleIoCs) > 0 {
			iocSummary := make([]map[string]string, 0, len(articleIoCs))
			for _, ioc := range articleIoCs {
				iocSummary = append(iocSummary, map[string]string{
					"type":        string(ioc.Type),
					"value":       ioc.Value,
					"description": ioc.Description,
				})
			}
			logger.Debug("extracted IoCs from article",
				"source_id", s.id,
				"article_title", article.Title,
				"article_url", article.Link,
				"ioc_count", len(articleIoCs),
				"iocs", iocSummary)
		}
	}

	// Batch save all IoCs
	if len(iocsToSave) > 0 {
		result, err := s.iocRepo.BatchUpsertIoCs(ctx, iocsToSave)
		stats.IoCsCreated += result.Created
		stats.IoCsUpdated += result.Updated
		stats.IoCsUnchanged += result.Unchanged
		if err != nil {
			logger.Error("failed to batch save IoCs",
				"source_id", s.id,
				"total_iocs", len(iocsToSave),
				"result", result,
				"error", err)
			stats.ErrorCount++
		}

		logger.Info("batch saved IoCs",
			"source_id", s.id,
			"created", result.Created,
			"updated", result.Updated,
			"unchanged", result.Unchanged,
			"total", len(iocsToSave))
	}

	// Update source state
	if len(newArticles) > 0 {
		latestArticle := rss.GetLatestArticle(newArticles)
		state.LastArticleID = latestArticle.GUID
		state.LastItemDate = latestArticle.PublishedAt
		state.ItemCount += int64(len(newArticles))
		state.LastFetchedAt = time.Now()
	}

	state.ErrorCount += int64(stats.ErrorCount)
	if stats.ErrorCount > 0 {
		state.LastError = "encountered errors during fetch"
	} else {
		state.LastError = ""
	}
	state.UpdatedAt = time.Now()

	if err := s.stateRepo.SaveRSSState(ctx, state); err != nil {
		logger.Error("failed to save RSS state",
			"source_id", s.id,
			"error", err)
	}

	stats.ProcessingTime = time.Since(startTime)
	return stats, nil
}
