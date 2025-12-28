package feed

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
	"github.com/secmon-lab/beehive/pkg/service/feed"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
)

// FeedSource implements Source interface for threat intelligence feeds
type FeedSource struct {
	id         string
	feedSchema types.FeedSchema
	url        string // Effective URL (explicit or default)
	tags       types.Tags
	maxItems   int

	// Dependencies (Feed-specific, no LLM extraction needed)
	iocRepo     interfaces.IoCRepository
	stateRepo   FeedStateRepository
	feedService *feed.Service
	extractor   *extractor.Extractor // embedding generation only
}

// New creates a new FeedSource from configuration
func New(
	id string,
	cfg config.FeedSource,
	iocRepo interfaces.IoCRepository,
	stateRepo FeedStateRepository,
	llmClient gollem.LLMClient,
) (*FeedSource, error) {
	// Validate and convert config.Schema to types.FeedSchema
	feedSchema, err := types.NewFeedSchema(cfg.Schema)
	if err != nil {
		return nil, goerr.Wrap(err, "invalid feed schema", goerr.V("id", id))
	}

	// Validate and convert config.Tags to types.Tags
	tags, err := types.NewTags(cfg.Tags)
	if err != nil {
		return nil, goerr.Wrap(err, "invalid tags", goerr.V("id", id))
	}

	// Get effective URL (explicit or default from schema)
	url := cfg.GetURL()
	if url == "" {
		return nil, goerr.New("no URL available for feed", goerr.V("id", id), goerr.V("schema", cfg.Schema))
	}

	return &FeedSource{
		id:          id,
		feedSchema:  feedSchema,
		url:         url,
		tags:        tags,
		maxItems:    cfg.MaxItems,
		iocRepo:     iocRepo,
		stateRepo:   stateRepo,
		feedService: feed.New(),
		extractor:   extractor.New(llmClient),
	}, nil
}

// ID returns the source identifier
func (s *FeedSource) ID() string {
	return s.id
}

// Type returns the source type
func (s *FeedSource) Type() model.SourceType {
	return model.SourceTypeFeed
}

// Tags returns the source tags as string slice
func (s *FeedSource) Tags() []string {
	return s.tags.Strings()
}

// Enabled returns whether this source is enabled
func (s *FeedSource) Enabled() bool {
	return true
}

// Fetch fetches and processes IoCs from the threat intelligence feed
func (s *FeedSource) Fetch(ctx context.Context) (*interfaces.FetchStats, error) {
	logger := logging.From(ctx)
	startTime := time.Now()
	stats := &interfaces.FetchStats{
		SourceID:   s.id,
		SourceType: model.SourceTypeFeed,
	}

	// Fetch feed entries using feedSchema.String() for parser selection
	entries, err := s.feedService.FetchFeed(ctx, s.url, s.feedSchema.String())
	if err != nil {
		return stats, goerr.Wrap(err, "failed to fetch feed",
			goerr.V("source_id", s.id),
			goerr.V("url", s.url),
			goerr.V("schema", s.feedSchema.String()))
	}

	stats.ItemsFetched = len(entries)

	// Apply max items limit if configured
	if s.maxItems > 0 && len(entries) > s.maxItems {
		entries = entries[:s.maxItems]
	}

	logger.Info("fetched feed entries",
		"source_id", s.id,
		"total_entries", stats.ItemsFetched,
		"processing_entries", len(entries))

	// Get existing IoCs for this source to implement differential update
	existingIoCs, err := s.iocRepo.ListIoCsBySource(ctx, s.id)
	if err != nil {
		logger.Warn("failed to list existing IoCs",
			"source_id", s.id,
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
		iocID := model.GenerateID(s.id, entry.Type, entry.Value, contextKey)
		seenIDs[iocID] = true

		ioc := &model.IoC{
			ID:          iocID,
			SourceID:    s.id,
			SourceType:  string(model.SourceTypeFeed),
			Type:        entry.Type,
			Value:       model.NormalizeValue(entry.Type, entry.Value),
			Description: entry.Description,
			SourceURL:   s.url,
			Context:     "", // Feeds don't have context
			Embedding:   make([]float32, model.EmbeddingDimension),
			Status:      model.IoCStatusActive,
		}

		// Generate embedding
		embedText := ioc.Value + " " + ioc.Description
		embedding, err := s.extractor.GenerateEmbedding(ctx, embedText)
		if err != nil {
			logger.Warn("failed to generate embedding",
				"source_id", s.id,
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
			stats.Errors++
		}

		logger.Info("batch saved IoCs",
			"source_id", s.id,
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
		inactiveResult, err := s.iocRepo.BatchUpsertIoCs(ctx, inactiveIoCs)
		if err != nil {
			logger.Error("failed to batch mark IoCs as inactive",
				"source_id", s.id,
				"total_inactive", len(inactiveIoCs),
				"updated_count", inactiveResult,
				"error", err)
			stats.Errors++
		}

		logger.Info("batch marked IoCs as inactive",
			"source_id", s.id,
			"inactive_count", inactiveResult.Updated,
			"total_inactive", len(inactiveIoCs))
	}

	// Update source state using FeedStateRepository
	state := &FeedState{
		SourceID:      s.id,
		LastFetchedAt: time.Now(),
		ItemCount:     int64(len(entries)),
		ErrorCount:    int64(stats.Errors),
		UpdatedAt:     time.Now(),
	}

	if stats.Errors > 0 {
		state.LastError = "encountered errors during fetch"
	}

	if err := s.stateRepo.SaveFeedState(ctx, state); err != nil {
		// Don't fail the entire fetch if state save fails
		if !errors.Is(err, ErrFeedStateNotFound) {
			logger.Error("failed to save feed state",
				"source_id", s.id,
				"error", err)
		}
	}

	stats.ProcessingTime = time.Since(startTime)
	return stats, nil
}
