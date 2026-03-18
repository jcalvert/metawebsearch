// multi.go
package metawebsearch

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
			// Use per-engine client if the engine needs a specific TLS profile
			client := m.Client
			if eng.ClientProfile != "" {
				override, profileErr := NewClient(ClientOpts{BrowserProfile: eng.ClientProfile})
				if profileErr != nil {
					mu.Lock()
					outputs = append(outputs, engineResult{
						name: eng.Name, err: fmt.Errorf("client profile %q: %w", eng.ClientProfile, profileErr), order: idx,
					})
					mu.Unlock()
					return
				}
				client = override
			}
			results, err := Execute(ctx, client, eng, query, opts)
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

	// If every engine failed, return an aggregate error.
	if len(sr.Results) == 0 && len(sr.Errors) > 0 {
		names := make([]string, 0, len(sr.Errors))
		for name := range sr.Errors {
			names = append(names, name)
		}
		sort.Strings(names)
		return sr, fmt.Errorf("all %d engines failed: %s", len(sr.Errors), strings.Join(names, ", "))
	}

	return sr, nil
}
