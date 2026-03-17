// google_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestGoogleParseResponse(t *testing.T) {
	data, err := os.ReadFile("testdata/google.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Google, "test", SearchOpts{MaxResults: 10})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}
	for _, r := range results {
		if r.Title == "" {
			t.Error("empty title")
		}
		if r.URL == "" || r.URL == "javascript:void(0)" {
			t.Errorf("bad URL: %q", r.URL)
		}
		if r.Engine != "google" {
			t.Errorf("Engine = %q, want google", r.Engine)
		}
	}
}

func TestGoogleRedirectUnwrapping(t *testing.T) {
	html := `<html><body>
		<div data-snc="1">
			<div role="link">Title</div>
			<a href="/url?q=https://real.com&amp;sa=U">link</a>
			<div data-sncf="1">Body</div>
		</div>
	</body></html>`

	client := newFakeClient(200, html)
	results, err := Execute(context.Background(), client, Google, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].URL != "https://real.com" {
		t.Errorf("URL = %q, want https://real.com", results[0].URL)
	}
}

func TestGoogleFiltersNonHTTPAndEmptyTitle(t *testing.T) {
	html := `<html><body>
		<div data-snc="1">
			<div role="link">Valid</div>
			<a href="/url?q=https://valid.com&amp;sa=U">link</a>
			<div data-sncf="1">Body</div>
		</div>
		<div data-snc="2">
			<div role="link">No HTTP</div>
			<a href="javascript:void(0)">link</a>
			<div data-sncf="1">Body</div>
		</div>
		<div data-snc="3">
			<div role="link"></div>
			<a href="/url?q=https://notitle.com&amp;sa=U">link</a>
			<div data-sncf="1">Body</div>
		</div>
	</body></html>`

	client := newFakeClient(200, html)
	results, err := Execute(context.Background(), client, Google, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (only the valid one)", len(results))
	}
	if results[0].Title != "Valid" {
		t.Errorf("Title = %q, want Valid", results[0].Title)
	}
}

func TestGoogleBuildRequest(t *testing.T) {
	req, err := Google.BuildRequest("golang tutorial", SearchOpts{
		Region:     "us-en",
		SafeSearch: "moderate",
	})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	if !strings.Contains(req.URL.String(), "google.com/search") {
		t.Errorf("URL = %q, want google.com/search", req.URL.String())
	}

	q := req.URL.Query()
	if q.Get("q") != "golang tutorial" {
		t.Errorf("q = %q, want 'golang tutorial'", q.Get("q"))
	}
	if q.Get("filter") != "1" {
		t.Errorf("filter = %q, want '1' (moderate)", q.Get("filter"))
	}
	if q.Get("hl") != "en-US" {
		t.Errorf("hl = %q, want 'en-US'", q.Get("hl"))
	}
	if q.Get("lr") != "lang_en" {
		t.Errorf("lr = %q, want 'lang_en'", q.Get("lr"))
	}
	if q.Get("cr") != "countryUS" {
		t.Errorf("cr = %q, want 'countryUS'", q.Get("cr"))
	}

	// Check user-agent header is set (GSA pattern)
	ua := req.Header.Get("User-Agent")
	if !strings.Contains(ua, "GSA/") {
		t.Errorf("User-Agent = %q, want GSA user-agent", ua)
	}
}

func TestGoogleBuildRequestSafeSearch(t *testing.T) {
	tests := []struct {
		safesearch string
		want       string
	}{
		{"on", "2"},
		{"moderate", "1"},
		{"off", "0"},
		{"", "1"}, // default to moderate
	}
	for _, tt := range tests {
		req, err := Google.BuildRequest("test", SearchOpts{SafeSearch: tt.safesearch})
		if err != nil {
			t.Fatalf("BuildRequest error for safesearch=%q: %v", tt.safesearch, err)
		}
		got := req.URL.Query().Get("filter")
		if got != tt.want {
			t.Errorf("safesearch=%q: filter = %q, want %q", tt.safesearch, got, tt.want)
		}
	}
}

func TestGoogleUserAgentPool(t *testing.T) {
	// Call getGoogleUA multiple times and ensure all return valid GSA user-agents
	seen := make(map[string]bool)
	for range 20 {
		ua := getGoogleUA()
		if !strings.Contains(ua, "GSA/") {
			t.Errorf("UA missing GSA/: %q", ua)
		}
		if !strings.Contains(ua, "iPhone") {
			t.Errorf("UA missing iPhone: %q", ua)
		}
		seen[ua] = true
	}
	// With 20 draws from ~30 options, we should see at least 2 distinct UAs
	if len(seen) < 2 {
		t.Errorf("expected variety in UAs, got %d distinct", len(seen))
	}
}
