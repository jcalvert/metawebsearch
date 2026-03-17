// registry.go
package metawebsearch

// AllEngines returns every built-in engine.
func AllEngines() []EngineConfig {
	return []EngineConfig{
		Google,
		DuckDuckGo,
		Brave,
		Mojeek,
		Yahoo,
		Yandex,
		Wikipedia,
		Grokipedia,
	}
}

// EngineByName looks up a single engine by name. Returns false if not found.
func EngineByName(name string) (EngineConfig, bool) {
	for _, e := range AllEngines() {
		if e.Name == name {
			return e, true
		}
	}
	return EngineConfig{}, false
}
