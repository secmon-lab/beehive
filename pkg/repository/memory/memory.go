package memory

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/source/feed"
	"github.com/secmon-lab/beehive/pkg/domain/source/rss"
)

type Memory struct {
	iocs         map[string]*model.IoC         // key: IoC ID
	sourceStates map[string]*model.SourceState // key: Source ID
	rssStates    map[string]*rss.RSSState      // key: Source ID
	feedStates   map[string]*feed.FeedState    // key: Source ID
	mu           sync.RWMutex
}

var _ interfaces.IoCRepository = &Memory{}
var _ interfaces.SourceStateRepository = &Memory{}
var _ rss.RSSStateRepository = &Memory{}
var _ feed.FeedStateRepository = &Memory{}

func New() *Memory {
	return &Memory{
		iocs:         make(map[string]*model.IoC),
		sourceStates: make(map[string]*model.SourceState),
		rssStates:    make(map[string]*rss.RSSState),
		feedStates:   make(map[string]*feed.FeedState),
	}
}

// GetIoC retrieves an IoC by ID
func (m *Memory) GetIoC(ctx context.Context, id string) (*model.IoC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ioc, ok := m.iocs[id]
	if !ok {
		return nil, interfaces.ErrIoCNotFound
	}

	// Return a copy to prevent external modification
	iocCopy := *ioc
	return &iocCopy, nil
}

// ListIoCsBySource lists all IoCs for a given source
func (m *Memory) ListIoCsBySource(ctx context.Context, sourceID string) ([]*model.IoC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*model.IoC
	for _, ioc := range m.iocs {
		if ioc.SourceID == sourceID {
			iocCopy := *ioc
			result = append(result, &iocCopy)
		}
	}

	return result, nil
}

// ListAllIoCs lists all IoCs across all sources
func (m *Memory) ListAllIoCs(ctx context.Context) ([]*model.IoC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*model.IoC
	for _, ioc := range m.iocs {
		iocCopy := *ioc
		result = append(result, &iocCopy)
	}

	return result, nil
}

// ListIoCs lists IoCs with pagination and sorting
func (m *Memory) ListIoCs(ctx context.Context, opts *model.IoCListOptions) (*model.IoCConnection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get all IoCs
	allIoCs := make([]*model.IoC, 0, len(m.iocs))
	for _, ioc := range m.iocs {
		iocCopy := *ioc
		allIoCs = append(allIoCs, &iocCopy)
	}

	// Sort
	if opts != nil && opts.SortField != "" {
		sortIoCs(allIoCs, opts.SortField, opts.SortOrder)
	}

	total := len(allIoCs)

	// Apply pagination
	if opts != nil {
		offset := opts.Offset
		limit := opts.Limit

		if offset < 0 {
			offset = 0
		}
		if limit <= 0 {
			limit = 20 // default
		}

		start := offset
		end := offset + limit

		if start > total {
			start = total
		}
		if end > total {
			end = total
		}

		allIoCs = allIoCs[start:end]
	}

	return &model.IoCConnection{
		Items: allIoCs,
		Total: total,
	}, nil
}

