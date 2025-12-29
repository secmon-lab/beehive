package feed

import (
	"context"
)

// Default URL for GreenSnow feed
const (
	GreenSnowBlocklistURL = "https://blocklist.greensnow.co/greensnow.txt"
)

// FetchGreenSnowBlocklist fetches GreenSnow blocklist
func (s *Service) FetchGreenSnowBlocklist(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = GreenSnowBlocklistURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"greensnow", "blocklist"})
}
