package memory

import (
	"context"
	"reflect"

	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func (m *Memory) GetSourceState(_ context.Context, id types.SourceID) (*model.SourceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.states[id]
	if !ok {
		return nil, notFound("source state", id)
	}
	return cloneSourceState(s), nil
}

func (m *Memory) ListSourceStates(_ context.Context) ([]*model.SourceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*model.SourceState, 0, len(m.states))
	for _, s := range m.states {
		out = append(out, cloneSourceState(s))
	}
	return out, nil
}

func (m *Memory) UpdateSourceState(_ context.Context, state *model.SourceState) error {
	if state == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prev := m.states[state.ID]
	if prev != nil && reflect.DeepEqual(prev, state) {
		// Write-cost optimisation: skip identical writes (mirrors the
		// Firestore implementation's behaviour).
		return nil
	}
	c := *state
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = nowUTC()
	}
	m.states[state.ID] = &c
	return nil
}

func (m *Memory) SetEnabledOverride(_ context.Context, id types.SourceID, override types.EnabledOverride) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.states[id]
	if !ok {
		// Create a fresh state document if none exists yet — the override
		// itself is reason enough to start tracking state.
		s = &model.SourceState{ID: id}
		m.states[id] = s
	}
	if s.EnabledOverride == override {
		return nil
	}
	s.EnabledOverride = override
	s.UpdatedAt = nowUTC()
	return nil
}
