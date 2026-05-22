package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/cli"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	gt.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	return path
}

// TestRun_ValidatePassesWithGoodTOML uses the real "rss" provider so the
// test exercises the production wiring (no test-only registry injection).
func TestRun_ValidatePassesWithGoodTOML(t *testing.T) {
	// validate now also asserts LLM config presence — supply harmless
	// values so the test focuses on TOML / provider checks.
	t.Setenv("BEEHIVE_LLM_PROVIDER", "gemini")
	t.Setenv("BEEHIVE_LLM_MODEL", "test-model")
	t.Setenv("BEEHIVE_LLM_API_KEY", "test-key")
	t.Setenv("BEEHIVE_LLM_ARGS", "project_id=test,location=us-central1")

	path := writeConfig(t, `
[[source]]
id = "ok-1"
name = "ok"
type = "rss"
url = "https://example.com/feed"
interval = "1h"
`)
	err := cli.Run(context.Background(), []string{
		"beehive", "validate",
		"--config", path,
		"--repo-backend", "memory",
	}, "test")
	gt.NoError(t, err)
}

func TestRun_ValidateFailsWithBadTOML(t *testing.T) {
	path := writeConfig(t, `
[[source]]
id = "bad-1"
name = "bad"
type = "definitely-not-registered"
url = "https://example.com/feed"
interval = "1h"
`)
	err := cli.Run(context.Background(), []string{
		"beehive", "validate",
		"--config", path,
		"--repo-backend", "memory",
	}, "test")
	gt.Error(t, err)
}

func TestRun_ValidateRequiresFirestoreConfig(t *testing.T) {
	// Ensure the operator's shell env (via zenv / direnv / etc.) does
	// not bleed into this test — the assertion is "missing config →
	// failure", so we must guarantee the values are missing.
	t.Setenv("BEEHIVE_FIRESTORE_PROJECT_ID", "")
	t.Setenv("BEEHIVE_FIRESTORE_DATABASE", "")

	// Empty config file is fine — the failure is on missing project_id.
	path := writeConfig(t, ``)
	err := cli.Run(context.Background(), []string{
		"beehive", "validate",
		"--config", path,
		"--repo-backend", "firestore",
	}, "test")
	gt.Error(t, err)
}
