package feed

import (
	"context"
)

// IPsum feeds - all are simple TXT format with one IP per line from GitHub

// Default URLs for IPsum feeds
const (
	IPsumLevel3URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/3.txt"
	IPsumLevel4URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/4.txt"
	IPsumLevel5URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/5.txt"
	IPsumLevel6URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/6.txt"
	IPsumLevel7URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/7.txt"
	IPsumLevel8URL = "https://raw.githubusercontent.com/stamparm/ipsum/master/levels/8.txt"
)

// FetchIPsumLevel3 fetches IPsum threat level 3 feed
func (s *Service) FetchIPsumLevel3(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel3URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-3"})
}

// FetchIPsumLevel4 fetches IPsum threat level 4 feed
func (s *Service) FetchIPsumLevel4(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel4URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-4"})
}

// FetchIPsumLevel5 fetches IPsum threat level 5 feed
func (s *Service) FetchIPsumLevel5(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel5URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-5"})
}

// FetchIPsumLevel6 fetches IPsum threat level 6 feed
func (s *Service) FetchIPsumLevel6(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel6URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-6"})
}

// FetchIPsumLevel7 fetches IPsum threat level 7 feed
func (s *Service) FetchIPsumLevel7(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel7URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-7"})
}

// FetchIPsumLevel8 fetches IPsum threat level 8 feed
func (s *Service) FetchIPsumLevel8(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = IPsumLevel8URL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"ipsum", "threat-level-8"})
}
