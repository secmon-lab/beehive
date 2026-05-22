// Package blog hosts BlogProvider implementations. The only one in MVP
// is "rss", which handles RSS / Atom / JSON Feed transparently via
// gofeed (it autodetects the format).
package blog

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/mmcdole/gofeed"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/htmltext"
)

// RSSTypeID is the type id used in TOML. RSS / Atom / JSON Feed all
// route here. Imported (not auto-registered) by pkg/service/providers
// at process startup.
const RSSTypeID = "rss"

// RSSProvider implements interfaces.BlogProvider. Concrete fields are
// kept minimal so tests can swap the HTTP client easily.
type RSSProvider struct {
	http *fetcher.HTTPClient
}

// New returns an RSSProvider sharing the package-wide HTTP client. A
// custom client may be passed for unit tests.
func New(c *fetcher.HTTPClient) *RSSProvider {
	return &RSSProvider{http: c}
}

func (p *RSSProvider) Kind() types.SourceKind { return types.KindBlog }

// Fetch implements interfaces.BlogProvider. The flow is:
//  1. GET the feed URL, hand the bytes to gofeed (autodetects RSS /
//     Atom / JSON Feed).
//  2. For each entry, GET the entry URL.
//  3. Run the HTML through htmltext.Extract to strip script / style /
//     nav / footer / ad boilerplate.
//  4. Return the resulting `FetchedArticle` slice.
//
// Errors fetching individual entries do not abort the whole feed — they
// are dropped and logged by the caller via the wrapped goerr.
func (p *RSSProvider) Fetch(ctx context.Context, src *model.Source) ([]*interfaces.FetchedArticle, error) {
	if src.URL == "" {
		return nil, goerr.New("rss provider requires url",
			goerr.V("source_id", src.ID),
			goerr.T(errutil.TagInvalidInput),
		)
	}
	feedBytes, err := p.http.Get(ctx, src.URL, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "get feed", goerr.V("source_id", src.ID))
	}
	parser := gofeed.NewParser()
	feed, err := parser.Parse(bytes.NewReader(feedBytes))
	if err != nil {
		return nil, goerr.Wrap(err, "parse feed",
			goerr.V("source_id", src.ID),
			goerr.V("url", src.URL),
			goerr.T(errutil.TagInvalidInput),
		)
	}

	out := make([]*interfaces.FetchedArticle, 0, len(feed.Items))
	for _, item := range feed.Items {
		if item.Link == "" {
			continue
		}
		article, err := p.fetchArticle(ctx, item)
		if err != nil {
			// One bad entry must not poison the whole feed.
			continue
		}
		out = append(out, article)
	}
	return out, nil
}

func (p *RSSProvider) fetchArticle(ctx context.Context, item *gofeed.Item) (*interfaces.FetchedArticle, error) {
	body, err := p.http.Get(ctx, item.Link, nil)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(item.Link)
	if err != nil {
		return nil, goerr.Wrap(err, "parse article url", goerr.V("url", item.Link))
	}
	article, err := htmltext.Extract(bytes.NewReader(body), parsedURL)
	if err != nil {
		// Fallback: use the raw body as best-effort text.
		return &interfaces.FetchedArticle{
			URL:         item.Link,
			Title:       item.Title,
			PublishedAt: itemPublished(item),
			BodyText:    strings.TrimSpace(string(body)),
			Summary:     item.Description,
		}, nil
	}
	return &interfaces.FetchedArticle{
		URL:         item.Link,
		Title:       firstNonEmpty(article.Title, item.Title),
		PublishedAt: itemPublished(item),
		BodyText:    article.BodyText,
		Summary:     firstNonEmpty(article.Excerpt, item.Description),
	}, nil
}

func itemPublished(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return item.PublishedParsed.UTC()
	}
	if item.UpdatedParsed != nil {
		return item.UpdatedParsed.UTC()
	}
	return time.Time{}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
