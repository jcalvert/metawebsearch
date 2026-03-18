// result.go
package metawebsearch

import (
	"net/http"
	"time"
)

// Result is a single search result from any engine.
type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Engine  string `json:"engine"`
}

// SearchOpts controls a search request.
type SearchOpts struct {
	MaxResults int
	Page       int    // 1-based page number (default: 1)
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

	// ClientProfile overrides the TLS client profile for this engine.
	// If set, Execute creates a dedicated client with this profile.
	// This is needed when the engine's User-Agent requires a matching
	// TLS fingerprint (e.g. Google's GSA UA needs Safari iOS profile).
	ClientProfile string

	MinDelay        time.Duration
	MaxRetries      int
	RetryableStatus func(statusCode int) bool
}
