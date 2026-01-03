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

// fetchRepository defines the repository methods required by FetchUseCase
type fetchRepository interface {
	interfaces.IoCRepository
	interfaces.SourceStateRepository
	interfaces.HistoryRepository
}

// FetchUseCase orchestrates the fetching of IoCs from various sources
type FetchUseCase struct {
	repo        fetchRepository
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
	repo fetchRepository,
	llmClient gollem.LLMClient,
) *FetchUseCase {
	// Initialize n-gram vectorizer for embedding generation
	vec := vectorizer.NewNGramVectorizer()

	return &FetchUseCase{
		repo:        repo,
		llmClient:   llmClient,
		rssService:  rss.New(),
		feedService: feed.New(),
		extractor:   extractor.New(llmClient, extractor.WithNGramVectorizer(vec)),
	}
}

// FetchAllSources fetches IoCs from all enabled sources, optionally filtered by tags
func (uc *FetchUseCase) FetchAllSources(ctx context.Context, sources map[string]model.Source, tags []string) ([]*model.History, error) {
	logger := logging.From(ctx)
	var allHistories []*model.History

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

		var history *model.History
		var err error

		switch source.Type {
		case model.SourceTypeRSS:
			history, err = uc.fetchRSS(ctx, sourceID, &source)
		case model.SourceTypeFeed:
			history, err = uc.fetchFeed(ctx, sourceID, &source)
		default:
			logger.Warn("unknown source type", "source_id", sourceID, "type", source.Type)
			continue
		}

		if err != nil {
			logger.Error("failed to fetch from source",
				"source_id", sourceID,
				"error", err)
			// Continue with other sources even if one fails
			// Create history for the failed fetch
			history = &model.History{
				ID:             model.GenerateHistoryID(),
				SourceID:       sourceID,
				SourceType:     source.Type,
				Status:         model.FetchStatusFailure,
				StartedAt:      time.Now(),
				CompletedAt:    time.Now(),
				ProcessingTime: 0,
				URLs:           []string{}, // URL unknown for failed fetch
				ItemsFetched:   0,
				IoCsExtracted:  0,
				IoCsCreated:    0,
				IoCsUpdated:    0,
				IoCsUnchanged:  0,
				ErrorCount:     1,
				Errors:         []*model.FetchError{model.ExtractErrorInfo(err)},
				CreatedAt:      time.Now(),
			}
			if histErr := uc.repo.SaveHistory(ctx, history); histErr != nil {
				logger.Error("failed to save fetch history",
					"source_id", sourceID,
					"history_id", history.ID,
					"error", histErr)
			}
		}

		allHistories = append(allHistories, history)
	}

	return allHistories, nil
}

