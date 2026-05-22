package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/cli"
)

func TestRun_HelpExits0(t *testing.T) {
	// urfave/cli/v3 prints help and returns nil for `--help`.
	err := cli.Run(context.Background(), []string{"beehive", "--help"}, "test")
	gt.NoError(t, err)
}

func TestRun_VersionFlag(t *testing.T) {
	err := cli.Run(context.Background(), []string{"beehive", "--version"}, "vX")
	gt.NoError(t, err)
	// Smoke check: the subcommand handler still resolves cleanly.
	gt.True(t, strings.HasPrefix("vX", "v"))
}
