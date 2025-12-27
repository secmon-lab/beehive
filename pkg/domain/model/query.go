package model

// IoCListOptions represents query options for listing IoCs
type IoCListOptions struct {
	Offset    int
	Limit     int
	SortField IoCSortField
	SortOrder SortOrder
}

// IoCSortField represents the field to sort IoCs by
type IoCSortField string

const (
	IoCSortByType        IoCSortField = "type"
	IoCSortByValue       IoCSortField = "value"
	IoCSortBySourceID    IoCSortField = "source_id"
	IoCSortByStatus      IoCSortField = "status"
	IoCSortByFirstSeenAt IoCSortField = "first_seen_at"
	IoCSortByUpdatedAt   IoCSortField = "updated_at"
)

// SortOrder represents the sort order
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// IoCConnection represents a paginated list of IoCs
type IoCConnection struct {
	Items []*IoC
	Total int
}
