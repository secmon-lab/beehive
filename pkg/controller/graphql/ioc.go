package graphql

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	gqlmodel "github.com/secmon-lab/beehive/pkg/domain/model/graphql"
)

func (r *queryResolver) ListIoCs(ctx context.Context, options *gqlmodel.IoCListOptions) (*gqlmodel.IoCConnection, error) {
	// Convert GraphQL input to domain model
	var opts *model.IoCListOptions
	if options != nil {
		opts = &model.IoCListOptions{
			Offset:    ptrIntValue(options.Offset),
			Limit:     ptrIntValue(options.Limit),
			SortField: toModelSortField(options.SortField),
			SortOrder: toModelSortOrder(options.SortOrder),
		}
	}

	connection, err := r.repo.ListIoCs(ctx, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list IoCs")
	}

	items := make([]*gqlmodel.IoC, len(connection.Items))
	for i, ioc := range connection.Items {
		items[i] = toGraphQLIoC(ioc)
	}

	return &gqlmodel.IoCConnection{
		Items: items,
		Total: connection.Total,
	}, nil
}

func ptrIntValue(ptr *int) int {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func toModelSortField(field *gqlmodel.IoCSortField) model.IoCSortField {
	if field == nil {
		return ""
	}
	switch *field {
	case gqlmodel.IoCSortFieldType:
		return model.IoCSortByType
	case gqlmodel.IoCSortFieldValue:
		return model.IoCSortByValue
	case gqlmodel.IoCSortFieldSourceID:
		return model.IoCSortBySourceID
	case gqlmodel.IoCSortFieldStatus:
		return model.IoCSortByStatus
	case gqlmodel.IoCSortFieldFirstSeenAt:
		return model.IoCSortByFirstSeenAt
	case gqlmodel.IoCSortFieldUpdatedAt:
		return model.IoCSortByUpdatedAt
	default:
		return ""
	}
}

func toModelSortOrder(order *gqlmodel.SortOrder) model.SortOrder {
	if order == nil {
		return ""
	}
	switch *order {
	case gqlmodel.SortOrderAsc:
		return model.SortOrderAsc
	case gqlmodel.SortOrderDesc:
		return model.SortOrderDesc
	default:
		return ""
	}
}

func (r *queryResolver) GetIoC(ctx context.Context, id string) (*gqlmodel.IoC, error) {
	ioc, err := r.repo.GetIoC(ctx, id)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get IoC", goerr.V("id", id))
	}

	return toGraphQLIoC(ioc), nil
}

func toGraphQLIoC(ioc *model.IoC) *gqlmodel.IoC {
	var sourceURL *string
	if ioc.SourceURL != "" {
		sourceURL = &ioc.SourceURL
	}

	return &gqlmodel.IoC{
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
