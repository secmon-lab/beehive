package feed

import (
	"context"
	"encoding/csv"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/utils/httpclient"
)

var (
	errParseFailed = goerr.New("failed to parse feed")
)

// SchemaDescriptions maps feed schema names to human-readable descriptions
var SchemaDescriptions = map[string]string{
	// Abuse.ch feeds
	"abuse_ch_urlhaus":         "URLhaus - Malicious URLs used for malware distribution",
	"abuse_ch_threatfox":       "ThreatFox - Indicators of Compromise (IOCs) shared by the community",
	"abuse_ch_feodotracker_ip": "Feodotracker - IP addresses of Feodo/Emotet/Dridex C&C servers",
	"abuse_ch_sslbl":           "SSL Blacklist - Malicious SSL certificates",

	// Blocklist.de feeds
	"blocklist_de_all":        "Blocklist.de All - All IP addresses that attacked servers",
	"blocklist_de_ssh":        "Blocklist.de SSH - IP addresses that attempted SSH brute force attacks",
	"blocklist_de_mail":       "Blocklist.de Mail - IP addresses that attempted mail server attacks",
	"blocklist_de_apache":     "Blocklist.de Apache - IP addresses that attempted Apache/web attacks",
	"blocklist_de_imap":       "Blocklist.de IMAP - IP addresses that attempted IMAP attacks",
	"blocklist_de_bots":       "Blocklist.de Bots - IP addresses identified as bots",
	"blocklist_de_bruteforce": "Blocklist.de Bruteforce - IP addresses that attempted brute force attacks",
	"blocklist_de_strongips":  "Blocklist.de StrongIPs - IP addresses with strong attack patterns",
	"blocklist_de_ftp":        "Blocklist.de FTP - IP addresses that attempted FTP attacks",

	// IPsum feeds
	"ipsum_level3": "IPsum Level 3 - Malicious IPs (aggregation level 3)",
	"ipsum_level4": "IPsum Level 4 - Malicious IPs (aggregation level 4)",
	"ipsum_level5": "IPsum Level 5 - Malicious IPs (aggregation level 5)",
	"ipsum_level6": "IPsum Level 6 - Malicious IPs (aggregation level 6)",
	"ipsum_level7": "IPsum Level 7 - Malicious IPs (aggregation level 7)",
	"ipsum_level8": "IPsum Level 8 - Malicious IPs (aggregation level 8)",

	// C2IntelFeeds
	"c2intel_ipc2s":             "C2IntelFeeds - Command and Control IP addresses",
	"c2intel_domain_c2s":        "C2IntelFeeds - Command and Control domains",
	"c2intel_domain_c2s_url":    "C2IntelFeeds - Command and Control domains with URLs",
	"c2intel_domain_c2s_url_ip": "C2IntelFeeds - Command and Control domains with URLs and IPs",

	// Montysecurity C2 Tracker feeds
	"montysecurity_brute_ratel":   "Montysecurity - Brute Ratel C4 C2 servers",
	"montysecurity_cobalt_strike": "Montysecurity - Cobalt Strike C2 servers",
	"montysecurity_sliver":        "Montysecurity - Sliver C2 servers",
	"montysecurity_metasploit":    "Montysecurity - Metasploit C2 servers",
	"montysecurity_havoc":         "Montysecurity - Havoc Framework C2 servers",
	"montysecurity_burpsuite":     "Montysecurity - BurpSuite Collaborator servers",
	"montysecurity_deimos":        "Montysecurity - Deimos C2 servers",
	"montysecurity_gophish":       "Montysecurity - GoPhish phishing servers",
	"montysecurity_mythic":        "Montysecurity - Mythic C2 servers",
	"montysecurity_nimplant":      "Montysecurity - NimPlant C2 servers",
	"montysecurity_panda":         "Montysecurity - PANDA C2 servers",
	"montysecurity_xmrig":         "Montysecurity - XMRig cryptocurrency miners",
	"montysecurity_all":           "Montysecurity - All C2 and malicious servers",

	// ThreatView.io feeds
	"threatview_ioc_tweets":    "ThreatView - IOCs extracted from Twitter",
	"threatview_cobalt_strike": "ThreatView - Cobalt Strike C2 servers",
	"threatview_ip_high":       "ThreatView - High confidence malicious IPs",
	"threatview_domain_high":   "ThreatView - High confidence malicious domains",
	"threatview_md5":           "ThreatView - Malicious file MD5 hashes",
	"threatview_url_high":      "ThreatView - High confidence malicious URLs",
	"threatview_sha":           "ThreatView - Malicious file SHA hashes",

	// Other feeds
	"emerging_threats_compromised_ip": "Emerging Threats - Known compromised or attacking IP addresses",
	"binarydefense_banlist":           "BinaryDefense - IP addresses observed attacking honeypots",
	"cinsscore_badguys":               "CI Army - Malicious IP addresses from various sources",
	"greensnow_blocklist":             "GreenSnow - IP addresses with suspicious activity",
}

