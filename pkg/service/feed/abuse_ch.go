package feed

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/httpclient"
)

var (
	errParseFailed = goerr.New("failed to parse feed")
)

// FeedEntry represents a single entry from a threat intelligence feed
type FeedEntry struct {
	ID          string // Unique identifier for this entry
	Type        model.IoCType
	Value       string
	Description string
	Tags        []string
	FirstSeen   time.Time
	LastSeen    time.Time
}

// Service provides threat intelligence feed fetching and parsing
type Service struct {
	client httpclient.HTTPClient
}

// New creates a new feed service
func New() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchAbuseCHURLhaus fetches and parses URLhaus feed from abuse.ch
// Format: id,dateadded,url,url_status,last_online,threat,tags,urlhaus_link,reporter
func (s *Service) FetchAbuseCHURLhaus(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch URLhaus feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1 // Allow variable number of fields

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
		if len(record) < 9 {
			continue
		}

		// Parse fields
		id := record[0]
		dateAdded := parseDate(record[1])
		urlValue := record[2]
		threat := record[5]
		tags := parseTags(record[6])

		entry := &FeedEntry{
			ID:          id,
			Type:        model.IoCTypeURL,
			Value:       urlValue,
			Description: threat,
			Tags:        tags,
			FirstSeen:   dateAdded,
			LastSeen:    dateAdded,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchAbuseCHThreatFox fetches and parses ThreatFox feed from abuse.ch
// Format: first_seen_utc,ioc_id,ioc_value,ioc_type,threat_type,fk_malware,malware_alias,malware_printable,last_seen_utc,confidence_level,reference,tags,anonymous,reporter
func (s *Service) FetchAbuseCHThreatFox(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch ThreatFox feed")
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
		if len(record) < 14 {
			continue
		}

		// Parse fields
		firstSeen := parseDate(record[0])
		id := record[1]
		iocValue := record[2]
		iocTypeStr := record[3]
		threatType := record[4]
		malware := record[7] // malware_printable
		tags := parseTags(record[11])

		// Map ThreatFox IOC type to our IOC type
		iocType := mapThreatFoxType(iocTypeStr)

		description := threatType
		if malware != "" {
			description = malware + ": " + threatType
		}

		entry := &FeedEntry{
			ID:          id,
			Type:        iocType,
			Value:       iocValue,
			Description: description,
			Tags:        tags,
			FirstSeen:   firstSeen,
			LastSeen:    firstSeen,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchFeed fetches a feed based on the schema name
func (s *Service) FetchFeed(ctx context.Context, feedURL, schema string) ([]*FeedEntry, error) {
	switch schema {
	case "abuse_ch_urlhaus":
		return s.FetchAbuseCHURLhaus(ctx, feedURL)
	case "abuse_ch_threatfox":
		return s.FetchAbuseCHThreatFox(ctx, feedURL)
	default:
		return nil, goerr.New("unsupported feed schema", goerr.V("schema", schema))
	}
}

// parseDate parses date string in various formats
func parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Now()
	}

	// Try common date formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Fallback to current time
	return time.Now()
}

// parseTags parses a comma-separated tag string
func parseTags(tagStr string) []string {
	tagStr = strings.TrimSpace(tagStr)
	if tagStr == "" {
		return nil
	}

	tags := strings.Split(tagStr, ",")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// mapThreatFoxType maps ThreatFox IOC type to our IOC type
func mapThreatFoxType(tfType string) model.IoCType {
	tfType = strings.ToLower(strings.TrimSpace(tfType))

	switch tfType {
	case "ip:port", "ip":
		return model.IoCTypeIPv4
	case "domain":
		return model.IoCTypeDomain
	case "url":
		return model.IoCTypeURL
	case "md5_hash":
		return model.IoCTypeMD5
	case "sha1_hash":
		return model.IoCTypeSHA1
	case "sha256_hash":
		return model.IoCTypeSHA256
	case "email":
		return model.IoCTypeEmail
	default:
		// Try to auto-detect
		return model.IoCTypeFilename // Default fallback
	}
}
