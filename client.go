package metawebsearch

import (
	"fmt"
	"net/http"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// ClientOpts configures the TLS-impersonating HTTP client.
type ClientOpts struct {
	BrowserProfile string // key into profiles.MappedTLSClients; empty = default
}

// NewClient creates an HTTPClient backed by bogdanfinn/tls-client.
// This is the only place in the codebase that imports tls-client directly;
// everything else uses the HTTPClient interface from result.go.
func NewClient(opts ClientOpts) (HTTPClient, error) {
	profile := profiles.DefaultClientProfile

	if opts.BrowserProfile != "" {
		p, ok := profiles.MappedTLSClients[opts.BrowserProfile]
		if !ok {
			return nil, fmt.Errorf("unknown browser profile: %q", opts.BrowserProfile)
		}
		profile = p
	}

	options := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(profile),
		tls_client.WithRandomTLSExtensionOrder(),
		tls_client.WithNotFollowRedirects(),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}

	return &tlsClientWrapper{client: client}, nil
}

// tlsClientWrapper adapts tls_client.HttpClient to our HTTPClient interface.
// It converts between stdlib net/http types and bogdanfinn/fhttp types
// so the rest of the codebase never imports fhttp.
type tlsClientWrapper struct {
	client tls_client.HttpClient
}

func (w *tlsClientWrapper) Do(req *http.Request) (*http.Response, error) {
	fReq := stdToFHTTPRequest(req)

	fResp, err := w.client.Do(fReq)
	if err != nil {
		return nil, err
	}

	return fhttpToStdResponse(fResp), nil
}

// stdToFHTTPRequest converts a stdlib *http.Request to a bogdanfinn/fhttp *Request.
func stdToFHTTPRequest(req *http.Request) *fhttp.Request {
	fReq := &fhttp.Request{
		Method:        req.Method,
		URL:           req.URL,
		Header:        fhttp.Header{},
		Body:          req.Body,
		ContentLength: req.ContentLength,
		Host:          req.Host,
	}
	for k, vs := range req.Header {
		for _, v := range vs {
			fReq.Header.Add(k, v)
		}
	}
	return fReq
}

// fhttpToStdResponse converts a bogdanfinn/fhttp *Response to a stdlib *http.Response.
func fhttpToStdResponse(resp *fhttp.Response) *http.Response {
	stdResp := &http.Response{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Proto:         resp.Proto,
		ProtoMajor:    resp.ProtoMajor,
		ProtoMinor:    resp.ProtoMinor,
		Header:        http.Header{},
		Body:          resp.Body,
		ContentLength: resp.ContentLength,
	}
	for k, vs := range resp.Header {
		for _, v := range vs {
			stdResp.Header.Add(k, v)
		}
	}
	return stdResp
}
