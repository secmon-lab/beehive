package feed

import (
	"context"
)

// Default URL for CINSscore feed
const (
	CINSscoreBadguysURL = "https://cinsscore.com/list/ci-badguys.txt"
)

// FetchCINSscoreBadguys fetches Bad Guys list from CINSscore
func (s *Service) FetchCINSscoreBadguys(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = CINSscoreBadguysURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"cinsscore", "badguy"})
}
