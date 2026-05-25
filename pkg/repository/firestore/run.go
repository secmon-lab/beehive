package firestore

import (
	"context"
	"errors"
	"slices"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

func (f *Firestore) GetRun(ctx context.Context, id types.RunID) (*model.Run, error) {
	doc, err := f.client.Collection(collectionRuns).Doc(string(id)).Get(ctx)
	if err != nil {
		return nil, wrapNotFound(err, "run", id)
	}
	var r model.Run
	if err := doc.DataTo(&r); err != nil {
		return nil, goerr.Wrap(err, "decode run", goerr.V("id", id))
	}
	return &r, nil
}

func (f *Firestore) CreateRun(ctx context.Context, r *model.Run) error {
	if r == nil || r.ID == "" {
		return goerr.New("run id is empty", goerr.T(errutil.TagInvalidInput))
	}
	if _, err := f.client.Collection(collectionRuns).Doc(string(r.ID)).Create(ctx, r); err != nil {
		return goerr.Wrap(err, "create run", goerr.V("id", r.ID))
	}
	return nil
}

func (f *Firestore) UpdateRun(ctx context.Context, r *model.Run) error {
	if r == nil || r.ID == "" {
		return goerr.New("run id is empty", goerr.T(errutil.TagInvalidInput))
	}
	if _, err := f.client.Collection(collectionRuns).Doc(string(r.ID)).Set(ctx, r); err != nil {
		return goerr.Wrap(err, "update run", goerr.V("id", r.ID))
	}
	return nil
}

// ListRecentRuns returns up to `limit` runs ordered by StartedAt
// descending. Needs a Firestore index on `StartedAt desc` — the
// startup-time error from Firestore will tell the operator which
// composite index to create.
func (f *Firestore) ListRecentRuns(ctx context.Context, limit int) ([]*model.Run, error) {
	if limit <= 0 {
		return nil, nil
	}
	iter := f.client.Collection(collectionRuns).
		OrderBy("StartedAt", firestore.Desc).
		Limit(limit).
		Documents(ctx)
	defer iter.Stop()

	out := make([]*model.Run, 0, limit)
	for {
		doc, err := iter.Next()
		if errors.Is(err, ErrIteratorDone()) {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "iterate runs")
		}
		var r model.Run
		if err := doc.DataTo(&r); err != nil {
			return nil, goerr.Wrap(err, "decode run", goerr.V("id", doc.Ref.ID))
		}
		out = append(out, &r)
	}
	return out, nil
}

// AppendRunSource transactionally appends to Run.Sources and SourceIDs.
func (f *Firestore) AppendRunSource(ctx context.Context, runID types.RunID, src *model.RunSource) error {
	if src == nil {
		return goerr.New("run source is nil", goerr.T(errutil.TagInvalidInput))
	}
	docRef := f.client.Collection(collectionRuns).Doc(string(runID))

	return f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			return wrapNotFound(err, "run", runID)
		}
		var run model.Run
		if err := snap.DataTo(&run); err != nil {
			return goerr.Wrap(err, "decode run", goerr.V("id", runID))
		}
		run.Sources = append(run.Sources, *src)
		if !slices.Contains(run.SourceIDs, src.SourceID) {
			run.SourceIDs = append(run.SourceIDs, src.SourceID)
		}
		if err := tx.Set(docRef, &run); err != nil {
			return goerr.Wrap(err, "set run", goerr.V("id", runID))
		}
		return nil
	})
}
