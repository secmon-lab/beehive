package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

// FetchStatus represents the status of a fetch operation
type FetchStatus string

const (
	FetchStatusSuccess        FetchStatus = "success" // All operations succeeded
	FetchStatusPartialSuccess FetchStatus = "partial" // Some errors occurred but some items were processed
	FetchStatusFailure        FetchStatus = "failure" // Complete failure
)

// FetchError represents a single error that occurred during ingestion
type FetchError struct {
	Message string            // Error message
	Values  map[string]string // Context information from goerr.Values()
}

// History represents the history of a single fetch operation
type History struct {
	ID             string        // Unique identifier (UUIDv7)
	SourceID       string        // Source identifier
	SourceType     SourceType    // RSS or Feed
	Status         FetchStatus   // Operation status
	StartedAt      time.Time     // Start time
	CompletedAt    time.Time     // Completion time
	ProcessingTime time.Duration // Processing duration
	URLs           []string      // URLs accessed during fetch

	// Statistics (from FetchStats)
	ItemsFetched  int // Number of items fetched
	IoCsExtracted int // Number of IoCs extracted
	IoCsCreated   int // Number of new IoCs created
	IoCsUpdated   int // Number of IoCs updated
	IoCsUnchanged int // Number of unchanged IoCs
	ErrorCount    int // Number of errors

	// Error details
	Errors []*FetchError // List of errors with context

	CreatedAt time.Time // Record creation time
}

// GenerateHistoryID generates a unique ID for a history record using UUIDv7
func GenerateHistoryID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// DetermineFetchStatus determines the status based on error count and items fetched
func DetermineFetchStatus(errorCount, itemsFetched int) FetchStatus {
	if errorCount == 0 && itemsFetched > 0 {
		return FetchStatusSuccess
	} else if itemsFetched == 0 && errorCount > 0 {
		return FetchStatusFailure
	} else if errorCount > 0 {
		return FetchStatusPartialSuccess
	}
	// No errors and no items (empty feed) is still considered success
	return FetchStatusSuccess
}

// ExtractErrorInfo extracts error information from an error
// If the error is a goerr error, it extracts Values() as context
func ExtractErrorInfo(err error) *FetchError {
	fetchErr := &FetchError{
		Message: err.Error(),
		Values:  make(map[string]string),
	}

	// Try to extract goerr.Values if available
	if goErr := goerr.Unwrap(err); goErr != nil {
		for k, v := range goErr.Values() {
			fetchErr.Values[k] = fmt.Sprintf("%v", v)
		}
	}

	return fetchErr
}
