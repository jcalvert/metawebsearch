// mojeek.go
package metawebsearch

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

// Mojeek is the EngineConfig for Mojeek web search.
// Ported from reference/ddgs/engines/mojeek.py.
var Mojeek = EngineConfig{
	Name:            "mojeek",
	MinDelay:        2 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    mojeekBuildRequest,
	ParseResponse:   mojeekParseResponse,
}

// mojeekSafeSearchMap maps SafeSearch option strings to Mojeek's cookie values.
var mojeekSafeSearchMap = map[string]string{
	"off":      "0",
	"moderate": "1",
	"on":       "2",
}

// mojeekBuildRequest constructs the HTTP GET request for a Mojeek search.
// Region and safesearch are set via cookies on www.mojeek.com.
// Ported from Mojeek.build_payload() in the reference implementation.
func mojeekBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.mojeek.com/search", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("q", query)
	req.URL.RawQuery = q.Encode()

	// Region -> cookies: arc = country (uppercase), lb = language
	region := opts.Region
	if region == "" {
		region = "us-en"
	}
	parts := strings.SplitN(strings.ToLower(region), "-", 2)
	country := strings.ToUpper(parts[0])
	lang := parts[0]
	if len(parts) == 2 {
		lang = parts[1]
	}

	req.AddCookie(&http.Cookie{Name: "arc", Value: country})
	req.AddCookie(&http.Cookie{Name: "lb", Value: lang})

	// SafeSearch -> cookie
	safesearch := strings.ToLower(opts.SafeSearch)
	if safesearch == "" {
		safesearch = "moderate"
	}
	if cookieVal, ok := mojeekSafeSearchMap[safesearch]; ok {
		req.AddCookie(&http.Cookie{Name: "safesearch", Value: cookieVal})
	} else {
		req.AddCookie(&http.Cookie{Name: "safesearch", Value: "1"}) // default moderate
	}

	// Set a browser-like user-agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return req, nil
}

// mojeekParseResponse parses the HTML response from Mojeek search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //ul[contains(@class, 'results')]/li
//   - title: .//h2//text()
//   - href:  .//h2/a/@href
//   - body:  .//p[@class='s']//text()
func mojeekParseResponse(resp *http.Response) ([]Result, error) {
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
		"href":  ".//h2/a/@href",
		"body":  ".//p[@class='s']//text()",
	}

	rows, err := XPathExtract(doc, "//ul[contains(@class, 'results')]/li", fields)
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
