package model

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
)

const (
	// EmbeddingDimension is the dimension size for text embeddings
	// Using 128 dimensions for efficient storage and retrieval
	// This value is fixed and cannot be changed as Firestore vector index
	// is created with a specific dimension size
	EmbeddingDimension = 128
)

// IoCType represents the type of IoC
type IoCType string

const (
	// Network IOCs
	IoCTypeIPv4    IoCType = "ipv4"
	IoCTypeIPv6    IoCType = "ipv6"
	IoCTypeDomain  IoCType = "domain"
	IoCTypeURL     IoCType = "url"
	IoCTypeEmail   IoCType = "email"
	IoCTypeMacAddr IoCType = "mac-addr"
	IoCTypeASN     IoCType = "asn"

	// File IOCs
	IoCTypeMD5      IoCType = "md5"
	IoCTypeSHA1     IoCType = "sha1"
	IoCTypeSHA256   IoCType = "sha256"
	IoCTypeFilename IoCType = "filename"

	// System IOCs
	IoCTypeProcess   IoCType = "process"      // Process name or command line
	IoCTypeMutex     IoCType = "mutex"        // Mutex name
	IoCTypeRegKey    IoCType = "registry-key" // Windows registry key
	IoCTypeUserAgent IoCType = "user-agent"   // HTTP User-Agent string
	IoCTypeCertHash  IoCType = "cert-hash"    // X.509 certificate hash
)

// IoCStatus represents the status of an IoC
type IoCStatus string

const (
	IoCStatusActive   IoCStatus = "active"   // Currently active IoC
	IoCStatusInactive IoCStatus = "inactive" // No longer active (removed from feed)
)

// IoC represents an Indicator of Compromise
type IoC struct {
	ID          string             // Unique identifier: hash(SourceID + Type + normalized Value + ContextKey)
	SourceID    string             // Source identifier from config
	SourceType  string             // "rss" or "feed"
	Type        IoCType            // IoC type
	Value       string             // IoC value (IP, domain, hash, etc) - normalized
	Description string             // Human-readable description
	SourceURL   string             // Original URL where this IoC was found
	Context     string             // Additional context (article text, surrounding text, etc)
	Embedding   firestore.Vector32 // Vector embedding for semantic search
	Status      IoCStatus          // Active or inactive status
	FirstSeenAt time.Time          // First time this IoC was observed
	UpdatedAt   time.Time          // Last update time
}

// IoCContextKey represents a context-aware unique key for deduplication.
// Each feed/source type can define its own deduplication strategy using this key.
type IoCContextKey string

var (
	// ErrInvalidIoCType is returned when an invalid IoC type is provided
	ErrInvalidIoCType = goerr.New("invalid IoC type")
	// ErrInvalidIoCValue is returned when an invalid IoC value is provided
	ErrInvalidIoCValue = goerr.New("invalid IoC value")
)

// GenerateID generates a unique ID for an IoC based on source ID, type, normalized value, and context key.
// The context key allows different feed types to implement their own deduplication semantics.
func GenerateID(sourceID string, iocType IoCType, value string, contextKey IoCContextKey) string {
	normalized := NormalizeValue(iocType, value)
	hash := sha256.Sum256([]byte(
		sourceID + ":" +
			string(iocType) + ":" +
			normalized + ":" +
			string(contextKey),
	))
	return fmt.Sprintf("ioc_%x", hash[:16]) // Use first 128 bits
}

// GenerateContextKey generates a context-aware key for IoC deduplication.
// This function implements feed-type-specific deduplication strategies:
//
// For "feed" type (e.g., abuse.ch):
//   - If feed provides entry_id: uses entry_id as context (same entry = same IoC)
//   - If no entry_id: uses empty string to deduplicate by IoC value within the source
//
// For "rss" type (blog feeds):
//   - Uses article_guid as context (same article + same IoC value = same IoC)
//   - Different articles with same IoC value are treated as separate observations
//
// For feeds without unique identifiers:
//   - Returns empty string, causing deduplication to occur purely by IoC value within the source
//   - This means multiple occurrences of the same IoC value in the feed will be merged into one
//
// Parameters can include:
//   - "entry_id": Unique identifier from feed entry (abuse.ch feeds)
//   - "article_guid": Article GUID from RSS feed
//   - Any other feed-specific identifier
func GenerateContextKey(sourceType string, params map[string]string) IoCContextKey {
	switch sourceType {
	case "feed":
		// For feeds with entry IDs (abuse.ch URLhaus, ThreatFox, etc.)
		// Use the entry ID to ensure each feed entry creates a separate IoC record
		if entryID := params["entry_id"]; entryID != "" {
			return IoCContextKey(entryID)
		}
		// If no entry_id is provided, deduplicate by IoC value only
		// This means the same IoC appearing multiple times in the feed will be merged
		return IoCContextKey("")

	case "rss":
		// For RSS feeds, use article GUID as context
		// This ensures IoCs from the same article are deduplicated,
		// but the same IoC from different articles creates separate records
		if articleGUID := params["article_guid"]; articleGUID != "" {
			return IoCContextKey(articleGUID)
		}
		// Fallback: use article URL if GUID is not available
		if articleURL := params["article_url"]; articleURL != "" {
			return IoCContextKey(articleURL)
		}
		// If neither is available, deduplicate by IoC value only
		return IoCContextKey("")

	default:
		// For unknown source types, create a deterministic context from all parameters
		// This provides extensibility for future feed types
		if len(params) == 0 {
			return IoCContextKey("")
		}

		var keys []string
		for k := range params {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			parts = append(parts, k+"="+params[k])
		}
		return IoCContextKey(strings.Join(parts, "&"))
	}
}

