# Configuration

All beehive configuration is environment-variable driven. The subset required by each subcommand is enforced at startup by `beehive validate`.

## HTTP server

| Variable             | Default  | Notes                                                                 |
|----------------------|----------|-----------------------------------------------------------------------|
| `BEEHIVE_HTTP_ADDR`  | `:8080`  | Listener address.                                                     |

There is no built-in authentication. Front the service with an authenticator (Cloud Run IAM, IAP, or the upcoming Rego-based authz layer).

## Logging

| Variable               | Default | Notes                       |
|------------------------|---------|-----------------------------|
| `BEEHIVE_LOG_LEVEL`    | `info`  | `debug`/`info`/`warn`/`error`. |
| `BEEHIVE_LOG_FORMAT`   | `json`  | `json` or `text` (clog).    |

## Source catalog

| Variable               | Default      | Notes                                                       |
|------------------------|--------------|-------------------------------------------------------------|
| `BEEHIVE_CONFIG`       | `./config.toml` | Path to the single TOML config file. Override with `--config` / `-c`. |

## Repository backend

| Variable                       | Default      | Notes                                                           |
|--------------------------------|--------------|-----------------------------------------------------------------|
| `BEEHIVE_REPO_BACKEND`         | `firestore`  | `firestore` for production, `memory` for local / tests.        |
| `BEEHIVE_FIRESTORE_PROJECT_ID` | — (required) | GCP project hosting the Firestore database.                    |
| `BEEHIVE_FIRESTORE_DATABASE`   | — (required) | Named database. The `(default)` database is intentionally not assumed. |

## LLM (gollem)

| Variable                | Default   | Notes                                                                         |
|-------------------------|-----------|-------------------------------------------------------------------------------|
| `BEEHIVE_LLM_PROVIDER`  | `gemini`  | `gemini`, `vertex`, `claude`, `openai`, etc.                                  |
| `BEEHIVE_LLM_MODEL`     | provider-specific | e.g. `gemini-2.5-pro`.                                                   |
| `BEEHIVE_LLM_API_KEY`   | — (required for API-key providers) | Not used by Vertex (uses ADC).                                                |
| `BEEHIVE_LLM_ARGS`      | empty     | Comma-separated `key=value`. For Vertex: `project_id=foo,location=us-central1`. Always use `project_id` (never bare `project`). |

## Provider credentials

Providers that talk to credentialed APIs read their own environment variables. The naming convention is `BEEHIVE_PROVIDER_<UPPER_SNAKE>_*`.

| Variable                                | Provider           |
|-----------------------------------------|--------------------|
| `BEEHIVE_PROVIDER_ABUSEIPDB_API_KEY`    | `abuseipdb_blacklist` |

`beehive validate` reports a missing required env var as a clear error with the `export ...` line you need.

## Test-time only

| Variable                          | Purpose                                                 |
|-----------------------------------|---------------------------------------------------------|
| `TEST_FIRESTORE_PROJECT_ID`       | Firestore emulator / sandbox project for the repository harness. |
| `TEST_FIRESTORE_DATABASE_ID`      | Database name. Tests skip the Firestore arm if either is unset. |
