//go:build integration

// Live integration tests. Disabled under the default `go test` invocation
// because they hit api.crawlbase.com and consume real request credits.
//
// To run:
//
//	CRAWLBASE_TOKEN=...  CRAWLBASE_JS_TOKEN=...  go test -tags=integration ./...
//
// Each test consumes ~1 credit. The suite is intentionally small.
package crawlbase_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/crawlbase/crawlbase-go"
)

func token(t *testing.T, env string) string {
	t.Helper()
	v := os.Getenv(env)
	if v == "" {
		t.Skipf("%s not set — skipping integration test", env)
	}
	return v
}

// TestIntegration_CrawlingGetStatic exercises the basic Crawling API
// path against a stable, low-cost target (httpbin echoes the request
// headers back, so we can sanity-check that we crawled a real page).
func TestIntegration_CrawlingGetStatic(t *testing.T) {
	api, err := crawlbase.NewCrawlingAPI(token(t, "CRAWLBASE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := api.GetWithContext(ctx, "https://httpbin.org/headers", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("StatusCode=%d PCStatus=%d OriginalStatus=%d body_len=%d",
		res.StatusCode, res.PCStatus, res.OriginalStatus, len(res.Body))

	if res.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", res.StatusCode)
	}
	if res.PCStatus != 200 {
		t.Fatalf("PCStatus = %d, want 200", res.PCStatus)
	}
	if !strings.Contains(res.Body, "headers") {
		t.Errorf("body missing expected 'headers' marker; got %q", res.Body[:min(200, len(res.Body))])
	}
}

// TestIntegration_CrawlingGetJSToken exercises the JavaScript-token
// path. We're not testing JS rendering per se — just confirming the
// JS token authenticates correctly through the same client interface.
func TestIntegration_CrawlingGetJSToken(t *testing.T) {
	api, err := crawlbase.NewCrawlingAPI(token(t, "CRAWLBASE_JS_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := api.GetWithContext(ctx, "https://example.com/", map[string]string{
		"page_wait": "1000",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("StatusCode=%d PCStatus=%d body_len=%d", res.StatusCode, res.PCStatus, len(res.Body))

	if res.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", res.StatusCode)
	}
	if res.PCStatus != 200 {
		t.Fatalf("PCStatus = %d, want 200 — got %d (body=%q)",
			res.PCStatus, res.PCStatus, res.Body[:min(300, len(res.Body))])
	}
	if !strings.Contains(strings.ToLower(res.Body), "example domain") {
		t.Errorf("body missing 'example domain' marker; got %q", res.Body[:min(200, len(res.Body))])
	}
}

// TestIntegration_ScraperViaCrawlingAPI exercises the modern way to
// run a scraper — through the Crawling API root with scraper=NAME —
// and verifies the JSON auto-parsing path.
func TestIntegration_ScraperViaCrawlingAPI(t *testing.T) {
	api, err := crawlbase.NewCrawlingAPI(token(t, "CRAWLBASE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	res, err := api.GetWithContext(ctx,
		"https://www.google.com/search?q=crawlbase",
		map[string]string{"scraper": "google-serp"},
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("StatusCode=%d PCStatus=%d body_len=%d json_keys=%v",
		res.StatusCode, res.PCStatus, len(res.Body), keysOf(res.JSON))

	if res.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200; body=%q",
			res.StatusCode, res.Body[:min(300, len(res.Body))])
	}
	if res.JSON == nil {
		t.Fatal("expected JSON to be auto-parsed; got nil")
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
