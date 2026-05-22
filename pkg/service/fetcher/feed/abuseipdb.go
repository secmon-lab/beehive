package feed

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// AbuseIPDBAPIKeyEnv names the env var that supplies the API key for
// the AbuseIPDB blacklist endpoint. The pkg/service/providers wiring
// reads it once at startup; the provider itself never touches os.Getenv.
const AbuseIPDBAPIKeyEnv = "BEEHIVE_PROVIDER_ABUSEIPDB_API_KEY"

const AbuseIPDBTypeID = "abuseipdb_blacklist"
const abuseIPDBURL = "https://api.abuseipdb.com/api/v2/blacklist"

// AbuseIPDBProvider pulls AbuseIPDB's blacklist feed.
type AbuseIPDBProvider struct {
	http   *fetcher.HTTPClient
	apiKey string
}

func NewAbuseIPDB(c *fetcher.HTTPClient, apiKey string) *AbuseIPDBProvider {
	return &AbuseIPDBProvider{http: c, apiKey: apiKey}
}

func (p *AbuseIPDBProvider) Kind() types.SourceKind { return types.KindFeed }

type abuseIPDBResponse struct {
	Data []struct {
		IPAddress            string `json:"ipAddress"`
		AbuseConfidenceScore int    `json:"abuseConfidenceScore"`
		LastReportedAt       string `json:"lastReportedAt"`
	} `json:"data"`
}

func (p *AbuseIPDBProvider) Fetch(ctx context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	if p.apiKey == "" {
		return nil, goerr.New("BEEHIVE_PROVIDER_ABUSEIPDB_API_KEY is required",
			goerr.T(errutil.TagInvalidInput))
	}
	body, err := p.http.Get(ctx, abuseIPDBURL, map[string]string{
		"Key":    p.apiKey,
		"Accept": "application/json",
	})
	if err != nil {
		return nil, err
	}
	var payload abuseIPDBResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, goerr.Wrap(err, "decode abuseipdb response")
	}
	return ParseAbuseIPDB(body)
}

// ParseAbuseIPDB is exported so unit tests can drive it with fixtures
// without needing a live HTTP client.
func ParseAbuseIPDB(raw []byte) ([]*interfaces.IoCSeed, error) {
	var payload abuseIPDBResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, goerr.Wrap(err, "decode abuseipdb response")
	}
	now := time.Now().UTC()
	out := make([]*interfaces.IoCSeed, 0, len(payload.Data))
	for _, d := range payload.Data {
		confidence := float64(d.AbuseConfidenceScore) / 100.0
		seed, ok := seedIP(d.IPAddress, confidence, now)
		if !ok {
			continue
		}
		if t, err := time.Parse(time.RFC3339, d.LastReportedAt); err == nil {
			seed.SeenAt = t.UTC()
		}
		out = append(out, seed)
	}
	_ = strconv.Itoa // silence unused if score arithmetic shape changes
	return out, nil
}
