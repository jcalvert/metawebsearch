// registry.go
package metawebsearch

// AllEngines returns every built-in engine.
func AllEngines() []EngineConfig {
	return []EngineConfig{
		DuckDuckGo,
		Brave,
		Mojeek,
		Yahoo,
		Yandex,
		Wikipedia,
		Grokipedia,
	}
}

// allKnownEngines includes every engine, even those excluded from AllEngines().
var allKnownEngines = []EngineConfig{
	Google,
	DuckDuckGo,
	Brave,
	Mojeek,
	Yahoo,
	Yandex,
	Wikipedia,
	Grokipedia,
}

// EngineByName looks up any engine by name, including engines not in
// AllEngines() (e.g. Google). Returns false if not found.
func EngineByName(name string) (EngineConfig, bool) {
	for _, e := range allKnownEngines {
		if e.Name == name {
			return e, true
		}
	}
	return EngineConfig{}, false
}
