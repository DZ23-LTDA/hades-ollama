//go:build !windows

package agent

import (
	"errors"
	"os"
)

func secureWorkspaceSnapshotDirectory(_ string, root *os.Root) error {
	if root == nil {
		return errors.New("snapshot directory handle is required")
	}
	return root.Chmod(".", 0o700)
}
