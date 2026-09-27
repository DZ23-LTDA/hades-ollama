//go:build !linux

package agent

import (
	"errors"
	"os/exec"
)

type sandboxControl struct{}

func newSandboxControl(string) (*sandboxControl, error) {
	return nil, errors.New("strict sandbox requires a Linux executor with delegated cgroup v2")
}

func trustedSandboxExecutable(string) (string, error) {
	return "", errors.New("sandbox helper resolution is available only on Linux")
}

func configureSandboxCommand(*exec.Cmd, *sandboxControl) error { return nil }
func closeSandboxControl(*sandboxControl)                      {}
func killSandboxControl(*sandboxControl)                       {}
