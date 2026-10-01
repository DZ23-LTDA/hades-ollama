//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package agent

import (
	"os"
	"path/filepath"
	"syscall"
)

// trustedSystemExecutableDirectory accepts only canonical root-owned directories
// whose ancestors are also root-owned and not writable by group or others.
func trustedSystemExecutableDirectory(path string) (string, bool) {
	resolved, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", false
	}
	canonical, err := filepath.EvalSymlinks(resolved)
	if err != nil || canonical != resolved {
		return "", false
	}
	for current := resolved; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
			return "", false
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 {
			return "", false
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	return resolved, true
}

func trustedSystemExecutableFile(path string, info os.FileInfo) bool {
	if info == nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	directory, ok := trustedSystemExecutableDirectory(filepath.Dir(resolved))
	if !ok || filepath.Dir(resolved) != directory {
		return false
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil || !resolvedInfo.Mode().IsRegular() || resolvedInfo.Mode().Perm()&0o022 != 0 {
		return false
	}
	stat, ok := resolvedInfo.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}
