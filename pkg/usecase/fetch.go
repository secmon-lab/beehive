package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/beehive/pkg/domain/extractor"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/vectorizer"
	"github.com/secmon-lab/beehive/pkg/service/feed"
	"github.com/secmon-lab/beehive/pkg/service/rss"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// FetchUseCase orchestrates the fetching of IoCs from various sources
type FetchUseCase struct {
	iocRepo     interfaces.IoCRepository
	stateRepo   interfaces.SourceStateRepository
	historyRepo interfaces.HistoryRepository
	llmClient   gollem.LLMClient
	rssService  *rss.Service
	feedService *feed.Service
	extractor   *extractor.Extractor
}

// FetchStats represents statistics from a fetch operation
type FetchStats struct {
	SourceID       string
	SourceType     string
	ItemsFetched   int
	IoCsExtracted  int
	IoCsCreated    int // New IoCs created
	IoCsUpdated    int // Existing IoCs updated (description/status changed)
	IoCsUnchanged  int // Existing IoCs unchanged (skipped)
	IoCsGeneric    int // Generic IoCs skipped
	ErrorCount     int
	ProcessingTime time.Duration
}

// NewFetchUseCase creates a new fetch use case
func NewFetchUseCase(
	iocRepo interfaces.IoCRepository,
	stateRepo interfaces.SourceStateRepository,
	historyRepo interfaces.HistoryRepository,
	llmClient gollem.LLMClient,
) *FetchUseCase {
	// Initialize n-gram vectorizer for embedding generation
	vec := vectorizer.NewNGramVectorizer()

	return &FetchUseCase{
		iocRepo:     iocRepo,
		stateRepo:   stateRepo,
		historyRepo: historyRepo,
		llmClient:   llmClient,
		rssService:  rss.New(),
		feedService: feed.New(),
		extractor:   extractor.New(llmClient, extractor.WithNGramVectorizer(vec)),
	}
}

// FetchAllSources fetches IoCs from all enabled sources, optionally filtered by tags
func (uc *FetchUseCase) FetchAllSources(ctx context.Context, sources map[string]model.Source, tags []string) ([]*FetchStats, error) {
	logger := logging.From(ctx)
	var allStats []*FetchStats

	for sourceID, source := range sources {
		// Skip disabled sources
		if !source.Enabled {
			logger.Info("skipping disabled source", "source_id", sourceID)
			continue
		}

		// Filter by tags if specified
		if len(tags) > 0 && !hasAnyTag(source.Tags, tags) {
			logger.Info("skipping source due to tag filter",
				"source_id", sourceID,
				"source_tags", source.Tags,
				"filter_tags", tags)
			continue
		}

		logger.Info("fetching from source", "source_id", sourceID, "type", source.Type)

		var stats *FetchStats
		var err error

		switch source.Type {
		case model.SourceTypeRSS:
			stats, err = uc.fetchRSS(ctx, sourceID, &source)
		case model.SourceTypeFeed:
			stats, err = uc.fetchFeed(ctx, sourceID, &source)
		default:
			logger.Warn("unknown source type", "source_id", sourceID, "type", source.Type)
			continue
		}

		if err != nil {
			logger.Error("failed to fetch from source",
				"source_id", sourceID,
				"error", err)
			// Continue with other sources even if one fails
			// Create stats with error information
			stats = &FetchStats{
				SourceID:   sourceID,
				SourceType: string(source.Type),
				ErrorCount: 1,
			}

			// Save history for the failed fetch
			history := &model.History{
				ID:             model.GenerateHistoryID(sourceID, time.Now()),
				SourceID:       sourceID,
				SourceType:     source.Type,
				Status:         model.FetchStatusFailure,
				StartedAt:      time.Now(),
				CompletedAt:    time.Now(),
				ProcessingTime: 0,
				ItemsFetched:   0,
				IoCsExtracted:  0,
				IoCsCreated:    0,
				IoCsUpdated:    0,
				IoCsUnchanged:  0,
				ErrorCount:     1,
				Errors:         []*model.FetchError{model.ExtractErrorInfo(err)},
				CreatedAt:      time.Now(),
			}
			if histErr := uc.historyRepo.SaveHistory(ctx, history); histErr != nil {
				logger.Error("failed to save fetch history",
					"source_id", sourceID,
					"history_id", history.ID,
					"error", histErr)
			}
		}

		allStats = append(allStats, stats)
	}

	return allStats, nil
}

