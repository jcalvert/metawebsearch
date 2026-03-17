// result_test.go
package metawebsearch

import (
	"fmt"
	"testing"
)

func TestResultFields(t *testing.T) {
	r := Result{
		Title:   "Example",
		URL:     "https://example.com",
		Snippet: "A snippet",
		Engine:  "google",
	}
	if r.Title != "Example" {
		t.Errorf("Title = %q, want %q", r.Title, "Example")
	}
	if r.Engine != "google" {
		t.Errorf("Engine = %q, want %q", r.Engine, "google")
	}
}

func TestSearchOptsDefaults(t *testing.T) {
	opts := SearchOpts{}
	if opts.MaxResults != 0 {
		t.Errorf("MaxResults = %d, want 0", opts.MaxResults)
	}
	if opts.Region != "" {
		t.Errorf("Region = %q, want empty", opts.Region)
	}
}

func TestSearchResultPartialErrors(t *testing.T) {
	sr := &SearchResult{
		Results: []Result{{Title: "A", URL: "https://a.com", Engine: "google"}},
		Errors:  map[string]error{"brave": fmt.Errorf("timeout")},
	}
	if len(sr.Results) != 1 {
		t.Errorf("Results len = %d, want 1", len(sr.Results))
	}
	if sr.Errors["brave"] == nil {
		t.Error("expected error for brave")
	}
}
