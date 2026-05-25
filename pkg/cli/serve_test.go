package cli_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/beehive/pkg/cli"
	"github.com/secmon-lab/beehive/pkg/utils/safe"
)

// freeAddr returns an OS-assigned address we can hand to `serve` so the
// test does not collide with other parallel runs.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	gt.NoError(t, err)
	addr := l.Addr().String()
	gt.NoError(t, l.Close())
	return addr
}

func TestRun_ServeBootsAndServesHealth(t *testing.T) {
	addr := freeAddr(t)

	// LLM is mandatory at startup — supply harmless values so the
	// pre-flight Validate passes. Boot stops at HTTP listen; the LLM
	// is only actually invoked when a blog source is fetched.
	t.Setenv("BEEHIVE_LLM_PROVIDER", "gemini")
	t.Setenv("BEEHIVE_LLM_MODEL", "test-model")
	t.Setenv("BEEHIVE_LLM_API_KEY", "test-key")
	t.Setenv("BEEHIVE_LLM_ARGS", "project_id=test,location=us-central1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- cli.Run(ctx, []string{
			"beehive", "serve",
			"--http-addr", addr,
			"--repo-backend", "memory",
		}, "test")
	}()

	// Poll the health endpoint until it answers (server is ready) or we
	// time out. 2 seconds is generous for an in-process boot.
	url := "http://" + addr + "/api/v1/health"
	deadline := time.Now().Add(2 * time.Second)
	var resp *http.Response
	var err error
	for time.Now().Before(deadline) {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	gt.NoError(t, err).Required()
	defer safe.Close(ctx, resp.Body)
	gt.Equal(t, resp.StatusCode, http.StatusOK)

	cancel()

	// serve should exit cleanly within the shutdown grace period.
	select {
	case err := <-errCh:
		gt.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not exit after context cancellation")
	}
}
