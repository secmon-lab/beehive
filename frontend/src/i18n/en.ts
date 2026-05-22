// English dictionary. Keys are dotted paths describing the location of
// the string ("nav.sources"). Add new keys here when you introduce a
// user-visible string — never inline literals in JSX.

export const en = {
  // Navigation
  "nav.sources": "Sources",
  "nav.iocs": "IoCs",
  "nav.runs": "Runs",

  // Sources page
  "sources.title": "Sources",
  "sources.subtitle": "Sources are defined in a single config.toml file. Edit it and redeploy to change the catalog.",
  "sources.column.name": "Name",
  "sources.column.kind": "Kind",
  "sources.column.type": "Type",
  "sources.column.interval": "Interval",
  "sources.column.disabled": "Disabled",
  "sources.column.override": "Override",
  "sources.column.lastStatus": "Last status",
  "sources.column.lastFetchedAt": "Last fetched",
  "sources.empty": "No sources defined yet.",
  "sources.action.fetch": "Fetch now",
  "sources.action.fetchAll": "Fetch all due",

  // IoCs page
  "iocs.title": "IoC lookup",
  "iocs.subtitle": "Look up a single indicator by exact (type, value). Bulk queries go through BigQuery.",
  "iocs.placeholder.value": "value",
  "iocs.action.lookup": "Lookup",
  "iocs.notFound": "Not found",
  "iocs.recent.heading": "Recent IoCs",
  "iocs.recent.empty": "No IoCs yet.",
  "iocs.column.type": "Type",
  "iocs.column.value": "Value",
  "iocs.column.firstSeenAt": "First seen",
  "iocs.column.lastSeenAt": "Last seen",

  // Runs page
  "runs.title": "Runs",
  "runs.subtitle": "Recent fetch invocations. Each row is one POST /api/v1/fetch call.",
  "runs.empty": "No runs yet.",
} as const;
