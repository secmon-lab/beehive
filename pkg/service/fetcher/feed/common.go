// Package feed hosts FeedProvider implementations. Each file is one
// site-specific provider; they share the SSRF-safe HTTP client and a
// few small helpers in this file.
package feed

import (
	"bufio"
	"bytes"
	"net/netip"
	"strings"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/types"
)

// iterText calls fn for each non-empty, non-comment line of data.
// Comments begin with "#" (and may be preceded by whitespace).
func iterText(data []byte, fn func(line string)) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fn(line)
	}
}

// seedIP builds an IoCSeed from a candidate IPv4 / IPv6 string. Returns
// (nil, false) when the input does not parse as a public IP (private /
// loopback / link-local addresses are filtered out at the seed level —
// they should never appear in a public IoC feed).
func seedIP(value string, confidence float64, now time.Time) (*interfaces.IoCSeed, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return nil, false
	}
	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return nil, false
	}
	t := types.IoCTypeIPv4
	if addr.Is6() && !addr.Is4In6() {
		t = types.IoCTypeIPv6
	}
	return &interfaces.IoCSeed{
		Type:       t,
		Value:      addr.Unmap().String(),
		Raw:        value,
		Confidence: confidence,
		SeenAt:     now,
	}, true
}

// seedURL builds an IoCSeed for a URL value, after best-effort
// normalisation (lowercase scheme + host).
func seedURL(value string, confidence float64, now time.Time) *interfaces.IoCSeed {
	v := strings.TrimSpace(value)
	return &interfaces.IoCSeed{
		Type:       types.IoCTypeURL,
		Value:      v,
		Raw:        value,
		Confidence: confidence,
		SeenAt:     now,
	}
}
