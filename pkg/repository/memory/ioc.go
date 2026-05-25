package memory

import (
	"context"
	"sort"
	"time"

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

// BulkUpsertIoCs applies UpsertIoCWithRef to each pair sequentially.
// The in-memory backend has no notion of "batch" — it is cheap enough
// to loop — so this method exists solely to satisfy the repository
// contract that the Firestore backend exploits for BulkWriter.
//
// Returns the count of distinct IoC.IDs in the input (matches the
// Firestore implementation's notion of "intended persisted count" so
// the caller's bookkeeping does not depend on the backend).
func (m *Memory) BulkUpsertIoCs(ctx context.Context, pairs []model.IoCWithRef) (int, error) {
	seen := make(map[types.IoCID]bool, len(pairs))
	for i, p := range pairs {
		if p.IoC == nil || p.IoC.ID == "" {
			return 0, goerr.New("ioc id is empty",
				goerr.V("index", i),
				goerr.T(errutil.TagInvalidInput))
		}
		if err := m.UpsertIoCWithRef(ctx, p.IoC, p.Ref); err != nil {
			return 0, goerr.Wrap(err, "bulk upsert ioc",
				goerr.V("index", i),
				goerr.V("id", p.IoC.ID),
				goerr.V("count", len(pairs)))
		}
		seen[p.IoC.ID] = true
	}
	return len(seen), nil
}

// CountIoCsOfType walks the in-memory map and tallies entries of the
// requested type. The fan-out across all types lives in the usecase
// layer (so the implementation strategy — sequential or concurrent —
// is a single decision point).
func (m *Memory) CountIoCsOfType(_ context.Context, t types.IoCType) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var n int64
	for _, i := range m.iocs {
		if i.Type == t {
			n++
		}
	}
	return n, nil
}

// ListRecentIoCs returns up to `limit` IoCs ordered by LastSeenAt
// descending. Ties on LastSeenAt fall back to ID for a stable order.
func (m *Memory) ListRecentIoCs(ctx context.Context, limit int) ([]*model.IoC, error) {
	return m.ListRecentIoCsAfter(ctx, limit, nil)
}

// ListRecentIoCsAfter returns up to `limit` IoCs strictly older than
// the supplied LastSeenAt cursor. Pass nil to start at the head.
func (m *Memory) ListRecentIoCsAfter(_ context.Context, limit int, after *time.Time) ([]*model.IoC, error) {
	if limit <= 0 {
		return nil, nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*model.IoC, 0, len(m.iocs))
	for _, i := range m.iocs {
		if after != nil && !i.LastSeenAt.Before(*after) {
			continue
		}
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

// GetIoCCounts returns the in-memory counter snapshot. Defaults to the
// zero-valued shape when nothing has been saved yet.
func (m *Memory) GetIoCCounts(_ context.Context) (*model.IoCCounts, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.iocCounts == nil {
		return model.ZeroIoCCounts(), nil
	}
	out := &model.IoCCounts{
		ByType:    make(map[types.IoCType]int64, len(m.iocCounts.ByType)),
		Total:     m.iocCounts.Total,
		UpdatedAt: m.iocCounts.UpdatedAt,
	}
	for k, v := range m.iocCounts.ByType {
		out.ByType[k] = v
	}
	return out, nil
}

// SaveIoCCounts replaces the in-memory counter snapshot.
func (m *Memory) SaveIoCCounts(_ context.Context, counts *model.IoCCounts) error {
	if counts == nil {
		return goerr.New("counts is nil", goerr.T(errutil.TagInvalidInput))
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c := &model.IoCCounts{
		ByType:    make(map[types.IoCType]int64, len(counts.ByType)),
		Total:     counts.Total,
		UpdatedAt: counts.UpdatedAt,
	}
	for k, v := range counts.ByType {
		c.ByType[k] = v
	}
	m.iocCounts = c
	return nil
}
