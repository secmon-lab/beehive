package firestore

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// ListRecentIoCs returns up to `limit` IoCs ordered by LastSeenAt
// descending. Needs a Firestore index on `LastSeenAt desc` — the
// startup-time error from Firestore will tell the operator which
// composite index to create.
func (f *Firestore) ListRecentIoCs(ctx context.Context, limit int) ([]*model.IoC, error) {
	if limit <= 0 {
		return nil, nil
	}
	iter := f.client.Collection(collectionIoCs).
		OrderBy("LastSeenAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	out := make([]*model.IoC, 0, limit)
	for {
		doc, err := iter.Next()
		if errors.Is(err, ErrIteratorDone()) {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "iterate iocs")
		}
		var i model.IoC
		if err := doc.DataTo(&i); err != nil {
			return nil, goerr.Wrap(err, "decode ioc", goerr.V("id", doc.Ref.ID))
		}
		out = append(out, &i)
	}
	return out, nil
}

func (f *Firestore) GetIoC(ctx context.Context, id types.IoCID) (*model.IoC, error) {
	doc, err := f.client.Collection(collectionIoCs).Doc(string(id)).Get(ctx)
	if err != nil {
		return nil, wrapNotFound(err, "ioc", id)
	}
	var i model.IoC
	if err := doc.DataTo(&i); err != nil {
		return nil, goerr.Wrap(err, "decode ioc", goerr.V("id", id))
	}
	return &i, nil
}

// UpsertIoCWithRef implements the cost-optimised upsert: new IoC -> full
// create (Raw fixed); existing IoC -> only bump LastSeenAt; new ref ->
// create; existing ref -> no-op.
//
// Firestore transactions require **all reads to happen before all
// writes** ("firestore: read after write in transaction" otherwise), so
// we gather both snapshots first and execute every mutation at the end.
func (f *Firestore) UpsertIoCWithRef(ctx context.Context, ioc *model.IoC, ref *model.IoCRef) error {
	if ioc == nil || ioc.ID == "" {
		return goerr.New("ioc id is empty", goerr.T(errutil.TagInvalidInput))
	}
	iocRef := f.client.Collection(collectionIoCs).Doc(string(ioc.ID))

	var refRef *firestore.DocumentRef
	if ref != nil {
		refID := model.ComputeRefID(ref.SourceID, ref.ArticleID, ref.RunID)
		refRef = iocRef.Collection(collectionRefs).Doc(string(refID))
	}

	return f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// ---- reads (must happen before any write) ----
		iocSnap, iocErr := tx.Get(iocRef)
		if iocErr != nil && !isNotFound(iocErr) {
			return goerr.Wrap(iocErr, "get ioc for upsert", goerr.V("id", ioc.ID))
		}

		var refExists bool
		if refRef != nil {
			_, rerr := tx.Get(refRef)
			switch {
			case rerr == nil:
				refExists = true
			case isNotFound(rerr):
				refExists = false
			default:
				return goerr.Wrap(rerr, "get existing ref")
			}
		}

		// ---- writes ----
		if iocErr == nil {
			// Existing IoC — bump LastSeenAt only if it is strictly newer.
			var prev model.IoC
			if err := iocSnap.DataTo(&prev); err != nil {
				return goerr.Wrap(err, "decode existing ioc", goerr.V("id", ioc.ID))
			}
			if !ioc.LastSeenAt.IsZero() && ioc.LastSeenAt.After(prev.LastSeenAt) {
				if err := tx.Update(iocRef, []firestore.Update{
					{Path: "LastSeenAt", Value: ioc.LastSeenAt},
				}); err != nil {
					return goerr.Wrap(err, "update LastSeenAt")
				}
			}
		} else {
			// New IoC — create with the full payload (Raw included).
			if err := tx.Create(iocRef, ioc); err != nil {
				return goerr.Wrap(err, "create ioc", goerr.V("id", ioc.ID))
			}
		}

		if refRef != nil && !refExists {
			if err := tx.Create(refRef, ref); err != nil {
				return goerr.Wrap(err, "create ref")
			}
		}
		return nil
	})
}
