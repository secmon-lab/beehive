package feed_test

import (
	"context"
	_ "embed"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/service/feed"
)

//go:embed testdata/urlhaus_sample.csv
var urlhausSampleData []byte

//go:embed testdata/threatfox_sample.csv
var threatfoxSampleData []byte

func TestService_FetchAbuseCHURLhaus(t *testing.T) {
	ctx := context.Background()

	t.Run("parse real URLhaus feed data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(urlhausSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
		gt.NoError(t, err)

		// Real data has 11 entries (excluding comments/header)
		gt.Array(t, entries).Length(11).Describe("should parse all 11 URLhaus entries from real data")

		// Verify first entry with exact expected values
		gt.Array(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.ID).Equal("3741935").Describe("first entry ID")
			gt.V(t, first.Type).Equal(model.IoCTypeURL).Describe("first entry type")
			gt.V(t, first.Value).Equal("https://sivqen.a8riculmarb1e.ru/0dh149h0").Describe("first entry URL")
			gt.S(t, first.Description).Contains("malware_download").Describe("first entry description should contain threat type")
			gt.Array(t, first.Tags).Length(1).Describe("first entry should have 1 tag")
			gt.V(t, first.Tags[0]).Equal("ClearFake").Describe("first entry tag")

			// Parse expected timestamp: "2025-12-24 07:20:09"
			expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2025-12-24 07:20:09")
			gt.V(t, first.FirstSeen).Equal(expectedTime).Describe("first entry timestamp")
		})

		// Verify second entry
		gt.Array(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.ID).Equal("3741934").Describe("second entry ID")
			gt.V(t, second.Value).Equal("https://sivqen.a8riculmarb1e.ru/wf7eqkdv").Describe("second entry URL")
		})

		// Verify third entry with multiple tags
		gt.Array(t, entries).At(2, func(t testing.TB, third *feed.FeedEntry) {
			gt.V(t, third.ID).Equal("3741933").Describe("third entry ID")
			gt.V(t, third.Value).Equal("http://182.126.117.23:55214/bin.sh").Describe("third entry URL")
			gt.Array(t, third.Tags).Length(4).Describe("third entry should have 4 tags")
			gt.Array(t, third.Tags).Has("32-bit").Describe("third entry should have 32-bit tag")
			gt.Array(t, third.Tags).Has("elf").Describe("third entry should have elf tag")
			gt.Array(t, third.Tags).Has("mips").Describe("third entry should have mips tag")
			gt.Array(t, third.Tags).Has("Mozi").Describe("third entry should have Mozi tag")
		})

		// Verify last entry
		gt.Array(t, entries).At(10, func(t testing.TB, last *feed.FeedEntry) {
			gt.V(t, last.ID).Equal("3741925").Describe("last entry ID")
			gt.V(t, last.Value).Equal("http://222.140.199.253:32946/bin.sh").Describe("last entry URL")
		})
	})

	t.Run("handle empty feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Comment only\n"))
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
		gt.NoError(t, err)
		gt.Array(t, entries).Length(0).Describe("empty feed should return zero entries")
	})

	t.Run("handle HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		svc := feed.New()
		_, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
		gt.Error(t, err).Describe("should error on HTTP 404")
	})

	t.Run("handle invalid CSV", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("invalid,\"unclosed quote"))
		}))
		defer server.Close()

		svc := feed.New()
		_, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
		gt.Error(t, err).Describe("should error on malformed CSV")
	})
}

