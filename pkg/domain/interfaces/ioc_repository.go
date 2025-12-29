package interfaces

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

var (
	// ErrIoCNotFound is returned when an IoC is not found
	ErrIoCNotFound = goerr.New("IoC not found")
)

// BatchUpsertResult represents the result of a batch upsert operation
type BatchUpsertResult struct {
	Created   int // Number of new IoCs created
	Updated   int // Number of existing IoCs updated
	Unchanged int // Number of existing IoCs unchanged (skipped)
}

// IoCRepository defines the interface for IoC persistence
type IoCRepository interface {
	GetIoC(ctx context.Context, id string) (*model.IoC, error)
	ListIoCsBySource(ctx context.Context, sourceID string) ([]*model.IoC, error)
	ListAllIoCs(ctx context.Context) ([]*model.IoC, error)
	ListIoCs(ctx context.Context, opts *model.IoCListOptions) (*model.IoCConnection, error)
	UpsertIoC(ctx context.Context, ioc *model.IoC) error
	// BatchUpsertIoCs upserts multiple IoCs in a single batch operation
	// Returns the result with created/updated/unchanged counts and any error
	BatchUpsertIoCs(ctx context.Context, iocs []*model.IoC) (*BatchUpsertResult, error)
	// FindNearestIoCs performs vector similarity search
	// Returns IoCs ordered by similarity to the query vector (most similar first)
	FindNearestIoCs(ctx context.Context, queryVector []float32, limit int) ([]*model.IoC, error)
}
