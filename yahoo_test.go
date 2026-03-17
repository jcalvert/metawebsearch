// yahoo_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestYahooParseResponse(t *testing.T) {
	resetRateLimit("yahoo")
	data, err := os.ReadFile("testdata/yahoo.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Yahoo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2 (Bing ad should be filtered)", len(results))
	}

	// Verify first result
	if results[0].Title != "Yahoo Result One" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Yahoo Result One")
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://example.com")
	}
	if results[0].Snippet != "First snippet text here." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "First snippet text here.")
	}

	// Verify second result
	if results[1].Title != "Yahoo Result Two" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Yahoo Result Two")
	}
	if results[1].URL != "https://second.com" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://second.com")
	}
	if results[1].Snippet != "Second snippet." {
		t.Errorf("results[1].Snippet = %q, want %q", results[1].Snippet, "Second snippet.")
	}

	// Verify engine name is set
	for _, r := range results {
		if r.Engine != "yahoo" {
			t.Errorf("Engine = %q, want %q", r.Engine, "yahoo")
		}
	}
}

func TestYahooUnwrapsRedirects(t *testing.T) {
	resetRateLimit("yahoo")
	html := `<html><body>
		<div class="relsrch">
			<div class="Title"><h3><a href="https://r.search.yahoo.com/cb/RU=https%3A%2F%2Funwrapped.example.com%2Fpath/RK=2/RS=xyz">Unwrapped</a></h3></div>
			<div class="Text">Body text.</div>
		</div>
	</body></html>`

	client := newFakeClient(200, html)
	results, err := Execute(context.Background(), client, Yahoo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].URL != "https://unwrapped.example.com/path" {
		t.Errorf("URL = %q, want %q", results[0].URL, "https://unwrapped.example.com/path")
	}
}

func TestYahooFiltersBingAds(t *testing.T) {
	resetRateLimit("yahoo")
	html := `<html><body>
		<div class="relsrch">
			<div class="Title"><h3><a href="https://valid.com">Valid Result</a></h3></div>
			<div class="Text">Valid body.</div>
		</div>
		<div class="relsrch">
			<div class="Title"><h3><a href="https://www.bing.com/aclick?ld=abc123&u=something">Bing Ad</a></h3></div>
			<div class="Text">Ad body.</div>
		</div>
		<div class="relsrch">
			<div class="Title"><h3><a href="https://www.bing.com/aclick?ld=xyz789">Another Ad</a></h3></div>
			<div class="Text">Another ad body.</div>
		</div>
	</body></html>`

	client := newFakeClient(200, html)
	results, err := Execute(context.Background(), client, Yahoo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1 (Bing ads filtered)", len(results))
	}
	if results[0].Title != "Valid Result" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Valid Result")
	}
}

func TestYahooBuildRequest(t *testing.T) {
	req, err := Yahoo.BuildRequest("golang tutorial", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	// URL should contain search.yahoo.com/search
	if !strings.Contains(req.URL.String(), "search.yahoo.com/search") {
		t.Errorf("URL = %q, want search.yahoo.com/search", req.URL.String())
	}

	// URL path should contain random _ylt and _ylu tokens
	if !strings.Contains(req.URL.Path, "_ylt=") {
		t.Errorf("URL path = %q, want _ylt= token", req.URL.Path)
	}
	if !strings.Contains(req.URL.Path, "_ylu=") {
		t.Errorf("URL path = %q, want _ylu= token", req.URL.Path)
	}

	// Query param should be "p", not "q"
	q := req.URL.Query()
	if q.Get("p") != "golang tutorial" {
		t.Errorf("p = %q, want %q", q.Get("p"), "golang tutorial")
	}

	// Verify "q" is not set
	if q.Get("q") != "" {
		t.Errorf("unexpected q param: %q", q.Get("q"))
	}
}

func TestYahooBuildRequestPagination(t *testing.T) {
	// Page 1: no "b" param
	req1, err := Yahoo.BuildRequest("test", SearchOpts{Page: 1})
	if err != nil {
		t.Fatalf("BuildRequest page 1 error: %v", err)
	}
	if req1.URL.Query().Get("b") != "" {
		t.Errorf("page 1: b = %q, want empty", req1.URL.Query().Get("b"))
	}

	// Page 2: b = (2-1)*7+1 = 8
	req2, err := Yahoo.BuildRequest("test", SearchOpts{Page: 2})
	if err != nil {
		t.Fatalf("BuildRequest page 2 error: %v", err)
	}
	if req2.URL.Query().Get("b") != "8" {
		t.Errorf("page 2: b = %q, want %q", req2.URL.Query().Get("b"), "8")
	}

	// Page 3: b = (3-1)*7+1 = 15
	req3, err := Yahoo.BuildRequest("test", SearchOpts{Page: 3})
	if err != nil {
		t.Fatalf("BuildRequest page 3 error: %v", err)
	}
	if req3.URL.Query().Get("b") != "15" {
		t.Errorf("page 3: b = %q, want %q", req3.URL.Query().Get("b"), "15")
	}
}

func TestYahooBuildRequestTimeLimit(t *testing.T) {
	req, err := Yahoo.BuildRequest("test", SearchOpts{TimeLimit: "d"})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	if req.URL.Query().Get("btf") != "d" {
		t.Errorf("btf = %q, want %q", req.URL.Query().Get("btf"), "d")
	}
}

func TestYahooRandomTokensVary(t *testing.T) {
	// Call BuildRequest multiple times and verify tokens change
	seen := make(map[string]bool)
	for range 10 {
		req, err := Yahoo.BuildRequest("test", SearchOpts{})
		if err != nil {
			t.Fatalf("BuildRequest error: %v", err)
		}
		seen[req.URL.Path] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected variety in URL paths (random tokens), got %d distinct", len(seen))
	}
}
