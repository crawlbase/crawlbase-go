package crawlbase

import "context"

// ScraperAPI returns parsed JSON for sites Crawlbase has a built-in
// scraper for (Amazon, Google, LinkedIn, Instagram, eBay, etc.). Same
// transport as CrawlingAPI; different endpoint that runs the page
// through the right parser before returning.
//
// See https://crawlbase.com/docs/scrapers for the catalog of supported
// sites and their per-site parameter reference.
type ScraperAPI struct {
	*baseClient
}

// NewScraperAPI constructs a Scraper API client with the given token.
// Returns ErrTokenRequired on an empty token.
func NewScraperAPI(token string) (*ScraperAPI, error) {
	bc, err := newBaseClient(token, "scraper")
	if err != nil {
		return nil, err
	}
	return &ScraperAPI{baseClient: bc}, nil
}

// Get scrapes targetURL using the scraper named in options["scraper"]
// (e.g. "amazon-product-details"). Returns parsed JSON in Response.Body
// and pre-decoded into Response.JSON.
//
// Only GET is supported on the Scraper API — POST and PUT are not
// available (mirrors the Node / Python / Ruby SDKs).
func (a *ScraperAPI) Get(targetURL string, options map[string]string) (*Response, error) {
	return a.GetWithContext(context.Background(), targetURL, options)
}

// GetWithContext is Get with cancellation / deadline / trace propagation.
func (a *ScraperAPI) GetWithContext(ctx context.Context, targetURL string, options map[string]string) (*Response, error) {
	return a.request(ctx, "GET", targetURL, options, nil, "")
}
