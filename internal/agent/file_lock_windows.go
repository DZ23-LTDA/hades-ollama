//go:build windows

package agent

import (
	"context"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// withFileLock serializes updates across processes. The lock file itself is
// never removed: contenders always coordinate on the same persistent file.
func withFileLock(path string, run func() error) error {
	return withFileLockContext(context.Background(), path, run)
}

func withFileLockContext(ctx context.Context, path string, run func() error) error {
	if ctx == nil || path == "" || run == nil {
		return fmt.Errorf("file lock context, path, and callback are required")
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode file lock path: %w", err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return fmt.Errorf("open file lock: %w", err)
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return fmt.Errorf("open file lock: invalid handle")
	}
	defer file.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return fmt.Errorf("stat file lock: %w", err)
	}
	fileType, err := windows.GetFileType(handle)
	if err != nil {
		return fmt.Errorf("inspect file lock type: %w", err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 || fileType != windows.FILE_TYPE_DISK {
		return fmt.Errorf("file lock is not a regular file")
	}
	overlapped := &windows.Overlapped{}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped)
		if err == nil {
			break
		}
		if err != windows.ERROR_LOCK_VIOLATION {
			return fmt.Errorf("acquire file lock: %w", err)
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	defer windows.UnlockFileEx(handle, 0, 1, 0, overlapped) //nolint:errcheck
	return run()
}
