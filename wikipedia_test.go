// wikipedia_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestWikipediaParseResponse(t *testing.T) {
	resetRateLimit("wikipedia")
	data, err := os.ReadFile("testdata/wikipedia_opensearch.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Wikipedia, "golang", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	// Disambiguation entry should be filtered out, leaving 2 results
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// First result
	if results[0].Title != "Go (programming language)" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Go (programming language)")
	}
	if results[0].URL != "https://en.wikipedia.org/wiki/Go_(programming_language)" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://en.wikipedia.org/wiki/Go_(programming_language)")
	}
	if results[0].Snippet != "Go is a programming language." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "Go is a programming language.")
	}

	// Second result
	if results[1].Title != "Go (game)" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Go (game)")
	}
	if results[1].URL != "https://en.wikipedia.org/wiki/Go_(game)" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://en.wikipedia.org/wiki/Go_(game)")
	}
	if results[1].Snippet != "Go is a board game." {
		t.Errorf("results[1].Snippet = %q, want %q", results[1].Snippet, "Go is a board game.")
	}

	// Verify engine name is set
	for _, r := range results {
		if r.Engine != "wikipedia" {
			t.Errorf("Engine = %q, want %q", r.Engine, "wikipedia")
		}
	}
}

func TestWikipediaBuildRequest(t *testing.T) {
	req, err := Wikipedia.BuildRequest("golang", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	// Default language should be "en"
	if req.URL.Host != "en.wikipedia.org" {
		t.Errorf("Host = %q, want %q", req.URL.Host, "en.wikipedia.org")
	}
	if req.URL.Path != "/w/api.php" {
		t.Errorf("Path = %q, want %q", req.URL.Path, "/w/api.php")
	}

	q := req.URL.Query()
	if q.Get("action") != "opensearch" {
		t.Errorf("action = %q, want %q", q.Get("action"), "opensearch")
	}
	if q.Get("profile") != "fuzzy" {
		t.Errorf("profile = %q, want %q", q.Get("profile"), "fuzzy")
	}
	if q.Get("search") != "golang" {
		t.Errorf("search = %q, want %q", q.Get("search"), "golang")
	}
}

func TestWikipediaBuildRequestWithRegion(t *testing.T) {
	req, err := Wikipedia.BuildRequest("test", SearchOpts{Region: "de-de"})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	// Language extracted from region "de-de" should be "de"
	if req.URL.Host != "de.wikipedia.org" {
		t.Errorf("Host = %q, want %q", req.URL.Host, "de.wikipedia.org")
	}

	// Also test "us-en" -> "en"
	req2, err := Wikipedia.BuildRequest("test", SearchOpts{Region: "us-en"})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	if req2.URL.Host != "en.wikipedia.org" {
		t.Errorf("Host = %q, want %q", req2.URL.Host, "en.wikipedia.org")
	}

	// Test "fr-fr" -> "fr"
	req3, err := Wikipedia.BuildRequest("test", SearchOpts{Region: "fr-fr"})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	if req3.URL.Host != "fr.wikipedia.org" {
		t.Errorf("Host = %q, want %q", req3.URL.Host, "fr.wikipedia.org")
	}
}

func TestWikipediaMaxResults(t *testing.T) {
	req, err := Wikipedia.BuildRequest("golang", SearchOpts{MaxResults: 5})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	q := req.URL.Query()
	if q.Get("limit") != "5" {
		t.Errorf("limit = %q, want %q", q.Get("limit"), "5")
	}

	// Default (MaxResults=0) should use a reasonable default
	req2, err := Wikipedia.BuildRequest("golang", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	q2 := req2.URL.Query()
	if q2.Get("limit") == "" || q2.Get("limit") == "0" {
		t.Errorf("default limit = %q, want a positive number", q2.Get("limit"))
	}
}
