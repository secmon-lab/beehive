package feed

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
)

const ThreatFoxTypeID = "threatfox_recent"
const threatFoxURL = "https://threatfox-api.abuse.ch/api/v1/"

type ThreatFoxProvider struct {
	http *fetcher.HTTPClient
}

func NewThreatFox(c *fetcher.HTTPClient) *ThreatFoxProvider {
	return &ThreatFoxProvider{http: c}
}

func (p *ThreatFoxProvider) Kind() types.SourceKind { return types.KindFeed }

type threatFoxResp struct {
	Data []struct {
		IOC        string `json:"ioc"`
		IOCType    string `json:"ioc_type"`
		Confidence int    `json:"confidence_level"`
		FirstSeen  string `json:"first_seen"`
	} `json:"data"`
}

func (p *ThreatFoxProvider) Fetch(ctx context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	// ThreatFox uses a POST body that selects the recent endpoint.
	// For MVP we hit the public "get_iocs" endpoint via GET semantics
	// the public mirror exposes — refine when a real source picks
	// this up.
	body, err := p.http.Get(ctx, threatFoxURL+"?query=get_iocs&days=1", nil)
	if err != nil {
		return nil, err
	}
	return ParseThreatFox(body)
}

func ParseThreatFox(raw []byte) ([]*interfaces.IoCSeed, error) {
	var payload threatFoxResp
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, goerr.Wrap(err, "decode threatfox response")
	}
	now := time.Now().UTC()
	out := make([]*interfaces.IoCSeed, 0, len(payload.Data))
	for _, d := range payload.Data {
		seed := mapThreatFox(d.IOC, d.IOCType, float64(d.Confidence)/100.0, now)
		if seed == nil {
			continue
		}
		if t, err := time.Parse(time.RFC3339, d.FirstSeen); err == nil {
			seed.SeenAt = t.UTC()
		}
		out = append(out, seed)
	}
	return out, nil
}

func mapThreatFox(value, kind string, confidence float64, now time.Time) *interfaces.IoCSeed {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch kind {
	case "ip:port":
		// "1.2.3.4:80" — split off the port.
		host := value
		if i := strings.LastIndex(value, ":"); i > 0 {
			host = value[:i]
		}
		if seed, ok := seedIP(host, confidence, now); ok {
			return seed
		}
		return nil
	case "domain":
		return &interfaces.IoCSeed{
			Type: types.IoCTypeDomain, Value: strings.ToLower(value), Raw: value,
			Confidence: confidence, SeenAt: now,
		}
	case "url":
		return seedURL(value, confidence, now)
	case "md5_hash":
		return &interfaces.IoCSeed{Type: types.IoCTypeMD5, Value: strings.ToLower(value), Raw: value, Confidence: confidence, SeenAt: now}
	case "sha1_hash":
		return &interfaces.IoCSeed{Type: types.IoCTypeSHA1, Value: strings.ToLower(value), Raw: value, Confidence: confidence, SeenAt: now}
	case "sha256_hash":
		return &interfaces.IoCSeed{Type: types.IoCTypeSHA256, Value: strings.ToLower(value), Raw: value, Confidence: confidence, SeenAt: now}
	default:
		return nil
	}
}
