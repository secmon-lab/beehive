# BigQuery integration

beehive **does not** export to BigQuery itself. The Firestore → BigQuery path is delegated to GCP-managed plumbing, so you can pick whichever option suits your environment without changing beehive code.

## Option A — Managed export + Load (batch)

Best for daily / hourly snapshots. No additional running services.

1. Schedule a Cloud Scheduler job to invoke `gcloud firestore export`:
    ```bash
    gcloud firestore export gs://<your-bucket>/firestore/$(date -u +%Y%m%dT%H%M%S) \
        --project="$PROJECT_ID" --database="$DATABASE_ID"
    ```
2. After the export finishes, ingest the snapshot into BigQuery:
    ```bash
    bq load \
        --source_format=DATASTORE_BACKUP \
        --replace \
        "$DATASET.$TABLE" \
        "gs://<your-bucket>/firestore/<timestamp>/all_namespaces/kind_<Kind>/all_namespaces_kind_<Kind>.export_metadata"
    ```
   Repeat once per collection (`iocs`, `iocs/refs`, `runs`, `articles`, `states`, `locks`).

## Option B — Firestore-to-BigQuery Extension (streaming)

Best for near-real-time analytics. Requires the Firebase Extensions runtime and a Cloud Functions deployment.

1. Install the [`firestore-bigquery-export`](https://extensions.dev/extensions/firebase/firestore-bigquery-export) extension.
2. Repeat the installation per collection — the extension installs one Cloud Function per collection it watches.
3. The extension creates a changelog table and a "view" table that mirrors the current state of the collection.

## Sample queries

The schemas below correspond to the Go types in `pkg/domain/model` and the field-name policy in CLAUDE.md §3 (Repository fields are PascalCase in Firestore; BigQuery typically lowercases them automatically when ingesting via Datastore Backup).

Top hostnames seen across all blog sources in the last 7 days:

```sql
SELECT JSON_EXTRACT_SCALAR(ref.SourceID) AS source_id,
       ioc.Value AS domain,
       COUNT(*) AS occurrences
FROM `iocs.refs` AS ref
JOIN `iocs.iocs` AS ioc ON ioc.ID = ref.IoCID
WHERE ioc.Type = 'domain'
  AND ref.SeenAt > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
GROUP BY source_id, domain
ORDER BY occurrences DESC
LIMIT 50;
```

## What stays inside beehive

beehive does ensure that the Firestore documents it writes are friendly to BigQuery ingestion:

- All timestamps are UTC `time.Time` (BigQuery `TIMESTAMP`).
- Arrays use `[]string` / `[]SourceID` (`REPEATED STRING` in BigQuery).
- No nested struct soup — embedded sub-documents are kept flat where possible.
- The document ID is also stored as the `ID` field so JOINs that need both `__name__` and an explicit column work.
