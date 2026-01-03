package graphql_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	gqlcontroller "github.com/secmon-lab/beehive/pkg/controller/graphql"
	httpcontroller "github.com/secmon-lab/beehive/pkg/controller/http"
	"github.com/secmon-lab/beehive/pkg/domain/model"
	"github.com/secmon-lab/beehive/pkg/repository/memory"
	"github.com/secmon-lab/beehive/pkg/usecase"
)

// CountingRepository wraps memory repository to count GetState and BatchGetStates calls
type CountingRepository struct {
	*memory.Memory
	getStateCallCount       atomic.Int32
	batchGetStatesCallCount atomic.Int32
}

func (r *CountingRepository) GetState(ctx context.Context, sourceID string) (*model.SourceState, error) {
	r.getStateCallCount.Add(1)
	return r.Memory.GetState(ctx, sourceID)
}

func (r *CountingRepository) BatchGetStates(ctx context.Context, sourceIDs []string) (map[string]*model.SourceState, error) {
	r.batchGetStatesCallCount.Add(1)
	return r.Memory.BatchGetStates(ctx, sourceIDs)
}

func (r *CountingRepository) GetCallCount() int32 {
	return r.getStateCallCount.Load()
}

func (r *CountingRepository) GetBatchCallCount() int32 {
	return r.batchGetStatesCallCount.Load()
}

func (r *CountingRepository) ResetCallCount() {
	r.getStateCallCount.Store(0)
	r.batchGetStatesCallCount.Store(0)
}

func TestGraphQL_ListSources_N_Plus_1_Problem(t *testing.T) {
	ctx := context.Background()

	// Create a temporary sources config file with multiple sources
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "sources.toml")

	sourcesConfig := `
[rss.source-1]
url = "https://example.com/rss1"
tags = ["test"]

[rss.source-2]
url = "https://example.com/rss2"
tags = ["test"]

[feed.source-3]
schema = "abuse_ch_urlhaus"
url = "https://example.com/feed1"
tags = ["test"]

[feed.source-4]
schema = "abuse_ch_threatfox"
url = "https://example.com/feed2"
tags = ["test"]

[rss.source-5]
url = "https://example.com/rss3"
disabled = true
tags = ["test"]
`
	err := os.WriteFile(configPath, []byte(sourcesConfig), 0644)
	gt.NoError(t, err)

	// Create counting repository
	countingRepo := &CountingRepository{Memory: memory.New()}

	// Create source states for some sources
	now := time.Now()
	states := []*model.SourceState{
		{
			SourceID:      "source-1",
			LastFetchedAt: now.Add(-1 * time.Hour),
			ItemCount:     100,
			ErrorCount:    0,
			UpdatedAt:     now.Add(-1 * time.Hour),
		},
		{
			SourceID:      "source-2",
			LastFetchedAt: now.Add(-2 * time.Hour),
			ItemCount:     50,
			ErrorCount:    1,
			LastError:     "some error",
			UpdatedAt:     now.Add(-2 * time.Hour),
		},
		{
			SourceID:      "source-3",
			LastFetchedAt: now.Add(-30 * time.Minute),
			ItemCount:     200,
			ErrorCount:    0,
			UpdatedAt:     now.Add(-30 * time.Minute),
		},
	}

	for _, state := range states {
		err := countingRepo.SaveState(ctx, state)
		gt.NoError(t, err)
	}

	uc := usecase.New(countingRepo)
	fetchUC := usecase.NewFetchUseCase(countingRepo, nil)
	resolver, err := gqlcontroller.NewResolver(countingRepo, uc, fetchUC, configPath)
	gt.NoError(t, err)
	server := httpcontroller.New(resolver)

	query := `
		query {
			listSources {
				id
				type
				url
				enabled
				state {
					sourceID
					itemCount
					errorCount
				}
			}
		}
	`

	// Reset counter before request
	countingRepo.ResetCallCount()

	reqBody := map[string]interface{}{
		"query": query,
	}
	body, err := json.Marshal(reqBody)
	gt.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	t.Logf("Response: %s", w.Body.String())
	gt.N(t, w.Code).Equal(http.StatusOK).Describef("HTTP status should be 200, got %d: %s", w.Code, w.Body.String())

	var resp graphQLResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	gt.NoError(t, err)
	if len(resp.Errors) > 0 {
		t.Logf("GraphQL errors: %+v", resp.Errors)
	}
	gt.N(t, len(resp.Errors)).Equal(0).Describe("should have no GraphQL errors")

	var data struct {
		ListSources []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			URL     string `json:"url"`
			Enabled bool   `json:"enabled"`
			State   *struct {
				SourceID   string `json:"sourceID"`
				ItemCount  int    `json:"itemCount"`
				ErrorCount int    `json:"errorCount"`
			} `json:"state"`
		} `json:"listSources"`
	}
	err = json.Unmarshal(resp.Data, &data)
	gt.NoError(t, err)

	// Verify sources are returned
	gt.A(t, data.ListSources).Length(5).Describe("should return all 5 sources")

	// Check call counts - verify N+1 problem is resolved with data loader
	getStateCallCount := countingRepo.GetCallCount()
	batchGetStatesCallCount := countingRepo.GetBatchCallCount()
	sourceCount := len(data.ListSources)

	t.Logf("GetState called %d times, BatchGetStates called %d times for %d sources",
		getStateCallCount, batchGetStatesCallCount, sourceCount)

	// With data loader implemented: should use batch API (1 call) instead of individual calls (N calls)
	gt.N(t, int(getStateCallCount)).Equal(0).Describe("should NOT call GetState individually (N+1 problem resolved)")
	gt.N(t, int(batchGetStatesCallCount)).Equal(1).Describe("should call BatchGetStates once (data loader batching)")

	// Verify state data is correct for sources that have state
	var source1Found bool
	for _, src := range data.ListSources {
		if src.ID == "source-1" {
			source1Found = true
			gt.V(t, src.State).NotNil().Describe("source-1 should have state")
			if src.State != nil {
				gt.S(t, src.State.SourceID).Equal("source-1").Describe("source-1 state sourceID")
				gt.N(t, src.State.ItemCount).Equal(100).Describe("source-1 state itemCount")
				gt.N(t, src.State.ErrorCount).Equal(0).Describe("source-1 state errorCount")
			}
		}
	}
	gt.V(t, source1Found).Equal(true).Describe("source-1 should be in results")
}
