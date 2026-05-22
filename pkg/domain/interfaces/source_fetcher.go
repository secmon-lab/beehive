package interfaces

import (
	"context"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// Provider is the union type the dispatcher uses to look up an
// implementation by Source.Type. Concrete providers also implement either
// BlogProvider or FeedProvider, never both.
type Provider interface {
	Kind() types.SourceKind
}

// BlogProvider returns full-body articles. The default implementation
// (type = "rss") uses gofeed for feed parsing, fetches each entry URL,
// and runs the HTML through go-readability to strip boilerplate
// (script / style / nav / footer / ads).
type BlogProvider interface {
	Provider
	Fetch(ctx context.Context, src *model.Source) ([]*FetchedArticle, error)
}

// FetchedArticle is the in-flight value returned by a BlogProvider before
// the use case decides whether to create or update the persistent Article.
type FetchedArticle struct {
	URL         string
	Title       string
	PublishedAt time.Time
	BodyText    string // boilerplate already stripped
	Summary     string // optional, comes from the feed if available
}

// FeedProvider returns IoC seeds directly without LLM involvement. Used
// for sites that publish curated IoC lists (e.g. AbuseIPDB blacklist).
type FeedProvider interface {
	Provider
	Fetch(ctx context.Context, src *model.Source) ([]*IoCSeed, error)
}

// IoCSeed is the pre-normalization, pre-persistence shape of an IoC. The
// json tags are intentionally present so this type doubles as the LLM
// structured-response schema for blog extraction (see CLAUDE.md).
//
// SeenAt is filled in by the use case (provider returns the zero value).
type IoCSeed struct {
	Type       types.IoCType `json:"type"`
	Value      string        `json:"value"`         // pre-normalization
	Raw        string        `json:"raw,omitempty"` // optional defang-original
	Confidence float64       `json:"confidence"`
	SeenAt     time.Time     `json:"-"`
}
