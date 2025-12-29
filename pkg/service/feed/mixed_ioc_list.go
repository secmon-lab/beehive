package feed

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/httpclient"
)

var (
	// Regular expressions for IoC type detection
	md5Regex    = regexp.MustCompile(`^[a-f0-9]{32}$`)
	sha1Regex   = regexp.MustCompile(`^[a-f0-9]{40}$`)
	sha256Regex = regexp.MustCompile(`^[a-f0-9]{64}$`)
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
)

// FetchMixedIoCList fetches and parses feeds with mixed IoC types (one IoC per line)
// This is a generic function for TXT format feeds with various IoC types.
// Comments (lines starting with #) and empty lines are skipped.
// IoC type is automatically detected based on the value format.
func (s *Service) FetchMixedIoCList(ctx context.Context, feedURL string, tags []string) ([]*FeedEntry, error) {
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch mixed IoC list feed")
	}

	lines := strings.Split(string(data), "\n")
	var entries []*FeedEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Detect IoC type
		iocType := detectIoCType(line)
		if iocType == "" {
			// Skip unrecognized IoC types
			continue
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

// detectIoCType automatically detects the IoC type based on the value format
func detectIoCType(value string) model.IoCType {
	value = strings.TrimSpace(value)
	valueLower := strings.ToLower(value)

	// Check for URL (must check before domain)
	if strings.HasPrefix(valueLower, "http://") || strings.HasPrefix(valueLower, "https://") ||
		strings.HasPrefix(valueLower, "ftp://") {
		return model.IoCTypeURL
	}

	// Check for IP address
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() == nil {
			return model.IoCTypeIPv6
		}
		return model.IoCTypeIPv4
	}

	// Check for hash values
	if md5Regex.MatchString(valueLower) {
		return model.IoCTypeMD5
	}
	if sha1Regex.MatchString(valueLower) {
		return model.IoCTypeSHA1
	}
	if sha256Regex.MatchString(valueLower) {
		return model.IoCTypeSHA256
	}

	// Check for URL (more lenient check)
	if u, err := url.Parse(value); err == nil && u.Scheme != "" && u.Host != "" {
		return model.IoCTypeURL
	}

	// Check for domain
	if domainRegex.MatchString(value) {
		return model.IoCTypeDomain
	}

	// Unknown type - return empty string to skip
	return ""
}
