# beehive

[![test](https://github.com/secmon-lab/beehive/actions/workflows/test.yml/badge.svg)](https://github.com/secmon-lab/beehive/actions/workflows/test.yml) [![lint](https://github.com/secmon-lab/beehive/actions/workflows/lint.yml/badge.svg)](https://github.com/secmon-lab/beehive/actions/workflows/lint.yml) [![gosec](https://github.com/secmon-lab/beehive/actions/workflows/gosec.yml/badge.svg)](https://github.com/secmon-lab/beehive/actions/workflows/gosec.yml) [![trivy](https://github.com/secmon-lab/beehive/actions/workflows/trivy.yml/badge.svg)](https://github.com/secmon-lab/beehive/actions/workflows/trivy.yml) [![e2e](https://github.com/secmon-lab/beehive/actions/workflows/e2e.yml/badge.svg)](https://github.com/secmon-lab/beehive/actions/workflows/e2e.yml)

Pull-driven crawler that extracts IoCs (Indicators of Compromise) from security blogs and IoC feeds. Designed for Cloud Run zero scale: an external scheduler (e.g. Cloud Scheduler) hits `POST /api/v1/fetch` and beehive runs all due sources synchronously.

- **Backend**: Go (chi + urfave/cli)
- **Frontend**: React 19 + Vite + TypeScript (3 minimal pages: Sources / IoCs / Runs)
- **Datastore**: Firestore (single binary, frontend embedded via `//go:embed`)
- **LLM**: [gollem](https://github.com/m-mizutani/gollem) for IoC extraction from blog bodies

## Quick start

```bash
# 1. configure
cp examples/config.toml ./config.toml
export BEEHIVE_FIRESTORE_PROJECT_ID=my-project
export BEEHIVE_FIRESTORE_DATABASE=beehive
export BEEHIVE_LLM_PROVIDER=gemini
export BEEHIVE_LLM_MODEL=gemini-2.5-pro
export BEEHIVE_LLM_API_KEY=...

# 2. validate config (TOML + provider + env)
go run . validate

# 3. run server
go run . serve
# in another terminal:
curl -X POST http://localhost:8080/api/v1/fetch
```

## Source definition (TOML, IaC)

Sources are defined in a single TOML file (default `./config.toml`, override with `--config` / `-c` / `BEEHIVE_CONFIG`). There is no API for creating/editing sources; change the TOML and redeploy. See `examples/config.toml` for a starting point.

```toml
[[source]]
id = "trendmicro-research"
name = "Trend Micro Research"
type = "rss"
url = "https://www.trendmicro.com/en_us/research.rss"
interval = "1h"

[[source]]
id = "abuseipdb-blacklist"
name = "AbuseIPDB Blacklist"
type = "abuseipdb_blacklist"
interval = "6h"
```

A source is enabled by default. Set `disabled = true` to pause it without removing.

## IoC consumption

- **BigQuery (recommended for bulk / SQL)**: configure GCP-side `Firestore → BigQuery` export ([managed export](https://cloud.google.com/firestore/docs/manage-data/export-import) or the [`firestore-bigquery-export` extension](https://extensions.dev/extensions/firebase/firestore-bigquery-export)). beehive itself does not write to BigQuery.
- **HTTP API (exact match lookup)**: `GET /api/v1/iocs/lookup?type=ipv4&value=1.2.3.4`. Returns 404 if not found. List / bulk / streaming endpoints are not provided; use BigQuery for that.

## Architecture & docs

See [`docs/architecture.md`](docs/architecture.md), [`docs/configuration.md`](docs/configuration.md), [`docs/sources.md`](docs/sources.md), [`docs/bigquery.md`](docs/bigquery.md), and the design spec in `.spec/` for details.

## License

Apache-2.0
