//go:build windows || darwin

package updater

import "testing"

func TestUpdateURLAllowedOnlyForThisDistribution(t *testing.T) {
	old := UpdateCheckURLBase
	t.Cleanup(func() { UpdateCheckURLBase = old })
	UpdateCheckURLBase = "https://ollama.com/api/update"

	for _, u := range []string{
		"https://github.com/ollama/ollama/releases/download/v0.34.4/OllamaSetup.exe",
		"https://ollama.com/download/OllamaSetup.exe",
		"https://github.com/DZ23-LTDA/ollama-classe-a-plus.evil.com/releases/download/x.exe",
		"not a url",
		"",
	} {
		if updateURLAllowed(u) {
			t.Errorf("updateURLAllowed(%q) = true, want false", u)
		}
	}
	if !updateURLAllowed("https://github.com/DZ23-LTDA/ollama-classe-a-plus/releases/download/v0.2.0/OllamaClasseAPlusSetup.exe") {
		t.Error("DZ23 release URL should be allowed")
	}

	UpdateCheckURLBase = "http://127.0.0.1:4567/update.json"
	if !updateURLAllowed("http://127.0.0.1:4567/download") {
		t.Error("loopback test server URL on the same host should be allowed")
	}
	if updateURLAllowed("http://127.0.0.1:9999/download") {
		t.Error("loopback URL on a different port must not be allowed")
	}
}
