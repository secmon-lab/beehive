package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	firestorepb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/source/feed"
	"github.com/secmon-lab/beehive/pkg/domain/source/rss"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	collectionIoCs         = "iocs"
	collectionSources      = "sources"
	subcollectionHistories = "histories"
)

// firestoreSourceState is the Firestore representation of SourceState
// This keeps the domain model free from infrastructure concerns
type firestoreSourceState struct {
	SourceID      string    `firestore:"source_id"`
	LastFetchedAt time.Time `firestore:"last_fetched_at"`
	LastItemID    string    `firestore:"last_item_id"`
	LastItemDate  time.Time `firestore:"last_item_date"`
	ItemCount     int64     `firestore:"item_count"`
	ErrorCount    int64     `firestore:"error_count"`
	LastError     string    `firestore:"last_error"`
	UpdatedAt     time.Time `firestore:"updated_at"`
}

// toFirestoreSourceState converts domain model to Firestore representation
func toFirestoreSourceState(state *model.SourceState) *firestoreSourceState {
	return &firestoreSourceState{
		SourceID:      state.SourceID,
		LastFetchedAt: state.LastFetchedAt,
		LastItemID:    state.LastItemID,
		LastItemDate:  state.LastItemDate,
		ItemCount:     state.ItemCount,
		ErrorCount:    state.ErrorCount,
		LastError:     state.LastError,
		UpdatedAt:     state.UpdatedAt,
	}
}

// toDomainSourceState converts Firestore representation to domain model
func toDomainSourceState(fs *firestoreSourceState) *model.SourceState {
	return &model.SourceState{
		SourceID:      fs.SourceID,
		LastFetchedAt: fs.LastFetchedAt,
		LastItemID:    fs.LastItemID,
		LastItemDate:  fs.LastItemDate,
		ItemCount:     fs.ItemCount,
		ErrorCount:    fs.ErrorCount,
		LastError:     fs.LastError,
		UpdatedAt:     fs.UpdatedAt,
	}
}

// firestoreFetchError is the Firestore representation of FetchError
type firestoreFetchError struct {
	Message string            `firestore:"message"`
	Values  map[string]string `firestore:"values"`
}

// firestoreHistory is the Firestore representation of History
// This keeps the domain model free from infrastructure concerns
type firestoreHistory struct {
	ID             string                `firestore:"id"`
	SourceID       string                `firestore:"source_id"`
	SourceType     string                `firestore:"source_type"`
	Status         string                `firestore:"status"`
	StartedAt      time.Time             `firestore:"started_at"`
	CompletedAt    time.Time             `firestore:"completed_at"`
	ProcessingTime int64                 `firestore:"processing_time"` // nanoseconds
	ItemsFetched   int                   `firestore:"items_fetched"`
	IoCsExtracted  int                   `firestore:"iocs_extracted"`
	IoCsCreated    int                   `firestore:"iocs_created"`
	IoCsUpdated    int                   `firestore:"iocs_updated"`
	IoCsUnchanged  int                   `firestore:"iocs_unchanged"`
	ErrorCount     int                   `firestore:"error_count"`
	Errors         []firestoreFetchError `firestore:"errors"`
	CreatedAt      time.Time             `firestore:"created_at"`
}

// toFirestoreHistory converts domain model to Firestore representation
func toFirestoreHistory(history *model.History) *firestoreHistory {
	fh := &firestoreHistory{
		ID:             history.ID,
		SourceID:       history.SourceID,
		SourceType:     string(history.SourceType),
		Status:         string(history.Status),
		StartedAt:      history.StartedAt,
		CompletedAt:    history.CompletedAt,
		ProcessingTime: int64(history.ProcessingTime),
		ItemsFetched:   history.ItemsFetched,
		IoCsExtracted:  history.IoCsExtracted,
		IoCsCreated:    history.IoCsCreated,
		IoCsUpdated:    history.IoCsUpdated,
		IoCsUnchanged:  history.IoCsUnchanged,
		ErrorCount:     history.ErrorCount,
		CreatedAt:      history.CreatedAt,
	}

	if history.Errors != nil {
		fh.Errors = make([]firestoreFetchError, len(history.Errors))
		for i, err := range history.Errors {
			fh.Errors[i] = firestoreFetchError{
				Message: err.Message,
				Values:  err.Values,
			}
		}
	}

	return fh
}

