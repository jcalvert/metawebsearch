# metawebsearch

Multi-engine web search library for Go. Scrapes Google, DuckDuckGo, Brave, Mojeek, Yahoo, Yandex, Wikipedia, and Grokipedia behind a common interface.

Inspired by [deedy5/ddgs](https://github.com/deedy5/ddgs).

## Installation

```bash
go get github.com/jcalvert/metawebsearch
```

## Usage

### Single engine

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jcalvert/metawebsearch"
)

func main() {
	client, err := metawebsearch.NewClient(metawebsearch.ClientOpts{})
	if err != nil {
		log.Fatal(err)
	}

	results, err := metawebsearch.Execute(
		context.Background(),
		client,
		metawebsearch.Google,
		"golang concurrency",
		metawebsearch.SearchOpts{MaxResults: 10},
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range results {
		fmt.Printf("%s\n  %s\n\n", r.Title, r.URL)
	}
}
```

### Multi-engine

```go
ms := &metawebsearch.MultiSearch{
	Client:  client,
	Engines: metawebsearch.AllEngines(),
}

sr, err := ms.Search(context.Background(), "golang concurrency", metawebsearch.SearchOpts{
	MaxResults: 10,
})
if err != nil {
	log.Fatal(err)
}

for _, r := range sr.Results {
	fmt.Printf("[%s] %s\n  %s\n\n", r.Engine, r.Title, r.URL)
}

for name, err := range sr.Errors {
	fmt.Printf("engine %s failed: %v\n", name, err)
}
```

## Supported Engines

| Engine       | Var / Name        |
|--------------|-------------------|
| Google       | `Google`          |
| DuckDuckGo   | `DuckDuckGo`      |
| Brave        | `Brave`           |
| Mojeek       | `Mojeek`          |
| Yahoo        | `Yahoo`           |
| Yandex       | `Yandex`          |
| Wikipedia    | `Wikipedia`       |
| Grokipedia   | `Grokipedia`      |

You can also look up an engine by name at runtime:

```go
eng, ok := metawebsearch.EngineByName("google")
```

## Testing

Unit tests (no network):

```bash
go test -race ./...
```

Integration tests (live HTTP, requires internet):

```bash
go test -tags=integration ./... -v
```

## License

See [LICENSE](LICENSE).
