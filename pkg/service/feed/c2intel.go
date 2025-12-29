package feed

import (
	"context"
	"encoding/csv"
	"io"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/httpclient"
)

// C2IntelFeeds - CSV format feeds from GitHub

// Default URLs for C2IntelFeeds
const (
	C2IntelIPListURL              = "https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/IPC2s-30day.csv"
	C2IntelDomainListURL          = "https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2s-30day-filter-abused.csv"
	C2IntelDomainWithURLURL       = "https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2swithURL-30day-filter-abused.csv"
	C2IntelDomainWithURLWithIPURL = "https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2swithURLwithIP-30day-filter-abused.csv"
)

// FetchC2IntelIPList fetches IP C2s feed
// Format: #ip,ioc (CSV with IP addresses and descriptions)
func (s *Service) FetchC2IntelIPList(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = C2IntelIPListURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch C2Intel IP feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 2 {
			continue
		}

		ip := strings.TrimSpace(record[0])
		description := strings.TrimSpace(record[1])

		entry := &FeedEntry{
			ID:          "",
			Type:        model.IoCTypeIPv4,
			Value:       ip,
			Description: description,
			Tags:        []string{"c2", "command-control", "c2intel"},
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchC2IntelDomainList fetches Domain C2s feed
// Format: #domain,ioc (CSV with domains and descriptions)
func (s *Service) FetchC2IntelDomainList(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = C2IntelDomainListURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch C2Intel Domain feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 2 {
			continue
		}

		domain := strings.TrimSpace(record[0])
		description := strings.TrimSpace(record[1])

		entry := &FeedEntry{
			ID:          "",
			Type:        model.IoCTypeDomain,
			Value:       domain,
			Description: description,
			Tags:        []string{"c2", "command-control", "c2intel"},
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchC2IntelDomainWithURL fetches Domain C2s with URLs feed
// This feed contains domains that may include URLs
func (s *Service) FetchC2IntelDomainWithURL(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = C2IntelDomainWithURLURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch C2Intel Domain with URL feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 2 {
			continue
		}

		value := strings.TrimSpace(record[0])
		description := strings.TrimSpace(record[1])

		// Detect if it's a URL or domain
		iocType := model.IoCTypeDomain
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			iocType = model.IoCTypeURL
		}

		entry := &FeedEntry{
			ID:          "",
			Type:        iocType,
			Value:       value,
			Description: description,
			Tags:        []string{"c2", "command-control", "c2intel"},
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchC2IntelDomainWithURLWithIP fetches Domain C2s with URLs and IPs feed
// This feed may contain domains, URLs, or IPs
func (s *Service) FetchC2IntelDomainWithURLWithIP(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = C2IntelDomainWithURLWithIPURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch C2Intel comprehensive feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 2 {
			continue
		}

		value := strings.TrimSpace(record[0])
		description := strings.TrimSpace(record[1])

		// Auto-detect IoC type
		iocType := detectIoCType(value)
		if iocType == "" {
			// Skip if type cannot be determined
			continue
		}

		entry := &FeedEntry{
			ID:          "",
			Type:        iocType,
			Value:       value,
			Description: description,
			Tags:        []string{"c2", "command-control", "c2intel"},
			FirstSeen:   time.Now(),
			LastSeen:    time.Now(),
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