// toDomainHistory converts Firestore representation to domain model
func toDomainHistory(fh *firestoreHistory) *model.History {
	history := &model.History{
		ID:             fh.ID,
		SourceID:       fh.SourceID,
		SourceType:     model.SourceType(fh.SourceType),
		Status:         model.FetchStatus(fh.Status),
		StartedAt:      fh.StartedAt,
		CompletedAt:    fh.CompletedAt,
		ProcessingTime: time.Duration(fh.ProcessingTime),
		ItemsFetched:   fh.ItemsFetched,
		IoCsExtracted:  fh.IoCsExtracted,
		IoCsCreated:    fh.IoCsCreated,
		IoCsUpdated:    fh.IoCsUpdated,
		IoCsUnchanged:  fh.IoCsUnchanged,
		ErrorCount:     fh.ErrorCount,
		CreatedAt:      fh.CreatedAt,
	}

	if fh.Errors != nil {
		history.Errors = make([]*model.FetchError, len(fh.Errors))
		for i, err := range fh.Errors {
			history.Errors[i] = &model.FetchError{
				Message: err.Message,
				Values:  err.Values,
			}
		}
	}

	return history
}

type Firestore struct {
	client *firestore.Client
}

var _ interfaces.IoCRepository = &Firestore{}
var _ interfaces.SourceStateRepository = &Firestore{}
var _ interfaces.HistoryRepository = &Firestore{}
var _ rss.RSSStateRepository = &Firestore{}
var _ feed.FeedStateRepository = &Firestore{}

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

// ListAllIoCs lists all IoCs across all sources
func (f *Firestore) ListAllIoCs(ctx context.Context) ([]*model.IoC, error) {
	docs, err := f.client.Collection(collectionIoCs).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list all IoCs")
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

// ListIoCs lists IoCs with pagination and sorting options
func (f *Firestore) ListIoCs(ctx context.Context, opts *model.IoCListOptions) (*model.IoCConnection, error) {
	// Start with base query
	query := f.client.Collection(collectionIoCs).Query

	// Apply sorting
	if opts != nil && opts.SortField != "" {
		fieldPath, direction := getSortParams(opts.SortField, opts.SortOrder)
		query = query.OrderBy(fieldPath, direction)
	} else {
		// Default sort by UpdatedAt descending
		query = query.OrderBy("UpdatedAt", firestore.Desc)
	}

	// Get total count using aggregation query
	// NOTE: If filters are added in the future, the same filters must be applied here
	aggregationQuery := f.client.Collection(collectionIoCs).NewAggregationQuery().WithCount("total")
	aggregationResults, err := aggregationQuery.Get(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get total count")
	}

	// Extract count from aggregation result (AggregationResult is map[string]interface{})
	totalValue, ok := aggregationResults["total"]
	if !ok {
		return nil, goerr.New("total count not found in aggregation result")
	}

	// Convert count value to int
	// Firestore aggregation returns *firestorepb.Value (protobuf type)
	pbValue, ok := totalValue.(*firestorepb.Value)
	if !ok {
		return nil, goerr.New("total count has unexpected type",
			goerr.V("type", fmt.Sprintf("%T", totalValue)),
			goerr.V("value", totalValue))
	}
	total := int(pbValue.GetIntegerValue())

	// Apply pagination using Firestore's Offset and Limit
	offset := 0
	limit := 20 // default
	if opts != nil {
		if opts.Offset > 0 {
			offset = opts.Offset
		}
		if opts.Limit > 0 {
			limit = opts.Limit
		}
	}

	// Apply Firestore pagination
	query = query.Offset(offset).Limit(limit)

	// Execute query
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to query IoCs")
	}

	// Parse results
	var iocs []*model.IoC
	for _, doc := range docs {
		var ioc model.IoC
		if err := doc.DataTo(&ioc); err != nil {
			return nil, goerr.Wrap(err, "failed to decode IoC",
				goerr.V("doc_id", doc.Ref.ID))
		}
		iocs = append(iocs, &ioc)
	}

	return &model.IoCConnection{
		Items: iocs,
		Total: total,
	}, nil
}

