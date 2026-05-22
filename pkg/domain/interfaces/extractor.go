package interfaces

import (
	"context"
)

// Extractor turns an article body into a list of candidate IoC seeds.
// The MVP implementation in pkg/service/extractor uses gollem with a
// JSON-schema-constrained response so the seeds come out as a clean array.
type Extractor interface {
	Extract(ctx context.Context, body string) ([]*IoCSeed, error)
}
