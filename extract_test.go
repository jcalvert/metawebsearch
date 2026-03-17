// extract_test.go
package metawebsearch

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestXPathExtract(t *testing.T) {
	raw := `<html><body>
		<div class="result">
			<h2><a href="https://example.com">Example Title</a></h2>
			<p class="snippet">A snippet of text</p>
		</div>
		<div class="result">
			<h2><a href="https://other.com">Other Title</a></h2>
			<p class="snippet">Another snippet</p>
		</div>
	</body></html>`
	doc, _ := html.Parse(strings.NewReader(raw))

	results, err := XPathExtract(doc, "//div[@class='result']", map[string]string{
		"title": ".//h2//text()",
		"href":  ".//h2/a/@href",
		"body":  ".//p[@class='snippet']//text()",
	})
	if err != nil {
		t.Fatalf("XPathExtract error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0]["title"] != "Example Title" {
		t.Errorf("title = %q, want %q", results[0]["title"], "Example Title")
	}
	if results[0]["href"] != "https://example.com" {
		t.Errorf("href = %q, want %q", results[0]["href"], "https://example.com")
	}
	if results[0]["body"] != "A snippet of text" {
		t.Errorf("body = %q, want %q", results[0]["body"], "A snippet of text")
	}
	if results[1]["title"] != "Other Title" {
		t.Errorf("title = %q, want %q", results[1]["title"], "Other Title")
	}
	if results[1]["href"] != "https://other.com" {
		t.Errorf("href = %q, want %q", results[1]["href"], "https://other.com")
	}
}

func TestXPathExtractEmpty(t *testing.T) {
	raw := `<html><body><p>No results here</p></body></html>`
	doc, _ := html.Parse(strings.NewReader(raw))

	results, err := XPathExtract(doc, "//div[@class='result']", map[string]string{
		"title": ".//h2//text()",
	})
	if err != nil {
		t.Fatalf("XPathExtract error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestXPathExtractBadXPath(t *testing.T) {
	raw := `<html><body></body></html>`
	doc, _ := html.Parse(strings.NewReader(raw))

	_, err := XPathExtract(doc, "[[[invalid", map[string]string{})
	if err == nil {
		t.Error("expected error for invalid XPath, got nil")
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"  hello   world  ", "hello world"},
		{"line1\nline2\n  line3", "line1 line2 line3"},
		{"", ""},
		{"\t\ttabbed\t\ttext\t\t", "tabbed text"},
		{"already clean", "already clean"},
		{" \n \r \t ", ""},
	}
	for _, tt := range tests {
		got := CleanText(tt.in)
		if got != tt.want {
			t.Errorf("CleanText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestUnwrapDDGRedirect(t *testing.T) {
	href := "//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.com&rut=abc"
	got := UnwrapRedirect(href, RedirectDDG)
	if got != "https://example.com" {
		t.Errorf("UnwrapRedirect DDG = %q, want %q", got, "https://example.com")
	}
}

func TestUnwrapDDGRedirectPassthrough(t *testing.T) {
	href := "https://example.com/direct"
	got := UnwrapRedirect(href, RedirectDDG)
	if got != href {
		t.Errorf("UnwrapRedirect DDG passthrough = %q, want %q", got, href)
	}
}

func TestUnwrapYahooRedirect(t *testing.T) {
	href := "https://r.search.yahoo.com/something/RU=https%3A%2F%2Fexample.com/RK=2/RS=abc"
	got := UnwrapRedirect(href, RedirectYahoo)
	if got != "https://example.com" {
		t.Errorf("UnwrapRedirect Yahoo = %q, want %q", got, "https://example.com")
	}
}

func TestUnwrapYahooRedirectPassthrough(t *testing.T) {
	href := "https://example.com/direct"
	got := UnwrapRedirect(href, RedirectYahoo)
	if got != href {
		t.Errorf("UnwrapRedirect Yahoo passthrough = %q, want %q", got, href)
	}
}

func TestUnwrapGoogleRedirect(t *testing.T) {
	href := "/url?q=https://example.com&sa=U&ved=abc"
	got := UnwrapRedirect(href, RedirectGoogle)
	if got != "https://example.com" {
		t.Errorf("UnwrapRedirect Google = %q, want %q", got, "https://example.com")
	}
}

func TestUnwrapGoogleRedirectPassthrough(t *testing.T) {
	href := "https://example.com/direct"
	got := UnwrapRedirect(href, RedirectGoogle)
	if got != href {
		t.Errorf("UnwrapRedirect Google passthrough = %q, want %q", got, href)
	}
}

func TestUnwrapRedirectNone(t *testing.T) {
	href := "https://example.com"
	got := UnwrapRedirect(href, RedirectNone)
	if got != href {
		t.Errorf("UnwrapRedirect None = %q, want %q", got, href)
	}
}
