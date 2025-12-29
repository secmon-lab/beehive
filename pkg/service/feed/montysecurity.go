package feed

import (
	"context"
)

// Montysecurity C2 Tracker feeds - all are simple TXT format with one IP per line from GitHub

// Default URLs for Montysecurity C2 Tracker feeds
const (
	MontysecurityBruteRatelURL   = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Brute%20Ratel%20C4%20IPs.txt"
	MontysecurityCobaltStrikeURL = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Cobalt%20Strike%20C2%20IPs.txt"
	MontysecuritySliverURL       = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Sliver%20C2%20IPs.txt"
	MontysecurityMetasploitURL   = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Metasploit%20Framework%20C2%20IPs.txt"
	MontysecurityHavocURL        = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Havoc%20C2%20IPs.txt"
	MontysecurityBurpSuiteURL    = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/BurpSuite%20IPs.txt"
	MontysecurityDeimosURL       = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Deimos%20C2%20IPs.txt"
	MontysecurityGoPhishURL      = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/GoPhish%20IPs.txt"
	MontysecurityMythicURL       = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Mythic%20C2%20IPs.txt"
	MontysecurityNimPlantURL     = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/NimPlant%20C2%20IPs.txt"
	MontysecurityPANDAURL        = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/PANDA%20C2%20IPs.txt"
	MontysecurityXMRigURL        = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/XMRig%20Monero%20Cryptominer%20IPs.txt"
	MontysecurityAllURL          = "https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/all.txt"
)

// FetchMontysecurityBruteRatel fetches Brute Ratel C4 IPs
func (s *Service) FetchMontysecurityBruteRatel(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityBruteRatelURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "brute-ratel"})
}

// FetchMontysecurityCobaltStrike fetches Cobalt Strike C2 IPs
func (s *Service) FetchMontysecurityCobaltStrike(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityCobaltStrikeURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "cobalt-strike"})
}

// FetchMontysecuritySliver fetches Sliver C2 IPs
func (s *Service) FetchMontysecuritySliver(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecuritySliverURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "sliver"})
}

// FetchMontysecurityMetasploit fetches Metasploit Framework C2 IPs
func (s *Service) FetchMontysecurityMetasploit(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityMetasploitURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "metasploit"})
}

// FetchMontysecurityHavoc fetches Havoc C2 IPs
func (s *Service) FetchMontysecurityHavoc(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityHavocURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "havoc"})
}

// FetchMontysecurityBurpSuite fetches BurpSuite IPs
func (s *Service) FetchMontysecurityBurpSuite(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityBurpSuiteURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "burpsuite"})
}

// FetchMontysecurityDeimos fetches Deimos C2 IPs
func (s *Service) FetchMontysecurityDeimos(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityDeimosURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "deimos"})
}

// FetchMontysecurityGoPhish fetches GoPhish IPs
func (s *Service) FetchMontysecurityGoPhish(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityGoPhishURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "gophish"})
}

// FetchMontysecurityMythic fetches Mythic C2 IPs
func (s *Service) FetchMontysecurityMythic(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityMythicURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "mythic"})
}

// FetchMontysecurityNimPlant fetches NimPlant C2 IPs
func (s *Service) FetchMontysecurityNimPlant(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityNimPlantURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "nimplant"})
}

// FetchMontysecurityPANDA fetches PANDA C2 IPs
func (s *Service) FetchMontysecurityPANDA(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityPANDAURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "panda"})
}

// FetchMontysecurityXMRig fetches XMRig Monero Cryptominer IPs
func (s *Service) FetchMontysecurityXMRig(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityXMRigURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "xmrig", "cryptominer"})
}

// FetchMontysecurityAll fetches all C2 IPs from Montysecurity
func (s *Service) FetchMontysecurityAll(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = MontysecurityAllURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"c2", "montysecurity"})
}
