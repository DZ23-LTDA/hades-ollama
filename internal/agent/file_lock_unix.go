//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package agent

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// withFileLock serializes updates across processes. The lock file itself is
// never removed: unlinking a live lock would allow a second inode to be locked.
func withFileLock(path string, run func() error) error {
	if path == "" || run == nil {
		return fmt.Errorf("file lock path and callback are required")
	}
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("open file lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return fmt.Errorf("open file lock: invalid descriptor")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file lock: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("file lock is not a regular file")
	}
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer unix.Flock(fd, unix.LOCK_UN) //nolint:errcheck
	return run()
}
