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
// path against a stable target. Assert on CBStatus (the billed verdict)
// rather than StatusCode alone — the two can diverge when the target
// returns a non-2xx original_status but Crawlbase still delivers body.
func TestIntegration_CrawlingGetStatic(t *testing.T) {
	api, err := crawlbase.NewCrawlingAPI(token(t, "CRAWLBASE_TOKEN"))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	res, err := api.GetWithContext(ctx, "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("StatusCode=%d CBStatus=%d OriginalStatus=%d body_len=%d",
		res.StatusCode, res.CBStatus, res.OriginalStatus, len(res.Body))

	if res.CBStatus != 200 {
		t.Fatalf("CBStatus = %d, want 200", res.CBStatus)
	}
	if len(res.Body) == 0 {
		t.Fatal("expected non-empty body")
	}
	if !strings.Contains(strings.ToLower(res.Body), "example domain") {
		t.Errorf("body missing expected 'example domain' marker; got %q", res.Body[:min(200, len(res.Body))])
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

	t.Logf("StatusCode=%d CBStatus=%d body_len=%d", res.StatusCode, res.CBStatus, len(res.Body))

	if res.StatusCode != 200 {
		t.Fatalf("StatusCode = %d, want 200", res.StatusCode)
	}
	if res.CBStatus != 200 {
		t.Fatalf("CBStatus = %d, want 200 — got %d (body=%q)",
			res.CBStatus, res.CBStatus, res.Body[:min(300, len(res.Body))])
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

	t.Logf("StatusCode=%d CBStatus=%d body_len=%d json_keys=%v",
		res.StatusCode, res.CBStatus, len(res.Body), keysOf(res.JSON))

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
