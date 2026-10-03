package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserProcessEnvironmentDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("PATH", "safe-path")
	t.Setenv("OLLAMA_AGENT_BROWSER_EXECUTABLE", "chromium")
	t.Setenv("OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "1")
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "secret-credential")
	t.Setenv("AWS_ACCESS_KEY_ID", "secret-cloud-key")
	t.Setenv("HTTP_PROXY", "http://user:password@proxy.example")

	workspace := filepath.Join(os.TempDir(), "workspace")
	got := browserProcessEnvironment(workspace)
	joined := strings.Join(got, "\n")
	for _, secret := range []string{"secret-credential", "secret-cloud-key", "user:password", "OLLAMA_AGENT_CREDENTIAL_KEY", "AWS_ACCESS_KEY_ID", "HTTP_PROXY="} {
		if strings.Contains(joined, secret) {
			t.Fatalf("browser process environment contains forbidden value %q: %q", secret, joined)
		}
	}
	if !strings.Contains(joined, "PATH=safe-path") {
		t.Fatalf("browser process environment omitted PATH: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_EXECUTABLE=chromium") {
		t.Fatalf("browser process environment omitted configured executable: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE=1") {
		t.Fatalf("browser process environment omitted loopback test setting: %q", joined)
	}
	if !strings.Contains(joined, "OLLAMA_AGENT_BROWSER_ROOT="+filepath.Join(workspace, ".browser")) {
		t.Fatalf("browser process environment omitted browser root: %q", joined)
	}
}
