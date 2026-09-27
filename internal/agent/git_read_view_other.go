//go:build !linux

package agent

import (
	"errors"
	"os"
)

type gitReadView struct {
	directory string
	gitDir    string
}

func (*gitReadView) Close() error { return nil }

func newGitReadView(*os.Root) (*gitReadView, error) {
	return nil, errors.New("descriptor-bound Git metadata views are unsupported on this platform")
}
