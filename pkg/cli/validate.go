package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/secmon-lab/beehive/pkg/service/providers"
	"github.com/secmon-lab/beehive/pkg/service/source_catalog"
	"github.com/urfave/cli/v3"
)

func cmdValidate(cfg *Configs) *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "Validate beehive config (TOML sources, provider registry, env vars).",
		Flags: combineFlags(
			cfg.Sources.Flags(),
			cfg.LLM.Flags(),
			cfg.Repo.Flags(),
			cfg.Firestore.Flags(),
		),
		Action: func(ctx context.Context, _ *cli.Command) error {
			return runValidate(ctx, cfg)
		},
	}
}

func runValidate(_ context.Context, cfg *Configs) error {
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	cat, err := source_catalog.Load(cfg.Sources.Path)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("✗ load failed:"), err)
		return err
	}

	reg, err := providers.All(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("✗ provider registry:"), err)
		return err
	}

	problems := source_catalog.Validate(cat, reg)

	// Backend-specific config (Firestore project_id / database).
	var backendProblems []string
	if err := cfg.Firestore.Validate(cfg.Repo.Backend); err != nil {
		backendProblems = append(backendProblems, err.Error())
	}
	if err := cfg.LLM.Validate(); err != nil {
		backendProblems = append(backendProblems, err.Error())
	}

	if len(problems) == 0 && len(backendProblems) == 0 {
		_, _ = fmt.Fprintf(os.Stdout, "%s %d sources in %s\n",
			green("✓"), len(cat.Sources), cfg.Sources.Path)
		_, _ = fmt.Fprintf(os.Stdout, "%s all providers registered\n", green("✓"))
		if cfg.Repo.Backend == "firestore" {
			_, _ = fmt.Fprintf(os.Stdout, "%s firestore project_id=%s database=%s\n",
				green("✓"), cfg.Firestore.ProjectID, cfg.Firestore.Database)
		}
		_, _ = fmt.Fprintln(os.Stdout, green("OK"))
		return nil
	}

	for _, p := range problems {
		_, _ = fmt.Fprintf(os.Stderr, "%s %s\n", red("✗"), p.Error())
	}
	for _, p := range backendProblems {
		_, _ = fmt.Fprintf(os.Stderr, "%s %s\n", red("✗"), p)
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s %d problem(s) found\n", yellow(""), len(problems)+len(backendProblems))
	return fmt.Errorf("validation failed")
}
