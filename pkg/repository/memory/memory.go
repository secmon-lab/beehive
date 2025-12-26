package memory

import (
	"context"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

var (
	errIoCNotFound   = goerr.New("IoC not found")
	errStateNotFound = goerr.New("source state not found")
)

type Memory struct {
	iocs         map[string]*model.IoC         // key: IoC ID
	sourceStates map[string]*model.SourceState // key: Source ID
	mu           sync.RWMutex
}

var _ interfaces.IoCRepository = &Memory{}
var _ interfaces.SourceStateRepository = &Memory{}

func New() *Memory {
	return &Memory{
		iocs:         make(map[string]*model.IoC),
		sourceStates: make(map[string]*model.SourceState),
	}
}

// GetIoC retrieves an IoC by ID
func (m *Memory) GetIoC(ctx context.Context, id string) (*model.IoC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ioc, ok := m.iocs[id]
	if !ok {
		return nil, goerr.Wrap(errIoCNotFound, "IoC not found", goerr.V("id", id))
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
		// Existing IoC - only update if description or status changed
		needsUpdate := existing.Description != ioc.Description || existing.Status != ioc.Status
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
			// Existing IoC - only update if description or status changed
			needsUpdate := existing.Description != ioc.Description || existing.Status != ioc.Status
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
		return nil, goerr.Wrap(errStateNotFound, "source state not found", goerr.V("source_id", sourceID))
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
