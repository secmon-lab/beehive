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

//go:embed testdata/c2intel_ipc2s_sample.csv
var c2intelIPSampleData []byte

//go:embed testdata/c2intel_domain_sample.csv
var c2intelDomainSampleData []byte

func TestService_FetchC2IntelIPList(t *testing.T) {
	ctx := context.Background()

	t.Run("parse C2Intel IP feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(c2intelIPSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchC2IntelIPList(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 IPs from C2Intel IP sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry should be IPv4")
			gt.V(t, first.Value).Equal("192.0.2.1").Describe("first entry IP")
			gt.S(t, first.Description).Equal("Possible Cobaltstrike C2 IP").Describe("first entry description")
			gt.A(t, first.Tags).Length(3).Describe("should have 3 tags")
			gt.A(t, first.Tags).Has("c2").Describe("should have c2 tag")
			gt.A(t, first.Tags).Has("command-control").Describe("should have command-control tag")
			gt.A(t, first.Tags).Has("c2intel").Describe("should have c2intel tag")
		})

		// Verify second entry
		gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.Value).Equal("192.0.2.2").Describe("second entry IP")
		})
	})
}

func TestService_FetchC2IntelDomainList(t *testing.T) {
	ctx := context.Background()

	t.Run("parse C2Intel Domain feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(c2intelDomainSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchC2IntelDomainList(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 domains from C2Intel Domain sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeDomain).Describe("first entry should be Domain")
			gt.V(t, first.Value).Equal("malicious1.example.com").Describe("first entry domain")
			gt.S(t, first.Description).Equal("Possible Cobalt Strike C2 Domain").Describe("first entry description")
			gt.A(t, first.Tags).Has("c2").Describe("should have c2 tag")
		})

		// Verify third entry
		gt.A(t, entries).At(2, func(t testing.TB, third *feed.FeedEntry) {
			gt.V(t, third.Value).Equal("evil3.test.local").Describe("third entry domain")
		})
	})
}

func TestService_FetchC2IntelIPList_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchC2IntelIPList(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchC2IntelDomainList_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchC2IntelDomainList(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchC2IntelDomainWithURL_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchC2IntelDomainWithURL(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchC2IntelDomainWithURLWithIP_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchC2IntelDomainWithURLWithIP(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
