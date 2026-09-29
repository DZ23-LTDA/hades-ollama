package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMCPRegisterExecutableValidationIsPortable(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, mode os.FileMode) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		return path
	}

	register := func(command string) error {
		return NewMCPManager().Register(MCPServerConfig{ID: "probe", Command: command, TimeoutSeconds: 5})
	}
	rejected := func(t *testing.T, command string) {
		t.Helper()
		err := register(command)
		if err == nil || !strings.Contains(err.Error(), "executable regular file") {
			t.Fatalf("Register(%q) err = %v, want executable rejection", command, err)
		}
	}

	rejected(t, dir)
	rejected(t, filepath.Join(dir, "missing"))

	if runtime.GOOS == "windows" {
		// No execute bit exists on Windows; the extension decides, and batch
		// scripts stay rejected because cmd.exe argument parsing is unsafe.
		rejected(t, write("server.cmd", 0o755))
		rejected(t, write("server.bat", 0o755))
		rejected(t, write("server", 0o755))
		for _, path := range []string{
			`\\server\share\server.exe`,
			`\\?\C:\tools\server.exe`,
			`C:\tools\notes.txt:payload.exe`,
			`server.exe`,
		} {
			if isExecutableMode(path, nil) {
				t.Fatalf("%q must not be accepted as an MCP executable", path)
			}
		}
		for _, name := range []string{"server.exe", "SERVER.EXE", "server.com"} {
			if !isExecutableMode(write(name, 0o644), nil) {
				t.Fatalf("%s should be accepted as an executable image", name)
			}
		}
		return
	}

	rejected(t, write("plain", 0o644))
	info, err := os.Stat(write("server", 0o700))
	if err != nil {
		t.Fatal(err)
	}
	if !isExecutableMode("server", info) {
		t.Fatal("owner-executable file should be accepted")
	}
}
