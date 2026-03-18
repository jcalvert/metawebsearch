# metawebsearch

Multi-engine web search library for Go. Queries DuckDuckGo, Brave, Mojeek, Yahoo, Yandex, Wikipedia, and Grokipedia behind a common interface, with concurrent multi-engine dispatch, URL deduplication, and browser-grade TLS fingerprinting.

Inspired by [deedy5/ddgs](https://github.com/deedy5/ddgs).

## Installation

```bash
go get github.com/jcalvert/metawebsearch
```

Requires Go 1.24+.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    mws "github.com/jcalvert/metawebsearch"
)

func main() {
    client, err := mws.NewClient(mws.ClientOpts{})
    if err != nil {
        log.Fatal(err)
    }

    ms := mws.MultiSearch{
        Client:  client,
        Engines: mws.AllEngines(),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    sr, err := ms.Search(ctx, "golang concurrency", mws.SearchOpts{MaxResults: 10})
    if err != nil {
        log.Fatal(err)
    }

    for _, r := range sr.Results {
        fmt.Printf("[%s] %s\n  %s\n\n", r.Engine, r.Title, r.URL)
    }

    // Check for per-engine errors (partial failure is normal)
    for name, err := range sr.Errors {
        fmt.Printf("  %s: %v\n", name, err)
    }
}
```

## Single Engine

Use `Execute` to query one engine directly:

```go
results, err := mws.Execute(
    ctx,
    client,
    mws.DuckDuckGo,
    "search query",
    mws.SearchOpts{MaxResults: 10},
)
```

## Engines

`AllEngines()` returns the default set:

| Engine     | Name           | MinDelay | Notes                           |
|------------|----------------|----------|---------------------------------|
| DuckDuckGo | `"duckduckgo"` | 2s       | POST to html.duckduckgo.com     |
| Brave      | `"brave"`      | 2s       | Cookie-based region/safesearch  |
| Mojeek     | `"mojeek"`     | 2s       | Independent search index        |
| Yahoo      | `"yahoo"`      | 2s       | Unwraps /RU= redirect URLs      |
| Yandex     | `"yandex"`     | 2s       | Uses /search/site/ endpoint     |
| Wikipedia  | `"wikipedia"`  | 1s       | OpenSearch JSON API              |
| Grokipedia | `"grokipedia"` | 1s       | Typeahead JSON API               |

**Google** is available via `EngineByName("google")` but excluded from the default set. As of early 2026, Google requires JavaScript execution to serve search results, which breaks all HTTP-based scrapers (including the [reference Python implementation](https://github.com/deedy5/ddgs)). The Google engine code is maintained in the repo for when a workaround is found.

### Picking Specific Engines

```go
// Use only DuckDuckGo and Brave
ms := mws.MultiSearch{
    Client:  client,
    Engines: []mws.EngineConfig{mws.DuckDuckGo, mws.Brave},
}
```

### Runtime Lookup

```go
eng, ok := mws.EngineByName("brave")
if ok {
    results, err := mws.Execute(ctx, client, eng, "query", mws.SearchOpts{})
}
```

## Rate Limiting and Retries

Each engine has built-in rate limiting and retry behavior. These are configured via fields on `EngineConfig`.

### How Rate Limiting Works

The `Execute` pipeline enforces a per-engine minimum delay between requests. If you call `Execute` for the same engine twice in quick succession, the second call blocks until `MinDelay` has elapsed since the last request to that engine.

Rate limiting is **global per engine name** (not per client), so multiple goroutines sharing the same engine will respect the same rate limit.

### How Retries Work

When an engine returns a retryable HTTP status (default: 202, 429, 503), `Execute` retries with exponential backoff:

- First retry waits at least **5 seconds** (or `MinDelay`, whichever is greater)
- Each subsequent retry doubles the wait: 5s, 10s, 20s...
- After `MaxRetries` attempts (default: 3), it returns an error
- Retries respect the context — a cancelled context aborts immediately

### Customizing an Engine

Every field on `EngineConfig` is public. Copy a built-in engine and adjust:

```go
// Slow down Brave to avoid rate limiting
myBrave := mws.Brave
myBrave.MinDelay = 5 * time.Second
myBrave.MaxRetries = 5

// Make DuckDuckGo retry on 403
myDDG := mws.DuckDuckGo
myDDG.RetryableStatus = func(code int) bool {
    return code == 202 || code == 403 || code == 429 || code == 503
}

ms := mws.MultiSearch{
    Client:  client,
    Engines: []mws.EngineConfig{myBrave, myDDG, mws.Mojeek},
}
```

### Engine Defaults

| Field             | Default (if zero)                        |
|-------------------|------------------------------------------|
| `MinDelay`        | 2s (used for rate limiting between calls)|
| `MaxRetries`      | 3                                        |
| `RetryableStatus` | 202, 429, 503                            |

The minimum retry backoff is 5 seconds for engines with `MinDelay >= 1s`. This prevents hammering search engines during retry loops.

## Search Options

```go
opts := mws.SearchOpts{
    MaxResults: 10,          // max results to request (engine-dependent)
    Region:     "us-en",     // country-language code
    SafeSearch: "moderate",  // "on", "moderate", "off"
    TimeLimit:  "w",         // "d" (day), "w" (week), "m" (month), "y" (year)
    Page:       1,           // 1-based page number
}
```

Not all engines support all options. Unsupported options are silently ignored.

| Option       | DuckDuckGo | Brave | Mojeek | Yahoo | Yandex | Wikipedia | Grokipedia |
|--------------|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `MaxResults` | -   | -   | -   | -   | -   | yes | yes |
| `Region`     | -   | yes | yes | -   | -   | yes | -   |
| `SafeSearch`  | -   | yes | yes | -   | -   | -   | -   |
| `TimeLimit`  | -   | yes | -   | yes | -   | -   | -   |
| `Page`       | -   | -   | -   | yes | yes | -   | -   |

## MultiSearch Behavior

`MultiSearch.Search` dispatches all engines concurrently and returns a `*SearchResult`:

```go
type SearchResult struct {
    Results []Result            // deduplicated by URL, ordered by engine
    Errors  map[string]error    // per-engine errors (partial failure)
}
```

- **Partial failure**: If some engines fail and others succeed, you get results from the successful ones plus error details for the failed ones. `Search` itself never returns an error.
- **Deduplication**: Results are deduplicated by URL. When the same URL appears from multiple engines, the first engine (by order in the `Engines` slice) wins.
- **Ordering**: Results are grouped by engine in the order engines appear in the `Engines` slice.

## TLS Client

All HTTP requests go through [bogdanfinn/tls-client](https://github.com/bogdanfinn/tls-client), which impersonates real browser TLS fingerprints (JA3, HTTP/2 SETTINGS, header order). This is critical for avoiding bot detection.

```go
// Default Chrome profile
client, err := mws.NewClient(mws.ClientOpts{})

// Specific browser profile
client, err := mws.NewClient(mws.ClientOpts{
    BrowserProfile: "chrome_131",
})
```

See [tls-client profiles](https://github.com/bogdanfinn/tls-client/blob/master/profiles/profiles.go) for available profile names.

Browser-like headers (Accept, Accept-Language, Sec-Ch-Ua, Sec-Fetch-*, etc.) are automatically added to every request by the `Execute` pipeline. Engine-specific headers set in `BuildRequest` take precedence.

## Result Type

```go
type Result struct {
    Title   string `json:"title"`
    URL     string `json:"url"`
    Snippet string `json:"snippet"`
    Engine  string `json:"engine"`
}
```

Results include JSON tags for easy serialization.

## Test Binary

A simple CLI is included for manual testing:

```bash
go run ./cmd/search/ "your query here"
```

Outputs JSON with results grouped by engine and any errors.

## Testing

Unit tests (no network, fast):

```bash
go test ./...
```

With race detector:

```bash
go test -race ./...
```

Integration tests (live HTTP, hits real search engines):

```bash
go test -tags=integration ./... -v
```

## License

MIT. See [LICENSE](LICENSE).
