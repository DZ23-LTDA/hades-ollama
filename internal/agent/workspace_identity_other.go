//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows

package agent

import "errors"

func workspaceDirectoryIdentity(string) (string, error) {
	return "", errors.New("stable workspace identity is unavailable on this platform")
}
