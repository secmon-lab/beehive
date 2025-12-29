package feed

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/httpclient"
)

// FetchSimpleIPList fetches and parses simple IP list feeds (one IP per line)
// This is a generic function for TXT format feeds with one IP address per line.
// Comments (lines starting with #) and empty lines are skipped.
func (s *Service) FetchSimpleIPList(ctx context.Context, feedURL string, tags []string) ([]*FeedEntry, error) {
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch simple IP list feed")
	}

	lines := strings.Split(string(data), "\n")
	var entries []*FeedEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse IP address
		ip := net.ParseIP(line)
		if ip == nil {
			// Skip invalid IP addresses silently
			continue
		}

		// Determine IP type (IPv4 or IPv6)
		iocType := model.IoCTypeIPv4
		if ip.To4() == nil {
			iocType = model.IoCTypeIPv6
		}

		entry := &FeedEntry{
			ID:        "", // No unique ID for simple lists
			Type:      iocType,
			Value:     line,
			Tags:      tags,
			FirstSeen: time.Now(),
			LastSeen:  time.Now(),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
