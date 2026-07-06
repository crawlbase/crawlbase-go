package crawlbase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewCrawlingAPIRequiresToken verifies the constructor enforces a
// non-empty token.
func TestNewCrawlingAPIRequiresToken(t *testing.T) {
	if _, err := NewCrawlingAPI(""); !errors.Is(err, ErrTokenRequired) {
		t.Fatalf("expected ErrTokenRequired, got %v", err)
	}
}

// TestRequestBuildsCorrectURL verifies token + url + options arrive as
// expected query parameters at the upstream. Uses httptest so we don't
// hit the real Crawlbase API in unit tests.
func TestRequestBuildsCorrectURL(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("pc_status", "200")
		w.Header().Set("original_status", "200")
		w.Header().Set("url", "https://example.com/")
		_, _ = io.WriteString(w, "<html>ok</html>")
	}))
	defer srv.Close()

	api, err := NewCrawlingAPI("test-token")
	if err != nil {
		t.Fatal(err)
	}
	api.endpoint = srv.URL

	res, err := api.Get("https://example.com/", map[string]string{
		"country":   "DE",
		"page_wait": "2000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", res.StatusCode)
	}
	if res.CBStatus != 200 {
		t.Errorf("CBStatus = %d, want 200 (lifted from header)", res.CBStatus)
	}
	if res.PCStatus != res.CBStatus {
		t.Errorf("PCStatus = %d, want %d (mirrors CBStatus)", res.PCStatus, res.CBStatus)
	}
	if res.URL != "https://example.com/" {
		t.Errorf("URL = %q, want example.com", res.URL)
	}
	if !strings.Contains(res.Body, "<html>ok</html>") {
		t.Errorf("Body missing expected content, got %q", res.Body)
	}

	// Verify the upstream got the right query params.
	q := got.URL.Query()
	if q.Get("token") != "test-token" {
		t.Errorf("token query = %q, want test-token", q.Get("token"))
	}
	if q.Get("url") != "https://example.com/" {
		t.Errorf("url query = %q, want example.com", q.Get("url"))
	}
	if q.Get("country") != "DE" {
		t.Errorf("country = %q, want DE", q.Get("country"))
	}
	if q.Get("page_wait") != "2000" {
		t.Errorf("page_wait = %q, want 2000", q.Get("page_wait"))
	}
}

// TestScraperViaCrawlingAPI confirms a scraper=NAME call against the
// modern Crawling API endpoint round-trips JSON correctly. The legacy
// standalone /scraper endpoint is closed to new sign-ups; modern
// scraping rides on the Crawling API root with the scraper option.
func TestScraperViaCrawlingAPI(t *testing.T) {
	var q map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"test","price":"$10"}`)
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	res, err := api.Get("https://www.amazon.com/dp/X", map[string]string{
		"scraper": "amazon-product-details",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := q["scraper"]; len(got) != 1 || got[0] != "amazon-product-details" {
		t.Errorf("scraper query = %v, want [amazon-product-details]", got)
	}
	if res.JSON == nil {
		t.Fatal("expected JSON to be auto-parsed")
	}
	if res.JSON["name"] != "test" {
		t.Errorf("JSON.name = %v, want test", res.JSON["name"])
	}
}

// TestPostFormEncoded verifies the POST body shape — url.Values gets
// form-encoded and Content-Type defaults to application/x-www-form-urlencoded.
func TestPostFormEncoded(t *testing.T) {
	var bodyGot string
	var ctGot string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctGot = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		bodyGot = string(b)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	_, err := api.Post("https://producthunt.com/search", map[string]string{
		"text": "example",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ctGot != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ctGot)
	}
	if !strings.Contains(bodyGot, "text=example") {
		t.Errorf("body = %q, missing text=example", bodyGot)
	}
}

// TestContextCancellation verifies a cancelled context aborts the
// request before a long-running response can come back.
func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block for a second — context should cancel before this finishes.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			_, _ = io.WriteString(w, "too late")
		}
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, err := api.GetWithContext(ctx, "https://example.com/", nil); err == nil {
		t.Fatal("expected context-cancellation error, got nil")
	}
}

// TestCBStatusFromCBHeader verifies cb_status is read into CBStatus.
func TestCBStatusFromCBHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cb_status", "525")
		_, _ = io.WriteString(w, "")
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	res, err := api.Get("https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.CBStatus != 525 {
		t.Errorf("CBStatus = %d, want 525 (read from cb_status header)", res.CBStatus)
	}
	if res.PCStatus != 525 {
		t.Errorf("PCStatus = %d, want 525 (mirrors CBStatus)", res.PCStatus)
	}
}

// TestCBStatusFallbackToPCStatus verifies legacy pc_status is accepted
// when cb_status is absent.
func TestCBStatusFallbackToPCStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("pc_status", "200")
		_, _ = io.WriteString(w, "<html>ok</html>")
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	res, err := api.Get("https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.CBStatus != 200 {
		t.Errorf("CBStatus = %d, want 200 (read from pc_status header)", res.CBStatus)
	}
	if res.PCStatus != 200 {
		t.Errorf("PCStatus = %d, want 200 (mirrors CBStatus)", res.PCStatus)
	}
}

// TestCBStatusPrefersCBOverPC verifies cb_status wins when both headers
// are present.
func TestCBStatusPrefersCBOverPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cb_status", "525")
		w.Header().Set("pc_status", "200")
		_, _ = io.WriteString(w, "")
	}))
	defer srv.Close()

	api, _ := NewCrawlingAPI("t")
	api.endpoint = srv.URL

	res, err := api.Get("https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.CBStatus != 525 {
		t.Errorf("CBStatus = %d, want 525 (cb_status preferred)", res.CBStatus)
	}
	if res.PCStatus != 525 {
		t.Errorf("PCStatus = %d, want 525 (mirrors CBStatus)", res.PCStatus)
	}
}
