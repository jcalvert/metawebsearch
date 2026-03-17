// yahoo.go
package metawebsearch

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

// Yahoo is the EngineConfig for Yahoo web search.
// Ported from reference/ddgs/engines/yahoo.py.
var Yahoo = EngineConfig{
	Name:            "yahoo",
	MinDelay:        2 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    yahooBuildRequest,
	ParseResponse:   yahooParseResponse,
	PostProcess:     yahooPostProcess,
}

// yahooRandomToken generates a URL-safe base64 token of the given byte length.
// Mirrors Python's secrets.token_urlsafe(n).
func yahooRandomToken(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)
}

// yahooBuildRequest constructs the HTTP GET request for a Yahoo search.
// The URL embeds random _ylt and _ylu tokens in the path, and uses "p" as the
// query parameter (not "q").
// Ported from Yahoo.build_payload() in the reference implementation.
func yahooBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	// Python: token_urlsafe(24 * 3 // 4) = token_urlsafe(18) -> 18 random bytes -> 24 base64 chars
	// Python: token_urlsafe(47 * 3 // 4) = token_urlsafe(35) -> 35 random bytes -> 47 base64 chars
	ylt := yahooRandomToken(18)
	ylu := yahooRandomToken(35)

	searchURL := fmt.Sprintf("https://search.yahoo.com/search;_ylt=%s;_ylu=%s", ylt, ylu)

	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("p", query)

	// Pagination: 7 results per page, offset (page-1)*7+1
	page := opts.Page
	if page > 1 {
		q.Set("b", fmt.Sprintf("%d", (page-1)*7+1))
	}

	// Time limit
	if opts.TimeLimit != "" {
		q.Set("btf", opts.TimeLimit)
	}

	req.URL.RawQuery = q.Encode()

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return req, nil
}

// yahooParseResponse parses the HTML response from Yahoo search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //div[contains(@class, 'relsrch')]
//   - title: .//div[contains(@class, 'Title')]//h3//text()
//   - href:  .//div[contains(@class, 'Title')]//a/@href
//   - body:  .//div[contains(@class, 'Text')]//text()
func yahooParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	fields := map[string]string{
		"title": ".//div[contains(@class, 'Title')]//h3//text()",
		"href":  ".//div[contains(@class, 'Title')]//a/@href",
		"body":  ".//div[contains(@class, 'Text')]//text()",
	}

	rows, err := XPathExtract(doc, "//div[contains(@class, 'relsrch')]", fields)
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

// yahooPostProcess filters Bing ad clicks, unwraps /RU= redirects, and
// removes results with empty titles or non-HTTP URLs.
// Ported from Yahoo.post_extract_results() in the reference implementation.
func yahooPostProcess(results []Result) []Result {
	var filtered []Result
	for _, r := range results {
		// Filter Bing ad click URLs
		if strings.HasPrefix(r.URL, "https://www.bing.com/aclick?") {
			continue
		}

		// Unwrap /RU=.../RK= redirect URLs
		r.URL = UnwrapRedirect(r.URL, RedirectYahoo)

		// Keep only results with a title and an HTTP(S) URL
		if r.Title != "" && strings.HasPrefix(r.URL, "http") {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
