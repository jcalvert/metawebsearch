// multi_test.go
package metawebsearch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMultiSearchConcurrentDispatch(t *testing.T) {
	engine1 := EngineConfig{
		Name: "engine1",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 1", URL: "https://a.com"}}, nil
		},
	}
	engine2 := EngineConfig{
		Name: "engine2",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 2", URL: "https://b.com"}}, nil
		},
	}

	ms := MultiSearch{
		Client:  newFakeClient(200, ""),
		Engines: []EngineConfig{engine1, engine2},
	}
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(sr.Results) != 2 {
		t.Errorf("got %d results, want 2", len(sr.Results))
	}
}

func TestMultiSearchDeduplicatesByURL(t *testing.T) {
	engine1 := EngineConfig{
		Name: "engine1",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 1", URL: "https://same.com"}}, nil
		},
	}
	engine2 := EngineConfig{
		Name: "engine2",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 2", URL: "https://same.com"}}, nil
		},
	}

	ms := MultiSearch{
		Client:  newFakeClient(200, ""),
		Engines: []EngineConfig{engine1, engine2},
	}
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(sr.Results) != 1 {
		t.Errorf("got %d results, want 1 (dedup)", len(sr.Results))
	}
	// First engine wins
	if sr.Results[0].Title != "From 1" {
		t.Errorf("Title = %q, want 'From 1' (first engine wins)", sr.Results[0].Title)
	}
}

func TestMultiSearchDeduplicatesNormalized(t *testing.T) {
	engine1 := EngineConfig{
		Name: "engine1",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 1", URL: "https://example.com/page/"}}, nil
		},
	}
	engine2 := EngineConfig{
		Name: "engine2",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "From 2", URL: "https://Example.COM/page#section"}}, nil
		},
	}

	ms := MultiSearch{
		Client:  newFakeClient(200, ""),
		Engines: []EngineConfig{engine1, engine2},
	}
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(sr.Results) != 1 {
		t.Errorf("got %d results, want 1 (normalized dedup: trailing slash, case, fragment)", len(sr.Results))
	}
}

func TestMultiSearchPartialFailure(t *testing.T) {
	engine1 := EngineConfig{
		Name: "good",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "OK", URL: "https://ok.com"}}, nil
		},
	}
	engine2 := EngineConfig{
		Name: "bad",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return nil, fmt.Errorf("engine broken")
		},
	}

	ms := MultiSearch{
		Client:  newFakeClient(200, ""),
		Engines: []EngineConfig{engine1, engine2},
	}
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Search error: %v (should succeed with partial results)", err)
	}
	if len(sr.Results) != 1 {
		t.Errorf("got %d results, want 1", len(sr.Results))
	}
	if sr.Errors["bad"] == nil {
		t.Error("expected error for 'bad' engine")
	}
	if sr.Errors["good"] != nil {
		t.Error("unexpected error for 'good' engine")
	}
}

func TestMultiSearchEngineTimeout(t *testing.T) {
	// A fast engine returns 200 immediately. A slow engine always returns 429,
	// which would normally trigger retries for many seconds. The per-engine
	// timeout should cancel the slow engine so it doesn't block the result.
	fast := EngineConfig{
		Name: "fast",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com/fast?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "Fast", URL: "https://fast.com"}}, nil
		},
	}
	slow := EngineConfig{
		Name:            "slow",
		MinDelay:        time.Millisecond,
		MaxRetries:      10, // would take very long without timeout
		RetryableStatus: func(code int) bool { return code == 429 },
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com/slow?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return nil, nil
		},
	}

	client := &fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/slow" {
				return &http.Response{
					StatusCode: 429,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		},
	}

	ms := MultiSearch{
		Client:        client,
		Engines:       []EngineConfig{fast, slow},
		EngineTimeout: 200 * time.Millisecond,
	}

	start := time.Now()
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(sr.Results) != 1 || sr.Results[0].Title != "Fast" {
		t.Errorf("got results %v, want [Fast]", sr.Results)
	}
	if sr.Errors["slow"] == nil {
		t.Error("expected timeout error for slow engine")
	}
	if elapsed > 2*time.Second {
		t.Errorf("Search took %v, expected < 2s (per-engine timeout should cap it)", elapsed)
	}
}

func TestMultiSearchAllFail(t *testing.T) {
	engine := EngineConfig{
		Name: "broken",
		BuildRequest: func(q string, o SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+q, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return nil, fmt.Errorf("broken")
		},
	}

	ms := MultiSearch{
		Client:  newFakeClient(200, ""),
		Engines: []EngineConfig{engine},
	}
	sr, err := ms.Search(context.Background(), "test", SearchOpts{})
	if err == nil {
		t.Fatal("expected error when all engines fail")
	}
	if len(sr.Results) != 0 {
		t.Errorf("got %d results, want 0", len(sr.Results))
	}
	if sr.Errors["broken"] == nil {
		t.Error("expected error for 'broken' engine")
	}
}
