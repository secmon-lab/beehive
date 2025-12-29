package types

import "github.com/m-mizutani/goerr/v2"

// FeedSchema represents a threat intelligence feed schema
// A schema defines:
// - Data format/structure for parsing
// - Parser implementation to use
// - Default fetch URL for the feed
type FeedSchema string

const (
	// Abuse.ch feeds
	FeedSchemaAbuseCHURLhaus      FeedSchema = "abuse_ch_urlhaus"
	FeedSchemaAbuseCHThreatFox    FeedSchema = "abuse_ch_threatfox"
	FeedSchemaAbuseCHFeodotracker FeedSchema = "abuse_ch_feodotracker_ip"
	FeedSchemaAbuseCHSSLBlacklist FeedSchema = "abuse_ch_sslbl"

	// Blocklist.de feeds
	FeedSchemaBlocklistDeAll        FeedSchema = "blocklist_de_all"
	FeedSchemaBlocklistDeSSH        FeedSchema = "blocklist_de_ssh"
	FeedSchemaBlocklistDeMail       FeedSchema = "blocklist_de_mail"
	FeedSchemaBlocklistDeApache     FeedSchema = "blocklist_de_apache"
	FeedSchemaBlocklistDeIMAP       FeedSchema = "blocklist_de_imap"
	FeedSchemaBlocklistDeBots       FeedSchema = "blocklist_de_bots"
	FeedSchemaBlocklistDeBruteforce FeedSchema = "blocklist_de_bruteforce"
	FeedSchemaBlocklistDeStrongIPs  FeedSchema = "blocklist_de_strongips"
	FeedSchemaBlocklistDeFTP        FeedSchema = "blocklist_de_ftp"

	// IPsum feeds
	FeedSchemaIPsumLevel3 FeedSchema = "ipsum_level3"
	FeedSchemaIPsumLevel4 FeedSchema = "ipsum_level4"
	FeedSchemaIPsumLevel5 FeedSchema = "ipsum_level5"
	FeedSchemaIPsumLevel6 FeedSchema = "ipsum_level6"
	FeedSchemaIPsumLevel7 FeedSchema = "ipsum_level7"
	FeedSchemaIPsumLevel8 FeedSchema = "ipsum_level8"

	// C2IntelFeeds
	FeedSchemaC2IntelIPList              FeedSchema = "c2intel_ipc2s"
	FeedSchemaC2IntelDomainList          FeedSchema = "c2intel_domain_c2s"
	FeedSchemaC2IntelDomainWithURL       FeedSchema = "c2intel_domain_c2s_url"
	FeedSchemaC2IntelDomainWithURLWithIP FeedSchema = "c2intel_domain_c2s_url_ip"

	// Montysecurity C2 Tracker feeds
	FeedSchemaMontysecurityBruteRatel   FeedSchema = "montysecurity_brute_ratel"
	FeedSchemaMontysecurityCobaltStrike FeedSchema = "montysecurity_cobalt_strike"
	FeedSchemaMontysecuritySliver       FeedSchema = "montysecurity_sliver"
	FeedSchemaMontysecurityMetasploit   FeedSchema = "montysecurity_metasploit"
	FeedSchemaMontysecurityHavoc        FeedSchema = "montysecurity_havoc"
	FeedSchemaMontysecurityBurpSuite    FeedSchema = "montysecurity_burpsuite"
	FeedSchemaMontysecurityDeimos       FeedSchema = "montysecurity_deimos"
	FeedSchemaMontysecurityGoPhish      FeedSchema = "montysecurity_gophish"
	FeedSchemaMontysecurityMythic       FeedSchema = "montysecurity_mythic"
	FeedSchemaMontysecurityNimPlant     FeedSchema = "montysecurity_nimplant"
	FeedSchemaMontysecurityPANDA        FeedSchema = "montysecurity_panda"
	FeedSchemaMontysecurityXMRig        FeedSchema = "montysecurity_xmrig"
	FeedSchemaMontysecurityAll          FeedSchema = "montysecurity_all"

	// ThreatView.io feeds
	FeedSchemaThreatViewIOCTweets    FeedSchema = "threatview_ioc_tweets"
	FeedSchemaThreatViewCobaltStrike FeedSchema = "threatview_cobalt_strike"
	FeedSchemaThreatViewIPHigh       FeedSchema = "threatview_ip_high"
	FeedSchemaThreatViewDomainHigh   FeedSchema = "threatview_domain_high"
	FeedSchemaThreatViewMD5          FeedSchema = "threatview_md5"
	FeedSchemaThreatViewURLHigh      FeedSchema = "threatview_url_high"
	FeedSchemaThreatViewSHA          FeedSchema = "threatview_sha"

	// Other feeds
	FeedSchemaEmergingThreatsCompromisedIP FeedSchema = "emerging_threats_compromised_ip"
	FeedSchemaBinarydefenseBanlist         FeedSchema = "binarydefense_banlist"
	FeedSchemaCINSscoreBadguys             FeedSchema = "cinsscore_badguys"
	FeedSchemaGreenSnowBlocklist           FeedSchema = "greensnow_blocklist"
)

