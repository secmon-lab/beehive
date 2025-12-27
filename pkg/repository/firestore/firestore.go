package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionIoCs         = "iocs"
	collectionSourceStates = "source_states"
)

type Firestore struct {
	client *firestore.Client
}

var _ interfaces.IoCRepository = &Firestore{}
var _ interfaces.SourceStateRepository = &Firestore{}

func New(ctx context.Context, projectID string, opts ...Option) (*Firestore, error) {
	var options options
	for _, opt := range opts {
		opt(&options)
	}

	var client *firestore.Client
	var err error

	if options.databaseID != "" {
		// Use specific database
		client, err = firestore.NewClientWithDatabase(ctx, projectID, options.databaseID)
	} else {
		// Use default database
		client, err = firestore.NewClient(ctx, projectID)
	}

	if err != nil {
		return nil, goerr.Wrap(err, "failed to create firestore client",
			goerr.V("project_id", projectID),
			goerr.V("database_id", options.databaseID))
	}

	return &Firestore{
		client: client,
	}, nil
}

type options struct {
	databaseID string
}

type Option func(*options)

// WithDatabaseID sets the Firestore database ID
func WithDatabaseID(databaseID string) Option {
	return func(o *options) {
		o.databaseID = databaseID
	}
}

func (f *Firestore) Close() error {
	if f.client != nil {
		return f.client.Close()
	}
	return nil
}

// GetIoC retrieves an IoC by ID
func (f *Firestore) GetIoC(ctx context.Context, id string) (*model.IoC, error) {
	doc, err := f.client.Collection(collectionIoCs).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(interfaces.ErrIoCNotFound, "IoC not found", goerr.V("id", id))
		}
		return nil, goerr.Wrap(err, "failed to get IoC from firestore", goerr.V("id", id))
	}

	var ioc model.IoC
	if err := doc.DataTo(&ioc); err != nil {
		return nil, goerr.Wrap(err, "failed to decode IoC", goerr.V("id", id))
	}

	return &ioc, nil
}

// ListIoCsBySource lists all IoCs for a given source
func (f *Firestore) ListIoCsBySource(ctx context.Context, sourceID string) ([]*model.IoC, error) {
	docs, err := f.client.Collection(collectionIoCs).
		Where("SourceID", "==", sourceID).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list IoCs by source",
			goerr.V("source_id", sourceID))
	}

	var iocs []*model.IoC
	for _, doc := range docs {
		var ioc model.IoC
		if err := doc.DataTo(&ioc); err != nil {
			return nil, goerr.Wrap(err, "failed to decode IoC",
				goerr.V("doc_id", doc.Ref.ID))
		}
		iocs = append(iocs, &ioc)
	}

	return iocs, nil
}

// UpsertIoC inserts or updates an IoC
func (f *Firestore) UpsertIoC(ctx context.Context, ioc *model.IoC) error {
	if err := model.ValidateIoC(ioc); err != nil {
		return goerr.Wrap(err, "invalid IoC")
	}

	docRef := f.client.Collection(collectionIoCs).Doc(ioc.ID)

	// Check if document exists
	doc, err := docRef.Get(ctx)
	now := time.Now()

	if err != nil {
		if status.Code(err) == codes.NotFound {
			// New IoC
			ioc.FirstSeenAt = now
			ioc.UpdatedAt = now
		} else {
			return goerr.Wrap(err, "failed to check IoC existence",
				goerr.V("id", ioc.ID))
		}
	} else {
		// Existing IoC
		var existing model.IoC
		if err := doc.DataTo(&existing); err != nil {
			return goerr.Wrap(err, "failed to decode existing IoC",
				goerr.V("id", ioc.ID))
		}
		// Only update if description or status changed
		needsUpdate := existing.Description != ioc.Description || existing.Status != ioc.Status
		if !needsUpdate {
			// Skip - no changes needed
			return nil
		}
		// Update: preserve FirstSeenAt, update UpdatedAt
		ioc.FirstSeenAt = existing.FirstSeenAt
		ioc.UpdatedAt = now
	}

	// Save to Firestore
	if _, err := docRef.Set(ctx, ioc); err != nil {
		return goerr.Wrap(err, "failed to save IoC to firestore",
			goerr.V("id", ioc.ID))
	}

	return nil
}

