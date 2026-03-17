// multi.go
package metawebsearch

import (
	"context"
	"sort"
	"sync"
)

// MultiSearch dispatches a query to multiple engines concurrently.
type MultiSearch struct {
	Client  HTTPClient
	Engines []EngineConfig
}

// Search runs all engines concurrently, deduplicates by URL, collects per-engine errors.
func (m *MultiSearch) Search(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	type engineResult struct {
		name    string
		results []Result
		err     error
		order   int
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		outputs []engineResult
	)

	for i, engine := range m.Engines {
		wg.Add(1)
		go func(eng EngineConfig, idx int) {
			defer wg.Done()
			results, err := Execute(ctx, m.Client, eng, query, opts)
			mu.Lock()
			outputs = append(outputs, engineResult{
				name: eng.Name, results: results, err: err, order: idx,
			})
			mu.Unlock()
		}(engine, i)
	}
	wg.Wait()

	// Sort by original engine order
	sort.Slice(outputs, func(i, j int) bool {
		return outputs[i].order < outputs[j].order
	})

	// Deduplicate by URL, collect errors
	seen := make(map[string]bool)
	sr := &SearchResult{Errors: make(map[string]error)}

	for _, o := range outputs {
		if o.err != nil {
			sr.Errors[o.name] = o.err
			continue
		}
		for _, r := range o.results {
			if !seen[r.URL] {
				seen[r.URL] = true
				sr.Results = append(sr.Results, r)
			}
		}
	}

	return sr, nil
}
