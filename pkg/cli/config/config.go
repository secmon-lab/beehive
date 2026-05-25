// Package config translates env vars / CLI flags into typed values that
// the rest of beehive consumes. Each Config struct describes the inputs
// of one subsystem so that wiring stays explicit at main.
package config

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Logger collects the user-facing logger settings. The flag surface
// mirrors secmon-lab/hecatoncheires so CLI users get clog console output
// with goerr-aware attributes by default; Cloud Run deployments flip
// BEEHIVE_LOG_FORMAT=json to fall back to structured logs.
type Logger struct {
	Level      string // "debug"|"info"|"warn"|"error"
	FormatName string // "console"|"json"
	Output     string // "stdout"|"stderr"|"-"|<path>
	Quiet      bool
	Stacktrace bool

	closer func()
}

func (c *Logger) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Category:    "logging",
			Aliases:     []string{"l"},
			Sources:     cli.EnvVars("BEEHIVE_LOG_LEVEL"),
			Usage:       "Set log level [debug|info|warn|error]",
			Value:       "info",
			Destination: &c.Level,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Category:    "logging",
			Aliases:     []string{"f"},
			Sources:     cli.EnvVars("BEEHIVE_LOG_FORMAT"),
			Usage:       "Set log format [auto|console|json] (auto = console on TTY, json otherwise)",
			Value:       "auto",
			Destination: &c.FormatName,
		},
		&cli.StringFlag{
			Name:        "log-output",
			Category:    "logging",
			Aliases:     []string{"o"},
			Sources:     cli.EnvVars("BEEHIVE_LOG_OUTPUT"),
			Usage:       "Set log output ('-', 'stdout', 'stderr', or a file path)",
			Value:       "stderr",
			Destination: &c.Output,
		},
		&cli.BoolFlag{
			Name:        "log-quiet",
			Category:    "logging",
			Aliases:     []string{"q"},
			Usage:       "Quiet mode (drop every log record)",
			Sources:     cli.EnvVars("BEEHIVE_LOG_QUIET"),
			Destination: &c.Quiet,
		},
		&cli.BoolFlag{
			Name:        "log-stacktrace",
			Category:    "logging",
			Aliases:     []string{"s"},
			Usage:       "Show goerr stacktraces (console format only)",
			Sources:     cli.EnvVars("BEEHIVE_LOG_STACKTRACE"),
			Destination: &c.Stacktrace,
			Value:       true,
		},
	}
}

// LogValue lets the logger config show up as a structured attribute
// when echoed back through slog.
func (c Logger) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("level", c.Level),
		slog.String("format", c.FormatName),
		slog.String("output", c.Output),
	)
}

// Configure resolves the flag values into a *slog.Logger, installs it
// as the package default, and returns a closer the caller must invoke
// at shutdown (typically deferred). The closer is always safe to call,
// even when Configure returned an error.
func (c *Logger) Configure() (func(), error) {
	c.Close()
	c.closer = func() {}
	if c.Quiet {
		logging.Quiet()
		return c.closer, nil
	}

	format, err := parseFormat(c.FormatName)
	if err != nil {
		return c.closer, err
	}

	level, err := parseLevel(c.Level)
	if err != nil {
		return c.closer, err
	}

	output, closer, err := openOutput(c.Output)
	if err != nil {
		return c.closer, err
	}
	c.closer = closer

	logger := logging.New(output, level, format, c.Stacktrace)
	logging.SetDefault(logger)
	return c.closer, nil
}

// Close releases any file handle opened by Configure. Safe to call
// multiple times.
func (c *Logger) Close() {
	if c.closer != nil {
		c.closer()
		c.closer = nil
	}
}

func parseFormat(s string) (logging.Format, error) {
	switch strings.ToLower(s) {
	case "", "auto":
		return logging.FormatAuto, nil
	case "console", "text":
		return logging.FormatConsole, nil
	case "json":
		return logging.FormatJSON, nil
	default:
		return 0, goerr.New("invalid log format (want auto|console|json)",
			goerr.V("format", s),
			goerr.T(errutil.TagInvalidInput))
	}
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, goerr.New("invalid log level (want debug|info|warn|error)",
			goerr.V("level", s),
			goerr.T(errutil.TagInvalidInput))
	}
}

func openOutput(spec string) (io.Writer, func(), error) {
	noop := func() {}
	switch spec {
	case "", "stderr":
		return os.Stderr, noop, nil
	case "stdout", "-":
		return os.Stdout, noop, nil
	default:
		f, err := os.OpenFile(filepath.Clean(spec), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return nil, noop, goerr.Wrap(err, "open log file",
				goerr.V("path", spec))
		}
		closer := func() {
			if err := f.Close(); err != nil {
				errutil.Handle(context.Background(),
					goerr.Wrap(err, "close log file", goerr.V("path", spec)))
			}
		}
		return f, closer, nil
	}
}

// HTTP holds listener-related settings.
type HTTP struct {
	Addr string
}

func (c *HTTP) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "http-addr",
			Sources:     cli.EnvVars("BEEHIVE_HTTP_ADDR"),
			Value:       ":8080",
			Destination: &c.Addr,
		},
	}
}

// Sources points at the single TOML file that defines every Source.
// Multi-file directory layout was considered and rejected — see
// CLAUDE.md §0 (no abstraction without a concrete need).
type Sources struct {
	Path string
}

func (c *Sources) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Sources:     cli.EnvVars("BEEHIVE_CONFIG"),
			Value:       "./config.toml",
			Destination: &c.Path,
		},
	}
}

// Firestore captures the project / database selectors. Required when
// BEEHIVE_REPO_BACKEND=firestore.
type Firestore struct {
	ProjectID string
	Database  string
}

