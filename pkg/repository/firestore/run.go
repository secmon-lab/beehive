package firestore

import (
	"context"
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
