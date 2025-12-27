package interfaces

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

var (
	// ErrSourceStateNotFound is returned when a source state is not found
	ErrSourceStateNotFound = goerr.New("source state not found")
)

// SourceStateRepository defines the interface for source state persistence
type SourceStateRepository interface {
	GetState(ctx context.Context, sourceID string) (*model.SourceState, error)
	SaveState(ctx context.Context, state *model.SourceState) error
}
