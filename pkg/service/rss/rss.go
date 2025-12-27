package rss

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/m-mizutani/goerr/v2"
	"github.com/mmcdole/gofeed"
)

var (
	errFetchFailed = goerr.New("failed to fetch RSS feed")
	errParseFailed = goerr.New("failed to parse HTML")
)

// Article represents a parsed RSS article
type Article struct {
	GUID        string
	Title       string
	Link        string
	Description string
	Content     string // Full article content (HTML parsed)
	PublishedAt time.Time
}

// Service provides RSS feed fetching and parsing
type Service struct {
	client *http.Client
	parser *gofeed.Parser
}

// New creates a new RSS service
func New() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		parser: gofeed.NewParser(),
	}
}

// FetchFeed fetches and parses an RSS feed from the given URL
func (s *Service) FetchFeed(ctx context.Context, feedURL string) ([]*Article, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request", goerr.V("url", feedURL))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(errFetchFailed, "failed to fetch RSS feed",
			goerr.V("url", feedURL), goerr.V("error", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.Wrap(errFetchFailed, "non-200 status code",
			goerr.V("url", feedURL),
			goerr.V("status_code", resp.StatusCode))
	}

	feed, err := s.parser.Parse(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse RSS feed", goerr.V("url", feedURL))
	}

	articles := make([]*Article, 0, len(feed.Items))
	for _, item := range feed.Items {
		article := &Article{
			GUID:        item.GUID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PublishedAt: parsePublishedDate(item),
		}

		// Use GUID if available, otherwise use Link
		if article.GUID == "" {
			article.GUID = article.Link
		}

		articles = append(articles, article)
	}

	return articles, nil
}

// FetchArticleContent fetches and extracts the main content from an article URL
func (s *Service) FetchArticleContent(ctx context.Context, articleURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", articleURL, nil)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create request", goerr.V("url", articleURL))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", goerr.Wrap(errFetchFailed, "failed to fetch article",
			goerr.V("url", articleURL), goerr.V("error", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", goerr.Wrap(errFetchFailed, "non-200 status code",
			goerr.V("url", articleURL),
			goerr.V("status_code", resp.StatusCode))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", goerr.Wrap(errParseFailed, "failed to parse HTML",
			goerr.V("url", articleURL))
	}

	// Extract main content using common selectors
	content := extractMainContent(doc)

	return content, nil
}

// extractMainContent tries to extract the main article content from HTML
func extractMainContent(doc *goquery.Document) string {
	// Try common content selectors in order of specificity
	selectors := []string{
		"article",
		"main",
		".post-content",
		".article-content",
		".entry-content",
		"#content",
		".content",
	}

	var content string
	for _, selector := range selectors {
		selection := doc.Find(selector)
		if selection.Length() > 0 {
			content = selection.First().Text()
			if len(content) > 100 { // Only use if substantial content found
				break
			}
		}
	}

	// Fallback to body if no content found
	if content == "" {
		content = doc.Find("body").Text()
	}

	// Clean up whitespace
	content = strings.TrimSpace(content)
	content = strings.Join(strings.Fields(content), " ")

	return content
}

// parsePublishedDate attempts to parse the published date from an RSS item
func parsePublishedDate(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	if item.UpdatedParsed != nil {
		return *item.UpdatedParsed
	}
	return time.Now() // Fallback to current time
}

// FilterNewArticles filters articles that are newer than the given last item date/ID
func FilterNewArticles(articles []*Article, lastItemID string, lastItemDate time.Time) []*Article {
	if lastItemID == "" && lastItemDate.IsZero() {
		// No previous state, return all articles
		return articles
	}

	var filtered []*Article
	for _, article := range articles {
		// If we find the last processed item, stop here
		if article.GUID == lastItemID {
			break
		}

		// Include if newer than last item date
		if !lastItemDate.IsZero() && article.PublishedAt.After(lastItemDate) {
			filtered = append(filtered, article)
		} else if lastItemDate.IsZero() {
			// If no date available, include until we hit the last ID
			filtered = append(filtered, article)
		}
	}

	return filtered
}

// GetLatestArticle returns the most recent article from a list
func GetLatestArticle(articles []*Article) *Article {
	if len(articles) == 0 {
		return nil
	}

	latest := articles[0]
	for _, article := range articles[1:] {
		if article.PublishedAt.After(latest.PublishedAt) {
			latest = article
		}
	}

	return latest
}

// ExtractTextFromHTML extracts plain text from HTML content
func ExtractTextFromHTML(htmlContent string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", goerr.Wrap(errParseFailed, "failed to parse HTML content")
	}

	text := doc.Text()
	text = strings.TrimSpace(text)
	text = strings.Join(strings.Fields(text), " ")

	return text, nil
}
