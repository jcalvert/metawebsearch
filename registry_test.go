// registry_test.go
package metawebsearch

import "testing"

func TestAllEngines(t *testing.T) {
	engines := AllEngines()
	if len(engines) != 7 {
		t.Errorf("AllEngines() returned %d engines, want 7", len(engines))
	}
	names := make(map[string]bool)
	for _, e := range engines {
		names[e.Name] = true
	}
	for _, want := range []string{"duckduckgo", "brave", "mojeek", "yahoo", "yandex", "wikipedia", "grokipedia"} {
		if !names[want] {
			t.Errorf("missing engine: %s", want)
		}
	}
}

func TestEngineByName(t *testing.T) {
	e, ok := EngineByName("google")
	if !ok {
		t.Fatal("EngineByName(google) not found")
	}
	if e.Name != "google" {
		t.Errorf("Name = %q, want google", e.Name)
	}

	_, ok = EngineByName("nonexistent")
	if ok {
		t.Error("EngineByName(nonexistent) should return false")
	}
}
