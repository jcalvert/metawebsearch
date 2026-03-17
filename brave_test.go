// brave_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestBraveParseResponse(t *testing.T) {
	resetRateLimit("brave")
	data, err := os.ReadFile("testdata/brave.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Brave, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Verify first result
	if results[0].Title != "Brave Result One" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Brave Result One")
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://example.com")
	}
	if results[0].Snippet != "First result snippet text." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "First result snippet text.")
	}

	// Verify second result
	if results[1].Title != "Brave Result Two" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Brave Result Two")
	}
	if results[1].URL != "https://second.com" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://second.com")
	}

	// Verify engine name is set
	for _, r := range results {
		if r.Engine != "brave" {
			t.Errorf("Engine = %q, want %q", r.Engine, "brave")
		}
	}
}

func TestBraveBuildRequest(t *testing.T) {
	req, err := Brave.BuildRequest("golang tutorial", SearchOpts{
		Region:     "us-en",
		SafeSearch: "moderate",
	})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	if !strings.Contains(req.URL.String(), "search.brave.com/search") {
		t.Errorf("URL = %q, want search.brave.com/search", req.URL.String())
	}

	q := req.URL.Query()
	if q.Get("q") != "golang tutorial" {
		t.Errorf("q = %q, want %q", q.Get("q"), "golang tutorial")
	}
	if q.Get("source") != "web" {
		t.Errorf("source = %q, want %q", q.Get("source"), "web")
	}

	// Check user-agent header is set
	ua := req.Header.Get("User-Agent")
	if ua == "" {
		t.Error("User-Agent header not set")
	}
}

func TestBraveBuildRequestWithRegion(t *testing.T) {
	req, err := Brave.BuildRequest("test", SearchOpts{
		Region: "de-de",
	})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	// Verify cookie is set for the region country code
	var foundCountry bool
	for _, c := range req.Cookies() {
		if c.Name == "country" && c.Value == "de" {
			foundCountry = true
		}
	}
	if !foundCountry {
		t.Error("expected cookie 'country=de' for region de-de")
	}

	// Verify useLocation cookie
	var foundUseLocation bool
	for _, c := range req.Cookies() {
		if c.Name == "useLocation" && c.Value == "0" {
			foundUseLocation = true
		}
	}
	if !foundUseLocation {
		t.Error("expected cookie 'useLocation=0'")
	}
}

func TestBraveBuildRequestSafeSearch(t *testing.T) {
	tests := []struct {
		safesearch string
		wantCookie string // expected safesearch cookie value, empty means no cookie
	}{
		{"on", "strict"},
		{"off", "off"},
		{"moderate", ""},  // moderate = no safesearch cookie (default)
		{"", ""},          // empty = moderate = no safesearch cookie
	}

	for _, tt := range tests {
		req, err := Brave.BuildRequest("test", SearchOpts{SafeSearch: tt.safesearch})
		if err != nil {
			t.Fatalf("BuildRequest error for safesearch=%q: %v", tt.safesearch, err)
		}

		var safesearchVal string
		for _, c := range req.Cookies() {
			if c.Name == "safesearch" {
				safesearchVal = c.Value
			}
		}

		if tt.wantCookie == "" && safesearchVal != "" {
			t.Errorf("safesearch=%q: got safesearch cookie %q, want none", tt.safesearch, safesearchVal)
		}
		if tt.wantCookie != "" && safesearchVal != tt.wantCookie {
			t.Errorf("safesearch=%q: safesearch cookie = %q, want %q", tt.safesearch, safesearchVal, tt.wantCookie)
		}
	}
}

func TestBraveBuildRequestTimeLimit(t *testing.T) {
	tests := []struct {
		timeLimit string
		wantTF    string
	}{
		{"d", "pd"},
		{"w", "pw"},
		{"m", "pm"},
		{"y", "py"},
		{"", ""},
	}

	for _, tt := range tests {
		req, err := Brave.BuildRequest("test", SearchOpts{TimeLimit: tt.timeLimit})
		if err != nil {
			t.Fatalf("BuildRequest error for timeLimit=%q: %v", tt.timeLimit, err)
		}

		got := req.URL.Query().Get("tf")
		if got != tt.wantTF {
			t.Errorf("timeLimit=%q: tf = %q, want %q", tt.timeLimit, got, tt.wantTF)
		}
	}
}