// UpsertIoC inserts or updates an IoC
func (m *Memory) UpsertIoC(ctx context.Context, ioc *model.IoC) error {
	if err := model.ValidateIoC(ioc); err != nil {
		return goerr.Wrap(err, "invalid IoC")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Check if IoC already exists
	if existing, ok := m.iocs[ioc.ID]; ok {
		// Existing IoC - check if any field changed (for feed sources, update if anything changed)
		needsUpdate := existing.Description != ioc.Description ||
			existing.Status != ioc.Status ||
			existing.SourceURL != ioc.SourceURL ||
			existing.Context != ioc.Context
		if !needsUpdate {
			// Skip - no changes needed
			return nil
		}
		// Update: preserve FirstSeenAt, update UpdatedAt
		ioc.FirstSeenAt = existing.FirstSeenAt
		ioc.UpdatedAt = now
	} else {
		// New IoC
		ioc.FirstSeenAt = now
		ioc.UpdatedAt = now
	}

	// Store a copy to prevent external modification
	iocCopy := *ioc
	m.iocs[ioc.ID] = &iocCopy

	return nil
}

// BatchUpsertIoCs upserts multiple IoCs in a single operation
func (m *Memory) BatchUpsertIoCs(ctx context.Context, iocs []*model.IoC) (*interfaces.BatchUpsertResult, error) {
	result := &interfaces.BatchUpsertResult{}

	if len(iocs) == 0 {
		return result, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	for _, ioc := range iocs {
		if err := model.ValidateIoC(ioc); err != nil {
			return result, goerr.Wrap(err, "invalid IoC in batch", goerr.V("id", ioc.ID))
		}

		// Check if IoC already exists
		if existing, ok := m.iocs[ioc.ID]; ok {
			// Existing IoC - check if any field changed (for feed sources, update if anything changed)
			needsUpdate := existing.Description != ioc.Description ||
				existing.Status != ioc.Status ||
				existing.SourceURL != ioc.SourceURL ||
				existing.Context != ioc.Context
			if !needsUpdate {
				// Skip - no changes needed
				result.Unchanged++
				continue
			}
			// Update: preserve FirstSeenAt, update UpdatedAt
			ioc.FirstSeenAt = existing.FirstSeenAt
			ioc.UpdatedAt = now
			result.Updated++
		} else {
			// New IoC
			ioc.FirstSeenAt = now
			ioc.UpdatedAt = now
			result.Created++
		}

		// Store a copy to prevent external modification
		iocCopy := *ioc
		m.iocs[ioc.ID] = &iocCopy
	}

	return result, nil
}

// GetState retrieves source state by source ID
func (m *Memory) GetState(ctx context.Context, sourceID string) (*model.SourceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sourceStates[sourceID]
	if !ok {
		return nil, interfaces.ErrSourceStateNotFound
	}

	// Return a copy to prevent external modification
	stateCopy := *state
	return &stateCopy, nil
}

// SaveState saves or updates source state
func (m *Memory) SaveState(ctx context.Context, state *model.SourceState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state.UpdatedAt = time.Now()

	// Store a copy to prevent external modification
	stateCopy := *state
	m.sourceStates[state.SourceID] = &stateCopy

	return nil
}

// GetRSSState retrieves RSS state by source ID
func (m *Memory) GetRSSState(ctx context.Context, sourceID string) (*rss.RSSState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.rssStates[sourceID]
	if !ok {
		return nil, rss.ErrRSSStateNotFound
	}

	// Return a copy to prevent external modification
	stateCopy := *state
	return &stateCopy, nil
}

// SaveRSSState saves or updates RSS state
func (m *Memory) SaveRSSState(ctx context.Context, state *rss.RSSState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state.UpdatedAt = time.Now()

	// Store a copy to prevent external modification
	stateCopy := *state
	m.rssStates[state.SourceID] = &stateCopy

	return nil
}

// GetFeedState retrieves Feed state by source ID
func (m *Memory) GetFeedState(ctx context.Context, sourceID string) (*feed.FeedState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.feedStates[sourceID]
	if !ok {
		return nil, feed.ErrFeedStateNotFound
	}

	// Return a copy to prevent external modification
	stateCopy := *state
	return &stateCopy, nil
}

// SaveFeedState saves or updates Feed state
func (m *Memory) SaveFeedState(ctx context.Context, state *feed.FeedState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	state.UpdatedAt = time.Now()

	// Store a copy to prevent external modification
	stateCopy := *state
	m.feedStates[state.SourceID] = &stateCopy

	return nil
}

// FindNearestIoCs performs in-memory vector similarity search
// This is a simple brute-force implementation for testing
func (m *Memory) FindNearestIoCs(ctx context.Context, queryVector []float32, limit int) ([]*model.IoC, error) {
	if len(queryVector) != model.EmbeddingDimension {
		return nil, goerr.New("invalid query vector dimension",
			goerr.V("expected", model.EmbeddingDimension),
			goerr.V("actual", len(queryVector)))
	}

	if limit <= 0 {
		limit = 10
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate similarity for all IoCs
	type iocWithSimilarity struct {
		ioc        *model.IoC
		similarity float64
	}

	var candidates []iocWithSimilarity

	for _, ioc := range m.iocs {
		if len(ioc.Embedding) != model.EmbeddingDimension {
			continue // Skip IoCs without valid embeddings
		}

		// Calculate cosine similarity
		similarity := cosineSimilarity(queryVector, ioc.Embedding)
		candidates = append(candidates, iocWithSimilarity{
			ioc:        ioc,
			similarity: similarity,
		})
	}

	// Sort by similarity (descending)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].similarity < candidates[j].similarity {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Take top N
	resultCount := limit
	if resultCount > len(candidates) {
		resultCount = len(candidates)
	}

	results := make([]*model.IoC, resultCount)
	for i := 0; i < resultCount; i++ {
		// Return copies to prevent external modification
		iocCopy := *candidates[i].ioc
		results[i] = &iocCopy
	}

	return results, nil
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
