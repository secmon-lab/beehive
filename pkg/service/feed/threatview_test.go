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

//go:embed testdata/threatview_ip_high_sample.txt
var threatviewIPHighSampleData []byte

//go:embed testdata/threatview_md5_sample.txt
var threatviewMD5SampleData []byte

func TestService_FetchThreatViewIPHigh(t *testing.T) {
	ctx := context.Background()

	t.Run("parse ThreatView IP High Confidence feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(threatviewIPHighSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchThreatViewIPHigh(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 IPs from ThreatView IP High sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry should be IPv4")
			gt.V(t, first.Value).Equal("192.0.2.150").Describe("first entry IP")
			gt.A(t, first.Tags).Length(3).Describe("should have 3 tags")
			gt.A(t, first.Tags).Has("threatview").Describe("should have threatview tag")
			gt.A(t, first.Tags).Has("high-confidence").Describe("should have high-confidence tag")
			gt.A(t, first.Tags).Has("ip").Describe("should have ip tag")
		})

		// Verify second entry
		gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.Value).Equal("192.0.2.151").Describe("second entry IP")
		})
	})
}

func TestService_FetchThreatViewMD5(t *testing.T) {
	ctx := context.Background()

	t.Run("parse ThreatView MD5 feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(threatviewMD5SampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchThreatViewMD5(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 MD5 hashes from ThreatView MD5 sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeMD5).Describe("first entry should be MD5")
			gt.V(t, first.Value).Equal("a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6").Describe("first entry MD5")
			gt.A(t, first.Tags).Has("threatview").Describe("should have threatview tag")
			gt.A(t, first.Tags).Has("md5").Describe("should have md5 tag")
		})

		// Verify third entry
		gt.A(t, entries).At(2, func(t testing.TB, third *feed.FeedEntry) {
			gt.V(t, third.Value).Equal("c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8").Describe("third entry MD5")
		})
	})
}

func TestService_FetchThreatViewIOCTweets_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewIOCTweets(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchThreatViewCobaltStrike_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	_, err := svc.FetchThreatViewCobaltStrike(ctx, "")
	gt.NoError(t, err)
	// Feed may be temporarily empty, just verify successful fetch
}

func TestService_FetchThreatViewIPHigh_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewIPHigh(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchThreatViewDomainHigh_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewDomainHigh(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchThreatViewMD5_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewMD5(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchThreatViewURLHigh_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewURLHigh(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchThreatViewSHA_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}
	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchThreatViewSHA(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
