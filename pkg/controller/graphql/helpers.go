package graphql

import (
	"sort"
	"time"

	"github.com/secmon-lab/beehive/pkg/domain/model"
	graphql1 "github.com/secmon-lab/beehive/pkg/domain/model/graphql"
)

func ptrIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func toModelSortField(field *graphql1.IoCSortField) model.IoCSortField {
	if field == nil {
		return ""
	}
	switch *field {
	case graphql1.IoCSortFieldType:
		return model.IoCSortByType
	case graphql1.IoCSortFieldValue:
		return model.IoCSortByValue
	case graphql1.IoCSortFieldSourceID:
		return model.IoCSortBySourceID
	case graphql1.IoCSortFieldStatus:
		return model.IoCSortByStatus
	case graphql1.IoCSortFieldFirstSeenAt:
		return model.IoCSortByFirstSeenAt
	case graphql1.IoCSortFieldUpdatedAt:
		return model.IoCSortByUpdatedAt
	default:
		return ""
	}
}

func toModelSortOrder(order *graphql1.SortOrder) model.SortOrder {
	if order == nil {
		return ""
	}
	switch *order {
	case graphql1.SortOrderAsc:
		return model.SortOrderAsc
	case graphql1.SortOrderDesc:
		return model.SortOrderDesc
	default:
		return ""
	}
}

func toGraphQLIoC(ioc *model.IoC) *graphql1.IoC {
	var sourceURL *string
	if ioc.SourceURL != "" {
		sourceURL = &ioc.SourceURL
	}

	return &graphql1.IoC{
		ID:          ioc.ID,
		SourceID:    ioc.SourceID,
		SourceType:  ioc.SourceType,
		Type:        string(ioc.Type),
		Value:       ioc.Value,
		Description: ioc.Description,
		SourceURL:   sourceURL,
		Context:     ioc.Context,
		Status:      string(ioc.Status),
		FirstSeenAt: ioc.FirstSeenAt,
		UpdatedAt:   ioc.UpdatedAt,
	}
}

func toGraphQLSourceState(state *model.SourceState) *graphql1.SourceState {
	var lastFetchedAt *time.Time
	var lastItemID *string
	var lastItemDate *time.Time
	var lastError *string

	if !state.LastFetchedAt.IsZero() {
		lastFetchedAt = &state.LastFetchedAt
	}
	if state.LastItemID != "" {
		lastItemID = &state.LastItemID
	}
	if !state.LastItemDate.IsZero() {
		lastItemDate = &state.LastItemDate
	}
	if state.LastError != "" {
		lastError = &state.LastError
	}

	return &graphql1.SourceState{
		SourceID:      state.SourceID,
		LastFetchedAt: lastFetchedAt,
		LastItemID:    lastItemID,
		LastItemDate:  lastItemDate,
		ItemCount:     int(state.ItemCount),
		ErrorCount:    int(state.ErrorCount),
		LastError:     lastError,
		UpdatedAt:     state.UpdatedAt,
	}
}

func toGraphQLHistory(h *model.History) *graphql1.History {
	errors := make([]*graphql1.FetchError, len(h.Errors))
	for i, e := range h.Errors {
		// Sort keys to ensure consistent order in GraphQL response
		keys := make([]string, 0, len(e.Values))
		for k := range e.Values {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		values := make([]*graphql1.KeyValue, 0, len(e.Values))
		for _, k := range keys {
			values = append(values, &graphql1.KeyValue{
				Key:   k,
				Value: e.Values[k],
			})
		}

		errors[i] = &graphql1.FetchError{
			Message: e.Message,
			Values:  values,
		}
	}

	return &graphql1.History{
		ID:             h.ID,
		SourceID:       h.SourceID,
		SourceType:     string(h.SourceType),
		Status:         string(h.Status),
		StartedAt:      h.StartedAt,
		CompletedAt:    h.CompletedAt,
		ProcessingTime: int(h.ProcessingTime / time.Millisecond),
		Urls:           h.URLs,
		ItemsFetched:   h.ItemsFetched,
		IoCsExtracted:  h.IoCsExtracted,
		IoCsCreated:    h.IoCsCreated,
		IoCsUpdated:    h.IoCsUpdated,
		IoCsUnchanged:  h.IoCsUnchanged,
		ErrorCount:     h.ErrorCount,
		Errors:         errors,
		CreatedAt:      h.CreatedAt,
	}
}
