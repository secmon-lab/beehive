# APT29 Infrastructure Analysis: Cozy Bear Campaign 2024

Threat intelligence teams have identified new infrastructure associated with APT29 (Cozy Bear), a nation-state threat actor.

## Command and Control Infrastructure

The group has deployed multiple C2 servers across different regions:

### Primary Infrastructure
- **C2 Server 1**: 203.0.113.15 (hosted in Eastern Europe, active since January 2024)
- **C2 Server 2**: 203.0.113.28 (backup server, intermittent activity)
- **Domain infrastructure**: cozy-updates-cdn.com (masquerading as CDN service)
- **Secondary domain**: software-dist-mirror.org (fake software distribution)

### Phishing Infrastructure
- **Phishing portal**: https://office365-secure-login.info/auth
- **Credential harvesting**: https://accounts-verify-security.net/validate
- **Email sender**: no-reply@microsoft-security-alerts.info

## Malware Analysis

### Initial Access Payload
- **File**: "SecurityUpdate_Jan2024.exe"
- **SHA256**: `3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d`
- **MD5**: `098f6bcd4621d373cade4e832627b4f6`

### Second-Stage Loader
- **File**: "wmiprvse.dll"
- **SHA1**: `2fd4e1c67a2d28fced849ee1bb76e7391b93eb12`

### Persistence Module
- **SHA256**: `d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592`

## Legitimate Tools Abused

APT29 has been observed using legitimate system administration tools:

- PowerShell Empire framework (https://github.com/EmpireProject/Empire) - note: legitimate red team tool
- Cobalt Strike (https://www.cobaltstrike.com) - commercial penetration testing software
- Mimikatz for credential extraction (https://github.com/gentilkiwi/mimikatz) - open-source security tool

## Network Indicators

Traffic analysis reveals connections to:
- Port 443 HTTPS to C2 domains
- Port 8080 for secondary C2 channel
- DNS queries to controlled nameservers at 203.0.113.53

Note: Normal enterprise traffic to microsoft.com, github.com, and googleapis.com was also observed but is unrelated to the threat activity.

## References and Methodology

Analysis methodology follows MITRE ATT&CK framework: https://attack.mitre.org

Additional context available in:
- CISA Alert: https://www.cisa.gov/news-events/cybersecurity-advisories
- Microsoft Threat Intelligence: https://www.microsoft.com/en-us/security/business/threat-intelligence

For questions about this analysis, contact threat-intel@security-vendor-example.com

**Published by**: ThreatWatch Research
**Blog URL**: https://blog.threatwatch.example/2024/01/apt29-analysis
