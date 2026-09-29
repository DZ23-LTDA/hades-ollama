//go:build !windows

package secrets

// On macOS/Linux the file is plain text protected by 0600 permissions, which
// internal/multillm verifies before reading it.
const fileExtension = ".key"

func protect(key string) ([]byte, error) { return []byte(key), nil }

// Future shells inherit the variable only from the app; nothing to persist.
func persistUserEnv(string, string) error { return nil }
