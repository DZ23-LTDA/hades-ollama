// Package secrets stores provider API keys for the multi-provider router in
// the same format internal/multillm reads: a per-user protected file whose
// path is published as <ENV>_FILE.
package secrets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MaxKeyBytes bounds a pasted key; real provider keys are far smaller.
const MaxKeyBytes = 8 << 10

var envNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)

// ErrInvalid is returned for malformed variable names or keys.
var ErrInvalid = errors.New("invalid credential")

func validate(envName, key string) error {
	if !envNamePattern.MatchString(envName) {
		return fmt.Errorf("%w: variable name %q", ErrInvalid, envName)
	}
	key = strings.TrimSpace(key)
	if key == "" || len(key) > MaxKeyBytes || strings.ContainsAny(key, "\r\n\x00") {
		return fmt.Errorf("%w: key must be a single non-empty line", ErrInvalid)
	}
	return nil
}

// Path returns where the credential for envName is stored under dir.
func Path(dir, envName string) string {
	return filepath.Join(dir, envName+fileExtension)
}

// Save protects key for the current user, writes it under dir and points
// <envName>_FILE at it for this process and, where supported, for future
// sessions of the user. The key itself is never placed in the environment.
func Save(dir, envName, key string) (string, error) {
	if err := validate(envName, key); err != nil {
		return "", err
	}
	payload, err := protect(strings.TrimSpace(key))
	if err != nil {
		return "", fmt.Errorf("protect credential: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := Path(dir, envName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", err
	}
	os.Unsetenv(envName)
	os.Setenv(envName+"_FILE", path)
	if err := persistUserEnv(envName, path); err != nil {
		return path, fmt.Errorf("credential saved but user environment not updated: %w", err)
	}
	return path, nil
}

// Remove deletes the stored credential and clears both variables.
func Remove(dir, envName string) error {
	if !envNamePattern.MatchString(envName) {
		return fmt.Errorf("%w: variable name %q", ErrInvalid, envName)
	}
	if err := os.Remove(Path(dir, envName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	os.Unsetenv(envName)
	os.Unsetenv(envName + "_FILE")
	return persistUserEnv(envName, "")
}

// Adopt points <envName>_FILE at a credential previously saved under dir
// when neither variable is set, so a process started without the user's
// updated environment still finds saved keys. It reports whether it did.
func Adopt(dir, envName string) bool {
	return AdoptFrom(dir, envName, envName)
}

// AdoptFrom points <envName>_FILE at the credential saved under a different
// (legacy) variable name, with the same rules as Adopt.
func AdoptFrom(dir, savedName, envName string) bool {
	if !envNamePattern.MatchString(savedName) || !envNamePattern.MatchString(envName) {
		return false
	}
	if strings.TrimSpace(os.Getenv(envName)) != "" || strings.TrimSpace(os.Getenv(envName+"_FILE")) != "" {
		return false
	}
	path := Path(dir, savedName)
	if info, err := os.Stat(path); err != nil || info.IsDir() || info.Size() == 0 {
		return false
	}
	os.Setenv(envName+"_FILE", path)
	return true
}

// Configured reports whether a credential is available for envName, either
// directly in the environment or through an existing <envName>_FILE.
func Configured(envName string) bool {
	if strings.TrimSpace(os.Getenv(envName)) != "" {
		return true
	}
	path := strings.TrimSpace(os.Getenv(envName + "_FILE"))
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}
