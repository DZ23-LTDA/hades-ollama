package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// genericInterpreters are executables whose whole purpose is to run whatever
// they are handed. Registering one as an "MCP server" turns plugin
// registration into arbitrary process execution on the host, so they are
// rejected unless the operator opts in per command.
var genericInterpreters = map[string]bool{
	"sh": true, "bash": true, "dash": true, "zsh": true, "ksh": true, "csh": true,
	"tcsh": true, "fish": true, "ash": true, "busybox": true,
	"powershell": true, "powershell.exe": true, "pwsh": true, "pwsh.exe": true,
	"cmd": true, "cmd.exe": true, "wscript.exe": true, "cscript.exe": true,
	"env": true, "xargs": true, "nohup": true, "setsid": true, "timeout": true,
	"sudo": true, "doas": true, "su": true, "nice": true, "stdbuf": true,
	"perl": true, "ruby": true, "php": true, "lua": true, "tclsh": true,
	"osascript": true, "expect": true,
}

// ErrMCPInterpreterCommand is returned when an MCP registration points at a
// generic interpreter instead of a real MCP server binary.
var ErrMCPInterpreterCommand = errors.New("MCP command is a generic interpreter")

// rejectGenericInterpreterCommand blocks shells and generic runners. The
// operator can still allow a specific absolute path by listing it in
// OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST (comma separated), which keeps the escape
// hatch explicit, auditable and per-path instead of blanket.
func rejectGenericInterpreterCommand(command string) error {
	base := strings.ToLower(filepath.Base(command))
	if !genericInterpreters[base] {
		return nil
	}
	if mcpCommandExplicitlyAllowed(command) {
		return nil
	}
	return fmt.Errorf("%w: %q; register the MCP server binary itself or list this exact path in OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST", ErrMCPInterpreterCommand, base)
}

func mcpCommandExplicitlyAllowed(command string) bool {
	for _, entry := range strings.Split(mcpCommandAllowlistEnv(), ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if entry == command {
			return true
		}
	}
	return false
}

func mcpCommandAllowlistEnv() string {
	return os.Getenv("OLLAMA_AGENT_MCP_COMMAND_ALLOWLIST")
}
