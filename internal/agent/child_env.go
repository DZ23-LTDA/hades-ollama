package agent

import "github.com/ollama/ollama/internal/procenv"

// minimalChildEnv builds the environment for a child process spawned by the
// runtime: the shared allowlist in internal/procenv plus the explicit
// "KEY=value" entries the caller needs. Dropping everything else keeps
// provider API keys, database passwords and the credential key out of
// processes that handle hostile content.
func minimalChildEnv(extra ...string) []string {
	return procenv.Minimal(extra...)
}
