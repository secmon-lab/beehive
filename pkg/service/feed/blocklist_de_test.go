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

//go:embed testdata/blocklist_de_ssh_sample.txt
var blocklistDeSSHSampleData []byte

func TestService_FetchBlocklistDeSSH(t *testing.T) {
	ctx := context.Background()

	t.Run("parse Blocklist.de SSH feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(blocklistDeSSHSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchBlocklistDeSSH(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all 10 IPs from Blocklist.de SSH sample")

		// Verify first entry
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("first entry should be IPv4")
			gt.V(t, first.Value).Equal("192.0.2.1").Describe("first entry IP")
			gt.A(t, first.Tags).Length(2).Describe("should have 2 tags")
			gt.A(t, first.Tags).Has("blocklist-de").Describe("should have blocklist-de tag")
			gt.A(t, first.Tags).Has("ssh-attack").Describe("should have ssh-attack tag")
		})

		// Verify second entry
		gt.A(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.Value).Equal("192.0.2.2").Describe("second entry IP")
		})
	})
}

func TestService_FetchBlocklistDeAll(t *testing.T) {
	ctx := context.Background()

	t.Run("parse Blocklist.de All feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(blocklistDeSSHSampleData) // Reuse SSH sample data
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchBlocklistDeAll(ctx, server.URL)
		gt.NoError(t, err)

		gt.A(t, entries).Length(10).Describe("should parse all IPs")

		// Verify tags are correct for "All" feed
		gt.A(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.A(t, first.Tags).Has("blocklist-de").Describe("should have blocklist-de tag")
			gt.A(t, first.Tags).Has("attack").Describe("should have attack tag")
		})
	})
}

func TestService_FetchBlocklistDeSSH_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeSSH(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeAll_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeAll(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeMail_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeMail(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeApache_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeApache(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeIMAP_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeIMAP(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeBots_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeBots(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeBruteforce_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeBruteforce(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeStrongIPs_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeStrongIPs(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}

func TestService_FetchBlocklistDeFTP_E2E(t *testing.T) {
	if os.Getenv("TEST_E2E") == "" {
		t.Skip("E2E test skipped (set TEST_E2E=1 to run)")
	}

	ctx := context.Background()
	svc := feed.New()
	entries, err := svc.FetchBlocklistDeFTP(ctx, "")
	gt.NoError(t, err)
	gt.True(t, len(entries) > 0).Describe("should fetch at least one entry from live feed")
}
