package multillm

import (
	"os"
	"strings"

	"github.com/ollama/ollama/internal/procenv"
)

// cliCommandEnv builds the environment for a CLI provider process: the shared
// minimal passthrough plus, when the provider declares one, its own API key
// variable. Credentials belonging to other providers, to the database or to the
// agent credential store are never exported to it.
func cliCommandEnv(provider Provider) []string {
	var extra []string
	names := append([]string{provider.APIKeyEnv}, provider.PassthroughEnv...)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || strings.ContainsAny(name, "=\x00\r\n") {
			continue
		}
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			extra = append(extra, name+"="+value)
		}
	}
	return procenv.Minimal(extra...)
}
