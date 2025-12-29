package feed

import (
	"context"
)

// ThreatView.io feeds - mixed IoC types (IP, Domain, URL, Hash) in TXT format

// Default URLs for ThreatView.io feeds
const (
	ThreatViewIOCTweetsURL    = "https://threatview.io/Downloads/Experimental-IOC-Tweets.txt"
	ThreatViewCobaltStrikeURL = "https://threatview.io/Downloads/High-Confidence-CobaltStrike-C2%20-Feeds.txt"
	ThreatViewIPHighURL       = "https://threatview.io/Downloads/IP-High-Confidence-Feed.txt"
	ThreatViewDomainHighURL   = "https://threatview.io/Downloads/DOMAIN-High-Confidence-Feed.txt"
	ThreatViewMD5URL          = "https://threatview.io/Downloads/MD5-HASH-ALL.txt"
	ThreatViewURLHighURL      = "https://threatview.io/Downloads/URL-High-Confidence-Feed.txt"
	ThreatViewSHAURL          = "https://threatview.io/Downloads/SHA-HASH-FEED.txt"
)

// FetchThreatViewIOCTweets fetches Experimental IOC Tweets feed
func (s *Service) FetchThreatViewIOCTweets(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewIOCTweetsURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "ioc-tweets"})
}

// FetchThreatViewCobaltStrike fetches High Confidence CobaltStrike C2 feed
func (s *Service) FetchThreatViewCobaltStrike(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewCobaltStrikeURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "cobalt-strike", "high-confidence"})
}

// FetchThreatViewIPHigh fetches IP High Confidence feed
func (s *Service) FetchThreatViewIPHigh(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewIPHighURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "high-confidence", "ip"})
}

// FetchThreatViewDomainHigh fetches Domain High Confidence feed
func (s *Service) FetchThreatViewDomainHigh(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewDomainHighURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "high-confidence", "domain"})
}

// FetchThreatViewMD5 fetches MD5 Hash feed
func (s *Service) FetchThreatViewMD5(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewMD5URL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "md5"})
}

// FetchThreatViewURLHigh fetches URL High Confidence feed
func (s *Service) FetchThreatViewURLHigh(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewURLHighURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "high-confidence", "url"})
}

// FetchThreatViewSHA fetches SHA Hash feed
func (s *Service) FetchThreatViewSHA(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = ThreatViewSHAURL
	}
	return s.FetchMixedIoCList(ctx, feedURL, []string{"threatview", "sha"})
}
