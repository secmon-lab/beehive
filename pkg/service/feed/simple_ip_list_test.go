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

//go:embed testdata/simple_ip_list_sample.txt
var simpleIPListSampleData []byte

func TestService_FetchSimpleIPList(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(simpleIPListSampleData)
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchSimpleIPList(ctx, server.URL, []string{"test-tag", "blocklist"})
	gt.NoError(t, err)

	// Should parse all 10 IP addresses (skip comments and empty lines)
	gt.A(t, entries).Length(10).Describe("should parse all 10 IP addresses")

	// Test first IPv4 entry
	gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
		gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry type should be IPv4")
		gt.V(t, first.Value).Equal("192.0.2.1").Describe("first entry value")
		gt.A(t, first.Tags).Length(2).Describe("should have 2 tags")
		gt.A(t, first.Tags).At(0, func(t testing.TB, tag string) {
			gt.V(t, tag).Equal("test-tag").Describe("first tag")
		})
	})

	// Test first IPv6 entry (index 5)
	gt.A(t, entries).At(5, func(t testing.TB, sixth *feed.FeedEntry) {
		gt.V(t, sixth.Type).Equal(model.IoCTypeIPv6).Describe("sixth entry type should be IPv6")
		gt.V(t, sixth.Value).Equal("2001:db8::1").Describe("sixth entry value")
	})
}

func TestService_FetchSimpleIPList_EmptyFeed(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Only comments\n# No IPs\n"))
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchSimpleIPList(ctx, server.URL, []string{"test"})
	gt.NoError(t, err)

	gt.A(t, entries).Length(0).Describe("should return empty list for feed with only comments")
}

func TestService_FetchSimpleIPList_InvalidIPs(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-an-ip\n192.0.2.1\ninvalid.ip.address\n198.51.100.10\n"))
	}))
	defer server.Close()

	svc := feed.New()
	entries, err := svc.FetchSimpleIPList(ctx, server.URL, []string{"test"})
	gt.NoError(t, err)

	// Should skip invalid IPs and only parse the 2 valid ones
	gt.A(t, entries).Length(2).Describe("should skip invalid IPs and parse only valid ones")
	gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
		gt.V(t, first.Value).Equal("192.0.2.1").Describe("first valid IP")
	})
	gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
		gt.V(t, second.Value).Equal("198.51.100.10").Describe("second valid IP")
	})
}
