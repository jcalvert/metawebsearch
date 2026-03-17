// grokipedia_test.go
package metawebsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestGrokipediaParseResponse(t *testing.T) {
	resetRateLimit("grokipedia")
	data, err := os.ReadFile("testdata/grokipedia.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Grokipedia, "golang", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// First result: title underscores stripped, snippet pre-\n\n stripped, slug-based URL
	if results[0].Title != "Golang" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Golang")
	}
	if results[0].URL != "https://grokipedia.com/page/golang" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://grokipedia.com/page/golang")
	}
	if results[0].Snippet != "Go is a statically typed, compiled programming language designed at Google." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "Go is a statically typed, compiled programming language designed at Google.")
	}

	// Second result
	if results[1].Title != "Go Game" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Go Game")
	}
	if results[1].URL != "https://grokipedia.com/page/go-game" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://grokipedia.com/page/go-game")
	}
	if results[1].Snippet != "Go is an abstract strategy board game." {
		t.Errorf("results[1].Snippet = %q, want %q", results[1].Snippet, "Go is an abstract strategy board game.")
	}

	// Verify engine name
	for _, r := range results {
		if r.Engine != "grokipedia" {
			t.Errorf("Engine = %q, want %q", r.Engine, "grokipedia")
		}
	}
}

func TestGrokipediaBuildRequest(t *testing.T) {
	req, err := Grokipedia.BuildRequest("golang", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	if req.URL.Host != "grokipedia.com" {
		t.Errorf("Host = %q, want %q", req.URL.Host, "grokipedia.com")
	}
	if req.URL.Path != "/api/typeahead" {
		t.Errorf("Path = %q, want %q", req.URL.Path, "/api/typeahead")
	}

	q := req.URL.Query()
	if q.Get("query") != "golang" {
		t.Errorf("query = %q, want %q", q.Get("query"), "golang")
	}
	// Default limit should be 1 (from reference)
	if q.Get("limit") != "1" {
		t.Errorf("limit = %q, want %q", q.Get("limit"), "1")
	}
}

func TestGrokipediaMaxResults(t *testing.T) {
	req, err := Grokipedia.BuildRequest("golang", SearchOpts{MaxResults: 5})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	q := req.URL.Query()
	if q.Get("limit") != "5" {
		t.Errorf("limit = %q, want %q", q.Get("limit"), "5")
	}

	// Default (MaxResults=0) should use 1
	req2, err := Grokipedia.BuildRequest("golang", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	q2 := req2.URL.Query()
	if q2.Get("limit") != "1" {
		t.Errorf("default limit = %q, want %q", q2.Get("limit"), "1")
	}
}

func TestGrokipediaSnippetProcessing(t *testing.T) {
	tests := []struct {
		name    string
		snippet string
		want    string
	}{
		{
			name:    "normal double newline",
			snippet: "Summary info\n\nActual content here.",
			want:    "Actual content here.",
		},
		{
			name:    "no double newline",
			snippet: "Just a single line of text.",
			want:    "Just a single line of text.",
		},
		{
			name:    "empty snippet",
			snippet: "",
			want:    "",
		},
		{
			name:    "multiple double newlines keeps after first",
			snippet: "Intro\n\nMiddle\n\nEnd",
			want:    "Middle End",
		},
		{
			name:    "double newline at start",
			snippet: "\n\nContent after blank start.",
			want:    "Content after blank start.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetRateLimit("grokipedia")
			fixture := `{"results": [{"title": "Test", "snippet": ` + jsonEscape(tt.snippet) + `, "slug": "test"}]}`
			client := newFakeClient(200, fixture)
			results, err := Execute(context.Background(), client, Grokipedia, "test", SearchOpts{})
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			if results[0].Snippet != tt.want {
				t.Errorf("Snippet = %q, want %q", results[0].Snippet, tt.want)
			}
		})
	}
}

// jsonEscape produces a JSON-encoded string (with quotes) for embedding in test fixtures.
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
