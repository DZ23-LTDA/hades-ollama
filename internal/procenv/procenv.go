// Package procenv builds the environment handed to child processes spawned by
// the agent runtime. It is shared so every spawn site uses the same allowlist
// instead of inheriting the server environment, which carries provider API
// keys, database passwords and the agent credential key.
package procenv

import (
	"os"
	"strings"
)

// passthrough lists the variables a child process may inherit.
var passthrough = []string{
	// POSIX basics.
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_ALL", "TZ",
	"TMPDIR", "TMP", "TEMP",
	// Display plumbing required by headful browsers and desktop automation.
	"DISPLAY", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR", "XDG_SESSION_TYPE",
	// Playwright/browser cache locations.
	"PLAYWRIGHT_BROWSERS_PATH", "PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD",
	// Windows basics.
	"SystemRoot", "SYSTEMROOT", "windir", "WINDIR", "COMSPEC", "PATHEXT",
	"LOCALAPPDATA", "LocalAppData", "APPDATA", "USERPROFILE", "USERNAME",
	"HOMEDRIVE", "HOMEPATH", "PROGRAMFILES", "ProgramFiles", "ProgramData",
	// macOS basics.
	"__CF_USER_TEXT_ENCODING",
}

// Minimal returns the allowlisted environment plus the explicit "KEY=value"
// entries the caller needs. Later entries win, so a caller can override an
// inherited value.
func Minimal(extra ...string) []string {
	env := make([]string, 0, len(passthrough)+len(extra))
	for _, name := range passthrough {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			env = append(env, name+"="+value)
		}
	}
	for _, entry := range extra {
		if strings.TrimSpace(entry) == "" || !strings.Contains(entry, "=") {
			continue
		}
		env = append(env, entry)
	}
	return env
}

// Passthrough returns a copy of the allowlist, for tests and diagnostics.
func Passthrough() []string {
	names := make([]string, len(passthrough))
	copy(names, passthrough)
	return names
}
