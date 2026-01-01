# Supply Chain Attack on Popular GitHub Action

A critical supply chain vulnerability has been discovered in a widely-used GitHub Action that could allow attackers to execute arbitrary code in CI/CD pipelines.

## Overview

Security researchers at Wiz have identified a serious vulnerability in the tj-actions/changed-files GitHub Action, which has over 50,000 dependent repositories. The vulnerability, tracked as CVE-2025-30066, allows malicious actors to inject commands through crafted filenames.

As reported by Wiz security researchers at https://www.wiz.io/blog/github-action-tj-actions-changed-files-supply-chain-attack-cve-2025-30066, this attack vector exploits improper input sanitization in the action's file processing logic.

## Technical Details

The vulnerability stems from the XZ Utils backdoor incident, which was extensively analyzed by Akamai. For background on similar supply chain attacks, see Akamai's blog post about the XZ backdoor at https://www.akamai.com/blog/security-research/critical-linux-backdoor-xz-utils-discovered-what-to-know.

Microsoft has also published guidance on mitigating prompt injection attacks, which share some similarities with this vulnerability. The vulnerability CVE-2025-32711 is documented at https://msrc.microsoft.com/update-guide/vulnerability/CVE-2025-32711.

## Attack Methodology

Attackers can exploit this vulnerability by creating a malicious repository with specially crafted filenames containing shell metacharacters. When the vulnerable GitHub Action processes these files, it executes the injected commands.

The attack infrastructure used in observed exploits includes:
- Command and control server at http://malicious-c2-server.example.com/api/exfil
- Malware payload hosted at https://evil-cdn.attackdomain.ru/payload.sh
- Stolen credentials exfiltrated to https://data-collector.badactor.xyz/collect

## Impact

This vulnerability affects thousands of CI/CD pipelines across GitHub. For more details on the broader implications, see Unit 42's analysis at https://unit42.paloaltonetworks.com/supply-chain-security-best-practices.

## Indicators of Compromise

The following indicators have been observed in active exploitation:

- C2 server domain: malicious-c2-server.example.com
- Data exfiltration endpoint: http://malicious-c2-server.example.com/api/exfil
- Malware dropper: https://evil-cdn.attackdomain.ru/payload.sh
- Malware distribution domain: evil-cdn.attackdomain.ru
- Data collection endpoint: https://data-collector.badactor.xyz/collect
- Data collector domain: data-collector.badactor.xyz
- SHA256 hash of malicious payload: a1b2c3d4e5f6789012345678901234567890123456789012345678901234

## Remediation

Organizations should immediately:
1. Update to tj-actions/changed-files version 42.1.0 or later
2. Review CI/CD logs for suspicious activity
3. Implement additional input validation as described in Microsoft's security guidance at https://microsoft.com/security/blog/mitigating-github-actions-risks

## References

- Wiz Security Research: https://www.wiz.io/blog/github-action-supply-chain-attack
- GitHub Advisory: https://github.com/advisories/GHSA-xxxx-yyyy-zzzz
- NIST CVE Database: https://nvd.nist.gov/vuln/detail/CVE-2025-30066
- CrowdStrike Threat Intelligence: https://www.crowdstrike.com/blog/supply-chain-attack-analysis
