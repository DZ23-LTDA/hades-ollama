//go:build windows

package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// isExecutableMode reports whether a regular file can be launched as an MCP
// server. Windows has no execute permission bit, so only native executable
// images on a local drive are accepted. Batch scripts (.bat/.cmd) are rejected
// on purpose: they run through cmd.exe, whose argument parsing enables command
// injection. Paths that Win32 resolves differently from the name we inspect —
// NTFS alternate data streams, UNC shares and \\?\ device paths — are rejected
// so the extension check cannot be smuggled past.
func isExecutableMode(path string, _ os.FileInfo) bool {
	// Anything rooted at a separator is a UNC share, a device path or a
	// drive-relative path; only plain "C:\..." paths are allowed.
	if path == "" || os.IsPathSeparator(path[0]) {
		return false
	}
	if volume := filepath.VolumeName(path); len(volume) != 2 || strings.Contains(path[len(volume):], ":") {
		return false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".com":
		return true
	default:
		return false
	}
}
