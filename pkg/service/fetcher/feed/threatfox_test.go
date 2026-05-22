package feed_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher/feed"
)

func TestParseThreatFox_MixedTypes(t *testing.T) {
	body := []byte(`{
        "data": [
            {"ioc": "1.2.3.4:80", "ioc_type": "ip:port", "confidence_level": 90, "first_seen": "2026-01-01T00:00:00Z"},
            {"ioc": "evil.example", "ioc_type": "domain", "confidence_level": 80, "first_seen": "2026-01-01T00:00:00Z"},
            {"ioc": "http://x.example/path", "ioc_type": "url", "confidence_level": 70, "first_seen": "2026-01-01T00:00:00Z"},
            {"ioc": "DEADBEEFDEADBEEFDEADBEEFDEADBEEF", "ioc_type": "md5_hash", "confidence_level": 75, "first_seen": "2026-01-01T00:00:00Z"},
            {"ioc": "skip", "ioc_type": "unknown", "confidence_level": 50, "first_seen": "2026-01-01T00:00:00Z"}
        ]
    }`)
	seeds, err := feed.ParseThreatFox(body)
	gt.NoError(t, err)
	gt.A(t, seeds).Length(4)
	gt.Equal(t, seeds[0].Type, types.IoCTypeIPv4)
	gt.Equal(t, seeds[1].Type, types.IoCTypeDomain)
	gt.Equal(t, seeds[2].Type, types.IoCTypeURL)
	gt.Equal(t, seeds[3].Type, types.IoCTypeMD5)
	// MD5 hash should be lowercased.
	gt.Equal(t, seeds[3].Value, "deadbeefdeadbeefdeadbeefdeadbeef")
}
