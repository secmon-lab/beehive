package feed

import (
	"context"
)

// Default URL for Emerging Threats feed
const (
	EmergingThreatsCompromisedIPURL = "https://rules.emergingthreats.net/blockrules/compromised-ips.txt"
)

// FetchEmergingThreatsCompromisedIP fetches Compromised IPs from Proofpoint Emerging Threats
func (s *Service) FetchEmergingThreatsCompromisedIP(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = EmergingThreatsCompromisedIPURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"emerging-threats", "compromised"})
}
