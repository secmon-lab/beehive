package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// Source represents a threat intelligence source
type Source interface {
	// Metadata
	ID() string
	Type() model.SourceType
	Tags() []string
	Enabled() bool

	// Operations
	Fetch(ctx context.Context) (*FetchStats, error)
}

// FetchStats represents statistics from a fetch operation
type FetchStats struct {
	SourceID       string
	SourceType     model.SourceType
	ItemsFetched   int
	IoCsExtracted  int
	IoCsCreated    int
	IoCsUpdated    int
	IoCsUnchanged  int
	ErrorCount     int
	ProcessingTime time.Duration
}
