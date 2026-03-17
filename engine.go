// engine.go
package metawebsearch

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// defaultRetryableStatus returns true for status codes that should trigger a retry.
func defaultRetryableStatus(code int) bool {
	return code == 202 || code == 429 || code == 503
}

// rateLimiter tracks per-engine last request times.
var (
	rateMu   sync.Mutex
	lastReqs = make(map[string]time.Time)
)

// Execute runs the full engine pipeline: BuildRequest -> HTTP -> ParseResponse -> PostProcess.
// Handles rate limiting and retries with exponential backoff.
func Execute(ctx context.Context, client HTTPClient, engine EngineConfig, query string, opts SearchOpts) ([]Result, error) {
	retryable := engine.RetryableStatus
	if retryable == nil {
		retryable = defaultRetryableStatus
	}

	maxRetries := engine.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	minDelay := engine.MinDelay
	if minDelay <= 0 {
		minDelay = 2 * time.Second
	}

	// Rate limit: wait if needed (only when engine specifies a MinDelay)
	if engine.MinDelay > 0 {
		enforceRateLimit(engine.Name, minDelay)
	}

	backoff := minDelay
	var lastStatus int

	for attempt := range maxRetries + 1 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%s: %w (after %d retries, last HTTP %d)", engine.Name, ctx.Err(), attempt, lastStatus)
			case <-time.After(backoff):
			}
			backoff *= 2
		}

		req, err := engine.BuildRequest(query, opts)
		if err != nil {
			return nil, fmt.Errorf("%s: build request: %w", engine.Name, err)
		}
		req = req.WithContext(ctx)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s: request failed: %w", engine.Name, err)
		}

		if retryable(resp.StatusCode) {
			lastStatus = resp.StatusCode
			resp.Body.Close()
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("%s: HTTP %d", engine.Name, resp.StatusCode)
		}

		results, err := engine.ParseResponse(resp)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: parse: %w", engine.Name, err)
		}

		if engine.PostProcess != nil {
			results = engine.PostProcess(results)
		}

		// Stamp engine name on all results
		for i := range results {
			results[i].Engine = engine.Name
		}

		recordRequest(engine.Name)
		return results, nil
	}

	return nil, fmt.Errorf("%s: HTTP %d after %d retries", engine.Name, lastStatus, maxRetries)
}

func enforceRateLimit(name string, minDelay time.Duration) {
	rateMu.Lock()
	last := lastReqs[name]
	rateMu.Unlock()

	if elapsed := time.Since(last); elapsed < minDelay {
		time.Sleep(minDelay - elapsed)
	}
}

func recordRequest(name string) {
	rateMu.Lock()
	lastReqs[name] = time.Now()
	rateMu.Unlock()
}
