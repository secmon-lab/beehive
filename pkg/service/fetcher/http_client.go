package fetcher

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/utils/errutil"
)

// Default HTTP defaults shared by all providers. Per-source knobs are
// intentionally NOT exposed — see CLAUDE.md §0.
const (
	defaultTimeout = 30 * time.Second
	defaultMaxBody = 8 << 20 // 8 MiB
)

// UserAgent is set by main() before any provider runs. Default value is
// safe to use directly during tests.
var UserAgent = "beehive/dev"

// HTTPClient is the shared fetch client. SSRF protection rejects any
// resolved address in private / loopback / link-local ranges before the
// connection is established.
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient constructs the shared client. Callers MUST close every
// response body via safe.Close.
func NewHTTPClient() *HTTPClient {
	transport := &http.Transport{
		DialContext: (&safeDialer{
			inner: &net.Dialer{Timeout: 10 * time.Second},
		}).DialContext,
		MaxIdleConns:    32,
		IdleConnTimeout: 30 * time.Second,
	}
	return &HTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   defaultTimeout,
		},
	}
}

// Get issues a GET, reads up to defaultMaxBody bytes, and returns the
// body slice. Non-2xx responses are treated as errors.
func (c *HTTPClient) Get(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "build request", goerr.V("url", url))
	}
	req.Header.Set("User-Agent", UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, goerr.Wrap(err, "http get", goerr.V("url", url))
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Only treat 5xx (and 429) as retryable / "busy". 4xx is a
		// caller / upstream mistake that retrying will not help with,
		// so we leave it untagged.
		opts := []goerr.Option{
			goerr.V("url", url),
			goerr.V("status", resp.StatusCode),
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			opts = append(opts, goerr.T(errutil.TagBusy))
		}
		return nil, goerr.New("non-2xx response", opts...)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, defaultMaxBody+1))
	if err != nil {
		return nil, goerr.Wrap(err, "read body", goerr.V("url", url))
	}
	if int64(len(body)) > defaultMaxBody {
		return nil, goerr.New("response body exceeds limit",
			goerr.V("url", url),
			goerr.V("limit", defaultMaxBody),
			goerr.T(errutil.TagInvalidInput),
		)
	}
	return body, nil
}

// ---- SSRF guard ----

type safeDialer struct {
	inner *net.Dialer
}

// DialContext resolves the host once, rejects the connection if ANY
// returned address is in a forbidden range, AND then dials the first
// allowed IP literal directly. Passing the original hostname back to
// the inner dialer would let it re-resolve, opening a DNS rebinding
// hole: the second lookup could return an internal address after the
// first one validated.
func (d *safeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	var allowed netip.Addr
	for _, ip := range ips {
		na, ok := netip.AddrFromSlice(ip.IP)
		if !ok {
			continue
		}
		na = na.Unmap()
		if isForbidden(na) {
			return nil, goerr.New("forbidden destination",
				goerr.V("host", host),
				goerr.V("ip", na.String()),
				goerr.T(errutil.TagInvalidInput),
			)
		}
		if !allowed.IsValid() {
			allowed = na
		}
	}
	if !allowed.IsValid() {
		return nil, goerr.New("no usable address resolved",
			goerr.V("host", host),
			goerr.T(errutil.TagInvalidInput),
		)
	}
	// Dial the validated IP literal, bypassing any further name lookup.
	return d.inner.DialContext(ctx, network, net.JoinHostPort(allowed.String(), port))
}

func isForbidden(a netip.Addr) bool {
	return a.IsLoopback() ||
		a.IsPrivate() ||
		a.IsLinkLocalUnicast() ||
		a.IsLinkLocalMulticast() ||
		a.IsMulticast() ||
		a.IsUnspecified()
}
