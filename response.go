package crawlbase

// Response is what every Crawlbase API verb returns on success. Fields
// follow the same naming convention used by the other Crawlbase SDKs
// (Python / Node / Ruby / PHP) so cross-language porting is mechanical.
//
// StatusCode is the HTTP status of the SDK's request to Crawlbase.
// PCStatus is Crawlbase's verdict on the *target* (the site you asked it
// to crawl). They can disagree — a target can return 200 with empty body,
// in which case StatusCode is 200 but PCStatus is 520. Always branch on
// PCStatus when deciding whether to retry. See
// https://crawlbase.com/docs/crawling-api/#errors for the full table.
type Response struct {
	// StatusCode is the HTTP status of the request to Crawlbase.
	StatusCode int

	// Body is the page content returned by the target (or a JSON envelope
	// when the call set format=json or scraper=NAME).
	Body string

	// Headers are the response headers, lower-cased on the way in.
	Headers map[string]string

	// PCStatus is the Crawlbase verdict on the target — pulled from the
	// `pc_status` (or `cb_status`) response header. Branch on this for
	// retry decisions. Zero when not present.
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
