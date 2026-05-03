package crawlbase

import (
	"context"
	"net/url"
	"strings"
)

// CrawlingAPI is a client for the general-purpose Crawlbase Crawling API.
// It's the engine the rest of the platform sits on top of — JS rendering,
// anti-bot bypass, residential proxy routing, and the scraper library
// are all reachable from here through the options map.
//
// See https://crawlbase.com/docs/crawling-api for the full parameter
// reference.
type CrawlingAPI struct {
	*baseClient
}

// NewCrawlingAPI constructs a Crawling API client with the given token.
// Token can be either the "normal" (TCP) token or the JavaScript token,
// depending on whether you need browser rendering. The client doesn't
// switch tokens per-call, so hold two clients if you alternate.
//
// The constructor returns ErrTokenRequired if token is empty.
func NewCrawlingAPI(token string) (*CrawlingAPI, error) {
	bc, err := newBaseClient(token, "")
	if err != nil {
		return nil, err
	}
	return &CrawlingAPI{baseClient: bc}, nil
}

// Get fetches targetURL through Crawlbase. Pass nil for options to send
// just the target; otherwise every Crawling API parameter is reachable
// here as a key in the options map (country, device, page_wait, scroll,
// scraper, async, callback, store, format, etc.).
func (a *CrawlingAPI) Get(targetURL string, options map[string]string) (*Response, error) {
	return a.GetWithContext(context.Background(), targetURL, options)
}

// GetWithContext is Get with cancellation / deadline / trace propagation.
// Use this from servers and any code path that should respect upstream
// timeouts.
func (a *CrawlingAPI) GetWithContext(ctx context.Context, targetURL string, options map[string]string) (*Response, error) {
	return a.request(ctx, "GET", targetURL, options, nil, "")
}

// Post sends data to targetURL through Crawlbase as an HTTP POST. The
// data argument can be:
//
//   - a [url.Values] for form-encoded bodies (default)
//   - a string for raw bodies (JSON, plain text, etc.)
//   - a []byte for raw bodies
//
// To send JSON, pass options["post_content_type"] = "application/json"
// and provide the JSON-encoded body as a string or []byte.
func (a *CrawlingAPI) Post(targetURL string, data any, options map[string]string) (*Response, error) {
	return a.PostWithContext(context.Background(), targetURL, data, options)
}

// PostWithContext is Post with cancellation / deadline / trace propagation.
func (a *CrawlingAPI) PostWithContext(ctx context.Context, targetURL string, data any, options map[string]string) (*Response, error) {
	body, contentType := encodePostBody(data, options)
	return a.request(ctx, "POST", targetURL, options, body, contentType)
}

// Put is the PUT counterpart to Post — same body-encoding rules, same
// options bag.
func (a *CrawlingAPI) Put(targetURL string, data any, options map[string]string) (*Response, error) {
	return a.PutWithContext(context.Background(), targetURL, data, options)
}

// PutWithContext is Put with cancellation / deadline / trace propagation.
func (a *CrawlingAPI) PutWithContext(ctx context.Context, targetURL string, data any, options map[string]string) (*Response, error) {
	body, contentType := encodePostBody(data, options)
	return a.request(ctx, "PUT", targetURL, options, body, contentType)
}

// encodePostBody resolves the data argument into an io.Reader + Content-Type
// header. Mirrors the Node SDK's behavior — url.Values defaults to form-
// encoded; string / []byte are passed through; the post_content_type option
// can override the header for JSON or other content types.
func encodePostBody(data any, options map[string]string) (body *strings.Reader, contentType string) {
	contentType = "application/x-www-form-urlencoded"
	if options != nil {
		if v, ok := options["post_content_type"]; ok && v != "" {
			contentType = v
			// Don't leak post_content_type into the query string.
			delete(options, "post_content_type")
		}
	}
	switch d := data.(type) {
	case nil:
		return strings.NewReader(""), contentType
	case string:
		return strings.NewReader(d), contentType
	case []byte:
		return strings.NewReader(string(d)), contentType
	case url.Values:
		return strings.NewReader(d.Encode()), contentType
	case map[string]string:
		v := url.Values{}
		for k, vv := range d {
			v.Set(k, vv)
		}
		return strings.NewReader(v.Encode()), contentType
	default:
		// Fallback — caller passed something we don't know how to encode.
		// Stringify it; the server will probably reject, but at least we
		// pass through a deterministic shape rather than panic.
		return strings.NewReader(""), contentType
	}
}
