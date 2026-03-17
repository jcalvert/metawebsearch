// Package metawebsearch provides a multi-engine web search library for Go.
//
// It scrapes Google, DuckDuckGo, Brave, Mojeek, Yahoo, Yandex, Wikipedia,
// and Grokipedia behind a common EngineConfig interface. Engines can be used
// individually via Execute or concurrently via MultiSearch.
//
// Browser impersonation (TLS + HTTP/2 fingerprinting) is handled by tls-client,
// wrapped behind the HTTPClient interface for testability.
package metawebsearch
