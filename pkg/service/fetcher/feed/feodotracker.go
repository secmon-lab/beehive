package feed

import (
	"context"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/fetcher"
)

const FeodoTrackerTypeID = "feodotracker_ipblocklist"
const feodoURL = "https://feodotracker.abuse.ch/downloads/ipblocklist.txt"

type FeodoTrackerProvider struct {
	http *fetcher.HTTPClient
}

func NewFeodoTracker(c *fetcher.HTTPClient) *FeodoTrackerProvider {
	return &FeodoTrackerProvider{http: c}
}

func (p *FeodoTrackerProvider) Kind() types.SourceKind { return types.KindFeed }

func (p *FeodoTrackerProvider) Fetch(ctx context.Context, _ *model.Source) ([]*interfaces.IoCSeed, error) {
	body, err := p.http.Get(ctx, feodoURL, nil)
	if err != nil {
		return nil, err
	}
	return ParseFeodoTracker(body), nil
}

func ParseFeodoTracker(raw []byte) []*interfaces.IoCSeed {
	now := time.Now().UTC()
	var out []*interfaces.IoCSeed
	iterText(raw, func(line string) {
		if seed, ok := seedIP(line, 0.95, now); ok {
			out = append(out, seed)
		}
	})
	return out
}
