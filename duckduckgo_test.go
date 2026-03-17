// duckduckgo_test.go
package metawebsearch

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestDuckDuckGoParseResponse(t *testing.T) {
	resetRateLimit("duckduckgo")
	data, err := os.ReadFile("testdata/duckduckgo.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, DuckDuckGo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Should have 2 results: the y.js tracking link should be filtered
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}

func TestDuckDuckGoFiltersTrackingLinks(t *testing.T) {
	resetRateLimit("duckduckgo")
	data, err := os.ReadFile("testdata/duckduckgo.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, DuckDuckGo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	for _, r := range results {
		if strings.Contains(r.URL, "y.js") {
			t.Errorf("tracking link not filtered: %s", r.URL)
		}
	}
}

func TestDuckDuckGoUnwrapsRedirects(t *testing.T) {
	resetRateLimit("duckduckgo")
	data, err := os.ReadFile("testdata/duckduckgo.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	client := newFakeClient(200, string(data))
	results, err := Execute(context.Background(), client, DuckDuckGo, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	found := false
	for _, r := range results {
		if r.URL == "https://real.com" {
			found = true
		}
	}
	if !found {
		t.Error("expected unwrapped URL https://real.com")
	}
}
