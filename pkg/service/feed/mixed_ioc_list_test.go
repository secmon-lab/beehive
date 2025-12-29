package feed_test

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/service/feed"
)

//go:embed testdata/mixed_ioc_list_sample.txt
var mixedIoCListSampleData []byte

func TestService_FetchMixedIoCList(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(mixedIoCListSampleData)
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchMixedIoCList(ctx, server.URL, []string{"mixed", "high-confidence"})
	gt.NoError(t, err)

	// Should parse all 10 IoCs (2 IPs, 2 domains, 2 URLs, 3 MD5, 1 SHA1)
	gt.A(t, entries).Length(10).Describe("should parse all 10 IoCs")

	// Test first IP entry
	gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
		gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry type should be IPv4")
		gt.V(t, first.Value).Equal("192.0.2.1").Describe("first entry value")
		gt.A(t, first.Tags).Length(2).Describe("should have 2 tags")
	})

	// Test second IP entry
	gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
		gt.V(t, second.Type).Equal(model.IoCTypeIPv4).Describe("second entry type should be IPv4")
		gt.V(t, second.Value).Equal("198.51.100.10").Describe("second entry value")
	})

	// Test first domain entry (index 2)
	gt.A(t, entries).At(2, func(t testing.TB, third *feed.FeedEntry) {
		gt.V(t, third.Type).Equal(model.IoCTypeDomain).Describe("third entry type should be domain")
		gt.V(t, third.Value).Equal("example.com").Describe("third entry value")
	})

	// Test first URL entry (index 4)
	gt.A(t, entries).At(4, func(t testing.TB, fifth *feed.FeedEntry) {
		gt.V(t, fifth.Type).Equal(model.IoCTypeURL).Describe("fifth entry type should be URL")
		gt.V(t, fifth.Value).Equal("http://malicious.example.com/path").Describe("fifth entry value")
	})

	// Test first MD5 entry (index 6)
	gt.A(t, entries).At(6, func(t testing.TB, seventh *feed.FeedEntry) {
		gt.V(t, seventh.Type).Equal(model.IoCTypeMD5).Describe("seventh entry type should be MD5")
		gt.V(t, seventh.Value).Equal("002b1550152a4ca76ff1b2497a6c016e").Describe("seventh entry value")
	})

	// Test SHA1 entry (index 9)
	gt.A(t, entries).At(9, func(t testing.TB, tenth *feed.FeedEntry) {
		gt.V(t, tenth.Type).Equal(model.IoCTypeSHA1).Describe("tenth entry type should be SHA1")
		gt.V(t, tenth.Value).Equal("da39a3ee5e6b4b0d3255bfef95601890afd80709").Describe("tenth entry value")
	})
}

func TestService_FetchMixedIoCList_EmptyFeed(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Only comments\n# No IoCs\n"))
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchMixedIoCList(ctx, server.URL, []string{"test"})
	gt.NoError(t, err)

	gt.A(t, entries).Length(0).Describe("should return empty list for feed with only comments")
}

func TestService_FetchMixedIoCList_UnrecognizedTypes(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-a-valid-ioc\n192.0.2.1\nrandom-text-123\nexample.com\n"))
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchMixedIoCList(ctx, server.URL, []string{"test"})
	gt.NoError(t, err)

	// Should skip unrecognized types and only parse valid IoCs (2 valid: 1 IP + 1 domain)
	gt.A(t, entries).Length(2).Describe("should skip unrecognized IoC types")
	gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
		gt.V(t, first.Value).Equal("192.0.2.1").Describe("first valid IoC is IP")
	})
	gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
		gt.V(t, second.Value).Equal("example.com").Describe("second valid IoC is domain")
	})
}