// getSortParams converts domain sort field to Firestore field path and direction
func getSortParams(sortField model.IoCSortField, sortOrder model.SortOrder) (string, firestore.Direction) {
	direction := firestore.Asc
	if sortOrder == model.SortOrderDesc {
		direction = firestore.Desc
	}

	var fieldPath string
	switch sortField {
	case model.IoCSortByType:
		fieldPath = "Type"
	case model.IoCSortByValue:
		fieldPath = "Value"
	case model.IoCSortBySourceID:
		fieldPath = "SourceID"
	case model.IoCSortByStatus:
		fieldPath = "Status"
	case model.IoCSortByFirstSeenAt:
		fieldPath = "FirstSeenAt"
	case model.IoCSortByUpdatedAt:
		fieldPath = "UpdatedAt"
	default:
		fieldPath = "UpdatedAt"
		direction = firestore.Desc
	}

	return fieldPath, direction
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
		// Check if any field changed (for feed sources, update if anything changed)
		needsUpdate := existing.Description != ioc.Description ||
			existing.Status != ioc.Status ||
			existing.SourceURL != ioc.SourceURL ||
			existing.Context != ioc.Context
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
// Uses BulkWriter which handles batching automatically (20 writes per batch)
// Processes in chunks to avoid loading too many documents at once with GetAll
func (f *Firestore) BatchUpsertIoCs(ctx context.Context, iocs []*model.IoC) (*interfaces.BatchUpsertResult, error) {
	result := &interfaces.BatchUpsertResult{}

	if len(iocs) == 0 {
		return result, nil
	}

	// Process in chunks of 1000 for GetAll to balance memory usage and number of requests
	const chunkSize = 1000

	for i := 0; i < len(iocs); i += chunkSize {
		end := i + chunkSize
		if end > len(iocs) {
			end = len(iocs)
		}

		chunk := iocs[i:end]
		chunkResult, err := f.writeBatch(ctx, chunk)
		result.Created += chunkResult.Created
		result.Updated += chunkResult.Updated
		result.Unchanged += chunkResult.Unchanged
		if err != nil {
			return result, goerr.Wrap(err, "batch write failed",
				goerr.V("chunk_start", i),
				goerr.V("chunk_size", len(chunk)),
				goerr.V("result", result))
		}
	}

	return result, nil
}

// writeBatch writes a chunk of IoCs using BulkWriter
func (f *Firestore) writeBatch(ctx context.Context, iocs []*model.IoC) (*interfaces.BatchUpsertResult, error) {
	result := &interfaces.BatchUpsertResult{}
	bulkWriter := f.client.BulkWriter(ctx)
	now := time.Now()

	// First, fetch existing documents using GetAll for better performance
	docRefs := make([]*firestore.DocumentRef, len(iocs))
	for i, ioc := range iocs {
		docRefs[i] = f.client.Collection(collectionIoCs).Doc(ioc.ID)
	}

	docs, err := f.client.GetAll(ctx, docRefs)
	if err != nil {
		return result, goerr.Wrap(err, "failed to fetch existing documents")
	}

	existingMap := make(map[string]*model.IoC)
	for _, doc := range docs {
		if doc.Exists() {
			var existing model.IoC
			if err := doc.DataTo(&existing); err == nil {
				existingMap[existing.ID] = &existing
			}
		}
	}

	// Prepare bulk writes
	for _, ioc := range iocs {
		if err := model.ValidateIoC(ioc); err != nil {
			return result, goerr.Wrap(err, "invalid IoC in batch", goerr.V("id", ioc.ID))
		}

		docRef := f.client.Collection(collectionIoCs).Doc(ioc.ID)

		if existing, ok := existingMap[ioc.ID]; ok {
			// Existing IoC - check if any field changed (for feed sources, update if anything changed)
			needsUpdate := existing.Description != ioc.Description ||
				existing.Status != ioc.Status ||
				existing.SourceURL != ioc.SourceURL ||
				existing.Context != ioc.Context
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
	doc, err := f.client.Collection(collectionSources).Doc(sourceID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(interfaces.ErrSourceStateNotFound, "source state not found", goerr.V("source_id", sourceID))
		}
		return nil, goerr.Wrap(err, "failed to get source state from firestore",
			goerr.V("source_id", sourceID))
	}

	var fsState firestoreSourceState
	if err := doc.DataTo(&fsState); err != nil {
		return nil, goerr.Wrap(err, "failed to decode source state",
			goerr.V("source_id", sourceID))
	}

	return toDomainSourceState(&fsState), nil
}

// SaveState saves or updates source state
func (f *Firestore) SaveState(ctx context.Context, state *model.SourceState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	// Update timestamp before saving
	state.UpdatedAt = time.Now()

	// Convert to Firestore representation
	fsState := toFirestoreSourceState(state)

	docRef := f.client.Collection(collectionSources).Doc(state.SourceID)
	if _, err := docRef.Set(ctx, fsState); err != nil {
		return goerr.Wrap(err, "failed to save source state to firestore",
			goerr.V("source_id", state.SourceID))
	}

	return nil
}

// BatchGetStates retrieves multiple source states in a single operation using GetAll
func (f *Firestore) BatchGetStates(ctx context.Context, sourceIDs []string) (map[string]*model.SourceState, error) {
	if len(sourceIDs) == 0 {
		return make(map[string]*model.SourceState), nil
	}

	// Create document references
	docRefs := make([]*firestore.DocumentRef, len(sourceIDs))
	for i, sourceID := range sourceIDs {
		docRefs[i] = f.client.Collection(collectionSources).Doc(sourceID)
	}

	// Batch get all documents
	docs, err := f.client.GetAll(ctx, docRefs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to batch get source states from firestore")
	}

	// Convert to domain model
	result := make(map[string]*model.SourceState, len(docs))
	for i, doc := range docs {
		if !doc.Exists() {
			// Skip non-existent documents
			continue
		}

		var fsState firestoreSourceState
		if err := doc.DataTo(&fsState); err != nil {
			return nil, goerr.Wrap(err, "failed to unmarshal source state",
				goerr.V("source_id", sourceIDs[i]))
		}

		result[sourceIDs[i]] = toDomainSourceState(&fsState)
	}

	return result, nil
}

// GetRSSState retrieves RSS state by source ID
func (f *Firestore) GetRSSState(ctx context.Context, sourceID string) (*rss.RSSState, error) {
	doc, err := f.client.Collection(collectionSources).Doc(sourceID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(rss.ErrRSSStateNotFound, "RSS state not found", goerr.V("source_id", sourceID))
		}
		return nil, goerr.Wrap(err, "failed to get RSS state from firestore",
			goerr.V("source_id", sourceID))
	}

	var state rss.RSSState
	if err := doc.DataTo(&state); err != nil {
		return nil, goerr.Wrap(err, "failed to decode RSS state",
			goerr.V("source_id", sourceID))
	}

	// SourceID is stored as document ID, not in data fields
	state.SourceID = sourceID

	return &state, nil
}

// SaveRSSState saves or updates RSS state
func (f *Firestore) SaveRSSState(ctx context.Context, state *rss.RSSState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	// Update timestamp before saving
	state.UpdatedAt = time.Now()

	docRef := f.client.Collection(collectionSources).Doc(state.SourceID)
	if _, err := docRef.Set(ctx, state); err != nil {
		return goerr.Wrap(err, "failed to save RSS state to firestore",
			goerr.V("source_id", state.SourceID))
	}

	return nil
}

// GetFeedState retrieves Feed state by source ID
func (f *Firestore) GetFeedState(ctx context.Context, sourceID string) (*feed.FeedState, error) {
	doc, err := f.client.Collection(collectionSources).Doc(sourceID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(feed.ErrFeedStateNotFound, "Feed state not found", goerr.V("source_id", sourceID))
		}
		return nil, goerr.Wrap(err, "failed to get Feed state from firestore",
			goerr.V("source_id", sourceID))
	}

	var state feed.FeedState
	if err := doc.DataTo(&state); err != nil {
		return nil, goerr.Wrap(err, "failed to decode Feed state",
			goerr.V("source_id", sourceID))
	}

	// SourceID is stored as document ID, not in data fields
	state.SourceID = sourceID

	return &state, nil
}

