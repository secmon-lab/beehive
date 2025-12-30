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

//go:embed testdata/ipsum_level3_sample.txt
var ipsumLevel3SampleData []byte

func TestService_FetchIPsumLevel3(t *testing.T) {
	ctx := context.Background()

	t.Run("parse IPsum Level 3 feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(ipsumLevel3SampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchIPsumLevel3(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 IPs from IPsum Level 3 sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry should be IPv4")
			gt.V(t, first.Value).Equal("192.0.2.100").Describe("first entry IP")
			gt.A(t, first.Tags).Length(2).Describe("should have 2 tags")
			gt.A(t, first.Tags).Has("ipsum").Describe("should have ipsum tag")
			gt.A(t, first.Tags).Has("threat-level-3").Describe("should have threat-level-3 tag")
		})

		// Verify third entry
		gt.A(t, entries).At(2, func(t testing.TB, third *feed.FeedEntry) {
			gt.V(t, third.Value).Equal("198.51.100.102").Describe("third entry IP")
		})
	})
}

func TestService_FetchIPsumLevel3_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel3(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchIPsumLevel4_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel4(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchIPsumLevel5_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel5(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchIPsumLevel6_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel6(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchIPsumLevel7_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel7(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchIPsumLevel8_E2E(t *testing.T) {
	t.Parallel()
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchIPsumLevel8(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
