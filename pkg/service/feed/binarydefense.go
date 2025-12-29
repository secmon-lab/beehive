package feed

import (
	"context"
)

// Default URL for Binarydefense feed
const (
	BinarydefenseBanlistURL = "https://www.binarydefense.com/banlist.txt"
)

// FetchBinarydefenseBanlist fetches Artillery Threat Intelligence banlist from Binarydefense
func (s *Service) FetchBinarydefenseBanlist(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BinarydefenseBanlistURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"binarydefense", "artillery"})
}
