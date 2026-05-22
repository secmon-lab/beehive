package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// lockTTL is the heartbeat-based lock TTL. Kept equal to the heartbeat
// interval so a single missed heartbeat lapses the lock (Fetch is
// idempotent, so the cost of a redundant run is small — see CLAUDE.md §6).
const lockTTL = 20 * time.Second

func (f *Firestore) lockDoc(kind, targetID string) *firestore.DocumentRef {
	return f.client.
		Collection(collectionLocks).Doc(kind).
		Collection(subLockEntries).Doc(targetID)
}

func (f *Firestore) TryAcquireLock(ctx context.Context, lock *model.Lock) (bool, error) {
	if lock == nil {
		return false, goerr.New("lock is nil")
	}
	docRef := f.lockDoc(lock.Kind, lock.TargetID)
	now := time.Now().UTC()
	acquired := false

	err := f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err == nil {
			var cur model.Lock
			if err := snap.DataTo(&cur); err != nil {
				return goerr.Wrap(err, "decode lock")
			}
			if cur.ExpiresAt.After(now) {
				acquired = false
				return nil
			}
		} else if !isNotFound(err) {
			return goerr.Wrap(err, "read lock")
		}

		updated := *lock
		if updated.AcquiredAt.IsZero() {
			updated.AcquiredAt = now
		}
		updated.LastHeartbeatAt = now
		updated.ExpiresAt = now.Add(lockTTL)
		if err := tx.Set(docRef, &updated); err != nil {
			return goerr.Wrap(err, "set lock")
		}
		acquired = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return acquired, nil
}

func (f *Firestore) RenewLock(ctx context.Context, lock *model.Lock) error {
	if lock == nil {
		return goerr.New("lock is nil")
	}
	docRef := f.lockDoc(lock.Kind, lock.TargetID)
	now := time.Now().UTC()
	return f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			return interfaces.ErrLockLost
		}
		var cur model.Lock
		if err := snap.DataTo(&cur); err != nil {
			return goerr.Wrap(err, "decode lock for renew")
		}
		if cur.Holder != lock.Holder {
			return interfaces.ErrLockLost
		}
		if err := tx.Update(docRef, []firestore.Update{
			{Path: "ExpiresAt", Value: now.Add(lockTTL)},
			{Path: "LastHeartbeatAt", Value: now},
		}); err != nil {
			return goerr.Wrap(err, "update lock heartbeat")
		}
		return nil
	})
}

func (f *Firestore) ReleaseLock(ctx context.Context, kind, targetID string) error {
	if _, err := f.lockDoc(kind, targetID).Delete(ctx); err != nil {
		return goerr.Wrap(err, "delete lock",
			goerr.V("kind", kind),
			goerr.V("target_id", targetID))
	}
	return nil
}
