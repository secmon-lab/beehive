package extractor_test

import (
	"context"
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/domain/extractor"
)

//go:embed testdata/malware_campaign.md
var malwareCampaignArticle string

//go:embed testdata/certificate_validation.md
var certificateValidationArticle string

//go:embed testdata/apt_group_analysis.md
var aptGroupAnalysisArticle string

//go:embed testdata/reference_urls_article.md
var referenceURLsArticle string

func TestExtractor_RealLLM(t *testing.T) {
	projectID := os.Getenv("TEST_GEMINI_PROJECT")
	location := os.Getenv("TEST_GEMINI_LOCATION")

	if projectID == "" || location == "" {
		t.Skip("TEST_GEMINI_PROJECT and TEST_GEMINI_LOCATION environment variables not set")
	}

	ctx := context.Background()
	llmClient, err := gemini.New(ctx, projectID, location,
		gemini.WithModel("gemini-2.0-flash-exp"),
		gemini.WithThinkingBudget(0),
	)
	gt.NoError(t, err)

	ext := extractor.New(llmClient)

	t.Run("malware campaign with clear IoCs", func(t *testing.T) {
		extracted, err := ext.ExtractFromArticle(ctx, "Advanced Malware Campaign", malwareCampaignArticle)
		gt.NoError(t, err)

		// Expected malicious IoCs that MUST be extracted
		// Note: LLM may extract URLs or domains - both are acceptable for phishing infrastructure
		expectedIoCs := map[string]string{
			"198.51.100.42":                                                    "ipv4",   // Primary C2 IP
			"evil-finance-server.xyz":                                          "domain", // C2 domain
			"http://198.51.100.42/downloads/update.exe":                        "url",    // Malware download URL
			"support@secure-banking-verify.net":                                "email",  // Attacker email
			"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": "sha256", // Trojan hash
			"5d41402abc4b2a76b9719d911017c592":                                 "md5",    // Credential stealer
			"aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d":                         "sha1",   // Backdoor hash
		}

		// Additional acceptable variations (URL vs domain extraction)
		acceptableVariations := map[string][]string{
			"secure-banking-verify.net": {"url", "domain"}, // May be extracted as URL or domain
		}

		// Forbidden indicators that must NOT be extracted
		forbiddenIndicators := []string{
			"nvd.nist.gov",
			"datatracker.ietf.org",
			"github.com/iagox86/dnscat2",
			"www.microsoft.com",
			"example-security-blog.com",
			"team@example-security-blog.com",
		}

		// Build map of extracted IoCs
		extractedMap := make(map[string]string)
		for _, ioc := range extracted {
			extractedMap[ioc.Value] = ioc.Type
			t.Logf("Extracted: type=%s value=%s description=%s",
				ioc.Type, ioc.Value, ioc.Description)
		}

		// Verify ALL expected IoCs were extracted
		for value, expectedType := range expectedIoCs {
			extractedType, found := extractedMap[value]
			if !found {
				t.Errorf("Expected IoC not extracted: value=%s type=%s", value, expectedType)
			} else if extractedType != expectedType {
				t.Errorf("IoC extracted with wrong type: value=%s expected_type=%s actual_type=%s",
					value, expectedType, extractedType)
			}
		}

		// Check acceptable variations (values that may be extracted in different forms)
		for value, acceptableTypes := range acceptableVariations {
			extractedType, found := extractedMap[value]
			if found {
				// Verify the type is one of the acceptable types
				typeOK := false
				for _, acceptableType := range acceptableTypes {
					if extractedType == acceptableType {
						typeOK = true
						break
					}
				}
				if !typeOK {
					t.Errorf("Variation extracted with unexpected type: value=%s acceptable_types=%v actual_type=%s",
						value, acceptableTypes, extractedType)
				}
			}
		}

		// Verify NO forbidden indicators were extracted
		for _, forbidden := range forbiddenIndicators {
			for extractedValue := range extractedMap {
				if strings.Contains(extractedValue, forbidden) {
					t.Errorf("Forbidden indicator extracted: %s (contains %s)", extractedValue, forbidden)
				}
			}
		}

		t.Logf("Extracted %d IoCs, expected %d", len(extractedMap), len(expectedIoCs))
	})

	t.Run("certificate validation with no real threats", func(t *testing.T) {
		extracted, err := ext.ExtractFromArticle(ctx, "Certificate Validation Updates", certificateValidationArticle)
		gt.NoError(t, err)

		// This article has NO malicious IoCs - it's purely documentation
		// Expected: 0 IoCs extracted (or at most empty result)
		expectedIoCs := map[string]string{
			// No expected IoCs - this is a documentation article
		}

		// All these must NOT be extracted
		forbiddenIndicators := []string{
			"cabforum.org",
			"datatracker.ietf.org",
			"blog.chromium.org",
			"letsencrypt.org",
			"pki.goog",
			"plus.google.com",
			"example.com",
			"example.net",
			"validation.letsencrypt.org",
			"8.8.8.8",
			"1.1.1.1",
			"127.0.0.1",
		}

		// Build map of extracted IoCs
		extractedMap := make(map[string]string)
		for _, ioc := range extracted {
			extractedMap[ioc.Value] = ioc.Type
			t.Logf("Extracted: type=%s value=%s description=%s",
				ioc.Type, ioc.Value, ioc.Description)
		}

		// Verify expected IoCs (should be none or very few)
		for value, expectedType := range expectedIoCs {
			extractedType, found := extractedMap[value]
			if !found {
				t.Errorf("Expected IoC not extracted: value=%s type=%s", value, expectedType)
			} else if extractedType != expectedType {
				t.Errorf("IoC extracted with wrong type: value=%s expected_type=%s actual_type=%s",
					value, expectedType, extractedType)
			}
		}

		// Verify NO forbidden indicators were extracted
		for _, forbidden := range forbiddenIndicators {
			for extractedValue := range extractedMap {
				if strings.Contains(extractedValue, forbidden) {
					t.Errorf("Forbidden documentation URL extracted: %s (contains %s)", extractedValue, forbidden)
				}
			}
		}

		// Documentation article should yield 0 IoCs
		gt.N(t, len(extractedMap)).Equal(0).Describe("documentation article should extract zero IoCs")
	})

	t.Run("APT group analysis with mixed content", func(t *testing.T) {
		extracted, err := ext.ExtractFromArticle(ctx, "APT29 Infrastructure Analysis", aptGroupAnalysisArticle)
		gt.NoError(t, err)

		// Expected malicious IoCs that MUST be extracted
		expectedIoCs := map[string]string{
			// C2 Infrastructure
			"203.0.113.15":             "ipv4",
			"203.0.113.28":             "ipv4",
			"cozy-updates-cdn.com":     "domain",
			"software-dist-mirror.org": "domain",

			// Phishing Infrastructure
			"https://office365-secure-login.info/auth":      "url",
			"https://accounts-verify-security.net/validate": "url",
			"no-reply@microsoft-security-alerts.info":       "email",

			// Malware Hashes
			"3e23e8160039594a33894f6564e1b1348bbd7a0088d42c4acb73eeaed59c009d": "sha256",
			"098f6bcd4621d373cade4e832627b4f6":                                 "md5",
			"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12":                         "sha1",
			"d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592": "sha256",

			// Network indicators
			"203.0.113.53": "ipv4",
		}

		// Forbidden indicators that must NOT be extracted
		forbiddenIndicators := []string{
			"github.com/EmpireProject",
			"github.com/gentilkiwi",
			"cobaltstrike.com",
			"attack.mitre.org",
			"cisa.gov",
			"microsoft.com/en-us/security",
			"blog.threatwatch.example",
			"threat-intel@security-vendor-example.com",
		}

		// Build map of extracted IoCs
		extractedMap := make(map[string]string)
		for _, ioc := range extracted {
			extractedMap[ioc.Value] = ioc.Type
			t.Logf("Extracted: type=%s value=%s description=%s",
				ioc.Type, ioc.Value, ioc.Description)
		}

		// Verify ALL expected IoCs were extracted
		for value, expectedType := range expectedIoCs {
			extractedType, found := extractedMap[value]
			if !found {
				t.Errorf("Expected IoC not extracted: value=%s type=%s", value, expectedType)
			} else if extractedType != expectedType {
				t.Errorf("IoC extracted with wrong type: value=%s expected_type=%s actual_type=%s",
					value, expectedType, extractedType)
			}
		}

		// Verify NO forbidden indicators were extracted
		for _, forbidden := range forbiddenIndicators {
			for extractedValue := range extractedMap {
				if strings.Contains(extractedValue, forbidden) {
					t.Errorf("Forbidden legitimate tool/reference extracted: %s (contains %s)",
						extractedValue, forbidden)
				}
			}
		}

		t.Logf("Extracted %d IoCs, expected %d", len(extractedMap), len(expectedIoCs))
	})

	t.Run("reference URLs should not be extracted as IoCs", func(t *testing.T) {
		extracted, err := ext.ExtractFromArticle(ctx, "Supply Chain Attack on Popular GitHub Action", referenceURLsArticle)
		gt.NoError(t, err)

		// Expected malicious IoCs that MUST be extracted (actual attack infrastructure)
		expectedIoCs := map[string]string{
			// C2 Infrastructure
			"malicious-c2-server.example.com":                   "domain",
			"http://malicious-c2-server.example.com/api/exfil": "url",
			"evil-cdn.attackdomain.ru":                          "domain",
			"https://evil-cdn.attackdomain.ru/payload.sh":       "url",
			"data-collector.badactor.xyz":                       "domain",
			"https://data-collector.badactor.xyz/collect":       "url",

			// Malware Hash
			"a1b2c3d4e5f6789012345678901234567890123456789012345678901234": "sha256",
		}

		// Forbidden indicators (reference URLs) that must NOT be extracted
		forbiddenIndicators := []string{
			// Security vendor blogs
			"wiz.io/blog",
			"akamai.com/blog",
			"unit42.paloaltonetworks.com",
			"crowdstrike.com/blog",

			// Official advisories
			"msrc.microsoft.com",
			"microsoft.com/security",

			// GitHub/NIST references
			"github.com/advisories",
			"nvd.nist.gov",
		}

		// Build map of extracted IoCs
		extractedMap := make(map[string]string)
		for _, ioc := range extracted {
			extractedMap[ioc.Value] = ioc.Type
			t.Logf("Extracted: type=%s value=%s description=%s",
				ioc.Type, ioc.Value, ioc.Description)
		}

		// Verify ALL expected IoCs were extracted
		for value, expectedType := range expectedIoCs {
			extractedType, found := extractedMap[value]
			if !found {
				t.Errorf("Expected malicious IoC not extracted: value=%s type=%s", value, expectedType)
			} else if extractedType != expectedType {
				t.Errorf("IoC extracted with wrong type: value=%s expected_type=%s actual_type=%s",
					value, expectedType, extractedType)
			}
		}

		// Verify NO forbidden reference URLs were extracted
		for _, forbidden := range forbiddenIndicators {
			for extractedValue := range extractedMap {
				if strings.Contains(extractedValue, forbidden) {
					t.Errorf("Forbidden reference URL extracted as IoC: %s (contains %s) - this is a security vendor/advisory, not attack infrastructure",
						extractedValue, forbidden)
				}
			}
		}

		// Additional validation: ensure we're not extracting CVE identifiers or CVE database URLs
		for extractedValue := range extractedMap {
			if strings.Contains(strings.ToUpper(extractedValue), "CVE-") {
				t.Errorf("CVE-related content incorrectly extracted: %s - CVE identifiers and CVE database URLs should NOT be extracted", extractedValue)
			}
		}

		t.Logf("Extracted %d IoCs, expected %d malicious indicators", len(extractedMap), len(expectedIoCs))
	})
}