// Default URLs for Abuse.ch feeds
const (
	AbuseCHURLhausURL      = "https://urlhaus.abuse.ch/downloads/csv_recent/"
	AbuseCHThreatFoxURL    = "https://threatfox.abuse.ch/export/csv/recent/"
	AbuseCHFeodotrackerURL = "https://feodotracker.abuse.ch/downloads/ipblocklist.txt"
	AbuseCHSSLBlacklistURL = "https://sslbl.abuse.ch/blacklist/sslblacklist.csv"
)

// FeedEntry represents a single entry from a threat intelligence feed
type FeedEntry struct {
	ID          string // Unique identifier for this entry
	Type        model.IoCType
	Value       string
	Description string
	Tags        []string
	FirstSeen   time.Time
	LastSeen    time.Time
}

// Service provides threat intelligence feed fetching and parsing
type Service struct {
	client httpclient.HTTPClient
}

// New creates a new feed service
func New() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchAbuseCHURLhaus fetches and parses URLhaus feed from abuse.ch
// Format: id,dateadded,url,url_status,last_online,threat,tags,urlhaus_link,reporter
func (s *Service) FetchAbuseCHURLhaus(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = AbuseCHURLhausURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch URLhaus feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 9 {
			continue
		}

		// Parse fields
		id := record[0]
		dateAdded := parseDate(record[1])
		urlValue := record[2]
		lastOnline := parseDate(record[4])
		threat := record[5]
		tags := parseTags(record[6])

		entry := &FeedEntry{
			ID:          id,
			Type:        model.IoCTypeURL,
			Value:       urlValue,
			Description: threat,
			Tags:        tags,
			FirstSeen:   dateAdded,
			LastSeen:    lastOnline,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchAbuseCHThreatFox fetches and parses ThreatFox feed from abuse.ch
// Format: first_seen_utc,ioc_id,ioc_value,ioc_type,threat_type,fk_malware,malware_alias,malware_printable,last_seen_utc,confidence_level,reference,tags,anonymous,reporter
func (s *Service) FetchAbuseCHThreatFox(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = AbuseCHThreatFoxURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch ThreatFox feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 14 {
			continue
		}

		// Parse fields
		firstSeen := parseDate(record[0])
		id := record[1]
		iocValue := record[2]
		iocTypeStr := record[3]
		threatType := record[4]
		malware := record[7] // malware_printable
		lastSeen := parseDate(record[8])
		tags := parseTags(record[11])

		// Map ThreatFox IOC type to our IOC type
		iocType := mapThreatFoxType(iocTypeStr)

		description := threatType
		if malware != "" {
			description = malware + ": " + threatType
		}

		entry := &FeedEntry{
			ID:          id,
			Type:        iocType,
			Value:       iocValue,
			Description: description,
			Tags:        tags,
			FirstSeen:   firstSeen,
			LastSeen:    lastSeen,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchAbuseCHFeodotracker fetches and parses Feodotracker IP blocklist from abuse.ch
// Format: Simple TXT with one IP per line, comments starting with #
func (s *Service) FetchAbuseCHFeodotracker(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = AbuseCHFeodotrackerURL
	}
	// Use the generic simple IP list helper with appropriate tags
	return s.FetchSimpleIPList(ctx, feedURL, []string{"feodotracker", "botnet", "c2"})
}

// FetchAbuseCHSSLBlacklist fetches and parses SSL Certificate Blacklist from abuse.ch
// Format: Listingdate,SHA1,Listingreason
func (s *Service) FetchAbuseCHSSLBlacklist(ctx context.Context, feedURL string) ([]*FeedEntry, error) {
	if feedURL == "" {
		feedURL = AbuseCHSSLBlacklistURL
	}
	data, err := httpclient.FetchWithClient(ctx, s.client, feedURL)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch SSL Blacklist feed")
	}

	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	var entries []*FeedEntry
	lineNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(errParseFailed, "failed to parse CSV",
				goerr.V("line", lineNum))
		}
		lineNum++

		// Skip header or empty lines
		if len(record) < 3 {
			continue
		}

		// Parse fields
		listingDate := parseDate(record[0])
		sha1Hash := strings.TrimSpace(record[1])
		reason := strings.TrimSpace(record[2])

		entry := &FeedEntry{
			ID:          sha1Hash, // Use SHA1 hash as ID
			Type:        model.IoCTypeSHA1,
			Value:       sha1Hash,
			Description: reason,
			Tags:        []string{"ssl-cert", "malware"},
			FirstSeen:   listingDate,
			LastSeen:    listingDate,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// FetchFeed fetches a feed based on the schema name
func (s *Service) FetchFeed(ctx context.Context, feedURL, schema string) ([]*FeedEntry, error) {
	switch schema {
	// Abuse.ch feeds
	case "abuse_ch_urlhaus":
		return s.FetchAbuseCHURLhaus(ctx, feedURL)
	case "abuse_ch_threatfox":
		return s.FetchAbuseCHThreatFox(ctx, feedURL)
	case "abuse_ch_feodotracker_ip":
		return s.FetchAbuseCHFeodotracker(ctx, feedURL)
	case "abuse_ch_sslbl":
		return s.FetchAbuseCHSSLBlacklist(ctx, feedURL)

	// Blocklist.de feeds
	case "blocklist_de_all":
		return s.FetchBlocklistDeAll(ctx, feedURL)
	case "blocklist_de_ssh":
		return s.FetchBlocklistDeSSH(ctx, feedURL)
	case "blocklist_de_mail":
		return s.FetchBlocklistDeMail(ctx, feedURL)
	case "blocklist_de_apache":
		return s.FetchBlocklistDeApache(ctx, feedURL)
	case "blocklist_de_imap":
		return s.FetchBlocklistDeIMAP(ctx, feedURL)
	case "blocklist_de_bots":
		return s.FetchBlocklistDeBots(ctx, feedURL)
	case "blocklist_de_bruteforce":
		return s.FetchBlocklistDeBruteforce(ctx, feedURL)
	case "blocklist_de_strongips":
		return s.FetchBlocklistDeStrongIPs(ctx, feedURL)
	case "blocklist_de_ftp":
		return s.FetchBlocklistDeFTP(ctx, feedURL)

	// IPsum feeds
	case "ipsum_level3":
		return s.FetchIPsumLevel3(ctx, feedURL)
	case "ipsum_level4":
		return s.FetchIPsumLevel4(ctx, feedURL)
	case "ipsum_level5":
		return s.FetchIPsumLevel5(ctx, feedURL)
	case "ipsum_level6":
		return s.FetchIPsumLevel6(ctx, feedURL)
	case "ipsum_level7":
		return s.FetchIPsumLevel7(ctx, feedURL)
	case "ipsum_level8":
		return s.FetchIPsumLevel8(ctx, feedURL)

	// C2IntelFeeds
	case "c2intel_ipc2s":
		return s.FetchC2IntelIPList(ctx, feedURL)
	case "c2intel_domain_c2s":
		return s.FetchC2IntelDomainList(ctx, feedURL)
	case "c2intel_domain_c2s_url":
		return s.FetchC2IntelDomainWithURL(ctx, feedURL)
	case "c2intel_domain_c2s_url_ip":
		return s.FetchC2IntelDomainWithURLWithIP(ctx, feedURL)

	// Montysecurity C2 Tracker feeds
	case "montysecurity_brute_ratel":
		return s.FetchMontysecurityBruteRatel(ctx, feedURL)
	case "montysecurity_cobalt_strike":
		return s.FetchMontysecurityCobaltStrike(ctx, feedURL)
	case "montysecurity_sliver":
		return s.FetchMontysecuritySliver(ctx, feedURL)
	case "montysecurity_metasploit":
		return s.FetchMontysecurityMetasploit(ctx, feedURL)
	case "montysecurity_havoc":
		return s.FetchMontysecurityHavoc(ctx, feedURL)
	case "montysecurity_burpsuite":
		return s.FetchMontysecurityBurpSuite(ctx, feedURL)
	case "montysecurity_deimos":
		return s.FetchMontysecurityDeimos(ctx, feedURL)
	case "montysecurity_gophish":
		return s.FetchMontysecurityGoPhish(ctx, feedURL)
	case "montysecurity_mythic":
		return s.FetchMontysecurityMythic(ctx, feedURL)
	case "montysecurity_nimplant":
		return s.FetchMontysecurityNimPlant(ctx, feedURL)
	case "montysecurity_panda":
		return s.FetchMontysecurityPANDA(ctx, feedURL)
	case "montysecurity_xmrig":
		return s.FetchMontysecurityXMRig(ctx, feedURL)
	case "montysecurity_all":
		return s.FetchMontysecurityAll(ctx, feedURL)

	// ThreatView.io feeds
	case "threatview_ioc_tweets":
		return s.FetchThreatViewIOCTweets(ctx, feedURL)
	case "threatview_cobalt_strike":
		return s.FetchThreatViewCobaltStrike(ctx, feedURL)
	case "threatview_ip_high":
		return s.FetchThreatViewIPHigh(ctx, feedURL)
	case "threatview_domain_high":
		return s.FetchThreatViewDomainHigh(ctx, feedURL)
	case "threatview_md5":
		return s.FetchThreatViewMD5(ctx, feedURL)
	case "threatview_url_high":
		return s.FetchThreatViewURLHigh(ctx, feedURL)
	case "threatview_sha":
		return s.FetchThreatViewSHA(ctx, feedURL)

	// Other feeds
	case "emerging_threats_compromised_ip":
		return s.FetchEmergingThreatsCompromisedIP(ctx, feedURL)
	case "binarydefense_banlist":
		return s.FetchBinarydefenseBanlist(ctx, feedURL)
	case "cinsscore_badguys":
		return s.FetchCINSscoreBadguys(ctx, feedURL)
	case "greensnow_blocklist":
		return s.FetchGreenSnowBlocklist(ctx, feedURL)

	default:
		return nil, goerr.New("unsupported feed schema", goerr.V("schema", schema))
	}
}

// parseDate parses date string in various formats
func parseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Now()
	}

	// Try common date formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	// Fallback to current time
	return time.Now()
}

// parseTags parses a comma-separated tag string
func parseTags(tagStr string) []string {
	tagStr = strings.TrimSpace(tagStr)
	if tagStr == "" {
		return nil
	}

	tags := strings.Split(tagStr, ",")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			result = append(result, tag)
		}
	}

	return result
}

// mapThreatFoxType maps ThreatFox IOC type to our IOC type
func mapThreatFoxType(tfType string) model.IoCType {
	tfType = strings.ToLower(strings.TrimSpace(tfType))

	switch tfType {
	case "ip:port", "ip":
		return model.IoCTypeIPv4
	case "domain":
		return model.IoCTypeDomain
	case "url":
		return model.IoCTypeURL
	case "md5_hash":
		return model.IoCTypeMD5
	case "sha1_hash":
		return model.IoCTypeSHA1
	case "sha256_hash":
		return model.IoCTypeSHA256
	case "email":
		return model.IoCTypeEmail
	default:
		// Try to auto-detect
		return model.IoCTypeFilename // Default fallback
	}
}
