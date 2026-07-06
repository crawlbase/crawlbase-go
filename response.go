package crawlbase

import "encoding/base64"

// Response is what every Crawlbase API verb returns on success.
//
// StatusCode is the HTTP status of the SDK's request to Crawlbase.
// CBStatus is Crawlbase's verdict on the *target* (the site you asked it
// to crawl). They can disagree — a target can return 200 with empty body,
// in which case StatusCode is 200 but CBStatus is 520. Always branch on
// CBStatus when deciding whether to retry. See
// https://crawlbase.com/docs/crawling-api/#errors for the full table.
type Response struct {
	// StatusCode is the HTTP status of the request to Crawlbase.
	StatusCode int

	// Body is the page content returned by the target (or a JSON envelope
	// when the call set format=json or scraper=NAME).
	Body string

	// Headers are the response headers, lower-cased on the way in.
	Headers map[string]string

	// CBStatus is the Crawlbase verdict on the target — pulled from the
	// `cb_status` (or legacy `pc_status`) response header. Branch on this
	// for retry decisions. Zero when not present.
	CBStatus int

	// Deprecated: Use CBStatus instead. PCStatus is kept for backward
	// compatibility and mirrors CBStatus. It will be removed in a future
	// major version.
	PCStatus int

	// OriginalStatus is the HTTP status the target returned to Crawlbase —
	// pulled from the `original_status` response header. Zero when not
	// present.
	OriginalStatus int

	// URL is the final URL after target-side redirects. Pulled from the
	// `url` response header.
	URL string

	// RID is the Crawlbase request identifier. Set when the call carried
	// async=true or store=true; empty otherwise.
	RID string

	// JSON is the response body pre-parsed into a generic map. Populated
	// only when the response Content-Type is JSON (e.g. scraper=... or
	// format=json calls). Use it to avoid double-parsing the body.
	JSON map[string]any
}

// ImageBytes decodes the base64-encoded screenshot in res.Body into raw
// image bytes ready for os.WriteFile / image.Decode. Use this on
// responses from screenshot calls (CrawlingAPI.Get with
// options["screenshot"] = "true").
//
// Returns an error if the body isn't valid base64 — verify
// res.StatusCode and res.CBStatus first.
func ImageBytes(res *Response) ([]byte, error) {
	return base64.StdEncoding.DecodeString(res.Body)
}
