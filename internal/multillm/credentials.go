package multillm

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxCredentialFileBytes = 64 << 10

func credentialValue(envName string) string {
	if envName == "" {
		return ""
	}
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	path := strings.TrimSpace(os.Getenv(envName + "_FILE"))
	if path == "" {
		// Keys saved from the desktop app live in the default credential
		// directory; read them there so a server started before the key was
		// saved (or from a stale environment) still finds it.
		path = defaultCredentialPath(envName)
		if path == "" {
			return ""
		}
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > maxCredentialFileBytes || !credentialFilePermissionsSafe(info) {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	value, err := decodeCredentialFile(path, raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

// CredentialValue returns the credential for envName from the environment,
// <envName>_FILE, or the desktop app's default credential directory.
func CredentialValue(envName string) string {
	return credentialValue(envName)
}

// DefaultCredentialDir is where the desktop app stores provider and connector
// keys: %LOCALAPPDATA%\Ollama DZ23\secrets on Windows, the user config
// directory elsewhere.
func DefaultCredentialDir() string {
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, "Ollama DZ23", "secrets")
		}
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "Ollama DZ23", "secrets")
}

func defaultCredentialPath(envName string) string {
	dir := DefaultCredentialDir()
	if dir == "" {
		return ""
	}
	path := filepath.Join(dir, envName+credentialFileExtension)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return ""
	}
	return path
}