func (c *Firestore) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project-id",
			Sources:     cli.EnvVars("BEEHIVE_FIRESTORE_PROJECT_ID"),
			Destination: &c.ProjectID,
		},
		&cli.StringFlag{
			Name:        "firestore-database",
			Sources:     cli.EnvVars("BEEHIVE_FIRESTORE_DATABASE"),
			Destination: &c.Database,
		},
	}
}

func (c *Firestore) Validate(backend string) error {
	if backend != "firestore" {
		return nil
	}
	if c.ProjectID == "" {
		return goerr.New("BEEHIVE_FIRESTORE_PROJECT_ID is required when backend=firestore",
			goerr.T(errutil.TagInvalidInput))
	}
	if c.Database == "" {
		return goerr.New("BEEHIVE_FIRESTORE_DATABASE is required (default fallback disabled)",
			goerr.T(errutil.TagInvalidInput))
	}
	return nil
}

// Repo selects between firestore and memory backends.
type Repo struct {
	Backend string
}

func (c *Repo) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "repo-backend",
			Sources:     cli.EnvVars("BEEHIVE_REPO_BACKEND"),
			Value:       "firestore",
			Destination: &c.Backend,
		},
	}
}

// LLM gathers gollem configuration. ARGS is intentionally kept as a flat
// key=value list — provider-specific parsing happens inside the gollem
// factory in pkg/service/extractor.
type LLM struct {
	Provider string
	Model    string
	APIKey   string
	Args     string
}

func (c *LLM) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "llm-provider",
			Sources:     cli.EnvVars("BEEHIVE_LLM_PROVIDER"),
			Value:       "gemini",
			Destination: &c.Provider,
		},
		&cli.StringFlag{
			Name:        "llm-model",
			Sources:     cli.EnvVars("BEEHIVE_LLM_MODEL"),
			Destination: &c.Model,
		},
		&cli.StringFlag{
			Name:        "llm-api-key",
			Sources:     cli.EnvVars("BEEHIVE_LLM_API_KEY"),
			Destination: &c.APIKey,
		},
		&cli.StringFlag{
			Name:        "llm-args",
			Sources:     cli.EnvVars("BEEHIVE_LLM_ARGS"),
			Destination: &c.Args,
		},
	}
}

// Validate refuses to start when the LLM is not fully configured for
// the chosen provider.
//
// Different providers have different auth surfaces:
//   - gemini: Vertex AI flow — ADC handles credentials, but
//     BEEHIVE_LLM_ARGS must carry project_id + location. No API key.
//   - claude: two paths, mutually exclusive.
//   - Vertex AI: BEEHIVE_LLM_ARGS must carry project_id + location;
//     ADC handles credentials. No API key.
//   - Anthropic direct API: BEEHIVE_LLM_API_KEY is required and
//     BEEHIVE_LLM_ARGS=project_id/location must NOT be set.
//
// blog-kind sources need the Extractor regardless of provider, so we
// always require provider + model. Model has no implicit default —
// gollem's provider-internal defaults are intentionally NOT relied on.
func (c *LLM) Validate() error {
	missing := []string{}
	if c.Provider == "" {
		missing = append(missing, "BEEHIVE_LLM_PROVIDER")
	}
	if c.Model == "" {
		missing = append(missing, "BEEHIVE_LLM_MODEL")
	}

	switch c.Provider {
	case "gemini":
		args, err := c.ArgsMap()
		if err != nil {
			return err
		}
		if args["project_id"] == "" {
			missing = append(missing, "BEEHIVE_LLM_ARGS=project_id=...")
		}
		if args["location"] == "" {
			missing = append(missing, "BEEHIVE_LLM_ARGS=location=...")
		}
	case "claude":
		args, err := c.ArgsMap()
		if err != nil {
			return err
		}
		hasVertex := args["project_id"] != "" && args["location"] != ""
		hasAPIKey := c.APIKey != ""
		switch {
		case hasVertex && hasAPIKey:
			return goerr.New("claude: BEEHIVE_LLM_ARGS=project_id/location and BEEHIVE_LLM_API_KEY are mutually exclusive — pick one auth path",
				goerr.V("provider", c.Provider),
				goerr.T(errutil.TagInvalidInput))
		case hasVertex, hasAPIKey:
			// OK
		default:
			missing = append(missing,
				"either BEEHIVE_LLM_ARGS=project_id=...,location=... (Vertex AI) or BEEHIVE_LLM_API_KEY (Anthropic direct API)")
		}
	case "":
		// provider already flagged above
	default:
		return goerr.New("unsupported BEEHIVE_LLM_PROVIDER (only gemini / claude)",
			goerr.V("provider", c.Provider),
			goerr.T(errutil.TagInvalidInput))
	}

	if len(missing) == 0 {
		return nil
	}
	return goerr.New("LLM configuration is incomplete — every blog-kind source needs it",
		goerr.V("missing", missing),
		goerr.V("provider", c.Provider),
		goerr.T(errutil.TagInvalidInput))
}

// ArgsMap parses Args ("k=v,k=v") into a map. Returns an empty map when
// Args is empty.
func (c *LLM) ArgsMap() (map[string]string, error) {
	out := map[string]string{}
	if c.Args == "" {
		return out, nil
	}
	for kv := range strings.SplitSeq(c.Args, ",") {
		kv = strings.TrimSpace(kv)
		if kv == "" {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq <= 0 {
			return nil, goerr.New("invalid BEEHIVE_LLM_ARGS entry",
				goerr.V("entry", kv),
				goerr.T(errutil.TagInvalidInput))
		}
		out[strings.TrimSpace(kv[:eq])] = strings.TrimSpace(kv[eq+1:])
	}
	return out, nil
}
