package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	gqlcontroller "github.com/secmon-lab/beehive/pkg/controller/graphql"
	httpcontroller "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
)

type graphQLResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []json.RawMessage `json:"errors,omitempty"`
}

// executeGraphQL executes a GraphQL query against the test server
func executeGraphQL(t *testing.T, server *httpcontroller.Server, query string, variables map[string]interface{}) *graphQLResponse {
	t.Helper()

	reqBody := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		reqBody["variables"] = variables
	}

	body, err := json.Marshal(reqBody)
	gt.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	gt.N(t, w.Code).Equal(http.StatusOK).Describef("HTTP status should be 200, got %d: %s", w.Code, w.Body.String())

	var resp graphQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	gt.NoError(t, err)

	return &resp
}

func TestGraphQL_Health(t *testing.T) {
	repo := memory.New()
	uc := usecase.New(repo)
	fetchUC := usecase.NewFetchUseCase(repo, nil)
	resolver, err := gqlcontroller.NewResolver(repo, uc, fetchUC, "")
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `query { health }`
	resp := executeGraphQL(t, server, query, nil)

	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no errors")

	var data struct {
		Health string `json:"health"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)
	gt.S(t, data.Health).Equal("OK").Describe("health check should return OK")
}

func TestGraphQL_ListIoCs(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	// Create test IoCs
	now := time.Now()
	testIoCs := []*model.IoC{
		{
			ID:          "ioc-001",
			SourceID:    "source-1",
			SourceType:  "feed",
			Type:        model.IoCTypeIPv4,
			Value:       "192.0.2.1",
			Description: "Test malicious IP",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-24 * time.Hour),
			UpdatedAt:   now,
		},
		{
			ID:          "ioc-002",
			SourceID:    "source-1",
			SourceType:  "feed",
			Type:        model.IoCTypeDomain,
			Value:       "evil.example.com",
			Description: "Test malicious domain",
			Status:      model.IoCStatusActive,
			FirstSeenAt: now.Add(-12 * time.Hour),
			UpdatedAt:   now,
		},
	}

	for _, ioc := range testIoCs {
		err := repo.UpsertIoC(ctx, ioc)
		gt.NoError(t, err)
	}

	uc := usecase.New(repo)
	resolver, err := gqlcontroller.NewResolver(repo, uc, usecase.NewFetchUseCase(repo, nil), "")
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `
		query {
			listIoCs {
				total
				items {
					id
					sourceID
					type
					value
					description
					status
				}
			}
		}
	`

	resp := executeGraphQL(t, server, query, nil)
	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no errors")

	var data struct {
		ListIoCs struct {
			Total int `json:"total"`
			Items []struct {
				ID          string `json:"id"`
				SourceID    string `json:"sourceID"`
				Type        string `json:"type"`
				Value       string `json:"value"`
				Description string `json:"description"`
				Status      string `json:"status"`
			} `json:"items"`
		} `json:"listIoCs"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)

	gt.N(t, data.ListIoCs.Total).Equal(2).Describe("total should be 2")
	gt.A(t, data.ListIoCs.Items).Length(2).Describe("should have 2 items")

	// Verify first IoC
	gt.A(t, data.ListIoCs.Items).At(0, func(t testing.TB, item struct {
		ID          string `json:"id"`
		SourceID    string `json:"sourceID"`
		Type        string `json:"type"`
		Value       string `json:"value"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}) {
		gt.S(t, item.SourceID).Equal("source-1").Describe("first IoC sourceID")
		gt.S(t, item.Type).Equal("ipv4").Describe("first IoC type")
		gt.S(t, item.Status).Equal("active").Describe("first IoC status")
	})
}

func TestGraphQL_GetIoC(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	// Create test IoC
	now := time.Now()
	testIoC := &model.IoC{
		ID:          "ioc-001",
		SourceID:    "source-1",
		SourceType:  "feed",
		Type:        model.IoCTypeIPv4,
		Value:       "192.0.2.1",
		Description: "Test malicious IP",
		Status:      model.IoCStatusActive,
		FirstSeenAt: now.Add(-24 * time.Hour),
		UpdatedAt:   now,
	}
	err := repo.UpsertIoC(ctx, testIoC)
	gt.NoError(t, err)

	uc := usecase.New(repo)
	resolver, err := gqlcontroller.NewResolver(repo, uc, usecase.NewFetchUseCase(repo, nil), "")
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `
		query($id: ID!) {
			getIoC(id: $id) {
				id
				sourceID
				type
				value
				description
				status
			}
		}
	`

	resp := executeGraphQL(t, server, query, map[string]interface{}{
		"id": "ioc-001",
	})
	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no errors")

	var data struct {
		GetIoC struct {
			ID          string `json:"id"`
			SourceID    string `json:"sourceID"`
			Type        string `json:"type"`
			Value       string `json:"value"`
			Description string `json:"description"`
			Status      string `json:"status"`
		} `json:"getIoC"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)

	gt.S(t, data.GetIoC.ID).Equal("ioc-001").Describe("IoC ID")
	gt.S(t, data.GetIoC.SourceID).Equal("source-1").Describe("IoC sourceID")
	gt.S(t, data.GetIoC.Type).Equal("ipv4").Describe("IoC type")
	gt.S(t, data.GetIoC.Value).Equal("192.0.2.1").Describe("IoC value")
	gt.S(t, data.GetIoC.Status).Equal("active").Describe("IoC status")
}

func TestGraphQL_ListHistories(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	// Create test histories
	now := time.Now()
	sourceID := "source-1"

	history1 := &model.History{
		ID:             model.GenerateHistoryID(),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeFeed,
		Status:         model.FetchStatusSuccess,
		StartedAt:      now.Add(-2 * time.Hour),
		CompletedAt:    now.Add(-2*time.Hour + 5*time.Second),
		ProcessingTime: 5 * time.Second,
		URLs:           []string{"https://example.com/feed1"},
		ItemsFetched:   100,
		IoCsExtracted:  80,
		IoCsCreated:    60,
		IoCsUpdated:    20,
		IoCsUnchanged:  0,
		ErrorCount:     0,
		Errors:         []*model.FetchError{},
		CreatedAt:      now.Add(-2 * time.Hour),
	}

	history2 := &model.History{
		ID:             model.GenerateHistoryID(),
		SourceID:       sourceID,
		SourceType:     model.SourceTypeFeed,
		Status:         model.FetchStatusPartialSuccess,
		StartedAt:      now.Add(-1 * time.Hour),
		CompletedAt:    now.Add(-1*time.Hour + 3*time.Second),
		ProcessingTime: 3 * time.Second,
		URLs:           []string{"https://example.com/feed2"},
		ItemsFetched:   50,
		IoCsExtracted:  40,
		IoCsCreated:    30,
		IoCsUpdated:    10,
		IoCsUnchanged:  0,
		ErrorCount:     2,
		Errors: []*model.FetchError{
			{Message: "failed to parse entry", Values: map[string]string{"line": "42"}},
		},
		CreatedAt: now.Add(-1 * time.Hour),
	}

	err := repo.SaveHistory(ctx, history1)
	gt.NoError(t, err)
	err = repo.SaveHistory(ctx, history2)
	gt.NoError(t, err)

	uc := usecase.New(repo)
	resolver, err := gqlcontroller.NewResolver(repo, uc, usecase.NewFetchUseCase(repo, nil), "")
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `
		query($sourceID: String!, $limit: Int, $offset: Int) {
			listHistories(sourceID: $sourceID, limit: $limit, offset: $offset) {
				total
				items {
					id
					sourceID
					sourceType
					status
					itemsFetched
					ioCsExtracted
					ioCsCreated
					ioCsUpdated
					errorCount
					errors {
						message
						values {
							key
							value
						}
					}
				}
			}
		}
	`

	resp := executeGraphQL(t, server, query, map[string]interface{}{
		"sourceID": sourceID,
		"limit":    10,
		"offset":   0,
	})
	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no errors")

	var data struct {
		ListHistories struct {
			Total *int `json:"total"`
			Items []struct {
				ID            string `json:"id"`
				SourceID      string `json:"sourceID"`
				SourceType    string `json:"sourceType"`
				Status        string `json:"status"`
				ItemsFetched  int    `json:"itemsFetched"`
				IoCsExtracted int    `json:"ioCsExtracted"`
				IoCsCreated   int    `json:"ioCsCreated"`
				IoCsUpdated   int    `json:"ioCsUpdated"`
				ErrorCount    int    `json:"errorCount"`
				Errors        []struct {
					Message string `json:"message"`
					Values  []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"values"`
				} `json:"errors"`
			} `json:"items"`
		} `json:"listHistories"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)

	// Total is not computed for performance reasons
	gt.A(t, data.ListHistories.Items).Length(2).Describe("should have 2 items")

	// Verify histories are sorted by newest first
	gt.A(t, data.ListHistories.Items).At(0, func(t testing.TB, item struct {
		ID            string `json:"id"`
		SourceID      string `json:"sourceID"`
		SourceType    string `json:"sourceType"`
		Status        string `json:"status"`
		ItemsFetched  int    `json:"itemsFetched"`
		IoCsExtracted int    `json:"ioCsExtracted"`
		IoCsCreated   int    `json:"ioCsCreated"`
		IoCsUpdated   int    `json:"ioCsUpdated"`
		ErrorCount    int    `json:"errorCount"`
		Errors        []struct {
			Message string `json:"message"`
			Values  []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"values"`
		} `json:"errors"`
	}) {
		gt.S(t, item.SourceID).Equal(sourceID).Describe("first history sourceID")
		gt.S(t, item.Status).Equal("partial").Describe("first history status")
		gt.N(t, item.ItemsFetched).Equal(50).Describe("first history items fetched")
		gt.N(t, item.ErrorCount).Equal(2).Describe("first history error count")
		gt.A(t, item.Errors).Length(1).Describe("first history should have 1 error")
	})
}

func TestGraphQL_GetHistory(t *testing.T) {
	ctx := context.Background()
	repo := memory.New()

	// Create test history
	now := time.Now()
	sourceID := "source-1"
	historyID := model.GenerateHistoryID()

	history := &model.History{
		ID:             historyID,
		SourceID:       sourceID,
		SourceType:     model.SourceTypeRSS,
		Status:         model.FetchStatusSuccess,
		StartedAt:      now,
		CompletedAt:    now.Add(3 * time.Second),
		ProcessingTime: 3 * time.Second,
		URLs:           []string{"https://example.com/rss", "https://example.com/article1"},
		ItemsFetched:   25,
		IoCsExtracted:  20,
		IoCsCreated:    15,
		IoCsUpdated:    5,
		IoCsUnchanged:  0,
		ErrorCount:     0,
		Errors:         []*model.FetchError{},
		CreatedAt:      now,
	}

	err := repo.SaveHistory(ctx, history)
	gt.NoError(t, err)

	uc := usecase.New(repo)
	resolver, err := gqlcontroller.NewResolver(repo, uc, usecase.NewFetchUseCase(repo, nil), "")
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `
		query($sourceID: String!, $id: ID!) {
			getHistory(sourceID: $sourceID, id: $id) {
				id
				sourceID
				sourceType
				status
				itemsFetched
				ioCsExtracted
				ioCsCreated
				ioCsUpdated
				errorCount
			}
		}
	`

	resp := executeGraphQL(t, server, query, map[string]interface{}{
		"sourceID": sourceID,
		"id":       historyID,
	})
	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no errors")

	var data struct {
		GetHistory struct {
			ID            string `json:"id"`
			SourceID      string `json:"sourceID"`
			SourceType    string `json:"sourceType"`
			Status        string `json:"status"`
			ItemsFetched  int    `json:"itemsFetched"`
			IoCsExtracted int    `json:"ioCsExtracted"`
			IoCsCreated   int    `json:"ioCsCreated"`
			IoCsUpdated   int    `json:"ioCsUpdated"`
			ErrorCount    int    `json:"errorCount"`
		} `json:"getHistory"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)

	gt.S(t, data.GetHistory.ID).Equal(historyID).Describe("history ID")
	gt.S(t, data.GetHistory.SourceID).Equal(sourceID).Describe("history sourceID")
	gt.S(t, data.GetHistory.SourceType).Equal("rss").Describe("history sourceType")
	gt.S(t, data.GetHistory.Status).Equal("success").Describe("history status")
	gt.N(t, data.GetHistory.ItemsFetched).Equal(25).Describe("history items fetched")
	gt.N(t, data.GetHistory.IoCsExtracted).Equal(20).Describe("history IoCs extracted")
	gt.N(t, data.GetHistory.IoCsCreated).Equal(15).Describe("history IoCs created")
	gt.N(t, data.GetHistory.IoCsUpdated).Equal(5).Describe("history IoCs updated")
	gt.N(t, data.GetHistory.ErrorCount).Equal(0).Describe("history error count")
}
