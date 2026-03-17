package metawebsearch

import "testing"

func TestNewClientReturnsHTTPClient(t *testing.T) {
	c, err := NewClient(ClientOpts{})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClientWithProfile(t *testing.T) {
	c, err := NewClient(ClientOpts{BrowserProfile: "chrome_131"})
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient() returned nil")
	}
}

func TestNewClientUnknownProfile(t *testing.T) {
	_, err := NewClient(ClientOpts{BrowserProfile: "netscape_4"})
	if err == nil {
		t.Fatal("NewClient() should error on unknown profile")
	}
}
