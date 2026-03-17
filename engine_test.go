// engine_test.go
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

// fakeHTTPClient returns canned responses for testing.
type fakeHTTPClient struct {
	handler func(req *http.Request) (*http.Response, error)
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f.handler(req)
}

func newFakeClient(statusCode int, body string) *fakeHTTPClient {
	return &fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		},
	}
}

func TestExecuteCallsPipeline(t *testing.T) {
	var buildCalled, parseCalled bool

	engine := EngineConfig{
		Name: "test",
		BuildRequest: func(query string, opts SearchOpts) (*http.Request, error) {
			buildCalled = true
			return http.NewRequest("GET", "https://example.com/search?q="+query, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			parseCalled = true
			return []Result{{Title: "Test", URL: "https://test.com", Engine: "test"}}, nil
		},
	}

	client := newFakeClient(200, "<html></html>")
	results, err := Execute(context.Background(), client, engine, "golang", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !buildCalled {
		t.Error("BuildRequest not called")
	}
	if !parseCalled {
		t.Error("ParseResponse not called")
	}
	if len(results) != 1 || results[0].Title != "Test" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestExecutePostProcess(t *testing.T) {
	engine := EngineConfig{
		Name: "test",
		BuildRequest: func(query string, opts SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com/search?q="+query, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{
				{Title: "Keep", URL: "https://keep.com", Engine: "test"},
				{Title: "Drop", URL: "https://ads.com", Engine: "test"},
			}, nil
		},
		PostProcess: func(results []Result) []Result {
			var filtered []Result
			for _, r := range results {
				if r.Title != "Drop" {
					filtered = append(filtered, r)
				}
			}
			return filtered
		},
	}

	client := newFakeClient(200, "")
	results, err := Execute(context.Background(), client, engine, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Keep" {
		t.Errorf("PostProcess not applied: %v", results)
	}
}

func TestExecuteRetriesOnRetryableStatus(t *testing.T) {
	attempts := 0
	client := &fakeHTTPClient{
		handler: func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
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

	engine := EngineConfig{
		Name: "test",
		BuildRequest: func(query string, opts SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+query, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return []Result{{Title: "OK", Engine: "test"}}, nil
		},
		MaxRetries:      3,
		MinDelay:        time.Millisecond, // fast for tests
		RetryableStatus: func(code int) bool { return code == 429 },
	}

	results, err := Execute(context.Background(), client, engine, "test", SearchOpts{})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result after retry, got %d", len(results))
	}
}

func TestExecuteFailsAfterMaxRetries(t *testing.T) {
	client := newFakeClient(429, "")

	engine := EngineConfig{
		Name: "test",
		BuildRequest: func(query string, opts SearchOpts) (*http.Request, error) {
			return http.NewRequest("GET", "https://example.com?q="+query, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return nil, nil
		},
		MaxRetries:      2,
		MinDelay:        time.Millisecond,
		RetryableStatus: func(code int) bool { return code == 429 },
	}

	_, err := Execute(context.Background(), client, engine, "test", SearchOpts{})
	if err == nil {
		t.Fatal("expected error after max retries")
	}
}

func TestExecuteRespectsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	engine := EngineConfig{
		Name: "test",
		BuildRequest: func(query string, opts SearchOpts) (*http.Request, error) {
			return http.NewRequestWithContext(ctx, "GET", "https://example.com?q="+query, nil)
		},
		ParseResponse: func(resp *http.Response) ([]Result, error) {
			return nil, nil
		},
		MaxRetries:      3,
		MinDelay:        time.Millisecond,
		RetryableStatus: func(code int) bool { return code == 429 },
	}

	client := newFakeClient(429, "")
	_, err := Execute(ctx, client, engine, "test", SearchOpts{})
	if err == nil {
		t.Fatal("expected context error")
	}
}

// Ensure the fmt import is used (needed by later engine tests in this package).
var _ = fmt.Sprintf
