package crawlbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// apiBase is the host that every Crawlbase API client targets. Override
// in tests via baseClient.endpoint if needed.
const apiBase = "https://api.crawlbase.com"

// defaultTimeout matches the timeout the other Crawlbase SDKs ship with.
const defaultTimeout = 90 * time.Second

// ErrTokenRequired is returned by the constructors when called with an
// empty token. Most other errors come straight from net/http and are
// returned to the caller as-is.
var ErrTokenRequired = errors.New("crawlbase: token is required")

// baseClient holds the state shared by every API client (CrawlingAPI,
// ScraperAPI, LeadsAPI, ScreenshotsAPI). Each public client embeds a
// pointer to one of these and provides verb-shaped methods on top.
//
// Fields are exported so callers can tune them after construction —
// e.g. set a longer Timeout for slow targets, or swap in an HTTPClient
// with a custom transport for tracing.
type baseClient struct {
	// Token is the Crawlbase API token. Required.
	Token string

	// Timeout applies to the whole HTTP request (dial + TLS + send +
	// receive). Defaults to 90s on construction.
	Timeout time.Duration

	// HTTPClient is the underlying *http.Client. Defaults to a fresh
	// client with Timeout set; replace it with your own (e.g. one
	// instrumented with go's httptrace) if you need request hooks.
	HTTPClient *http.Client

	// path is the API sub-path: "" for Crawling, "scraper" for Scraper,
	// "leads" for Leads, "screenshots" for Screenshots. Set by each
	// API's constructor; not exported.
	path string

	// endpoint overrides apiBase. Used by tests to point at httptest
	// servers; left empty in production.
	endpoint string
}

// newBaseClient is the shared constructor used by every public
// New*API() — keeps timeout / http client / token validation in one spot.
func newBaseClient(token, path string) (*baseClient, error) {
	if token == "" {
		return nil, ErrTokenRequired
	}
	timeout := defaultTimeout
	return &baseClient{
		Token:      token,
		Timeout:    timeout,
		HTTPClient: &http.Client{Timeout: timeout},
		path:       path,
	}, nil
}

// request is the single chokepoint every verb funnels through. It builds
// the URL, attaches the body for POST/PUT, sends the request through
// HTTPClient, and parses the response into a *Response.
//
// method must be one of "GET", "POST", "PUT". options keys are passed
// through verbatim as query params (URL-encoded); pass nil to send only
// the token + url. body and contentType apply on POST/PUT.
func (c *baseClient) request(
	ctx context.Context,
	method string,
	targetURL string,
	options map[string]string,
	body io.Reader,
	contentType string,
) (*Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint := c.endpoint
	if endpoint == "" {
		endpoint = apiBase
	}
	reqURL := buildURL(endpoint, c.path, c.Token, targetURL, options)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("crawlbase: build request: %w", err)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	// NOTE: do not set Accept-Encoding manually. net/http.Transport
	// adds "Accept-Encoding: gzip" itself and *automatically*
	// decompresses the response body before handing it to us — but
	// only when the caller didn't already set the header. If we set
	// it ourselves, the user is presumed to want raw encoded bytes
	// back, and ReadAll returns gzip-compressed data.
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: c.Timeout}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crawlbase: request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("crawlbase: read response: %w", err)
	}

	out := &Response{
		StatusCode: resp.StatusCode,
		Body:       string(bodyBytes),
		Headers:    lowerHeaders(resp.Header),
	}

	// Lift the Crawlbase-specific verdict fields out of the headers for
	// typed access. Prefer `cb_status`; fall back to legacy `pc_status`.
	if v := out.Headers["cb_status"]; v != "" {
		out.CBStatus, _ = strconv.Atoi(v)
	} else if v := out.Headers["pc_status"]; v != "" {
		out.CBStatus, _ = strconv.Atoi(v)
	}
	out.PCStatus = out.CBStatus
	if v := out.Headers["original_status"]; v != "" {
		out.OriginalStatus, _ = strconv.Atoi(v)
	}
	out.URL = out.Headers["url"]
	out.RID = out.Headers["rid"]

	// Auto-parse JSON content. Mirrors the Node SDK behavior — saves
	// callers a json.Unmarshal step on scraper / format=json responses.
	if ct := out.Headers["content-type"]; strings.Contains(ct, "json") && len(bodyBytes) > 0 {
		_ = json.Unmarshal(bodyBytes, &out.JSON)
		// If the JSON envelope carries url / cb_status / original_status,
		// prefer those over the headers (matches Node SDK behavior — the
		// JSON form is canonical for format=json calls).
		if out.JSON != nil {
			if v, ok := out.JSON["url"].(string); ok && v != "" {
				out.URL = v
			}
			if v, ok := out.JSON["cb_status"].(float64); ok {
				out.CBStatus = int(v)
			} else if v, ok := out.JSON["pc_status"].(float64); ok {
				out.CBStatus = int(v)
			}
			out.PCStatus = out.CBStatus
			if v, ok := out.JSON["original_status"].(float64); ok {
				out.OriginalStatus = int(v)
			}
		}
	}

	return out, nil
}

// buildURL constructs the final request URL: endpoint + path + ?token=X&url=Y&...
//
// Keep the order stable: token first, target url second, then options
// alphabetically. The Crawlbase API doesn't care about ordering, but
// stable URLs make request fingerprinting / caching / log diffing easier.
func buildURL(endpoint, path, token, targetURL string, options map[string]string) string {
	q := url.Values{}
	q.Set("token", token)
	if targetURL != "" {
		q.Set("url", targetURL)
	}
	for k, v := range options {
		// "method" is reserved internally — don't leak it into the URL.
		if k == "method" {
			continue
		}
		q.Set(k, v)
	}

	full := endpoint
	if path != "" {
		// Trim any trailing slash on endpoint, leading slash on path.
		full = strings.TrimRight(endpoint, "/") + "/" + strings.TrimLeft(path, "/")
	}
	return full + "?" + q.Encode()
}

// lowerHeaders flattens net/http's MIMEHeader (multi-valued) into a
// single map[string]string keyed by lower-case header name. Last value
// wins on the rare header that ships multiple values (Set-Cookie etc.).
func lowerHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		out[strings.ToLower(k)] = v[len(v)-1]
	}
	return out
}
