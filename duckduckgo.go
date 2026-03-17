// duckduckgo.go
package metawebsearch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

// DuckDuckGo is the EngineConfig for DuckDuckGo web search.
// Ported from reference/ddgs/engines/duckduckgo.py.
var DuckDuckGo = EngineConfig{
	Name:            "duckduckgo",
	MinDelay:        2 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    ddgBuildRequest,
	ParseResponse:   ddgParseResponse,
	PostProcess:     ddgPostProcess,
}

// ddgBuildRequest constructs the HTTP POST request for DuckDuckGo HTML search.
// Ported from Duckduckgo.build_payload() in the reference implementation.
func ddgBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	form := url.Values{"q": {query}}.Encode()

	req, err := http.NewRequest(http.MethodPost, "https://html.duckduckgo.com/html/", strings.NewReader(form))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

// ddgParseResponse parses the HTML response from DuckDuckGo search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //div[contains(@class, 'web-result')]//div[contains(@class, 'body')]
//   - title: .//h2//text()
//   - href:  .//h2//a/@href
//   - body:  .//a[contains(@class, 'result__snippet')]//text()
func ddgParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	fields := map[string]string{
		"title": ".//h2//text()",
		"href":  ".//h2//a/@href",
		"body":  ".//a[contains(@class, 'result__snippet')]//text()",
	}

	rows, err := XPathExtract(doc, "//div[contains(@class, 'web-result')]//div[contains(@class, 'body')]", fields)
	if err != nil {
		return nil, fmt.Errorf("xpath extract: %w", err)
	}

	var results []Result
	for _, row := range rows {
		results = append(results, Result{
			Title:   CleanText(row["title"]),
			URL:     CleanText(row["href"]),
			Snippet: CleanText(row["body"]),
		})
	}
	return results, nil
}

// ddgPostProcess filters out DuckDuckGo tracking links (y.js), unwraps DDG
// redirect URLs, and removes results with empty titles or non-HTTP URLs.
// Ported from Duckduckgo.post_extract_results() in the reference implementation.
func ddgPostProcess(results []Result) []Result {
	var filtered []Result
	for _, r := range results {
		// Filter tracking links (y.js)
		if strings.Contains(r.URL, "duckduckgo.com/y.js") {
			continue
		}

		// Unwrap DDG redirect URLs (//duckduckgo.com/l/?uddg=...)
		r.URL = UnwrapRedirect(r.URL, RedirectDDG)

		// Keep only results with a title and an HTTP(S) URL
		if r.Title != "" && strings.HasPrefix(r.URL, "http") {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
