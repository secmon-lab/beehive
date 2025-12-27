package memory

import (
	"sort"
	"strings"

	"github.com/secmon-lab/beehive/pkg/domain/model"
)

func sortIoCs(iocs []*model.IoC, sortField model.IoCSortField, sortOrder model.SortOrder) {
	desc := sortOrder == model.SortOrderDesc

	sort.Slice(iocs, func(i, j int) bool {
		var less bool

		switch sortField {
		case model.IoCSortByType:
			less = strings.ToLower(string(iocs[i].Type)) < strings.ToLower(string(iocs[j].Type))
		case model.IoCSortByValue:
			less = strings.ToLower(iocs[i].Value) < strings.ToLower(iocs[j].Value)
		case model.IoCSortBySourceID:
			less = strings.ToLower(iocs[i].SourceID) < strings.ToLower(iocs[j].SourceID)
		case model.IoCSortByStatus:
			less = strings.ToLower(string(iocs[i].Status)) < strings.ToLower(string(iocs[j].Status))
		case model.IoCSortByFirstSeenAt:
			less = iocs[i].FirstSeenAt.Before(iocs[j].FirstSeenAt)
		case model.IoCSortByUpdatedAt:
			less = iocs[i].UpdatedAt.Before(iocs[j].UpdatedAt)
		default:
			// Default sort by UpdatedAt descending
			less = iocs[i].UpdatedAt.After(iocs[j].UpdatedAt)
		}

		if desc {
			return !less
		}
		return less
	})
}