func TestService_FetchAbuseCHThreatFox(t *testing.T) {
	ctx := context.Background()

	t.Run("parse real ThreatFox feed data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(threatfoxSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchAbuseCHThreatFox(ctx, server.URL)
		gt.NoError(t, err)

		// Real data has 11 entries
		gt.Array(t, entries).Length(11).Describe("should parse all 11 ThreatFox entries from real data")

		// Verify first entry (ip:port) with exact values
		gt.Array(t, entries).At(0, func(t testing.TB, first *feed.FeedEntry) {
			gt.V(t, first.ID).Equal("1685590").Describe("first entry IOC ID")
			gt.V(t, first.Type).Equal(model.IoCTypeIPv4).Describe("ip:port should be mapped to IPv4")
			gt.V(t, first.Value).Equal("45.32.211.159:51515").Describe("first entry value")
			gt.S(t, first.Description).Contains("Mirai").Describe("first entry should contain malware name")
			gt.S(t, first.Description).Contains("botnet_cc").Describe("first entry should contain threat type")
			gt.Array(t, first.Tags).Length(1).Describe("first entry should have 1 tag")
			gt.V(t, first.Tags[0]).Equal("mirai").Describe("first entry tag")

			expectedTime, _ := time.Parse("2006-01-02 15:04:05", "2025-12-24 07:24:23")
			gt.V(t, first.FirstSeen).Equal(expectedTime).Describe("first entry timestamp")
		})

		// Verify second entry (Cobalt Strike)
		gt.Array(t, entries).At(1, func(t testing.TB, second *feed.FeedEntry) {
			gt.V(t, second.ID).Equal("1685599").Describe("second entry IOC ID")
			gt.V(t, second.Value).Equal("47.96.75.57:8081").Describe("second entry value")
			gt.S(t, second.Description).Contains("Cobalt Strike").Describe("second entry malware name")
			gt.Array(t, second.Tags).Length(3).Describe("second entry should have 3 tags")
			gt.Array(t, second.Tags).Has("AS37963").Describe("should have AS tag")
			gt.Array(t, second.Tags).Has("C2").Describe("should have C2 tag")
			gt.Array(t, second.Tags).Has("censys").Describe("should have censys tag")
		})

		// Verify URL type entry
		gt.Array(t, entries).At(3, func(t testing.TB, urlEntry *feed.FeedEntry) {
			gt.V(t, urlEntry.ID).Equal("1685601").Describe("URL entry IOC ID")
			gt.V(t, urlEntry.Type).Equal(model.IoCTypeURL).Describe("url type should be mapped to URL")
			gt.V(t, urlEntry.Value).Equal("https://google-drive.co/").Describe("URL entry value")
			gt.S(t, urlEntry.Description).Contains("Unknown malware").Describe("URL entry malware name")
			gt.S(t, urlEntry.Description).Contains("payload_delivery").Describe("URL entry threat type")
			gt.Array(t, urlEntry.Tags).Length(1).Describe("URL entry should have 1 tag")
			gt.V(t, urlEntry.Tags[0]).Equal("ClickFix").Describe("URL entry tag")
		})

		// Verify Sliver entry
		gt.Array(t, entries).At(4, func(t testing.TB, sliver *feed.FeedEntry) {
			gt.V(t, sliver.ID).Equal("1685602").Describe("Sliver entry IOC ID")
			gt.V(t, sliver.Value).Equal("216.40.86.158:443").Describe("Sliver entry value")
			gt.S(t, sliver.Description).Contains("Sliver").Describe("Sliver entry malware name")
			gt.Array(t, sliver.Tags).Length(4).Describe("Sliver entry should have 4 tags")
			gt.Array(t, sliver.Tags).Has("AS1054").Describe("should have AS1054 tag")
			gt.Array(t, sliver.Tags).Has("C2").Describe("should have C2 tag")
			gt.Array(t, sliver.Tags).Has("censys").Describe("should have censys tag")
			gt.Array(t, sliver.Tags).Has("ZONT-LLC").Describe("should have ZONT-LLC tag")
		})

		// Verify last entry
		gt.Array(t, entries).At(10, func(t testing.TB, last *feed.FeedEntry) {
			gt.V(t, last.ID).Equal("1685608").Describe("last entry IOC ID")
			gt.V(t, last.Value).Equal("223.76.218.105:9205").Describe("last entry value")
			gt.S(t, last.Description).Contains("Unknown malware").Describe("last entry malware name")
			gt.Array(t, last.Tags).Length(4).Describe("last entry should have 4 tags")
			gt.Array(t, last.Tags).Has("AS9808").Describe("should have AS9808 tag")
			gt.Array(t, last.Tags).Has("censys").Describe("should have censys tag")
			gt.Array(t, last.Tags).Has("GoPhish").Describe("should have GoPhish tag")
			gt.Array(t, last.Tags).Has("Phishing").Describe("should have Phishing tag")
		})
	})

	t.Run("handle empty feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Comment only\n"))
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchAbuseCHThreatFox(ctx, server.URL)
		gt.NoError(t, err)
		gt.Array(t, entries).Length(0).Describe("empty feed should return zero entries")
	})

	t.Run("handle HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		svc := feed.New()
		_, err := svc.FetchAbuseCHThreatFox(ctx, server.URL)
		gt.Error(t, err).Describe("should error on HTTP 500")
	})
}

func TestService_FetchFeed(t *testing.T) {
	ctx := context.Background()

	t.Run("dispatch to URLhaus", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(urlhausSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchFeed(ctx, server.URL, "abuse_ch_urlhaus")
		gt.NoError(t, err)
		gt.Array(t, entries).Length(11).Describe("should parse all URLhaus entries")

		// All should be URL type
		for i := 0; i < 11; i++ {
			idx := i
			gt.Array(t, entries).At(idx, func(t testing.TB, entry *feed.FeedEntry) {
				gt.V(t, entry.Type).Equal(model.IoCTypeURL).Describef("entry %d should be URL type", idx)
			})
		}
	})

	t.Run("dispatch to ThreatFox", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(threatfoxSampleData)
		}))
		defer server.Close()

		svc := feed.New()
		entries, err := svc.FetchFeed(ctx, server.URL, "abuse_ch_threatfox")
		gt.NoError(t, err)
		gt.Array(t, entries).Length(11).Describe("should parse all ThreatFox entries")
	})

	t.Run("unsupported schema", func(t *testing.T) {
		svc := feed.New()
		_, err := svc.FetchFeed(ctx, "http://example.com", "unknown_schema")
		gt.Error(t, err).Describe("should error on unsupported schema")
	})
}

func TestService_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("handle non-200 status codes", func(t *testing.T) {
		testCases := []struct {
			code int
			name string
		}{
			{400, "Bad Request"},
			{403, "Forbidden"},
			{404, "Not Found"},
			{500, "Internal Server Error"},
			{503, "Service Unavailable"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.code)
				}))
				defer server.Close()

				svc := feed.New()
				_, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
				gt.Error(t, err).Describef("should error on HTTP %d", tc.code)
			})
		}
	})

	t.Run("handle malformed CSV", func(t *testing.T) {
		// Only test actual CSV parsing errors like unclosed quotes
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`"field1","unclosed quote`))
		}))
		defer server.Close()

		svc := feed.New()
		_, err := svc.FetchAbuseCHURLhaus(ctx, server.URL)
		gt.Error(t, err) // CSV parser should error on unclosed quotes
	})
}
