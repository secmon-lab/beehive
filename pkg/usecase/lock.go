package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/async"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// LockHeartbeatInterval is exported for tests so they can override it
// to drive the heartbeat loop quickly.
//
// The Firestore lock TTL is 20s (see pkg/repository/firestore.lockTTL);
// the heartbeat MUST fire well before that to absorb scheduling jitter
// and network latency. Using TTL/3 leaves headroom for one missed tick
// without losing the lock.
var LockHeartbeatInterval = 7 * time.Second

// LockManager is a thin LockManager that drives the underlying
// repository, owning the heartbeat goroutine.
type LockManager struct {
	repo   interfaces.LockRepository
	holder string
}

// NewLockManager builds a LockManager with the given holder identity.
// `holder` should already be unique (hostname + pid + random suffix).
func NewLockManager(repo interfaces.LockRepository, holder string) *LockManager {
	return &LockManager{repo: repo, holder: holder}
}

// Acquire tries to take the lock and, on success, starts a heartbeat
// goroutine that bumps ExpiresAt on a fixed interval. The returned
// AcquiredLock must be `defer`-released.
func (m *LockManager) Acquire(ctx context.Context, kind, targetID string) (interfaces.AcquiredLock, error) {
	if m == nil || m.repo == nil {
		return nil, goerr.New("lock manager not initialised")
	}
	lock := &model.Lock{
		Kind:       kind,
		TargetID:   targetID,
		Holder:     m.holder,
		AcquiredAt: time.Now().UTC(),
	}
	ok, err := m.repo.TryAcquireLock(ctx, lock)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, interfaces.ErrLockBusy
	}

	hbCtx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)
	al := &acquired{
		repo:   m.repo,
		lock:   lock,
		cancel: cancel,
		done:   wg,
	}
	async.Dispatch(hbCtx, func(ctx context.Context) error {
		defer wg.Done()
		ticker := time.NewTicker(LockHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := m.repo.RenewLock(ctx, lock); err != nil {
					return goerr.Wrap(err, "renew lock",
						goerr.V("kind", kind),
						goerr.V("target_id", targetID),
						goerr.T(errutil.TagBusy))
				}
			}
		}
	})
	return al, nil
}

type acquired struct {
	repo     interfaces.LockRepository
	lock     *model.Lock
	cancel   context.CancelFunc
	done     *sync.WaitGroup
	released bool
}

func (a *acquired) Release(ctx context.Context) error {
	if a == nil || a.released {
		return nil
	}
	a.released = true
	a.cancel()
	a.done.Wait()
	return a.repo.ReleaseLock(ctx, a.lock.Kind, a.lock.TargetID)
}
