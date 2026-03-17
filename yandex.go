// yandex.go
package metawebsearch

import (
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

// Yandex is the EngineConfig for Yandex web search.
// Ported from reference/ddgs/engines/yandex.py.
var Yandex = EngineConfig{
	Name:            "yandex",
	MinDelay:        2 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    yandexBuildRequest,
	ParseResponse:   yandexParseResponse,
}

// yandexBuildRequest constructs the HTTP GET request for a Yandex search.
// The request targets /search/site/ with a random 7-digit searchid.
// Ignores region, safesearch, and timelimit per the reference implementation.
// Pagination uses a 0-indexed "p" parameter (page 1 -> no param, page 2 -> p=1, etc.).
func yandexBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, "https://yandex.com/search/site/", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("text", query)
	q.Set("web", "1")
	q.Set("searchid", fmt.Sprintf("%d", rand.IntN(9000000)+1000000))

	if opts.Page > 1 {
		q.Set("p", fmt.Sprintf("%d", opts.Page-1))
	}

	req.URL.RawQuery = q.Encode()

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return req, nil
}

// yandexParseResponse parses the HTML response from Yandex search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //li[contains(@class, 'serp-item')]
//   - title: .//h3//text()
//   - href:  .//h3//a/@href
//   - body:  .//div[contains(@class, 'text')]//text()
func yandexParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	fields := map[string]string{
		"title": ".//h3//text()",
		"href":  ".//h3//a/@href",
		"body":  ".//div[contains(@class, 'text')]//text()",
	}

	rows, err := XPathExtract(doc, "//li[contains(@class, 'serp-item')]", fields)
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
