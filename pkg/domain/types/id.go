package types

// SourceID identifies a Source. The value comes from the TOML `id` field
// (human-readable, stable). Used as Firestore document ID for states.
type SourceID string

func (id SourceID) String() string { return string(id) }

// ArticleID is a ULID assigned to a fetched article. Firestore document ID
// of an `articles/{ArticleID}` document.
type ArticleID string

func (id ArticleID) String() string { return string(id) }

// IoCID is the document ID of an `iocs/{IoCID}` document. Computed as the
// SHA-256 hex digest of `Type + "\t" + Value` so that the same IoC always
// yields the same ID across runs / instances.
type IoCID string

func (id IoCID) String() string { return string(id) }

// RefID identifies a single occurrence record in `iocs/{IoCID}/refs/{RefID}`.
// Computed as the SHA-256 hex digest of `SourceID + ":" + (ArticleID or RunID)`
// so that re-fetching the same article does not create duplicate refs.
type RefID string

func (id RefID) String() string { return string(id) }

// RunID is a ULID assigned to one POST /api/v1/fetch invocation.
type RunID string

func (id RunID) String() string { return string(id) }
