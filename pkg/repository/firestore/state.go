package firestore

import (
	"context"
	"errors"
	"reflect"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

func (f *Firestore) GetSourceState(ctx context.Context, id types.SourceID) (*model.SourceState, error) {
	doc, err := f.client.Collection(collectionStates).Doc(string(id)).Get(ctx)
	if err != nil {
		return nil, wrapNotFound(err, "source state", id)
	}
	var s model.SourceState
	if err := doc.DataTo(&s); err != nil {
		return nil, goerr.Wrap(err, "decode state", goerr.V("id", id))
	}
	// Fill ID from doc path so callers always see a populated value even
	// if the stored document forgot to set it.
	s.ID = id
	return &s, nil
}

func (f *Firestore) ListSourceStates(ctx context.Context) ([]*model.SourceState, error) {
	iter := f.client.Collection(collectionStates).Documents(ctx)
	defer iter.Stop()

	var out []*model.SourceState
	for {
		doc, err := iter.Next()
		if errors.Is(err, ErrIteratorDone()) {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "iterate states")
		}
		var s model.SourceState
		if err := doc.DataTo(&s); err != nil {
			return nil, goerr.Wrap(err, "decode state", goerr.V("id", doc.Ref.ID))
		}
		s.ID = types.SourceID(doc.Ref.ID)
		out = append(out, &s)
	}
	return out, nil
}

func (f *Firestore) UpdateSourceState(ctx context.Context, state *model.SourceState) error {
	if state == nil {
		return nil
	}
	docRef := f.client.Collection(collectionStates).Doc(string(state.ID))

	// Read the existing document (if any) to skip identical-value writes.
	snap, err := docRef.Get(ctx)
	if err != nil && !isNotFound(err) {
		return goerr.Wrap(err, "read state for diff", goerr.V("id", state.ID))
	}
	if snap != nil && snap.Exists() {
		var prev model.SourceState
		if err := snap.DataTo(&prev); err == nil {
			prev.ID = state.ID
			if reflect.DeepEqual(&prev, state) {
				return nil
			}
		}
	}
	if state.UpdatedAt.IsZero() {
		state.UpdatedAt = time.Now().UTC()
	}
	if _, err := docRef.Set(ctx, state); err != nil {
		return goerr.Wrap(err, "set state", goerr.V("id", state.ID))
	}
	return nil
}

func (f *Firestore) SetEnabledOverride(ctx context.Context, id types.SourceID, override types.EnabledOverride) error {
	docRef := f.client.Collection(collectionStates).Doc(string(id))
	now := time.Now().UTC()
	updates := []firestore.Update{
		{Path: "EnabledOverride", Value: override},
		{Path: "UpdatedAt", Value: now},
		{Path: "ID", Value: id},
	}
	// Try Update first; if the document does not exist, fall back to Set
	// with a fresh state. We do not use MergeAll so unrelated fields
	// would not be silently zeroed out (there are none on creation, but
	// the rule still stands).
	if _, err := docRef.Update(ctx, updates); err != nil {
		if !isNotFound(err) {
			return goerr.Wrap(err, "update override", goerr.V("id", id))
		}
		state := &model.SourceState{
			ID:              id,
			EnabledOverride: override,
			UpdatedAt:       now,
		}
		if _, err := docRef.Create(ctx, state); err != nil {
			return goerr.Wrap(err, "create state for override", goerr.V("id", id))
		}
	}
	return nil
}
