//go:build !linux

package agent

import (
	"context"
	"errors"
	"os"
)

func openWorkspaceSnapshotRepositoryRoot(context.Context, string) (string, *os.Root, error) {
	return "", nil, errors.New("descriptor-bound Git repository discovery is unsupported on this platform")
}
