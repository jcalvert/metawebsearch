// extract.go
package metawebsearch

import (
	"net/url"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// RedirectPattern identifies a URL redirect scheme.
type RedirectPattern int

const (
	RedirectNone   RedirectPattern = iota
	RedirectDDG                    // //duckduckgo.com/l/?uddg=...
	RedirectYahoo                  // .../RU=.../RK=...
	RedirectGoogle                 // /url?q=...
)

// XPathExtract finds containers via itemsXPath, then extracts fields from
// each container using the fields map (field name -> XPath expression).
// Mirrors ddgs's BaseSearchEngine.extract_results().
func XPathExtract(doc *html.Node, itemsXPath string, fields map[string]string) ([]map[string]string, error) {
	items, err := htmlquery.QueryAll(doc, itemsXPath)
	if err != nil {
		return nil, err
	}

	var results []map[string]string
	for _, item := range items {
		row := make(map[string]string, len(fields))
		for key, xpath := range fields {
			nodes, err := htmlquery.QueryAll(item, xpath)
			if err != nil {
				continue
			}
			var parts []string
			for _, n := range nodes {
				text := strings.TrimSpace(nodeText(n))
				if text != "" {
					parts = append(parts, text)
				}
			}
			row[key] = strings.Join(parts, " ")
		}
		results = append(results, row)
	}
	return results, nil
}

// nodeText extracts text content from an html.Node.
func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	return htmlquery.InnerText(n)
}

// CleanText trims whitespace and collapses internal whitespace/newlines.
func CleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// UnwrapRedirect extracts the real URL from a search engine redirect wrapper.
func UnwrapRedirect(href string, pattern RedirectPattern) string {
	switch pattern {
	case RedirectDDG:
		if !strings.Contains(href, "duckduckgo.com/l/") {
			return href
		}
		u, err := url.Parse(href)
		if err != nil {
			return href
		}
		if uddg := u.Query().Get("uddg"); uddg != "" {
			return uddg
		}
		return href

	case RedirectYahoo:
		if !strings.Contains(href, "/RU=") {
			return href
		}
		parts := strings.SplitN(href, "/RU=", 2)
		if len(parts) < 2 {
			return href
		}
		tail := parts[1]
		for _, sep := range []string{"/RK=", "/RS="} {
			if idx := strings.Index(tail, sep); idx >= 0 {
				tail = tail[:idx]
			}
		}
		decoded, err := url.QueryUnescape(tail)
		if err != nil {
			return tail
		}
		return decoded

	case RedirectGoogle:
		if !strings.HasPrefix(href, "/url?") {
			return href
		}
		u, err := url.Parse(href)
		if err != nil {
			return href
		}
		if q := u.Query().Get("q"); q != "" {
			return q
		}
		return href

	default:
		return href
	}
}