// fetchRSS fetches and processes IoCs from an RSS source
func (uc *FetchUseCase) fetchRSS(ctx context.Context, sourceID string, source *model.Source) (*model.History, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &FetchStats{
		SourceID:   sourceID,
		SourceType: string(model.SourceTypeRSS),
	}

	// Track errors with context for history
	var fetchErrors []*model.FetchError

	// Get previous state
	state, err := uc.repo.GetState(ctx, sourceID)
	if err != nil {
		if errors.Is(err, interfaces.ErrSourceStateNotFound) {
			// No previous state found - this is expected for first run
			logger.Info("no previous state found, starting fresh", "source_id", sourceID)
			state = &model.SourceState{
				SourceID: sourceID,
			}
		} else {
			// Unexpected error - fail fast to avoid re-processing
			return nil, goerr.Wrap(err, "failed to get source state")
		}
	}

	// Fetch RSS feed
	articles, err := uc.rssService.FetchFeed(ctx, source.URL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch RSS feed",
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
		result, err := uc.repo.BatchUpsertIoCs(ctx, iocsToSave)
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
	state.LastFetchedAt = time.Now()
	if len(newArticles) > 0 {
		latestArticle := rss.GetLatestArticle(newArticles)
		state.LastItemID = latestArticle.GUID
		state.LastItemDate = latestArticle.PublishedAt
	}
	state.ItemCount += int64(len(newArticles))

	state.ErrorCount += int64(stats.ErrorCount)
	state.LastStatus = string(model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched))
	if stats.ErrorCount > 0 {
		state.LastError = "encountered errors during fetch"
	} else {
		state.LastError = ""
	}

	// Ensure SourceID is set (should already be set from GetState or initialization)
	if state.SourceID == "" {
		logger.Warn("state.SourceID is empty, setting it",
			"source_id", sourceID)
		state.SourceID = sourceID
	}

	if err := uc.repo.SaveState(ctx, state); err != nil {
		logger.Error("failed to save source state",
			"source_id", sourceID,
			"state_source_id", state.SourceID,
			"error", err)
	}

	stats.ProcessingTime = time.Since(startTime)

	// Save ingestion history
	history := &model.History{
		ID:             model.GenerateHistoryID(),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeRSS,
		Status:         model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched),
		StartedAt:      startTime,
		CompletedAt:    time.Now(),
		ProcessingTime: stats.ProcessingTime,
		URLs:           []string{}, // TODO: track accessed URLs in usecase
		ItemsFetched:   stats.ItemsFetched,
		IoCsExtracted:  stats.IoCsExtracted,
		IoCsCreated:    stats.IoCsCreated,
		IoCsUpdated:    stats.IoCsUpdated,
		IoCsUnchanged:  stats.IoCsUnchanged,
		ErrorCount:     stats.ErrorCount,
		Errors:         fetchErrors,
		CreatedAt:      time.Now(),
	}

	logger.Info("attempting to save fetch history",
		"source_id", sourceID,
		"history_id", history.ID,
		"status", history.Status,
		"items_fetched", history.ItemsFetched)

	if err := uc.repo.SaveHistory(ctx, history); err != nil {
		// History save failure should not fail the fetch operation
		logger.Error("failed to save fetch history",
			"source_id", sourceID,
			"history_id", history.ID,
			"error", err)
	}

	return history, nil
}

// fetchFeed fetches and processes IoCs from a threat intelligence feed
func (uc *FetchUseCase) fetchFeed(ctx context.Context, sourceID string, source *model.Source) (*model.History, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &FetchStats{
		SourceID:   sourceID,
		SourceType: string(model.SourceTypeFeed),
	}

	// Track errors with context for history
	var fetchErrors []*model.FetchError

	if source.FeedConfig == nil || source.FeedConfig.Schema == "" {
		return nil, goerr.New("feed config not specified", goerr.V("source_id", sourceID))
	}

	// Get previous state
	state, err := uc.repo.GetState(ctx, sourceID)
	if err != nil {
		if errors.Is(err, interfaces.ErrSourceStateNotFound) {
			// No previous state found - this is expected for first run
			logger.Info("no previous state found, starting fresh", "source_id", sourceID)
			state = &model.SourceState{
				SourceID: sourceID,
			}
		} else {
			// Unexpected error - fail fast to avoid re-processing
			return nil, goerr.Wrap(err, "failed to get source state")
		}
	}

	// Fetch feed entries
	entries, err := uc.feedService.FetchFeed(ctx, source.URL, source.FeedConfig.Schema)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch feed",
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
	existingIoCs, err := uc.repo.ListIoCsBySource(ctx, sourceID)
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
		result, err := uc.repo.BatchUpsertIoCs(ctx, iocsToSave)
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
		inactiveCount, err := uc.repo.BatchUpsertIoCs(ctx, inactiveIoCs)
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
	state.LastFetchedAt = time.Now()
	state.ItemCount += int64(stats.ItemsFetched)
	state.ErrorCount += int64(stats.ErrorCount)
	state.LastStatus = string(model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched))

	if stats.ErrorCount > 0 {
		state.LastError = "encountered errors during fetch"
	} else {
		state.LastError = ""
	}

	// Ensure SourceID is set (defensive check)
	if state.SourceID == "" {
		logger.Warn("state.SourceID is empty, setting it",
			"source_id", sourceID)
		state.SourceID = sourceID
	}

	if err := uc.repo.SaveState(ctx, state); err != nil {
		logger.Error("failed to save source state",
			"source_id", sourceID,
			"state_source_id", state.SourceID,
			"error", err)
	}

	stats.ProcessingTime = time.Since(startTime)

	// Save ingestion history
	history := &model.History{
		ID:             model.GenerateHistoryID(),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeFeed,
		Status:         model.DetermineFetchStatus(stats.ErrorCount, stats.ItemsFetched),
		StartedAt:      startTime,
		CompletedAt:    time.Now(),
		ProcessingTime: stats.ProcessingTime,
		URLs:           []string{}, // TODO: track accessed URLs in usecase
		ItemsFetched:   stats.ItemsFetched,
		IoCsExtracted:  stats.IoCsExtracted,
		IoCsCreated:    stats.IoCsCreated,
		IoCsUpdated:    stats.IoCsUpdated,
		IoCsUnchanged:  stats.IoCsUnchanged,
		ErrorCount:     stats.ErrorCount,
		Errors:         fetchErrors,
		CreatedAt:      time.Now(),
	}

	logger.Info("attempting to save fetch history",
		"source_id", sourceID,
		"history_id", history.ID,
		"status", history.Status,
		"items_fetched", history.ItemsFetched)

	if err := uc.repo.SaveHistory(ctx, history); err != nil {
		// History save failure should not fail the fetch operation
		logger.Error("failed to save fetch history",
			"source_id", sourceID,
			"history_id", history.ID,
			"error", err)
	}

	return history, nil
}

