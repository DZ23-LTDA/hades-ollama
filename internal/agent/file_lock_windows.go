//go:build windows

package agent

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// withFileLock serializes updates across processes using a byte-range lock.
// The lock file is persistent so contenders always coordinate on one file.
func withFileLock(path string, run func() error) error {
	if path == "" || run == nil {
		return fmt.Errorf("file lock path and callback are required")
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
	defer windows.CloseHandle(handle)
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
	if err := windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped); err != nil {
		return fmt.Errorf("acquire file lock: %w", err)
	}
	defer windows.UnlockFileEx(handle, 0, 1, 0, overlapped) //nolint:errcheck
	return run()
}
