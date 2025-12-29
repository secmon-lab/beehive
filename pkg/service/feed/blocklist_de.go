package feed

import (
	"context"
)

// Blocklist.de feeds - all are simple TXT format with one IP per line

// Default URLs for Blocklist.de feeds
const (
	BlocklistDeAllURL        = "https://lists.blocklist.de/lists/all.txt"
	BlocklistDeSSHURL        = "https://lists.blocklist.de/lists/ssh.txt"
	BlocklistDeMailURL       = "https://lists.blocklist.de/lists/mail.txt"
	BlocklistDeApacheURL     = "https://lists.blocklist.de/lists/apache.txt"
	BlocklistDeIMAPURL       = "https://lists.blocklist.de/lists/imap.txt"
	BlocklistDeBotsURL       = "https://lists.blocklist.de/lists/bots.txt"
	BlocklistDeBruteforceURL = "https://lists.blocklist.de/lists/bruteforcelogin.txt"
	BlocklistDeStrongIPsURL  = "https://lists.blocklist.de/lists/strongips.txt"
	BlocklistDeFTPURL        = "https://lists.blocklist.de/lists/ftp.txt"
)

// FetchBlocklistDeAll fetches the All blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeAll(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeAllURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "attack"})
}

// FetchBlocklistDeSSH fetches the SSH attack blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeSSH(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeSSHURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "ssh-attack"})
}

// FetchBlocklistDeMail fetches the Mail attack blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeMail(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeMailURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "mail-attack"})
}

// FetchBlocklistDeApache fetches the Apache attack blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeApache(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeApacheURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "apache-attack"})
}

// FetchBlocklistDeIMAP fetches the IMAP attack blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeIMAP(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeIMAPURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "imap-attack"})
}

// FetchBlocklistDeBots fetches the Bots blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeBots(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeBotsURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "bot"})
}

// FetchBlocklistDeBruteforce fetches the Bruteforce Login blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeBruteforce(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeBruteforceURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "bruteforce"})
}

// FetchBlocklistDeStrongIPs fetches the Strong IPs blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeStrongIPs(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeStrongIPsURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "strong-attack"})
}

// FetchBlocklistDeFTP fetches the FTP attack blocklist from Blocklist.de
func (s *Service) FetchBlocklistDeFTP(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = BlocklistDeFTPURL
	}
	return s.FetchSimpleIPList(ctx, feedURL, []string{"blocklist-de", "ftp-attack"})
}
