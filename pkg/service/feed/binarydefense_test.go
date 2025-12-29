package feed_test

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/service/feed"
)

//go:embed testdata/binarydefense_sample.txt
var binarydefenseSampleData []byte

func TestService_FetchBinarydefenseBanlist(t *testing.T) {
	ctx := context.Background()

	t.Run("parse Binarydefense Banlist feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(binarydefenseSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchBinarydefenseBanlist(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 IPs from Binarydefense sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry should be IPv4")
			gt.V(t, first.Value).Equal("192.0.2.100").Describe("first entry IP")
			gt.A(t, first.Tags).Length(2).Describe("should have 2 tags")
			gt.A(t, first.Tags).Has("binarydefense").Describe("should have binarydefense tag")
			gt.A(t, first.Tags).Has("artillery").Describe("should have artillery tag")
		})

		// Verify second entry
		gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.Value).Equal("192.0.2.101").Describe("second entry IP")
		})
	})
}

func TestService_FetchBinarydefenseBanlist_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBinarydefenseBanlist(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
