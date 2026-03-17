// yandex_test.go
package metawebsearch

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"testing"
)

func TestYandexParseResponse(t *testing.T) {
	resetRateLimit("yandex")
	data, err := os.ReadFile("testdata/yandex.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, Yandex, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	// Verify first result
	if results[0].Title != "Yandex Result One" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Yandex Result One")
	}
	if results[0].URL != "https://example.com" {
		t.Errorf("results[0].URL = %q, want %q", results[0].URL, "https://example.com")
	}
	if results[0].Snippet != "First result snippet text." {
		t.Errorf("results[0].Snippet = %q, want %q", results[0].Snippet, "First result snippet text.")
	}

	// Verify second result
	if results[1].Title != "Yandex Result Two" {
		t.Errorf("results[1].Title = %q, want %q", results[1].Title, "Yandex Result Two")
	}
	if results[1].URL != "https://second.com" {
		t.Errorf("results[1].URL = %q, want %q", results[1].URL, "https://second.com")
	}
	if results[1].Snippet != "Second result snippet." {
		t.Errorf("results[1].Snippet = %q, want %q", results[1].Snippet, "Second result snippet.")
	}

	// Verify engine name is set
	for _, r := range results {
		if r.Engine != "yandex" {
			t.Errorf("Engine = %q, want %q", r.Engine, "yandex")
		}
	}
}

func TestYandexBuildRequest(t *testing.T) {
	req, err := Yandex.BuildRequest("golang tutorial", SearchOpts{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}

	if req.Method != http.MethodGet {
		t.Errorf("Method = %q, want GET", req.Method)
	}

	// URL should be yandex.com/search/site/
	if req.URL.Host != "yandex.com" {
		t.Errorf("Host = %q, want %q", req.URL.Host, "yandex.com")
	}
	if req.URL.Path != "/search/site/" {
		t.Errorf("Path = %q, want %q", req.URL.Path, "/search/site/")
	}

	q := req.URL.Query()

	// text param
	if q.Get("text") != "golang tutorial" {
		t.Errorf("text = %q, want %q", q.Get("text"), "golang tutorial")
	}

	// web param
	if q.Get("web") != "1" {
		t.Errorf("web = %q, want %q", q.Get("web"), "1")
	}

	// searchid should be exactly 7 digits
	searchid := q.Get("searchid")
	matched, _ := regexp.MatchString(`^\d{7}$`, searchid)
	if !matched {
		t.Errorf("searchid = %q, want 7-digit number", searchid)
	}

	// No "p" param on page 0/1
	if q.Get("p") != "" {
		t.Errorf("page 1: p = %q, want empty", q.Get("p"))
	}
}

func TestYandexBuildRequestPagination(t *testing.T) {
	// Page 1: no "p" param
	req1, err := Yandex.BuildRequest("test", SearchOpts{Page: 1})
	if err != nil {
		t.Fatalf("BuildRequest page 1 error: %v", err)
	}
	if req1.URL.Query().Get("p") != "" {
		t.Errorf("page 1: p = %q, want empty", req1.URL.Query().Get("p"))
	}

	// Page 2: p = 1 (0-indexed)
	req2, err := Yandex.BuildRequest("test", SearchOpts{Page: 2})
	if err != nil {
		t.Fatalf("BuildRequest page 2 error: %v", err)
	}
	if req2.URL.Query().Get("p") != "1" {
		t.Errorf("page 2: p = %q, want %q", req2.URL.Query().Get("p"), "1")
	}

	// Page 3: p = 2
	req3, err := Yandex.BuildRequest("test", SearchOpts{Page: 3})
	if err != nil {
		t.Fatalf("BuildRequest page 3 error: %v", err)
	}
	if req3.URL.Query().Get("p") != "2" {
		t.Errorf("page 3: p = %q, want %q", req3.URL.Query().Get("p"), "2")
	}
}

func TestYandexSearchIDVaries(t *testing.T) {
	// Call BuildRequest multiple times and verify searchid changes (randomness)
	seen := make(map[string]bool)
	for range 10 {
		req, err := Yandex.BuildRequest("test", SearchOpts{})
		if err != nil {
			t.Fatalf("BuildRequest error: %v", err)
		}
		seen[req.URL.Query().Get("searchid")] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected variety in searchid values, got %d distinct", len(seen))
	}
}
