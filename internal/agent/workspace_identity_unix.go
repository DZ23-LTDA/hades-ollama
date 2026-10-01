//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func workspaceDirectoryIdentity(path string) (string, error) {
	linkInfo, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 || !linkInfo.IsDir() {
		return "", errors.New("workspace must be a real directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return "", err
	}
	defer root.Close()
	openedInfo, err := root.Stat(".")
	if err != nil {
		return "", err
	}
	currentInfo, err := os.Lstat(filepath.Clean(path))
	if err != nil || currentInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(openedInfo, currentInfo) {
		return "", errors.New("workspace changed while its identity was captured")
	}
	stat, ok := openedInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return "", errors.New("workspace filesystem identity is unavailable")
	}
	return fmt.Sprintf("%x:%x", stat.Dev, stat.Ino), nil
}
