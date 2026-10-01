//go:build !windows

package agent

import "os"

// isExecutableMode reports whether a regular file can be launched as an MCP
// server. On POSIX systems this is the owner/group/other execute bit.
func isExecutableMode(_ string, info os.FileInfo) bool {
	return info.Mode()&0o111 != 0
}
