// engine.go
package metawebsearch

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
)

// defaultRetryableStatus returns true for status codes that should trigger a retry.
func defaultRetryableStatus(code int) bool {
	return code == 202 || code == 429 || code == 503
}

// defaultBrowserHeaders are the headers a real browser sends on every navigation
// request. These match what primp/impersonate="random" sends in the reference
// implementation. Without these, search engines detect us as bots immediately.
var defaultBrowserHeaders = map[string]string{
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
	"Accept-Language":           "en-US,en;q=0.9",
	// Accept-Encoding is deliberately NOT set here. tls-client's fhttp transport
	// handles it automatically: it sends the right Accept-Encoding as part of the
	// TLS profile, auto-decompresses the response, and strips the Content-Encoding
	// header. If we set Accept-Encoding ourselves, fhttp still auto-decompresses
	// but leaves the Content-Encoding header, which breaks our decompressBody.
	"Sec-Ch-Ua":                 `"Chromium";v="131", "Not_A Brand";v="24"`,
	"Sec-Ch-Ua-Mobile":          "?0",
	"Sec-Ch-Ua-Platform":        `"Windows"`,
	"Sec-Fetch-Dest":            "document",
	"Sec-Fetch-Mode":            "navigate",
	"Sec-Fetch-Site":            "none",
	"Sec-Fetch-User":            "?1",
	"Upgrade-Insecure-Requests": "1",
}

// applyDefaultHeaders sets browser-like headers on the request. Headers already
// set by the engine's BuildRequest are not overwritten.
func applyDefaultHeaders(req *http.Request) {
	for k, v := range defaultBrowserHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
	// Ensure User-Agent is set if engine didn't provide one
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	}

	// Sec-Ch-Ua headers are Chrome desktop specific. Remove them if the UA
	// indicates a non-Chrome client (e.g. GSA mobile, or API clients like
	// Wikipedia that set their own UA).
	ua := req.Header.Get("User-Agent")
	if !strings.Contains(ua, "Chrome/") {
		req.Header.Del("Sec-Ch-Ua")
		req.Header.Del("Sec-Ch-Ua-Mobile")
		req.Header.Del("Sec-Ch-Ua-Platform")
	}
}

// decompressBody reads the full response body and decompresses it if needed.
// tls-client's fhttp transport is inconsistent: it auto-decompresses some
// responses but not others, and never strips the Content-Encoding header.
// We handle this by reading the body, attempting decompression if the header
// says it's compressed, and falling back to the raw bytes if decompression
// fails (meaning the transport already handled it).
func decompressBody(resp *http.Response) {
	ce := strings.ToLower(resp.Header.Get("Content-Encoding"))
	if ce == "" {
		return
	}

	raw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || len(raw) == 0 {
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		return
	}

	var decompressed []byte
	switch ce {
	case "br":
		decompressed, err = io.ReadAll(brotli.NewReader(bytes.NewReader(raw)))
	case "gzip":
		var gr *gzip.Reader
		gr, err = gzip.NewReader(bytes.NewReader(raw))
		if err == nil {
			decompressed, err = io.ReadAll(gr)
			gr.Close()
		}
	case "deflate":
		decompressed, err = io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
	default:
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		return
	}

	if err != nil {
		// Decompression failed — transport already decompressed it
		resp.Body = io.NopCloser(bytes.NewReader(raw))
	} else {
		resp.Body = io.NopCloser(bytes.NewReader(decompressed))
	}
}

const minRetryBackoff = 5 * time.Second

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

	// Retry backoff: real engines (MinDelay >= 1s) get at least 5s between
	// retries to avoid anti-bot defenses. Test engines with tiny delays are
	// not subject to the floor.
	backoff := minDelay
	if engine.MinDelay >= time.Second && backoff < minRetryBackoff {
		backoff = minRetryBackoff
	}
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
		applyDefaultHeaders(req)

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

		decompressBody(resp)
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

// resetRateLimit clears rate limit state for an engine. Exported for testing.
func resetRateLimit(name string) {
	rateMu.Lock()
	delete(lastReqs, name)
	rateMu.Unlock()
}
