package memory

import (
	"context"
	"slices"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

func (m *Memory) GetRun(_ context.Context, id types.RunID) (*model.Run, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[id]
	if !ok {
		return nil, notFound("run", id)
	}
	return cloneRun(r), nil
}

func (m *Memory) CreateRun(_ context.Context, r *model.Run) error {
	if r == nil || r.ID == "" {
		return goerr.New("run id is empty", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.runs[r.ID]; exists {
		return goerr.New("run already exists", goerr.V("id", r.ID), goerr.T(errutil.TagConflict))
	}
	c := cloneRun(r)
	m.runs[r.ID] = c
	return nil
}

func (m *Memory) UpdateRun(_ context.Context, r *model.Run) error {
	if r == nil || r.ID == "" {
		return goerr.New("run id is empty", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.runs[r.ID]; !ok {
		return notFound("run", r.ID)
	}
	m.runs[r.ID] = cloneRun(r)
	return nil
}

func (m *Memory) AppendRunSource(_ context.Context, runID types.RunID, src *model.RunSource) error {
	if src == nil {
		return goerr.New("run source is nil", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[runID]
	if !ok {
		return notFound("run", runID)
	}
	run.Sources = append(run.Sources, *src)
	if !slices.Contains(run.SourceIDs, src.SourceID) {
		run.SourceIDs = append(run.SourceIDs, src.SourceID)
	}
	return nil
}
