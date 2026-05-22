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
		return nil, goerr.New("non-2xx response",
			goerr.V("url", url),
			goerr.V("status", resp.StatusCode),
			goerr.T(errutil.TagBusy),
		)
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

func (d *safeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
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
	}
	return d.inner.DialContext(ctx, network, addr)
}

func isForbidden(a netip.Addr) bool {
	return a.IsLoopback() ||
		a.IsPrivate() ||
		a.IsLinkLocalUnicast() ||
		a.IsLinkLocalMulticast() ||
		a.IsMulticast() ||
		a.IsUnspecified()
}