// fetchRSS fetches and processes IoCs from an RSS source
func (uc *FetchUseCase) fetchRSS(ctx context.Context, sourceID string, source *model.Source) (*FetchStats, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &FetchStats{
		SourceID:   sourceID,
		SourceType: string(model.SourceTypeRSS),
	}

	// Track errors with context for history
	var fetchErrors []*model.FetchError

	// Get previous state
	state, err := uc.stateRepo.GetState(ctx, sourceID)
	if err != nil {
		if errors.Is(err, interfaces.ErrSourceStateNotFound) {
			// No previous state found - this is expected for first run
			logger.Info("no previous state found, starting fresh", "source_id", sourceID)
			state = &model.SourceState{
				SourceID: sourceID,
			}
		} else {
			// Unexpected error - fail fast to avoid re-processing
			return stats, goerr.Wrap(err, "failed to get source state")
		}
	}

	// Fetch RSS feed
	articles, err := uc.rssService.FetchFeed(ctx, source.URL)
	if err != nil {
		return stats, goerr.Wrap(err, "failed to fetch RSS feed",
			goerr.V("source_id", sourceID),
			goerr.V("url", source.URL))
	}

	logger.Info("fetched articles from RSS",
		"source_id", sourceID,
		"total_articles", len(articles))

	// Filter for new articles only
	newArticles := rss.FilterNewArticles(articles, state.LastItemID, state.LastItemDate)
	stats.ItemsFetched = len(newArticles)

	// Apply max articles limit if configured
	if source.RSSConfig != nil && source.RSSConfig.MaxArticles > 0 {
		if len(newArticles) > source.RSSConfig.MaxArticles {
			newArticles = newArticles[:source.RSSConfig.MaxArticles]
		}
	}

	logger.Info("processing new articles",
		"source_id", sourceID,
		"new_articles", len(newArticles))

	// Accumulate IoCs for batch writing
	var iocsToSave []*model.IoC

	// Process each article
	for _, article := range newArticles {
		// Fetch article content
		content, err := uc.rssService.FetchArticleContent(ctx, article.Link)
		if err != nil {
			logger.Warn("failed to fetch article content",
				"source_id", sourceID,
				"url", article.Link,
				"error", err)
			stats.ErrorCount++
			fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
			continue
		}

		// Extract IoCs from article using LLM
		extracted, err := uc.extractor.ExtractFromArticle(ctx, article.Title, content)
		if err != nil {
			logger.Warn("failed to extract IoCs from article",
				"source_id", sourceID,
				"url", article.Link,
				"error", err)
			stats.ErrorCount++
			fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
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

			ioc, err := extractor.ConvertToIoC(sourceID, string(model.SourceTypeRSS), article.Link, ext, contextParams)
			if err != nil {
				logger.Warn("failed to convert extracted IoC",
					"source_id", sourceID,
					"error", err)
				stats.ErrorCount++
				fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
				continue
			}

			// Generate embedding
			embedText := ioc.Value + " " + ioc.Description
			embedding, err := uc.extractor.GenerateEmbedding(ctx, embedText)
			if err != nil {
				logger.Warn("failed to generate embedding",
					"source_id", sourceID,
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
				"source_id", sourceID,
				"article_title", article.Title,
				"article_url", article.Link,
				"ioc_count", len(articleIoCs),
				"iocs", iocSummary)
		}
	}

	// Batch save all IoCs
	if len(iocsToSave) > 0 {
		result, err := uc.iocRepo.BatchUpsertIoCs(ctx, iocsToSave)
		stats.IoCsCreated += result.Created
		stats.IoCsUpdated += result.Updated
		stats.IoCsUnchanged += result.Unchanged
		if err != nil {
			logger.Error("failed to batch save IoCs",
				"source_id", sourceID,
				"total_iocs", len(iocsToSave),
				"result", result,
				"error", err)
			stats.ErrorCount++
			fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
		}

		logger.Info("batch saved IoCs",
			"source_id", sourceID,
			"created", result.Created,
			"updated", result.Updated,
			"unchanged", result.Unchanged,
			"total", len(iocsToSave))
	}

	// Update source state
	if len(newArticles) > 0 {
		latestArticle := rss.GetLatestArticle(newArticles)
		state.LastItemID = latestArticle.GUID
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

	if err := uc.stateRepo.SaveState(ctx, state); err != nil {
		logger.Error("failed to save source state",
			"source_id", sourceID,
			"error", err)
	}

	stats.ProcessingTime = time.Since(startTime)

	// Save ingestion history
	history := &model.History{
		ID:             model.GenerateHistoryID(sourceID, startTime),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeRSS,
		Status:         model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched),
		StartedAt:      startTime,
		CompletedAt:    time.Now(),
		ProcessingTime: stats.ProcessingTime,
		ItemsFetched:   stats.ItemsFetched,
		IoCsExtracted:  stats.IoCsExtracted,
		IoCsCreated:    stats.IoCsCreated,
		IoCsUpdated:    stats.IoCsUpdated,
		IoCsUnchanged:  stats.IoCsUnchanged,
		ErrorCount:     stats.ErrorCount,
		Errors:         fetchErrors,
		CreatedAt:      time.Now(),
	}

	if err := uc.historyRepo.SaveHistory(ctx, history); err != nil {
		// History save failure should not fail the fetch operation
		logger.Error("failed to save fetch history",
			"source_id", sourceID,
			"history_id", history.ID,
			"error", err)
	}

	return stats, nil
}

// fetchFeed fetches and processes IoCs from a threat intelligence feed
func (uc *FetchUseCase) fetchFeed(ctx context.Context, sourceID string, source *model.Source) (*FetchStats, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &FetchStats{
		SourceID:   sourceID,
		SourceType: string(model.SourceTypeFeed),
	}

	// Track errors with context for history
	var fetchErrors []*model.FetchError

	if source.FeedConfig == nil || source.FeedConfig.Schema == "" {
		return stats, goerr.New("feed config not specified", goerr.V("source_id", sourceID))
	}

	// Fetch feed entries
	entries, err := uc.feedService.FetchFeed(ctx, source.URL, source.FeedConfig.Schema)
	if err != nil {
		return stats, goerr.Wrap(err, "failed to fetch feed",
			goerr.V("source_id", sourceID),
			goerr.V("url", source.URL),
			goerr.V("schema", source.FeedConfig.Schema))
	}

	stats.ItemsFetched = len(entries)

	// Apply max items limit if configured
	if source.FeedConfig.MaxItems > 0 && len(entries) > source.FeedConfig.MaxItems {
		entries = entries[:source.FeedConfig.MaxItems]
	}

	logger.Info("fetched feed entries",
		"source_id", sourceID,
		"total_entries", stats.ItemsFetched,
		"processing_entries", len(entries))

	// Get existing IoCs for this source to implement differential update
	existingIoCs, err := uc.iocRepo.ListIoCsBySource(ctx, sourceID)
	if err != nil {
		logger.Warn("failed to list existing IoCs",
			"source_id", sourceID,
			"error", err)
		existingIoCs = nil
	}

	// Build map of existing IoC IDs for quick lookup
	existingMap := make(map[string]*model.IoC)
	for _, ioc := range existingIoCs {
		existingMap[ioc.ID] = ioc
	}

	// Track which IoCs are still in the feed
	seenIDs := make(map[string]bool)

	// Accumulate IoCs for batch writing
	var iocsToSave []*model.IoC

	// Process each feed entry
	for _, entry := range entries {
		// Prepare context parameters for threat feeds
		// Use entry ID as primary context for deduplication
		contextParams := map[string]string{
			"entry_id": entry.ID,
		}

		// Generate context key and IoC ID
		contextKey := model.GenerateContextKey(string(model.SourceTypeFeed), contextParams)
		iocID := model.GenerateID(sourceID, entry.Type, entry.Value, contextKey)
		seenIDs[iocID] = true

		ioc := &model.IoC{
			ID:          iocID,
			SourceID:    sourceID,
			SourceType:  string(model.SourceTypeFeed),
			Type:        entry.Type,
			Value:       model.NormalizeValue(entry.Type, entry.Value),
			Description: entry.Description,
			SourceURL:   source.URL,
			Context:     "", // Feeds don't have context
			Embedding:   make([]float32, model.EmbeddingDimension),
			Status:      model.IoCStatusActive,
		}

		// Generate embedding
		embedText := ioc.Value + " " + ioc.Description
		embedding, err := uc.extractor.GenerateEmbedding(ctx, embedText)
		if err != nil {
			logger.Warn("failed to generate embedding",
				"source_id", sourceID,
				"ioc_id", ioc.ID,
				"error", err)
		} else {
			copy(ioc.Embedding, embedding)
		}

		iocsToSave = append(iocsToSave, ioc)
		stats.IoCsExtracted++
	}

	// Batch save all active IoCs
	if len(iocsToSave) > 0 {
		result, err := uc.iocRepo.BatchUpsertIoCs(ctx, iocsToSave)
		stats.IoCsCreated += result.Created
		stats.IoCsUpdated += result.Updated
		stats.IoCsUnchanged += result.Unchanged
		if err != nil {
			logger.Error("failed to batch save IoCs",
				"source_id", sourceID,
				"total_iocs", len(iocsToSave),
				"result", result,
				"error", err)
			stats.ErrorCount++
			fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
		}

		logger.Info("batch saved IoCs",
			"source_id", sourceID,
			"created", result.Created,
			"updated", result.Updated,
			"unchanged", result.Unchanged,
			"total", len(iocsToSave))
	}

	// Mark IoCs no longer in feed as inactive
	var inactiveIoCs []*model.IoC
	for existingID, existingIoC := range existingMap {
		if !seenIDs[existingID] && existingIoC.Status == model.IoCStatusActive {
			existingIoC.Status = model.IoCStatusInactive
			inactiveIoCs = append(inactiveIoCs, existingIoC)
		}
	}

	// Batch update inactive IoCs
	if len(inactiveIoCs) > 0 {
		inactiveCount, err := uc.iocRepo.BatchUpsertIoCs(ctx, inactiveIoCs)
		if err != nil {
			logger.Error("failed to batch mark IoCs as inactive",
				"source_id", sourceID,
				"total_inactive", len(inactiveIoCs),
				"updated_count", inactiveCount,
				"error", err)
			stats.ErrorCount++
			fetchErrors = append(fetchErrors, model.ExtractErrorInfo(err))
		}

		logger.Info("batch marked IoCs as inactive",
			"source_id", sourceID,
			"inactive_count", inactiveCount,
			"total_inactive", len(inactiveIoCs))
	}

	// Update source state
	state := &model.SourceState{
		SourceID:      sourceID,
		LastFetchedAt: time.Now(),
		ItemCount:     int64(len(entries)),
		ErrorCount:    int64(stats.ErrorCount),
	}

	if stats.ErrorCount > 0 {
		state.LastError = "encountered errors during fetch"
	}

	if err := uc.stateRepo.SaveState(ctx, state); err != nil {
		logger.Error("failed to save source state",
			"source_id", sourceID,
			"error", err)
	}

	stats.ProcessingTime = time.Since(startTime)

	// Save ingestion history
	history := &model.History{
		ID:             model.GenerateHistoryID(sourceID, startTime),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeFeed,
		Status:         model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched),
		StartedAt:      startTime,
		CompletedAt:    time.Now(),
		ProcessingTime: stats.ProcessingTime,
		ItemsFetched:   stats.ItemsFetched,
		IoCsExtracted:  stats.IoCsExtracted,
		IoCsCreated:    stats.IoCsCreated,
		IoCsUpdated:    stats.IoCsUpdated,
		IoCsUnchanged:  stats.IoCsUnchanged,
		ErrorCount:     stats.ErrorCount,
		Errors:         fetchErrors,
		CreatedAt:      time.Now(),
	}

	if err := uc.historyRepo.SaveHistory(ctx, history); err != nil {
		// History save failure should not fail the fetch operation
		logger.Error("failed to save fetch history",
			"source_id", sourceID,
			"history_id", history.ID,
			"error", err)
	}

	return stats, nil
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
