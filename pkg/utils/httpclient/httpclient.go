package httpclient

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/m-mizutani/goerr/v2"
)

var (
	// ErrFetchFailed is returned when HTTP fetch fails
	ErrFetchFailed = goerr.New("failed to fetch from URL")
)

// HTTPClient is an interface for HTTP client operations
// This allows for easy mocking in tests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// FetchWithContext fetches data from a URL with context support
// Uses a default HTTP client with 30-second timeout
func FetchWithContext(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return FetchWithClient(ctx, client, url)
}

// FetchWithClient fetches data from a URL using the provided HTTP client
func FetchWithClient(ctx context.Context, client HTTPClient, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create request", goerr.V("url", url))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(ErrFetchFailed, "HTTP request failed",
			goerr.V("url", url))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.Wrap(ErrFetchFailed, "non-200 status code",
			goerr.V("url", url),
			goerr.V("status_code", resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read response body",
			goerr.V("url", url))
	}

	return data, nil
}