// AllFeedSchemas returns all valid feed schemas
func AllFeedSchemas() []FeedSchema {
	return []FeedSchema{
		// Abuse.ch
		FeedSchemaAbuseCHURLhaus,
		FeedSchemaAbuseCHThreatFox,
		FeedSchemaAbuseCHFeodotracker,
		FeedSchemaAbuseCHSSLBlacklist,

		// Blocklist.de
		FeedSchemaBlocklistDeAll,
		FeedSchemaBlocklistDeSSH,
		FeedSchemaBlocklistDeMail,
		FeedSchemaBlocklistDeApache,
		FeedSchemaBlocklistDeIMAP,
		FeedSchemaBlocklistDeBots,
		FeedSchemaBlocklistDeBruteforce,
		FeedSchemaBlocklistDeStrongIPs,
		FeedSchemaBlocklistDeFTP,

		// IPsum
		FeedSchemaIPsumLevel3,
		FeedSchemaIPsumLevel4,
		FeedSchemaIPsumLevel5,
		FeedSchemaIPsumLevel6,
		FeedSchemaIPsumLevel7,
		FeedSchemaIPsumLevel8,

		// C2IntelFeeds
		FeedSchemaC2IntelIPList,
		FeedSchemaC2IntelDomainList,
		FeedSchemaC2IntelDomainWithURL,
		FeedSchemaC2IntelDomainWithURLWithIP,

		// Montysecurity
		FeedSchemaMontysecurityBruteRatel,
		FeedSchemaMontysecurityCobaltStrike,
		FeedSchemaMontysecuritySliver,
		FeedSchemaMontysecurityMetasploit,
		FeedSchemaMontysecurityHavoc,
		FeedSchemaMontysecurityBurpSuite,
		FeedSchemaMontysecurityDeimos,
		FeedSchemaMontysecurityGoPhish,
		FeedSchemaMontysecurityMythic,
		FeedSchemaMontysecurityNimPlant,
		FeedSchemaMontysecurityPANDA,
		FeedSchemaMontysecurityXMRig,
		FeedSchemaMontysecurityAll,

		// ThreatView.io
		FeedSchemaThreatViewIOCTweets,
		FeedSchemaThreatViewCobaltStrike,
		FeedSchemaThreatViewIPHigh,
		FeedSchemaThreatViewDomainHigh,
		FeedSchemaThreatViewMD5,
		FeedSchemaThreatViewURLHigh,
		FeedSchemaThreatViewSHA,

		// Others
		FeedSchemaEmergingThreatsCompromisedIP,
		FeedSchemaBinarydefenseBanlist,
		FeedSchemaCINSscoreBadguys,
		FeedSchemaGreenSnowBlocklist,
	}
}

// NewFeedSchema creates a validated feed schema
func NewFeedSchema(s string) (FeedSchema, error) {
	fs := FeedSchema(s)
	for _, valid := range AllFeedSchemas() {
		if fs == valid {
			return fs, nil
		}
	}
	return "", goerr.New("invalid feed schema",
		goerr.V("schema", s),
		goerr.V("valid_schemas", AllFeedSchemas()))
}

// DefaultURL returns the default fetch URL for this schema
// Returns empty string if feedURL should be provided explicitly
func (fs FeedSchema) DefaultURL() string {
	// Default URLs are defined in the feed service constants
	// This method is kept for backward compatibility but returns empty
	// The actual default URL logic is in pkg/service/feed/*.go files
	return ""
}

func (fs FeedSchema) String() string {
	return string(fs)
}
