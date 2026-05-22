package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/types"
	"github.com/secmon-lab/beehive/pkg/service/providers"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/urfave/cli/v3"
)

func cmdFetch(cfg *Configs) *cli.Command {
	return &cli.Command{
		Name:      "fetch",
		Usage:     "Run a one-shot fetch (optionally limited to one source-id).",
		ArgsUsage: "[source-id]",
		Flags: combineFlags(
			cfg.Repo.Flags(),
			cfg.Firestore.Flags(),
			cfg.Sources.Flags(),
			cfg.LLM.Flags(),
		),
		Action: func(ctx context.Context, c *cli.Command) error {
			var targetID types.SourceID
			if c.Args().Len() > 0 {
				targetID = types.SourceID(c.Args().First())
			}
			return runFetch(ctx, cfg, targetID)
		},
	}
}

func runFetch(ctx context.Context, cfg *Configs, target types.SourceID) error {
	if err := cfg.Firestore.Validate(cfg.Repo.Backend); err != nil {
		return err
	}
	if err := cfg.LLM.Validate(); err != nil {
		return err
	}
	repo, closer, err := buildRepo(ctx, cfg)
	if err != nil {
		return err
	}
	defer closer()

	reg, err := providers.All(nil)
	if err != nil {
		return goerr.Wrap(err, "build provider registry")
	}

	ext, err := buildExtractor(ctx, cfg)
	if err != nil {
		return goerr.Wrap(err, "build extractor")
	}

	bs := usecase.NewBootstrap()
	cat, err := bs.Ensure(ctx, cfg.Sources.Path, repo, reg)
	if err != nil {
		return goerr.Wrap(err, "bootstrap")
	}

	deps := usecase.Deps{
		Repo:      repo,
		Lock:      usecase.NewLockManager(repo, holderID()),
		Catalog:   cat,
		Registry:  reg,
		Extractor: ext,
	}

	run, err := usecase.FetchAll(ctx, deps, "cli", target)
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	if run.Failed > 0 {
		fmt.Fprintf(os.Stderr, "%s run %s — total=%d triggered=%d skipped=%d failed=%d\n",
			red("✗"), run.ID, run.Total, run.Triggered, run.Skipped, run.Failed)
	} else {
		fmt.Fprintf(os.Stdout, "%s run %s — total=%d triggered=%d skipped=%d\n",
			green("✓"), run.ID, run.Total, run.Triggered, run.Skipped)
	}
	body, _ := json.MarshalIndent(map[string]any{
		"runId":      run.ID,
		"total":      run.Total,
		"triggered":  run.Triggered,
		"skipped":    run.Skipped,
		"failed":     run.Failed,
		"status":     run.Status,
		"durationMs": run.DurationMs,
	}, "", "  ")
	fmt.Fprintln(os.Stdout, string(body))
	if run.Failed > 0 {
		return fmt.Errorf("fetch finished with %d failures", run.Failed)
	}
	return nil
}
