package interfaces

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

var (
	// ErrHistoryNotFound is returned when a history record is not found
	ErrHistoryNotFound = goerr.New("history not found")
)

// HistoryRepository defines the interface for ingestion history persistence
type HistoryRepository interface {
	// SaveHistory saves a fetch history record
	SaveHistory(ctx context.Context, history *model.History) error

	// ListHistoriesBySource retrieves histories for a specific source
	// Returns histories ordered by StartedAt descending (newest first)
	// limit: maximum number of records to return
	// offset: number of records to skip
	ListHistoriesBySource(ctx context.Context, sourceID string, limit, offset int) ([]*model.History, error)

	// GetHistory retrieves a specific history record
	GetHistory(ctx context.Context, sourceID string, historyID string) (*model.History, error)
}
