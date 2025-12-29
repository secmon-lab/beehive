# Threat Intelligence Feed Reference

This document provides a comprehensive reference for all threat intelligence feeds supported by Beehive.

## Table of Contents

- [Overview](#overview)
- [Feed Categories](#feed-categories)
- [Feed List](#feed-list)
  - [Abuse.ch Feeds](#abusech-feeds)
  - [Blocklist.de Feeds](#blocklistde-feeds)
  - [IPsum Feeds](#ipsum-feeds)
  - [C2IntelFeeds](#c2intelfeeds)
  - [Montysecurity C2 Tracker](#montysecurity-c2-tracker)
  - [ThreatView.io Feeds](#threatviewio-feeds)
  - [Other Feeds](#other-feeds)
- [Usage](#usage)
- [Feed Relationships](#feed-relationships)

## Overview

Beehive supports 54+ threat intelligence feeds from various sources. Each feed is identified by a schema name and can be fetched using the `FetchFeed` method or individual fetch methods.

All feeds have default URLs configured, so you can use them without specifying URLs explicitly. To override the default URL, pass a custom URL as the `feedURL` parameter.

## Feed Categories

Feeds are organized into the following categories:

1. **Malware & C2 Infrastructure**: Feeds tracking malware distribution and command & control servers
2. **Attack Sources**: Feeds tracking IPs engaged in various types of attacks
3. **Threat Intelligence**: Aggregated threat intelligence from multiple sources
4. **SSL/TLS Abuse**: Feeds tracking malicious SSL certificates

## Feed List

### Abuse.ch Feeds

[Abuse.ch](https://abuse.ch/) provides multiple feeds tracking malware, botnets, and malicious infrastructure.

#### URLhaus

- **Schema**: `abuse_ch_urlhaus`
- **Default URL**: `https://urlhaus.abuse.ch/downloads/csv_recent/`
- **IoC Types**: URL
- **Description**: Recent malicious URLs used for malware distribution
- **Format**: CSV (id, dateadded, url, url_status, last_online, threat, tags, urlhaus_link, reporter)

#### ThreatFox

- **Schema**: `abuse_ch_threatfox`
- **Default URL**: `https://threatfox.abuse.ch/export/csv/recent/`
- **IoC Types**: Domain, IPv4, IPv6, URL, MD5, SHA1, SHA256
- **Description**: Indicators of Compromise (IOCs) shared by the community
- **Format**: CSV (first_seen_utc, ioc_id, ioc_value, ioc_type, threat_type, fk_malware, malware_alias, malware_printable, confidence_level, reference, tags, reporter)

#### Feodotracker

- **Schema**: `abuse_ch_feodotracker_ip`
- **Default URL**: `https://feodotracker.abuse.ch/downloads/ipblocklist.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Feodo/Emotet/Dridex botnet C2 IP addresses
- **Format**: TXT (one IP per line)

#### SSL Blacklist

- **Schema**: `abuse_ch_sslbl`
- **Default URL**: `https://sslbl.abuse.ch/blacklist/sslblacklist.csv`
- **IoC Types**: SHA1
- **Description**: SSL certificates associated with malware C2 servers
- **Format**: CSV (Listingdate, SHA1, Listingreason)

### Blocklist.de Feeds

[Blocklist.de](https://www.blocklist.de/) provides IP blocklists for various types of attacks.

#### All Attacks

- **Schema**: `blocklist_de_all`
- **Default URL**: `https://lists.blocklist.de/lists/all.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: All attack sources tracked by Blocklist.de
- **Format**: TXT (one IP per line)
- **Relationship**: Superset of all other Blocklist.de feeds

#### SSH Attacks

- **Schema**: `blocklist_de_ssh`
- **Default URL**: `https://lists.blocklist.de/lists/ssh.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs attacking SSH services
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### Mail Attacks

- **Schema**: `blocklist_de_mail`
- **Default URL**: `https://lists.blocklist.de/lists/mail.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs attacking mail services (SMTP, IMAP, POP3)
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### Apache Attacks

- **Schema**: `blocklist_de_apache`
- **Default URL**: `https://lists.blocklist.de/lists/apache.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs attacking Apache web servers
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### IMAP Attacks

- **Schema**: `blocklist_de_imap`
- **Default URL**: `https://lists.blocklist.de/lists/imap.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs attacking IMAP services
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### Bots

- **Schema**: `blocklist_de_bots`
- **Default URL**: `https://lists.blocklist.de/lists/bots.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Known botnet IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### Bruteforce Login Attempts

- **Schema**: `blocklist_de_bruteforce`
- **Default URL**: `https://lists.blocklist.de/lists/bruteforcelogin.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs performing bruteforce login attempts
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### Strong IPs

- **Schema**: `blocklist_de_strongips`
- **Default URL**: `https://lists.blocklist.de/lists/strongips.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs with high attack frequency
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

#### FTP Attacks

- **Schema**: `blocklist_de_ftp`
- **Default URL**: `https://lists.blocklist.de/lists/ftp.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: IPs attacking FTP services
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `blocklist_de_all`

### IPsum Feeds

[IPsum](https://github.com/stamparm/ipsum) provides threat intelligence feeds with different threat levels.

#### Level 3

- **Schema**: `ipsum_level3`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/3.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 3 (low-medium) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Superset of levels 4-8

#### Level 4

- **Schema**: `ipsum_level4`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/4.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 4 (medium) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of level 3, superset of levels 5-8

#### Level 5

- **Schema**: `ipsum_level5`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/5.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 5 (medium-high) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of levels 3-4, superset of levels 6-8

#### Level 6

- **Schema**: `ipsum_level6`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/6.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 6 (high) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of levels 3-5, superset of levels 7-8

#### Level 7

- **Schema**: `ipsum_level7`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/7.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 7 (very high) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of levels 3-6, superset of level 8

#### Level 8

- **Schema**: `ipsum_level8`
- **Default URL**: `https://raw.githubusercontent.com/stamparm/ipsum/master/levels/8.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Threat level 8 (critical) malicious IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of levels 3-7 (most critical threats only)
- **Note**: Higher threat level feeds are subsets of lower levels

### C2IntelFeeds

[C2IntelFeeds](https://github.com/drb-ra/C2IntelFeeds) tracks Command & Control infrastructure.

#### IP C2s

- **Schema**: `c2intel_ipc2s`
- **Default URL**: `https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/IPC2s-30day.csv`
- **IoC Types**: IPv4
- **Description**: IP addresses used as C2 servers (last 30 days)
- **Format**: CSV (ip, ioc description)
- **Relationship**: Subset of `c2intel_domain_c2s_url_ip`

#### Domain C2s

- **Schema**: `c2intel_domain_c2s`
- **Default URL**: `https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2s-30day-filter-abused.csv`
- **IoC Types**: Domain
- **Description**: Domains used as C2 servers (last 30 days, filtered)
- **Format**: CSV (domain, ioc description)
- **Relationship**: Subset of `c2intel_domain_c2s_url_ip`

#### Domain C2s with URLs

- **Schema**: `c2intel_domain_c2s_url`
- **Default URL**: `https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2swithURL-30day-filter-abused.csv`
- **IoC Types**: Domain, URL
- **Description**: Domains and URLs used as C2 servers (last 30 days, filtered)
- **Format**: CSV (domain/url, ioc description)
- **Relationship**: Subset of `c2intel_domain_c2s_url_ip`

#### Comprehensive C2s

- **Schema**: `c2intel_domain_c2s_url_ip`
- **Default URL**: `https://raw.githubusercontent.com/drb-ra/C2IntelFeeds/master/feeds/domainC2swithURLwithIP-30day-filter-abused.csv`
- **IoC Types**: IPv4, Domain, URL
- **Description**: All C2 indicators (IPs, domains, URLs) from last 30 days
- **Format**: CSV (ioc, description)
- **Relationship**: Superset of `c2intel_ipc2s`, `c2intel_domain_c2s`, `c2intel_domain_c2s_url`

### Montysecurity C2 Tracker

[Montysecurity C2 Tracker](https://github.com/montysecurity/C2-Tracker) tracks C2 infrastructure for specific tools.

#### Brute Ratel C4

- **Schema**: `montysecurity_brute_ratel`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Brute%20Ratel%20C4%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Brute Ratel C4 C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Cobalt Strike

- **Schema**: `montysecurity_cobalt_strike`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Cobalt%20Strike%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Cobalt Strike C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Sliver

- **Schema**: `montysecurity_sliver`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Sliver%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Sliver C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Metasploit Framework

- **Schema**: `montysecurity_metasploit`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Metasploit%20Framework%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Metasploit Framework C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Havoc

- **Schema**: `montysecurity_havoc`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Havoc%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Havoc C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### BurpSuite

- **Schema**: `montysecurity_burpsuite`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/BurpSuite%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: BurpSuite Collaborator server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Deimos

- **Schema**: `montysecurity_deimos`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Deimos%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Deimos C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### GoPhish

- **Schema**: `montysecurity_gophish`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/GoPhish%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: GoPhish phishing server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### Mythic

- **Schema**: `montysecurity_mythic`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/Mythic%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Mythic C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### NimPlant

- **Schema**: `montysecurity_nimplant`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/NimPlant%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: NimPlant C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### PANDA

- **Schema**: `montysecurity_panda`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/PANDA%20C2%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: PANDA C2 server IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### XMRig Monero Cryptominer

- **Schema**: `montysecurity_xmrig`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/XMRig%20Monero%20Cryptominer%20IPs.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: XMRig Monero cryptominer IPs
- **Format**: TXT (one IP per line)
- **Relationship**: Subset of `montysecurity_all`

#### All Montysecurity

- **Schema**: `montysecurity_all`
- **Default URL**: `https://raw.githubusercontent.com/montysecurity/C2-Tracker/main/data/all.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: All C2 IPs tracked by Montysecurity
- **Format**: TXT (one IP per line)
- **Relationship**: Superset of all other Montysecurity feeds

### ThreatView.io Feeds

[ThreatView.io](https://threatview.io/) provides various threat intelligence feeds.

#### Experimental IOC Tweets

- **Schema**: `threatview_ioc_tweets`
- **Default URL**: `https://threatview.io/Downloads/Experimental-IOC-Tweets.txt`
- **IoC Types**: IPv4, Domain, URL, MD5, SHA256
- **Description**: IoCs extracted from Twitter (experimental)
- **Format**: TXT (one IoC per line, mixed types)

#### Cobalt Strike C2

- **Schema**: `threatview_cobalt_strike`
- **Default URL**: `https://threatview.io/Downloads/High-Confidence-CobaltStrike-C2%20-Feeds.txt`
- **IoC Types**: IPv4, Domain
- **Description**: High confidence Cobalt Strike C2 servers
- **Format**: TXT (one IoC per line)

#### IP High Confidence

- **Schema**: `threatview_ip_high`
- **Default URL**: `https://threatview.io/Downloads/IP-High-Confidence-Feed.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: High confidence malicious IPs
- **Format**: TXT (one IP per line)

#### Domain High Confidence

- **Schema**: `threatview_domain_high`
- **Default URL**: `https://threatview.io/Downloads/DOMAIN-High-Confidence-Feed.txt`
- **IoC Types**: Domain
- **Description**: High confidence malicious domains
- **Format**: TXT (one domain per line)

#### MD5

- **Schema**: `threatview_md5`
- **Default URL**: `https://threatview.io/Downloads/MD5-FEED.txt`
- **IoC Types**: MD5
- **Description**: Malicious file MD5 hashes
- **Format**: TXT (one hash per line)

#### URL High Confidence

- **Schema**: `threatview_url_high`
- **Default URL**: `https://threatview.io/Downloads/URL-High-Confidence-Feed.txt`
- **IoC Types**: URL
- **Description**: High confidence malicious URLs
- **Format**: TXT (one URL per line)

#### SHA256

- **Schema**: `threatview_sha`
- **Default URL**: `https://threatview.io/Downloads/SHA-256-FEED.txt`
- **IoC Types**: SHA256
- **Description**: Malicious file SHA256 hashes
- **Format**: TXT (one hash per line)

### Other Feeds

#### Emerging Threats Compromised IPs

- **Schema**: `emerging_threats_compromised_ip`
- **Default URL**: `https://rules.emergingthreats.net/blockrules/compromised-ips.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Known compromised hosts
- **Format**: TXT (one IP per line)

#### Binary Defense Banlist

- **Schema**: `binarydefense_banlist`
- **Default URL**: `https://www.binarydefense.com/banlist.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: Binary Defense Artillery threat intelligence banlist
- **Format**: TXT (one IP per line)

#### CINSscore Badguys

- **Schema**: `cinsscore_badguys`
- **Default URL**: `https://cinsscore.com/list/ci-badguys.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: CINS Army threat intelligence list
- **Format**: TXT (one IP per line)

#### GreenSnow Blocklist

- **Schema**: `greensnow_blocklist`
- **Default URL**: `https://blocklist.greensnow.co/greensnow.txt`
- **IoC Types**: IPv4, IPv6
- **Description**: GreenSnow malicious IP blocklist
- **Format**: TXT (one IP per line)

