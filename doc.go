// Package crawlbase is the official Go client for the Crawlbase API
// (https://crawlbase.com/docs/api-reference).
//
// The package exposes one client — [CrawlingAPI] — that covers every
// Crawlbase product through the unified Crawling API endpoint:
//
//   - Plain crawls (default usage)
//   - Built-in scrapers via options["scraper"] = "amazon-product-details" etc.
//   - Screenshots via options["screenshot"] = "true"
//   - Email extraction via options["scraper"] = "email-extractor"
//   - Async + webhook delivery via options["async"] / options["callback"]
//
// Idiomatic Go ergonomics, no external dependencies (only net/http +
// stdlib), sensible defaults.
//
// # Quickstart
//
//	api := crawlbase.NewCrawlingAPI("YOUR_TOKEN")
//	res, err := api.Get("https://github.com/anthropic", nil)
//	if err != nil { log.Fatal(err) }
//	if res.StatusCode == 200 {
//	    fmt.Println(res.Body)
//	}
//
// # Tokens
//
// Crawlbase issues two tokens per account — a "normal" (TCP) token for static
// HTML / JSON endpoints, and a "JavaScript" token for SPAs and pages that
// hide content behind client-side rendering. Each client is constructed with
// one token; if you alternate between them, hold two clients.
//
// # Options
//
// Every Crawling API parameter (country, device, page_wait, scroll, scraper,
// async, callback, etc. — see https://crawlbase.com/docs/crawling-api) is
// passed as an entry in the options map. Pass nil for no options.
//
//	api.Get(url, map[string]string{
//	    "country":   "DE",
//	    "page_wait": "2000",
//	    "scroll":    "true",
//	})
//
// # Context
//
// Every verb has a *WithContext variant for cancellation, deadlines, and
// trace propagation:
//
//	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
//	defer cancel()
//	res, err := api.GetWithContext(ctx, url, nil)
//
// # Response
//
// All verbs return a [Response] with the HTTP status, body, lower-cased
// headers, and the Crawlbase-specific verdict fields (PCStatus,
// OriginalStatus, URL, RID) lifted out of the headers for typed access.
package crawlbase