// SaveFeedState saves or updates Feed state
func (f *Firestore) SaveFeedState(ctx context.Context, state *feed.FeedState) error {
	if state.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}

	// Update timestamp before saving
	state.UpdatedAt = time.Now()

	docRef := f.client.Collection(collectionSources).Doc(state.SourceID)
	if _, err := docRef.Set(ctx, state); err != nil {
		return goerr.Wrap(err, "failed to save Feed state to firestore",
			goerr.V("source_id", state.SourceID))
	}

	return nil
}

// FindNearestIoCs performs vector similarity search using Firestore Vector Search
func (f *Firestore) FindNearestIoCs(ctx context.Context, queryVector []float32, limit int) ([]*model.IoC, error) {
	if len(queryVector) != model.EmbeddingDimension {
		return nil, goerr.Wrap(interfaces.ErrIoCNotFound, "invalid query vector dimension",
			goerr.V("expected", model.EmbeddingDimension),
			goerr.V("actual", len(queryVector)))
	}

	if limit <= 0 {
		return []*model.IoC{}, nil
	}

	// Convert []float32 to firestore.Vector32
	vectorValue := firestore.Vector32(queryVector)

	// Perform vector search using FindNearest
	docs, err := f.client.Collection(collectionIoCs).
		FindNearest("Embedding", vectorValue, limit, firestore.DistanceMeasureCosine, nil).
		Documents(ctx).
		GetAll()

	if err != nil {
		return nil, goerr.Wrap(err, "failed to perform vector search",
			goerr.V("limit", limit),
			goerr.V("vector_dim", len(queryVector)))
	}

	// Parse results
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

// SaveHistory saves a fetch history record to a subcollection
func (f *Firestore) SaveHistory(ctx context.Context, history *model.History) error {
	if history.SourceID == "" {
		return goerr.New("source ID cannot be empty")
	}
	if history.ID == "" {
		return goerr.New("history ID cannot be empty")
	}

	fh := toFirestoreHistory(history)

	// Path: sources/{sourceID}/histories/{historyID}
	docRef := f.client.Collection(collectionSources).
		Doc(history.SourceID).
		Collection(subcollectionHistories).
		Doc(history.ID)

	if _, err := docRef.Set(ctx, fh); err != nil {
		return goerr.Wrap(err, "failed to save history to firestore",
			goerr.V("source_id", history.SourceID),
			goerr.V("history_id", history.ID))
	}

	return nil
}

// ListHistoriesBySource retrieves histories for a specific source
func (f *Firestore) ListHistoriesBySource(ctx context.Context, sourceID string, limit, offset int) ([]*model.History, int, error) {
	// Path: sources/{sourceID}/histories
	historyCollection := f.client.Collection(collectionSources).
		Doc(sourceID).
		Collection(subcollectionHistories)

	// Get total count using aggregation query
	aggregationQuery := historyCollection.NewAggregationQuery().WithCount("total")
	aggregationResults, err := aggregationQuery.Get(ctx)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get total history count",
			goerr.V("source_id", sourceID))
	}

	// Extract count from aggregation result
	totalValue, ok := aggregationResults["total"]
	if !ok {
		return nil, 0, goerr.New("total count not found in aggregation result")
	}

	// Convert count value to int (Firestore returns protobuf value)
	pbValue, ok := totalValue.(*firestorepb.Value)
	if !ok {
		return nil, 0, goerr.New("total count has unexpected type",
			goerr.V("type", fmt.Sprintf("%T", totalValue)),
			goerr.V("value", totalValue))
	}
	total := int(pbValue.GetIntegerValue())

	// Build query with ordering
	query := historyCollection.OrderBy("started_at", firestore.Desc)

	// Apply offset
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Apply limit
	if limit > 0 {
		query = query.Limit(limit)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to list histories from firestore",
			goerr.V("source_id", sourceID),
			goerr.V("limit", limit),
			goerr.V("offset", offset))
	}

	var histories []*model.History
	for _, doc := range docs {
		var fh firestoreHistory
		if err := doc.DataTo(&fh); err != nil {
			return nil, 0, goerr.Wrap(err, "failed to decode history",
				goerr.V("doc_id", doc.Ref.ID),
				goerr.V("source_id", sourceID))
		}
		histories = append(histories, toDomainHistory(&fh))
	}

	return histories, total, nil
}

// GetHistory retrieves a specific history record
func (f *Firestore) GetHistory(ctx context.Context, sourceID string, historyID string) (*model.History, error) {
	// Path: sources/{sourceID}/histories/{historyID}
	doc, err := f.client.Collection(collectionSources).
		Doc(sourceID).
		Collection(subcollectionHistories).
		Doc(historyID).
		Get(ctx)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(interfaces.ErrHistoryNotFound, "history not found",
				goerr.V("source_id", sourceID),
				goerr.V("history_id", historyID))
		}
		return nil, goerr.Wrap(err, "failed to get history from firestore",
			goerr.V("source_id", sourceID),
			goerr.V("history_id", historyID))
	}

	var fh firestoreHistory
	if err := doc.DataTo(&fh); err != nil {
		return nil, goerr.Wrap(err, "failed to decode history",
			goerr.V("source_id", sourceID),
			goerr.V("history_id", historyID))
	}

	return toDomainHistory(&fh), nil
}