// NormalizeValue normalizes an IoC value based on its type
func NormalizeValue(iocType IoCType, value string) string {
	switch iocType {
	case IoCTypeIPv4, IoCTypeIPv6:
		// Parse and canonicalize IP address
		ip := net.ParseIP(strings.TrimSpace(value))
		if ip != nil {
			return ip.String()
		}
		return strings.ToLower(strings.TrimSpace(value))

	case IoCTypeDomain:
		// Lowercase and remove trailing dot
		domain := strings.ToLower(strings.TrimSpace(value))
		return strings.TrimSuffix(domain, ".")

	case IoCTypeURL:
		// Parse URL and normalize
		u, err := url.Parse(strings.TrimSpace(value))
		if err == nil {
			// Lowercase scheme and host
			u.Scheme = strings.ToLower(u.Scheme)
			u.Host = strings.ToLower(u.Host)
			return u.String()
		}
		return strings.TrimSpace(value)

	case IoCTypeEmail:
		// Lowercase email address
		return strings.ToLower(strings.TrimSpace(value))

	case IoCTypeMD5, IoCTypeSHA1, IoCTypeSHA256, IoCTypeCertHash:
		// Lowercase hash values
		return strings.ToLower(strings.TrimSpace(value))

	case IoCTypeMacAddr:
		// Normalize MAC address format (remove separators, lowercase)
		mac := strings.ToLower(strings.TrimSpace(value))
		mac = strings.ReplaceAll(mac, ":", "")
		mac = strings.ReplaceAll(mac, "-", "")
		return mac

	case IoCTypeASN:
		// Remove "AS" prefix if present, normalize to number
		asn := strings.TrimSpace(value)
		asn = strings.TrimPrefix(strings.ToUpper(asn), "AS")
		return asn

	default:
		// For other types, just trim whitespace
		return strings.TrimSpace(value)
	}
}

// ValidateIoC validates an IoC structure
func ValidateIoC(ioc *IoC) error {
	if ioc.SourceID == "" {
		return goerr.Wrap(ErrInvalidIoCValue, "source_id is required", goerr.V("field", "source_id"))
	}
	if ioc.Type == "" {
		return goerr.Wrap(ErrInvalidIoCType, "type is required", goerr.V("type", ioc.Type))
	}
	if ioc.Value == "" {
		return goerr.Wrap(ErrInvalidIoCValue, "value is required", goerr.V("field", "value"))
	}
	if ioc.Status != IoCStatusActive && ioc.Status != IoCStatusInactive {
		return goerr.Wrap(ErrInvalidIoCValue, "invalid status", goerr.V("field", "status"), goerr.V("value", ioc.Status))
	}
	if len(ioc.Embedding) > 0 && len(ioc.Embedding) != EmbeddingDimension {
		return goerr.Wrap(ErrInvalidIoCValue, "invalid embedding dimension",
			goerr.V("field", "embedding"),
			goerr.V("expected_dimension", EmbeddingDimension),
			goerr.V("actual_dimension", len(ioc.Embedding)))
	}
	return nil
}

var (
	md5Regex    = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
	sha1Regex   = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
	sha256Regex = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
	domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	emailRegex  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// DetectIoCType attempts to detect the type of an IoC value
func DetectIoCType(value string) IoCType {
	value = strings.TrimSpace(value)

	// Check hash types first (most specific patterns)
	if sha256Regex.MatchString(value) {
		return IoCTypeSHA256
	}
	if sha1Regex.MatchString(value) {
		return IoCTypeSHA1
	}
	if md5Regex.MatchString(value) {
		return IoCTypeMD5
	}

	// Check IP addresses
	if ip := net.ParseIP(value); ip != nil {
		if ip.To4() != nil {
			return IoCTypeIPv4
		}
		return IoCTypeIPv6
	}

	// Check URL
	if u, err := url.Parse(value); err == nil && u.Scheme != "" && u.Host != "" {
		return IoCTypeURL
	}

	// Check email
	if emailRegex.MatchString(value) {
		return IoCTypeEmail
	}

	// Check domain
	if domainRegex.MatchString(value) {
		return IoCTypeDomain
	}

	// Default: treat as generic value (could be filename, process, etc.)
	return IoCTypeFilename
}
