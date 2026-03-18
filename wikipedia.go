// wikipedia.go
package metawebsearch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Wikipedia is the EngineConfig for Wikipedia OpenSearch API.
// Ported from reference/ddgs/engines/wikipedia.py.
//
// Unlike other engines, Wikipedia returns JSON (OpenSearch format), not HTML.
// The response is a JSON array: ["query", ["titles..."], ["descriptions..."], ["urls..."]]
var Wikipedia = EngineConfig{
	Name:            "wikipedia",
	MinDelay:        1 * time.Second,
	MaxRetries:      2,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    wikipediaBuildRequest,
	ParseResponse:   wikipediaParseResponse,
}

// wikipediaBuildRequest constructs the HTTP GET request for a Wikipedia OpenSearch query.
// Region format is "country-lang" (e.g. "us-en"); the lang part determines the subdomain.
// If no region is provided, defaults to "en".
func wikipediaBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	lang := "en"
	if opts.Region != "" {
		parts := strings.SplitN(strings.ToLower(opts.Region), "-", 2)
		if len(parts) == 2 {
			lang = parts[1]
		}
	}

	limit := opts.MaxResults
	if limit <= 0 {
		limit = 10
	}

	baseURL := fmt.Sprintf("https://%s.wikipedia.org/w/api.php", lang)
	req, err := http.NewRequest(http.MethodGet, baseURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("action", "opensearch")
	q.Set("profile", "fuzzy")
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("search", query)
	req.URL.RawQuery = q.Encode()

	// Wikipedia API requires a descriptive User-Agent per their policy:
	// https://meta.wikimedia.org/wiki/User-Agent_policy
	req.Header.Set("User-Agent", "metawebsearch/0.1 (https://github.com/jcalvert/metawebsearch)")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// wikipediaParseResponse parses the OpenSearch JSON response from Wikipedia.
// Filters out disambiguation pages (titles containing "disambiguation").
func wikipediaParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	// OpenSearch response: ["query", ["titles..."], ["descs..."], ["urls..."]]
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	if len(raw) < 4 {
		return nil, fmt.Errorf("unexpected JSON structure: got %d elements, want 4", len(raw))
	}

	var titles []string
	if err := json.Unmarshal(raw[1], &titles); err != nil {
		return nil, fmt.Errorf("parse titles: %w", err)
	}

	var descriptions []string
	if err := json.Unmarshal(raw[2], &descriptions); err != nil {
		return nil, fmt.Errorf("parse descriptions: %w", err)
	}

	var urls []string
	if err := json.Unmarshal(raw[3], &urls); err != nil {
		return nil, fmt.Errorf("parse urls: %w", err)
	}

	var results []Result
	for i := range titles {
		// Filter disambiguation pages
		if strings.Contains(strings.ToLower(titles[i]), "disambiguation") {
			continue
		}

		snippet := ""
		if i < len(descriptions) {
			snippet = CleanText(descriptions[i])
		}

		href := ""
		if i < len(urls) {
			href = urls[i]
		}

		results = append(results, Result{
			Title:   CleanText(titles[i]),
			URL:     href,
			Snippet: snippet,
		})
	}

	return results, nil
}
