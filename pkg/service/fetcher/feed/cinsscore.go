package feed

import (
	"context"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
)

const CINSScoreTypeID = "cinsscore_badguys"
const cinsURL = "https://cinsscore.com/list/ci-badguys.txt"

// CINSScoreProvider downloads the plaintext list of IPs from CINS.
type CINSScoreProvider struct {
	http *fetcher.HTTPClient
}

func NewCINSScore(c *fetcher.HTTPClient) *CINSScoreProvider {
	return &CINSScoreProvider{http: c}
}

func (p *CINSScoreProvider) Kind() types.SourceKind { return types.KindFeed }

func (p *CINSScoreProvider) Fetch(ctx context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	body, err := p.http.Get(ctx, cinsURL, nil)
	if err != nil {
		return nil, err
	}
	return ParseCINSScore(body), nil
}

// ParseCINSScore is the pure parser, exposed for unit tests.
func ParseCINSScore(raw []byte) []*interfaces.IoCSeed {
	now := time.Now().UTC()
	var out []*interfaces.IoCSeed
	iterText(raw, func(line string) {
		if seed, ok := seedIP(line, 0.9, now); ok {
			out = append(out, seed)
		}
	})
	return out
}
