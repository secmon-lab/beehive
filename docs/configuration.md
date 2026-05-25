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
| `BEEHIVE_LLM_PROVIDER`  | `gemini`  | `gemini` or `claude`. No other values are accepted.                           |
| `BEEHIVE_LLM_MODEL`     | — (required) | e.g. `gemini-2.5-pro`, `claude-sonnet-4@20250514`, `claude-sonnet-4-5-20250929`. **Required** — beehive never falls back to provider-internal defaults. |
| `BEEHIVE_LLM_API_KEY`   | — (Anthropic direct API only) | Anthropic API key for `claude` via Anthropic's direct API. Not used by `gemini`. Mutually exclusive with `BEEHIVE_LLM_ARGS=project_id/location` on `claude`. |
| `BEEHIVE_LLM_ARGS`      | empty     | Comma-separated `key=value`. For Vertex AI (`gemini`, or `claude` via Vertex): `project_id=foo,location=us-central1`. Always use `project_id` (never bare `project`). |

### Provider auth paths

- **`gemini`** — Vertex AI only. Requires `BEEHIVE_LLM_ARGS=project_id=...,location=...`. Authentication is by ADC; `BEEHIVE_LLM_API_KEY` is unused.
- **`claude`** — two mutually exclusive paths, selected by what you configure:
  - **Vertex AI**: set `BEEHIVE_LLM_ARGS=project_id=...,location=...` (e.g. `location=global`). Authentication is by ADC. Do not set `BEEHIVE_LLM_API_KEY`.
  - **Anthropic direct API**: set `BEEHIVE_LLM_API_KEY`. Do not set `project_id` / `location` in `BEEHIVE_LLM_ARGS`.
  - Setting both, or neither, is a configuration error and `beehive validate` will refuse to start.

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
