package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-mizutani/goerr/v2"
	bhhttp "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/domain/interfaces"
	"github.com/secmon-lab/beehive/pkg/repository/firestore"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/service/extractor"
	"github.com/secmon-lab/beehive/pkg/service/providers"
	"github.com/secmon-lab/beehive/pkg/usecase"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdServe(cfg *Configs) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Run the beehive HTTP server (no internal scheduler).",
		Flags: combineFlags(
			cfg.HTTP.Flags(),
			cfg.Repo.Flags(),
			cfg.Firestore.Flags(),
			cfg.Sources.Flags(),
			cfg.LLM.Flags(),
		),
		Action: func(ctx context.Context, _ *cli.Command) error {
			return runServe(ctx, cfg)
		},
	}
}

func runServe(ctx context.Context, cfg *Configs) error {
	logger := logging.From(ctx)

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

	server := bhhttp.New(bhhttp.WithDeps(deps), bhhttp.WithCatalog(cat))

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           server.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.LogAttrs(ctx, slog.LevelInfo, "beehive serve started",
			slog.String("addr", cfg.HTTP.Addr),
			slog.String("repo_backend", cfg.Repo.Backend),
			slog.String("firestore_project_id", cfg.Firestore.ProjectID),
			slog.String("firestore_database", cfg.Firestore.Database),
			slog.String("sources_path", cfg.Sources.Path),
			slog.Int("sources", len(cat.Sources)),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- goerr.Wrap(err, "ListenAndServe")
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		logger.LogAttrs(context.Background(), slog.LevelInfo, "shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return goerr.Wrap(err, "Shutdown")
	}
	logger.LogAttrs(context.Background(), slog.LevelInfo, "beehive serve stopped")

	return nil
}

// buildRepo returns the configured Repository and a closer the caller
// must invoke before the process exits.
func buildRepo(ctx context.Context, cfg *Configs) (interfaces.Repository, func(), error) {
	switch cfg.Repo.Backend {
	case "memory":
		return memory.New(), func() {}, nil
	case "firestore", "":
		fs, err := firestore.New(ctx, cfg.Firestore.ProjectID, cfg.Firestore.Database)
		if err != nil {
			return nil, nil, err
		}
		return fs, func() {
			if err := fs.Close(); err != nil {
				errutil.Handle(ctx, goerr.Wrap(err, "close firestore client"))
			}
		}, nil
	default:
		return nil, nil, fmt.Errorf("unknown BEEHIVE_REPO_BACKEND: %q", cfg.Repo.Backend)
	}
}

// buildExtractor constructs the LLM-backed Extractor from the LLM
// config. Called from both `serve` and `fetch` — kept here so all CLI
// commands share one wiring path.
func buildExtractor(ctx context.Context, cfg *Configs) (interfaces.Extractor, error) {
	args, err := cfg.LLM.ArgsMap()
	if err != nil {
		return nil, err
	}
	llm, err := extractor.NewLLMClient(ctx, extractor.LLMConfig{
		Provider: cfg.LLM.Provider,
		Model:    cfg.LLM.Model,
		APIKey:   cfg.LLM.APIKey,
		Args:     args,
	})
	if err != nil {
		return nil, err
	}
	return extractor.New(llm), nil
}

// holderID derives a unique identifier for this process. Used as the
// lock holder so concurrent Cloud Run instances are distinguishable.
func holderID() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s/%d", host, os.Getpid())
}

func combineFlags(groups ...[]cli.Flag) []cli.Flag {
	out := []cli.Flag{}
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}
