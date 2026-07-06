package crawlbase_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/crawlbase/crawlbase-go"
)

// Three-line quickstart. Replace YOUR_TOKEN with the token from your
// Crawlbase dashboard — sign-up gives 1,000 free requests, no credit
// card.
func ExampleNewCrawlingAPI() {
	api, err := crawlbase.NewCrawlingAPI("YOUR_TOKEN")
	if err != nil {
		log.Fatal(err)
	}
	res, err := api.Get("https://github.com/anthropic", nil)
	if err != nil {
		log.Fatal(err)
	}
	if res.StatusCode == 200 {
		fmt.Println(len(res.Body), "bytes received")
	}
}

// Reading the Crawlbase verdict on the target. CBStatus is the field
// you branch on for retry decisions — see
// https://crawlbase.com/docs/crawling-api/#errors for the full table.
func ExampleResponse_CBStatus() {
	api, _ := crawlbase.NewCrawlingAPI(os.Getenv("CRAWLBASE_TOKEN"))
	res, _ := api.Get("https://example.com/", nil)

	switch res.CBStatus {
	case 200:
		// success
	case 520, 525:
		// 520 = empty body, 525 = anti-bot couldn't be solved.
		// Switch to JS token and retry.
	case 521, 522, 523:
		// Target unreachable. Backoff + retry.
	default:
		// Other failure — log + alert.
	}
}

// Use the JavaScript token to render SPAs. Combine page_wait /
// ajax_wait / scroll / css_click_selector based on what the target
// needs. Order to think about: a fixed wait, then network-idle, then
// scroll for lazy-load, then click for any gating UI element.
func ExampleCrawlingAPI_javascriptRendering() {
	api, _ := crawlbase.NewCrawlingAPI("YOUR_JS_TOKEN")
	res, err := api.Get("https://spa.example.com", map[string]string{
		"page_wait": "2000",
		"ajax_wait": "true",
		"scroll":    "true",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.StatusCode)
}

// Apply a built-in scraper via the Crawling API to skip the parser
// step on supported sites. The Body comes back as a JSON string and
// is also pre-decoded into res.JSON for direct field access.
func ExampleCrawlingAPI_scraper() {
	api, _ := crawlbase.NewCrawlingAPI(os.Getenv("CRAWLBASE_TOKEN"))
	res, err := api.Get(
		"https://www.amazon.com/dp/B08N5WRWNW",
		map[string]string{"scraper": "amazon-product-details"},
	)
	if err != nil {
		log.Fatal(err)
	}
	if name, ok := res.JSON["name"].(string); ok {
		fmt.Println(name)
	}
}

// Use a context with a deadline for any code path that should respect
// upstream cancellation — HTTP handlers, RPC servers, anything else
// where a hung request would propagate.
func ExampleCrawlingAPI_GetWithContext() {
	api, _ := crawlbase.NewCrawlingAPI("YOUR_TOKEN")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := api.GetWithContext(ctx, "https://example.com/", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.StatusCode)
}

// Capture a screenshot via the Crawling API. The Body is base64-
// encoded image bytes; use [ImageBytes] to decode.
func ExampleCrawlingAPI_screenshot() {
	api, _ := crawlbase.NewCrawlingAPI("YOUR_JS_TOKEN")
	res, err := api.Get("https://www.apple.com/", map[string]string{
		"screenshot": "true",
	})
	if err != nil {
		log.Fatal(err)
	}
	img, err := crawlbase.ImageBytes(res)
	if err != nil {
		log.Fatal(err)
	}
	_ = os.WriteFile("apple.png", img, 0o644)
}
