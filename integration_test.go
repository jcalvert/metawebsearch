//go:build integration

package metawebsearch

import (
	"context"
	"testing"
	"time"
)

func TestIntegrationEngines(t *testing.T) {
	client, err := NewClient(ClientOpts{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	engines := AllEngines()
	for _, engine := range engines {
		t.Run(engine.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := Execute(ctx, client, engine, "golang", SearchOpts{MaxResults: 5})
			if err != nil {
				t.Fatalf("%s: %v", engine.Name, err)
			}
			if len(results) == 0 {
				t.Fatalf("%s: no results", engine.Name)
			}
			for i, r := range results {
				if r.Title == "" {
					t.Errorf("%s result[%d]: empty title", engine.Name, i)
				}
				if r.URL == "" {
					t.Errorf("%s result[%d]: empty URL", engine.Name, i)
				}
			}
			t.Logf("%s: %d results", engine.Name, len(results))
		})
	}
}

func TestIntegrationMultiSearch(t *testing.T) {
	client, err := NewClient(ClientOpts{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ms := MultiSearch{Client: client, Engines: AllEngines()}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sr, err := ms.Search(ctx, "golang concurrency", SearchOpts{MaxResults: 5})
	if err != nil {
		t.Fatalf("MultiSearch: %v", err)
	}
	t.Logf("Results: %d, Errors: %d", len(sr.Results), len(sr.Errors))
	for name, e := range sr.Errors {
		t.Logf("  %s: %v", name, e)
	}
	if len(sr.Results) == 0 {
		t.Fatal("MultiSearch returned 0 results")
	}
}
