package agent

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed browser_helper.py
var browserHelper []byte

type browserOperatorTool struct{}

func (browserOperatorTool) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "browser.operator", Version: "1", Description: "Operar um browser Playwright local com destinos de rede validados", Risk: RiskExternalSideEffect, RequiresApproval: true, Scopes: []string{"browser:navigate", "browser:files", "browser:takeover"}}
}

func (browserOperatorTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	if toolContext.OrganizationID != LocalOrganizationID {
		return ToolResult{}, errors.New("browser operator is restricted to the single-user local organization until a strict OS and network sandbox is configured")
	}
	action := strings.TrimSpace(stringInput(input, "action", ""))
	if action == "" {
		return ToolResult{}, errors.New("browser action is required")
	}
	if action == "navigate" {
		if strings.TrimSpace(stringInput(input, "url", "")) == "" {
			return ToolResult{}, errors.New("browser navigate requires url")
		}
	}
	for _, key := range []string{"path", "save_path"} {
		if value := stringInput(input, key, ""); value != "" {
			if _, err := safeWorkspacePath(toolContext.Workspace, value); err != nil {
				return ToolResult{}, fmt.Errorf("browser %s: %w", key, err)
			}
		}
	}
	request := cloneMap(input)
	request["session_id"] = toolContext.MissionID
	requestData, err := json.Marshal(request)
	if err != nil {
		return ToolResult{}, err
	}
	temp, err := os.CreateTemp("", "ollama-agent-browser-*.py")
	if err != nil {
		return ToolResult{}, err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o700); err != nil {
		_ = temp.Close()
		return ToolResult{}, err
	}
	if _, err := temp.Write(browserHelper); err != nil {
		_ = temp.Close()
		return ToolResult{}, err
	}
	if err := temp.Close(); err != nil {
		return ToolResult{}, err
	}
	deadline, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	pythonExecutable, err := browserPythonExecutable()
	if err != nil {
		return ToolResult{}, err
	}
	command := exec.CommandContext(deadline, pythonExecutable, tempName)
	command.Stdin = bytes.NewReader(requestData)
	command.Env = browserProcessEnvironment(toolContext.Workspace)
	var stdout, stderr bytes.Buffer
	command.Stdout = &limitedBuffer{Buffer: &stdout, Limit: 256 << 10}
	command.Stderr = &limitedBuffer{Buffer: &stderr, Limit: 64 << 10}
	if err := command.Run(); err != nil {
		if stdout.Len() > 0 {
			var failure map[string]any
			if json.Unmarshal(stdout.Bytes(), &failure) == nil {
				return ToolResult{Value: failure}, fmt.Errorf("browser operator: %v", failure["error"])
			}
		}
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return ToolResult{Value: map[string]any{"stderr": message}}, fmt.Errorf("browser operator: %w: %s", err, message)
		}
		return ToolResult{Value: map[string]any{"stderr": message}}, err
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return ToolResult{}, fmt.Errorf("decode browser result: %w", err)
	}
	if result["error"] != nil {
		return ToolResult{Value: result}, fmt.Errorf("browser operator: %v", result["error"])
	}
	return ToolResult{Value: result}, nil
}

func browserProcessEnvironment(workspace string) []string {
	allowed := []string{
		"PATH", "PATHEXT", "SYSTEMROOT", "WINDIR", "TEMP", "TMP",
		"HOME", "USERPROFILE", "LOCALAPPDATA", "APPDATA", "XDG_CACHE_HOME",
		"PLAYWRIGHT_BROWSERS_PATH", "OLLAMA_AGENT_BROWSER_EXECUTABLE",
		"OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "LANG", "LC_ALL", "TZ",
	}
	environment := make([]string, 0, len(allowed)+1)
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok {
			environment = append(environment, name+"="+value)
		}
	}
	return append(environment, "OLLAMA_AGENT_BROWSER_ROOT="+filepath.Join(workspace, ".browser"))
}

func browserPythonExecutable() (string, error) {
	for _, candidate := range []string{"python3", "python"} {
		if executable, err := exec.LookPath(candidate); err == nil {
			return executable, nil
		}
	}
	return "", errors.New("browser operator requires python3 or python on PATH")
}
