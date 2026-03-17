// mojeek_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestMojeekParseResponse(t *testing.T) {
	resetRateLimit("mojeek")
	data, err := os.ReadFile("testdata/mojeek.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Mojeek, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Verify first result
	if results[0].Title != "Mojeek Result One" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Mojeek Result One")
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://example.com")
	}
	if results[0].Snippet != "First result snippet text." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "First result snippet text.")
	}

	// Verify second result
	if results[1].Title != "Mojeek Result Two" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Mojeek Result Two")
	}
	if results[1].URL != "https://second.com" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://second.com")
	}
	if results[1].Snippet != "Second result snippet." {
		t.Errorf("results[1].Snippet = %q, want %q", results[1].Snippet, "Second result snippet.")
	}

	// Verify engine name is set
	for _, r := range results {
		if r.Engine != "mojeek" {
			t.Errorf("Engine = %q, want %q", r.Engine, "mojeek")
		}
	}
}

func TestMojeekBuildRequest(t *testing.T) {
	req, err := Mojeek.BuildRequest("golang tutorial", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	if !strings.Contains(req.URL.String(), "www.mojeek.com/search") {
		t.Errorf("URL = %q, want www.mojeek.com/search", req.URL.String())
	}

	q := req.URL.Query()
	if q.Get("q") != "golang tutorial" {
		t.Errorf("q = %q, want %q", q.Get("q"), "golang tutorial")
	}

	// Check user-agent header is set
	ua := req.Header.Get("User-Agent")
	if ua == "" {
		t.Error("User-Agent header not set")
	}
}

func TestMojeekBuildRequestWithRegion(t *testing.T) {
	req, err := Mojeek.BuildRequest("test", SearchOpts{
		Region: "de-de",
	})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	// Verify arc cookie is set to uppercase country code
	var foundArc bool
	for _, c := range req.Cookies() {
		if c.Name == "arc" && c.Value == "DE" {
			foundArc = true
		}
	}
	if !foundArc {
		t.Errorf("expected cookie 'arc=DE' for region de-de, cookies: %v", req.Cookies())
	}

	// Verify lb cookie is set to language
	var foundLb bool
	for _, c := range req.Cookies() {
		if c.Name == "lb" && c.Value == "de" {
			foundLb = true
		}
	}
	if !foundLb {
		t.Errorf("expected cookie 'lb=de' for region de-de, cookies: %v", req.Cookies())
	}
}

func TestMojeekBuildRequestSafeSearch(t *testing.T) {
	tests := []struct {
		safesearch     string
		wantCookieVal  string
	}{
		{"off", "0"},
		{"moderate", "1"},
		{"on", "2"},
		{"", "1"}, // default = moderate
	}

	for _, tt := range tests {
		req, err := Mojeek.BuildRequest("test", SearchOpts{SafeSearch: tt.safesearch})
		if err != nil {
			t.Fatalf("BuildRequest error for safesearch=%q: %v", tt.safesearch, err)
		}

		var safesearchVal string
		for _, c := range req.Cookies() {
			if c.Name == "safesearch" {
				safesearchVal = c.Value
			}
		}

		if safesearchVal != tt.wantCookieVal {
			t.Errorf("safesearch=%q: safesearch cookie = %q, want %q", tt.safesearch, safesearchVal, tt.wantCookieVal)
		}
	}
}
