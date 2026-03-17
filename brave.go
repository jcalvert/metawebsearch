// brave.go
package metawebsearch

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

// Brave is the EngineConfig for Brave web search.
// Ported from reference/ddgs/engines/brave.py.
var Brave = EngineConfig{
	Name:            "brave",
	MinDelay:        2 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    braveBuildRequest,
	ParseResponse:   braveParseResponse,
}

// braveTimeLimitMap maps SearchOpts.TimeLimit values to Brave's "tf" query param.
var braveTimeLimitMap = map[string]string{
	"d": "pd",
	"w": "pw",
	"m": "pm",
	"y": "py",
}

// braveBuildRequest constructs the HTTP GET request for a Brave search.
// Region and safesearch are set via cookies on search.brave.com.
// Ported from Brave.build_payload() in the reference implementation.
func braveBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, "https://search.brave.com/search", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("q", query)
	q.Set("source", "web")

	// Time limit mapping
	if opts.TimeLimit != "" {
		if tf, ok := braveTimeLimitMap[opts.TimeLimit]; ok {
			q.Set("tf", tf)
		}
	}

	req.URL.RawQuery = q.Encode()

	// Region -> cookies (country code + useLocation)
	region := opts.Region
	if region == "" {
		region = "us-en"
	}
	parts := strings.SplitN(strings.ToLower(region), "-", 2)
	country := parts[0]

	req.AddCookie(&http.Cookie{Name: "country", Value: country})
	req.AddCookie(&http.Cookie{Name: "useLocation", Value: "0"})

	// SafeSearch -> cookie (only set if not moderate/default)
	safesearch := strings.ToLower(opts.SafeSearch)
	if safesearch == "" {
		safesearch = "moderate"
	}
	if safesearch != "moderate" {
		cookieVal := "off"
		if safesearch == "on" {
			cookieVal = "strict"
		}
		req.AddCookie(&http.Cookie{Name: "safesearch", Value: cookieVal})
	}

	// Set a browser-like user-agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	return req, nil
}

// braveParseResponse parses the HTML response from Brave search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //div[@data-type='web']
//   - title: .//div[(contains(@class,'title') or contains(@class,'sitename-container')) and position()=last()]//text()
//   - href:  .//a[div[contains(@class, 'title')]]/@href
//   - body:  .//div[contains(@class, 'snippet')]//div[contains(@class, 'content')]//text()
func braveParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	fields := map[string]string{
		"title": ".//div[(contains(@class,'title') or contains(@class,'sitename-container')) and position()=last()]//text()",
		"href":  ".//a[div[contains(@class, 'title')]]/@href",
		"body":  ".//div[contains(@class, 'snippet')]//div[contains(@class, 'content')]//text()",
	}

	rows, err := XPathExtract(doc, "//div[@data-type='web']", fields)
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
