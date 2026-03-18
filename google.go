// google.go
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

// osGSAMap maps iOS version strings to GSA (Google Search App) version strings.
// Ported from reference/ddgs/engines/google.py get_ua().
var osGSAMap = map[string][]string{
	"17_4":   {"315.0.630091404", "317.0.634488990"},
	"17_6_1": {"411.0.879111500"},
	"18_1_1": {"411.0.879111500"},
	"18_2":   {"173.0.391310503"},
	"18_6_2": {"397.0.836500703", "399.2.845414227", "410.0.875971614", "411.0.879111500"},
	"18_7_2": {"411.0.879111500"},
	"18_7_5": {"411.0.879111500"},
	"18_7_6": {"411.0.879111500"},
	"26_1_0": {"411.0.879111500"},
	"26_2_0": {"396.0.833910942", "409.0.872648028", "411.0.879111500"},
	"26_2_1": {"409.0.872648028", "411.0.879111500"},
	"26_3_0": {"406.0.862495628", "410.0.875971614", "411.0.879111500"},
	"26_3_1": {"370.0.762543316", "404.0.856692123", "408.0.868297084", "410.0.875971614", "411.0.879111500"},
	"26_4_0": {"411.0.879111500"},
}

// osVersions is a pre-computed slice of the map keys for random selection.
var osVersions []string

func init() {
	osVersions = make([]string, 0, len(osGSAMap))
	for k := range osGSAMap {
		osVersions = append(osVersions, k)
	}
}

// getGoogleUA returns a random GSA (Google Search App) user-agent string.
// Ported from reference/ddgs/engines/google.py get_ua().
func getGoogleUA() string {
	osVersion := osVersions[rand.IntN(len(osVersions))]
	gsaVersions := osGSAMap[osVersion]
	gsaVersion := gsaVersions[rand.IntN(len(gsaVersions))]
	return fmt.Sprintf(
		"Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/%s Mobile/15E148 Safari/604.1",
		osVersion, gsaVersion,
	)
}

// Google is the EngineConfig for Google web search.
var Google = EngineConfig{
	Name:            "google",
	ClientProfile:   "safari_ios_26_0",
	MinDelay:        3 * time.Second,
	MaxRetries:      3,
	RetryableStatus: defaultRetryableStatus,
	BuildRequest:    googleBuildRequest,
	ParseResponse:   googleParseResponse,
	PostProcess:     googlePostProcess,
}

// googleSafeSearchMap maps SafeSearch option strings to Google's "filter" parameter.
var googleSafeSearchMap = map[string]string{
	"on":       "2",
	"moderate": "1",
	"off":      "0",
}

// googleBuildRequest constructs the HTTP request for a Google search.
// Ported from Google.build_payload() in the reference implementation.
func googleBuildRequest(query string, opts SearchOpts) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.google.com/search", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("q", query)

	// SafeSearch -> filter param
	safesearch := strings.ToLower(opts.SafeSearch)
	if filterVal, ok := googleSafeSearchMap[safesearch]; ok {
		q.Set("filter", filterVal)
	} else {
		q.Set("filter", "1") // default to moderate
	}

	// Region -> hl, lr, cr params
	// Region format: "us-en" (country-lang)
	region := opts.Region
	if region == "" {
		region = "us-en"
	}
	parts := strings.SplitN(region, "-", 2)
	if len(parts) == 2 {
		country := parts[0]
		lang := parts[1]
		q.Set("hl", fmt.Sprintf("%s-%s", lang, strings.ToUpper(country)))
		q.Set("lr", fmt.Sprintf("lang_%s", lang))
		q.Set("cr", fmt.Sprintf("country%s", strings.ToUpper(country)))
	}

	req.URL.RawQuery = q.Encode()

	// GSA (Google Search App) user-agent, matching the reference implementation.
	// The Google EngineConfig sets ClientProfile to Safari iOS so the TLS
	// fingerprint matches this mobile UA.
	req.Header.Set("User-Agent", getGoogleUA())

	return req, nil
}

// googleParseResponse parses the HTML response from Google search.
// Uses XPath selectors ported from the reference implementation:
//   - items: //div[@data-snc]
//   - title: .//div[@role='link']//text()
//   - href:  .//a/@href
//   - body:  ./div[@data-sncf]//text()
func googleParseResponse(resp *http.Response) ([]Result, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := htmlquery.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	fields := map[string]string{
		"title": ".//div[@role='link']//text()",
		"href":  ".//a/@href",
		"body":  "./div[@data-sncf]//text()",
	}

	rows, err := XPathExtract(doc, "//div[@data-snc]", fields)
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

// googlePostProcess unwraps Google redirect URLs and filters out non-HTTP URLs
// and results with empty titles.
// Ported from Google.post_extract_results() in the reference implementation.
func googlePostProcess(results []Result) []Result {
	var filtered []Result
	for _, r := range results {
		// Unwrap /url?q= redirects
		r.URL = UnwrapRedirect(r.URL, RedirectGoogle)

		// Keep only results with a title and an HTTP(S) URL
		if r.Title != "" && strings.HasPrefix(r.URL, "http") {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
