// grokipedia.go
package metawebsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Grokipedia is the EngineConfig for the Grokipedia typeahead API.
// Ported from reference/ddgs/engines/grokipedia.py.
//
// JSON API: GET https://grokipedia.com/api/typeahead?query=<query>&limit=<limit>
// Returns: {"results": [{"title": "...", "snippet": "...", "slug": "..."}]}
var Grokipedia = EngineConfig{
	Name:            "grokipedia",
	MinDelay:        1 * time.Second,
	MaxRetries:      2,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    grokipediaBuildRequest,
	ParseResponse:   grokipediaParseResponse,
}

// grokipediaBuildRequest constructs the HTTP GET request for a Grokipedia typeahead query.
func grokipediaBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	limit := opts.MaxResults
	if limit <= 0 {
		limit = 1
	}

	req, err := http.NewRequest(http.MethodGet, "https://grokipedia.com/api/typeahead", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("query", query)
	q.Set("limit", fmt.Sprintf("%d", limit))
	req.URL.RawQuery = q.Encode()

	return req, nil
}

// grokipediaParseResponse parses the JSON response from Grokipedia.
func grokipediaParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Slug    string `json:"slug"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	var results []Result
	for _, item := range data.Results {
		// Title: strip leading/trailing underscores, replace internal underscores with spaces.
		title := strings.Trim(item.Title, "_")
		title = strings.ReplaceAll(title, "_", " ")
		title = CleanText(title)

		// Snippet: strip everything before first \n\n, keep only what follows.
		snippet := item.Snippet
		if idx := strings.Index(snippet, "\n\n"); idx >= 0 {
			snippet = snippet[idx+2:]
		}
		snippet = CleanText(snippet)

		// URL constructed from slug.
		href := fmt.Sprintf("https://grokipedia.com/page/%s", item.Slug)

		results = append(results, Result{
			Title:   title,
			URL:     href,
			Snippet: snippet,
		})
	}

	return results, nil
}
