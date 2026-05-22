package memory

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// TryAcquireLock returns (true, nil) if the lock was created or stolen
// from an expired holder, (false, nil) if an active holder still owns it.
func (m *Memory) TryAcquireLock(_ context.Context, lock *model.Lock) (bool, error) {
	if lock == nil {
		return false, goerr.New("lock is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := lockKey(lock.Kind, lock.TargetID)
	now := nowUTC()
	if cur, ok := m.locks[key]; ok {
		if cur.ExpiresAt.After(now) {
			return false, nil
		}
	}
	c := *lock
	if c.AcquiredAt.IsZero() {
		c.AcquiredAt = now
	}
	if c.LastHeartbeatAt.IsZero() {
		c.LastHeartbeatAt = now
	}
	if c.ExpiresAt.IsZero() {
		c.ExpiresAt = now.Add(20 * time.Second)
	}
	m.locks[key] = &c
	return true, nil
}

func (m *Memory) RenewLock(_ context.Context, lock *model.Lock) error {
	if lock == nil {
		return goerr.New("lock is nil")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := lockKey(lock.Kind, lock.TargetID)
	cur, ok := m.locks[key]
	if !ok || cur.Holder != lock.Holder {
		return interfaces.ErrLockLost
	}
	now := nowUTC()
	cur.ExpiresAt = now.Add(20 * time.Second)
	cur.LastHeartbeatAt = now
	return nil
}

func (m *Memory) ReleaseLock(_ context.Context, kind, targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.locks, lockKey(kind, targetID))
	return nil
}
