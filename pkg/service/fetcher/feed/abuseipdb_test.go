package feed_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

func TestParseAbuseIPDB(t *testing.T) {
	body := []byte(`{
        "data": [
            {"ipAddress": "8.8.8.8", "abuseConfidenceScore": 100, "lastReportedAt": "2026-01-01T00:00:00Z"},
            {"ipAddress": "127.0.0.1", "abuseConfidenceScore": 90, "lastReportedAt": "2026-01-01T00:00:00Z"},
            {"ipAddress": "2001:db8::1", "abuseConfidenceScore": 80, "lastReportedAt": "2026-01-01T00:00:00Z"}
        ]
    }`)
	seeds, err := feed.ParseAbuseIPDB(body)
	gt.NoError(t, err)
	// 127.0.0.1 is rejected (loopback); 8.8.8.8 and 2001:db8::1 remain.
	gt.A(t, seeds).Length(2)

	// First should be the ipv4.
	gt.Equal(t, seeds[0].Type, types.IoCTypeIPv4)
	gt.Equal(t, seeds[0].Value, "8.8.8.8")
	gt.Equal(t, seeds[0].Confidence, 1.0)

	// Second should be the ipv6.
	gt.Equal(t, seeds[1].Type, types.IoCTypeIPv6)
}

func TestParseAbuseIPDB_BadJSON(t *testing.T) {
	_, err := feed.ParseAbuseIPDB([]byte("not-json"))
	gt.Error(t, err)
}
