// Package config translates env vars / CLI flags into typed values that
// the rest of beehive consumes. Each Config struct describes the inputs
// of one subsystem so that wiring stays explicit at main.
package config

import (
	"io"
	"log/slog"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Logger collects the user-facing logger settings.
type Logger struct {
	Level  string // "debug"|"info"|"warn"|"error"
	Format string // "json"|"text"
}

func (c *Logger) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "log-level",
			Sources:     cli.EnvVars("BEEHIVE_LOG_LEVEL"),
			Value:       "info",
			Destination: &c.Level,
		},
		&cli.StringFlag{
			Name:        "log-format",
			Sources:     cli.EnvVars("BEEHIVE_LOG_FORMAT"),
			Value:       "json",
			Destination: &c.Format,
		},
	}
}

func (c *Logger) Build(w io.Writer) *slog.Logger {
	return logging.Build(w, logging.Config{
		Level:  parseLevel(c.Level),
		Format: c.Format,
	})
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
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
//   - openai / claude: API key flow — BEEHIVE_LLM_API_KEY is required.
//
// blog-kind sources need the Extractor regardless of provider, so we
// always require provider + model.
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
	case "openai", "claude":
		if c.APIKey == "" {
			missing = append(missing, "BEEHIVE_LLM_API_KEY")
		}
	case "":
		// provider already flagged above
	default:
		return goerr.New("unsupported BEEHIVE_LLM_PROVIDER (only gemini/openai/claude)",
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
