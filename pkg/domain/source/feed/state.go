package feed

import (
	"context"
	"errors"
	"time"
)

var ErrFeedStateNotFound = errors.New("feed state not found")

// FeedState represents the state of a feed source
type FeedState struct {
	SourceID      string
	LastFetchedAt time.Time
	ItemCount     int64
	ErrorCount    int64
	LastError     string
	UpdatedAt     time.Time
}

// FeedStateRepository manages Feed source state
type FeedStateRepository interface {
	GetFeedState(ctx context.Context, sourceID string) (*FeedState, error)
	SaveFeedState(ctx context.Context, state *FeedState) error
}
