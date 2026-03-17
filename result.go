// result.go
package metawebsearch

import (
	"net/http"
	"time"
)

// Result is a single search result from any engine.
type Result struct {
	Title   string
	URL     string
	Snippet string
	Engine  string
}

// SearchOpts controls a search request.
type SearchOpts struct {
	MaxResults int
	Region     string // e.g. "us-en"
	SafeSearch string // "on", "moderate", "off"
	TimeLimit  string // "d" (day), "w" (week), "m" (month), "y" (year)
}

// SearchResult is what MultiSearch returns.
type SearchResult struct {
	Results []Result
	Errors  map[string]error
}

// HTTPClient is the interface the pipeline calls. Tests substitute a fake.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// EngineConfig defines a search engine's scraping pipeline.
type EngineConfig struct {
	Name          string
	BuildRequest  func(query string, opts SearchOpts) (*http.Request, error)
	ParseResponse func(resp *http.Response) ([]Result, error)
	PostProcess   func(results []Result) []Result

	MinDelay        time.Duration
	MaxRetries      int
	RetryableStatus func(statusCode int) bool
}