// FetchSourceByID executes fetch for a specific source ID
func (uc *FetchUseCase) FetchSourceByID(ctx context.Context, sourcesMap map[string]model.Source, sourceID string) (*model.History, error) {
	logger := logging.From(ctx)

	// Find the source configuration
	source, ok := sourcesMap[sourceID]
	if !ok {
		return nil, goerr.New("source not found", goerr.V("source_id", sourceID))
	}

	logger.Info("fetching from source", "source_id", sourceID, "type", source.Type)

	var history *model.History
	var err error

	switch source.Type {
	case model.SourceTypeRSS:
		history, err = uc.fetchRSS(ctx, sourceID, &source)
	case model.SourceTypeFeed:
		history, err = uc.fetchFeed(ctx, sourceID, &source)
	default:
		return nil, goerr.New("unknown source type",
			goerr.V("source_id", sourceID),
			goerr.V("type", source.Type))
	}

	if err != nil {
		logger.Error("failed to fetch from source",
			"source_id", sourceID,
			"error", err)

		// Create history for failed fetch
		failedHistory := &model.History{
			ID:             model.GenerateHistoryID(),
			SourceID:       sourceID,
			SourceType:     source.Type,
			Status:         model.FetchStatusFailure,
			StartedAt:      time.Now(),
			CompletedAt:    time.Now(),
			ProcessingTime: 0,
			URLs:           []string{},
			ItemsFetched:   0,
			IoCsExtracted:  0,
			IoCsCreated:    0,
			IoCsUpdated:    0,
			IoCsUnchanged:  0,
			ErrorCount:     1,
			Errors:         []*model.FetchError{model.ExtractErrorInfo(err)},
			CreatedAt:      time.Now(),
		}

		if histErr := uc.repo.SaveHistory(ctx, failedHistory); histErr != nil {
			logger.Error("failed to save fetch history",
				"source_id", sourceID,
				"history_id", failedHistory.ID,
				"error", histErr)
			return nil, goerr.Wrap(histErr, "failed to save fetch history")
		}

		return failedHistory, nil
	}

	// Return the history created by fetchRSS or fetchFeed
	return history, nil
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
