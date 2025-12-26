package interfaces

import (
	"context"

	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// SourceStateRepository defines the interface for source state persistence
type SourceStateRepository interface {
	GetState(ctx context.Context, sourceID string) (*model.SourceState, error)
	SaveState(ctx context.Context, state *model.SourceState) error
}