// BatchUpsertIoCs upserts multiple IoCs in batches
// Firestore batch write limit is 500 operations per batch
func (f *Firestore) BatchUpsertIoCs(ctx context.Context, iocs []*model.IoC) (*interfaces.BatchUpsertResult, error) {
	result := &interfaces.BatchUpsertResult{}

	if len(iocs) == 0 {
		return result, nil
	}

	const batchSize = 500

	// Process in batches of 500 (Firestore limit)
	for i := 0; i < len(iocs); i += batchSize {
		end := i + batchSize
		if end > len(iocs) {
			end = len(iocs)
		}

		batch := iocs[i:end]
		batchResult, err := f.writeBatch(ctx, batch)
		result.Created += batchResult.Created
		result.Updated += batchResult.Updated
		result.Unchanged += batchResult.Unchanged
		if err != nil {
			return result, goerr.Wrap(err, "batch write failed",
				goerr.V("batch_start", i),
				goerr.V("batch_size", len(batch)),
				goerr.V("result", result))
		}
	}

	return result, nil
}

// writeBatch writes a single batch of IoCs (max 500)
func (f *Firestore) writeBatch(ctx context.Context, iocs []*model.IoC) (*interfaces.BatchUpsertResult, error) {
	result := &interfaces.BatchUpsertResult{}
	bulkWriter := f.client.BulkWriter(ctx)
	now := time.Now()

	// First, fetch existing documents to preserve FirstSeenAt
	existingMap := make(map[string]*model.IoC)
	for _, ioc := range iocs {
		doc, err := f.client.Collection(collectionIoCs).Doc(ioc.ID).Get(ctx)
		if err == nil {
			var existing model.IoC
			if err := doc.DataTo(&existing); err == nil {
				existingMap[ioc.ID] = &existing
			}
		}
		// Ignore not found errors - new IoCs
	}

	// Prepare bulk writes
	for _, ioc := range iocs {
		if err := model.ValidateIoC(ioc); err != nil {
			return result, goerr.Wrap(err, "invalid IoC in batch", goerr.V("id", ioc.ID))
		}

		docRef := f.client.Collection(collectionIoCs).Doc(ioc.ID)

		if existing, ok := existingMap[ioc.ID]; ok {
			// Existing IoC - only update if description or status changed
			needsUpdate := existing.Description != ioc.Description || existing.Status != ioc.Status
			if !needsUpdate {
				// Skip - no changes needed
				result.Unchanged++
				continue
			}
			// Update: preserve FirstSeenAt, update UpdatedAt
			ioc.FirstSeenAt = existing.FirstSeenAt
			ioc.UpdatedAt = now
			result.Updated++
		} else {
			// New IoC
			ioc.FirstSeenAt = now
			ioc.UpdatedAt = now
			result.Created++
		}

		if _, err := bulkWriter.Set(docRef, ioc); err != nil {
			bulkWriter.End()
			return result, goerr.Wrap(err, "failed to add document to bulk writer",
				goerr.V("ioc_id", ioc.ID))
		}
	}

	// Flush and wait for all operations to complete
	bulkWriter.Flush()
	bulkWriter.End()

	return result, nil
}

// GetState retrieves source state by source ID
func (f *Firestore) GetState(ctx context.Context, sourceID string) (*model.SourceState, error) {
	doc, err := f.client.Collection(collectionSourceStates).Doc(sourceID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(interfaces.ErrSourceStateNotFound, "source state not found", goerr.V("source_id", sourceID))
		}
		return nil, goerr.Wrap(err, "failed to get source state from firestore",
			goerr.V("source_id", sourceID))
	}

	var state model.SourceState
	if err := doc.DataTo(&state); err != nil {
		return nil, goerr.Wrap(err, "failed to decode source state",
			goerr.V("source_id", sourceID))
	}

	return &state, nil
}

// SaveState saves or updates source state
func (f *Firestore) SaveState(ctx context.Context, state *model.SourceState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	state.UpdatedAt = time.Now()

	docRef := f.client.Collection(collectionSourceStates).Doc(state.SourceID)
	if _, err := docRef.Set(ctx, state); err != nil {
		return goerr.Wrap(err, "failed to save source state to firestore",
			goerr.V("source_id", state.SourceID))
	}

	return nil
}
