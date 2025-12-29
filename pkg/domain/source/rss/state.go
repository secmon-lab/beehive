package rss

import (
	"context"
	"errors"
	"time"
)

var ErrRSSStateNotFound = errors.New("RSS state not found")

// RSSState represents the state of an RSS source
type RSSState struct {
	SourceID      string
	LastFetchedAt time.Time
	LastArticleID string // GUID
	LastItemDate  time.Time
	ItemCount     int64
	ErrorCount    int64
	LastError     string
	UpdatedAt     time.Time
}

// RSSStateRepository manages RSS source state
type RSSStateRepository interface {
	GetRSSState(ctx context.Context, sourceID string) (*RSSState, error)
	SaveRSSState(ctx context.Context, state *RSSState) error
}
