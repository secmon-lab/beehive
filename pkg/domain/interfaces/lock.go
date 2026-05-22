package interfaces

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
)

// ErrLockLost is returned by LockManager.Renew (and surfaced through *Lock)
// when the underlying Firestore document has been taken over by another
// holder. Callers should abort their work and let the new holder proceed.
var ErrLockLost = goerr.New("lock lost")

// LockManager hands out heartbeat-renewed locks. Callers MUST defer
// (*AcquiredLock).Release on success.
type LockManager interface {
	// Acquire returns a renewing AcquiredLock if the lock could be taken,
	// or (nil, ErrLockBusy) if it is currently held by someone else.
	// Real errors (Firestore failures etc.) are wrapped via goerr.
	Acquire(ctx context.Context, kind, targetID string) (AcquiredLock, error)
}

// ErrLockBusy is returned when another holder currently owns the lock.
// Distinct from ErrLockLost (which only surfaces after a successful
// acquisition).
var ErrLockBusy = goerr.New("lock busy")

// AcquiredLock represents a lock currently held by this process. The
// heartbeat goroutine is started by Acquire and stopped by Release.
type AcquiredLock interface {
	Release(ctx context.Context) error
}
