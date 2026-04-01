// multi.go
package metawebsearch

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// normalizeURL lowercases scheme and host, strips trailing slashes, default
// ports, and fragments so that equivalent URLs dedup correctly.
func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	// Strip default ports
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = u.Hostname()
	}
	// Strip trailing slash from path (but keep "/" for root)
	if len(u.Path) > 1 {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

// MultiSearch dispatches a query to multiple engines concurrently.
type MultiSearch struct {
	Client  HTTPClient
	Engines []EngineConfig

	// EngineTimeout is the maximum time to wait for any single engine.
	// If an engine exceeds this deadline (e.g. due to rate-limit retries),
	// its context is canceled and results from faster engines are returned.
	// Zero means 10 seconds.
	EngineTimeout time.Duration
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

	engineTimeout := m.EngineTimeout
	if engineTimeout <= 0 {
		engineTimeout = 10 * time.Second
	}

	for i, engine := range m.Engines {
		wg.Add(1)
		go func(eng EngineConfig, idx int) {
			defer wg.Done()
			engCtx, engCancel := context.WithTimeout(ctx, engineTimeout)
			defer engCancel()
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
			results, err := Execute(engCtx, client, eng, query, opts)
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

	// Deduplicate by normalized URL, collect errors
	seen := make(map[string]bool)
	sr := &SearchResult{Errors: make(map[string]error)}

	for _, o := range outputs {
		if o.err != nil {
			sr.Errors[o.name] = o.err
			continue
		}
		for _, r := range o.results {
			key := normalizeURL(r.URL)
			if !seen[key] {
				seen[key] = true
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
