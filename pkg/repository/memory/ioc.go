package memory

import (
	"context"
	"sort"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

func (m *Memory) GetIoC(_ context.Context, id types.IoCID) (*model.IoC, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	i, ok := m.iocs[id]
	if !ok {
		return nil, notFound("ioc", id)
	}
	return cloneIoC(i), nil
}

// UpsertIoCWithRef implements the write-cost-optimised upsert: Raw is set
// only on first creation, LastSeenAt is bumped on subsequent occurrences,
// and the ref document is skipped if it already exists.
func (m *Memory) UpsertIoCWithRef(_ context.Context, ioc *model.IoC, ref *model.IoCRef) error {
	if ioc == nil || ioc.ID == "" {
		return goerr.New("ioc id is empty", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	cur, exists := m.iocs[ioc.ID]
	if !exists {
		c := *ioc
		m.iocs[ioc.ID] = &c
	} else {
		// Update only LastSeenAt; Raw stays immutable.
		if !ioc.LastSeenAt.IsZero() && ioc.LastSeenAt.After(cur.LastSeenAt) {
			cur.LastSeenAt = ioc.LastSeenAt
		}
	}

	if ref == nil {
		return nil
	}
	refID := model.ComputeRefID(ref.SourceID, ref.ArticleID, ref.RunID)
	bucket, ok := m.refs[ioc.ID]
	if !ok {
		bucket = make(map[types.RefID]*model.IoCRef)
		m.refs[ioc.ID] = bucket
	}
	if _, dup := bucket[refID]; dup {
		// Identical ref already recorded — skip the write.
		return nil
	}
	rc := *ref
	bucket[refID] = &rc
	return nil
}

// ListRecentIoCs returns up to `limit` IoCs ordered by LastSeenAt
// descending. Ties on LastSeenAt fall back to ID for a stable order.
func (m *Memory) ListRecentIoCs(_ context.Context, limit int) ([]*model.IoC, error) {
	if limit <= 0 {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*model.IoC, 0, len(m.iocs))
	for _, i := range m.iocs {
		out = append(out, cloneIoC(i))
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].LastSeenAt.Equal(out[j].LastSeenAt) {
			return out[i].LastSeenAt.After(out[j].LastSeenAt)
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
