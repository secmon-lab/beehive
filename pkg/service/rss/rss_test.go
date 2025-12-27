package rss_test

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/service/rss"
)

//go:embed testdata/schneier_blog.xml
var schneierBlogData []byte

func TestService_FetchFeed(t *testing.T) {
	ctx := context.Background()

	t.Run("parse real RSS feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/atom+xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(schneierBlogData)
		}))
		defer server.Close()

		svc := rss.New()
		articles, err := svc.FetchFeed(ctx, server.URL)
		gt.NoError(t, err)

		// Real Schneier blog feed should have multiple articles (at least 3 from test data)
		gt.True(t, len(articles) >= 3).Describe("should have at least 3 articles from real feed")

		// Verify first article with exact expected values
		gt.A(t, articles).At(0, func(t testing.TB, first *rss.Article) {
			gt.S(t, first.Title).Equal("Denmark Accuses Russia of Conducting Two Cyberattacks").Describe("first article title")
			gt.S(t, first.Link).Equal("https://www.schneier.com/blog/archives/2025/12/denmark-accuses-russia-of-conducting-two-cyberattacks.html").Describe("first article link")
			gt.S(t, first.GUID).Equal("https://www.schneier.com/?p=71373").Describe("first article GUID")
			expectedTime, _ := time.Parse(time.RFC3339, "2025-12-23T12:02:32Z")
			gt.V(t, first.PublishedAt).Equal(expectedTime).Describe("first article published time")
			gt.S(t, first.Description).Contains("Danish Defence Intelligence Service").Describe("first article description should contain expected text")
		})

		// Verify second article
		gt.A(t, articles).At(1, func(t testing.TB, second *rss.Article) {
			gt.S(t, second.Title).Equal("Microsoft Is Finally Killing RC4").Describe("second article title")
			gt.S(t, second.Link).Equal("https://www.schneier.com/blog/archives/2025/12/microsoft-is-finally-killing-rc4.html").Describe("second article link")
			gt.S(t, second.GUID).Equal("https://www.schneier.com/?p=71371").Describe("second article GUID")
			expectedTime, _ := time.Parse(time.RFC3339, "2025-12-22T17:05:09Z")
			gt.V(t, second.PublishedAt).Equal(expectedTime).Describe("second article published time")
		})

		// Verify third article
		gt.A(t, articles).At(2, func(t testing.TB, third *rss.Article) {
			gt.S(t, third.Title).Equal("Friday Squid Blogging: Petting a Squid").Describe("third article title")
			gt.S(t, third.GUID).Equal("https://www.schneier.com/?p=71304").Describe("third article GUID")
		})
	})

	t.Run("handle HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		svc := rss.New()
		_, err := svc.FetchFeed(ctx, server.URL)
		gt.Error(t, err).Describe("should error on HTTP 404")
	})

	t.Run("handle invalid XML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not xml"))
		}))
		defer server.Close()

		svc := rss.New()
		_, err := svc.FetchFeed(ctx, server.URL)
		gt.Error(t, err).Describe("should error on invalid XML")
	})
}

func TestService_FilterNewArticles(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(schneierBlogData)
	}))
	defer server.Close()

	svc := rss.New()
	articles, err := svc.FetchFeed(ctx, server.URL)
	gt.NoError(t, err)

	// Ensure we have at least 3 articles for filtering tests
	gt.True(t, len(articles) >= 3).Describe("need at least 3 articles for filter tests")

	t.Run("no filter returns all articles", func(t *testing.T) {
		// Pass empty lastItemID and a date way in the past (before all articles)
		veryOldDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		filtered := rss.FilterNewArticles(articles, "", veryOldDate)
		gt.V(t, len(filtered)).Equal(len(articles)).Describe("should return all articles when no filter applied")
	})

	t.Run("filter by last item ID", func(t *testing.T) {
		// Filter using third article's GUID (index 2) - should return first two articles
		filtered := rss.FilterNewArticles(articles, articles[2].GUID, articles[0].PublishedAt.Add(-24*time.Hour))
		gt.A(t, filtered).Length(2).Describe("should return only first two articles")

		// Verify the filtered articles are the first two
		gt.A(t, filtered).At(0, func(t testing.TB, first *rss.Article) {
			gt.S(t, first.GUID).Equal(articles[0].GUID).Describe("first filtered article should match original first")
		})
		gt.A(t, filtered).At(1, func(t testing.TB, second *rss.Article) {
			gt.S(t, second.GUID).Equal(articles[1].GUID).Describe("second filtered article should match original second")
		})
	})

	t.Run("filter by date", func(t *testing.T) {
		// Filter articles newer than the second article's publish time
		filtered := rss.FilterNewArticles(articles, "", articles[1].PublishedAt)

		// Should only get the first article (which is newer than articles[1])
		gt.A(t, filtered).Length(1).Describe("should return only first article")
		gt.A(t, filtered).At(0, func(t testing.TB, first *rss.Article) {
			gt.S(t, first.GUID).Equal(articles[0].GUID).Describe("filtered article should be the newest one")
		})
	})
}

func TestService_GetLatestArticle(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(schneierBlogData)
	}))
	defer server.Close()

	svc := rss.New()
	articles, err := svc.FetchFeed(ctx, server.URL)
	gt.NoError(t, err)
	gt.True(t, len(articles) >= 1).Describe("need at least one article")

	t.Run("get latest from multiple articles", func(t *testing.T) {
		latest := rss.GetLatestArticle(articles)
		gt.V(t, latest).NotNil().Describe("should return non-nil latest article")

		// Latest should be the first article (most recent)
		gt.S(t, latest.GUID).Equal(articles[0].GUID).Describe("latest should be first article in feed")
		gt.S(t, latest.Title).Equal("Denmark Accuses Russia of Conducting Two Cyberattacks").Describe("latest article title")
	})

	t.Run("empty list returns nil", func(t *testing.T) {
		latest := rss.GetLatestArticle([]*rss.Article{})
		gt.V(t, latest).Nil().Describe("should return nil for empty article list")
	})
}

func TestService_FetchArticleContent(t *testing.T) {
	ctx := context.Background()

	t.Run("extract content from HTML", func(t *testing.T) {
		htmlContent := `
<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
	<header>Header content</header>
	<article>
		<h1>Article Title</h1>
		<p>This is the main content of the article.</p>
		<p>It has multiple paragraphs.</p>
	</article>
	<footer>Footer content</footer>
</body>
</html>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(htmlContent))
		}))
		defer server.Close()

		svc := rss.New()
		content, err := svc.FetchArticleContent(ctx, server.URL)
		gt.NoError(t, err)
		gt.S(t, content).Contains("Article Title").Describe("extracted content should contain article title")
		gt.S(t, content).Contains("main content").Describe("extracted content should contain main text")
	})

	t.Run("handle HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		svc := rss.New()
		_, err := svc.FetchArticleContent(ctx, server.URL)
		gt.Error(t, err).Describe("should error on HTTP 404")
	})
}

func TestService_ExtractTextFromHTML(t *testing.T) {
	t.Run("extract plain text from HTML", func(t *testing.T) {
		html := `<html><body><p>Hello <strong>World</strong></p></body></html>`
		text, err := rss.ExtractTextFromHTML(html)
		gt.NoError(t, err)
		gt.S(t, text).Contains("Hello World").Describe("should extract text from HTML tags")
	})

	t.Run("handle plain text", func(t *testing.T) {
		html := `Just plain text without tags`
		text, err := rss.ExtractTextFromHTML(html)
		gt.NoError(t, err)
		gt.S(t, text).Contains("plain text").Describe("should handle plain text")
	})
}
