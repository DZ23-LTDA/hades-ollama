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
	return ToolDescriptor{Name: "browser.operator", Version: "1", Description: "Operar um browser Playwright em sessão isolada", Risk: RiskExternalSideEffect, RequiresApproval: true, Scopes: []string{"browser:navigate", "browser:files", "browser:takeover"}}
}

func (browserOperatorTool) Execute(ctx context.Context, toolContext ToolContext, input map[string]any) (ToolResult, error) {
	action := strings.TrimSpace(stringInput(input, "action", ""))
	if action == "" {
		return ToolResult{}, errors.New("browser action is required")
	}
	if action == "navigate" {
		if strings.TrimSpace(stringInput(input, "url", "")) == "" {
			return ToolResult{}, errors.New("browser navigate requires url")
		}
	}
	containedPaths := map[string]string{}
	for _, key := range []string{"path", "save_path"} {
		if value := stringInput(input, key, ""); value != "" {
			contained, err := safeWorkspacePath(toolContext.Workspace, value)
			if err != nil {
				return ToolResult{}, fmt.Errorf("browser %s: %w", key, err)
			}
			containedPaths[key] = contained
		}
	}
	request := cloneMap(input)
	// O helper resolve caminho relativo contra o CWD do servidor, não contra o
	// workspace. Remover as chaves e reinserir só os caminhos já validados
	// garante que um valor null/não-string (que o helper converteria em "None"
	// e escreveria fora do workspace) nunca seja repassado.
	for _, key := range []string{"path", "save_path"} {
		delete(request, key)
	}
	for key, contained := range containedPaths {
		request[key] = contained
	}
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
	command.Env = browserChildEnv(toolContext.Workspace)
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

// browserChildEnv builds the environment for the Playwright helper. The helper
// navigates hostile content, so inheriting the server environment would hand it
// provider API keys and database passwords. It gets the minimal passthrough
// plus the browser subsystem's own non-secret configuration.
func browserChildEnv(workspace string) []string {
	browserEnv := []string{"OLLAMA_AGENT_BROWSER_ROOT=" + filepath.Join(workspace, ".browser")}
	for _, name := range []string{"OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "OLLAMA_AGENT_BROWSER_EXECUTABLE"} {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			browserEnv = append(browserEnv, name+"="+value)
		}
	}
	return minimalChildEnv(browserEnv...)
}

func browserPythonExecutable() (string, error) {
	for _, candidate := range []string{"python3", "python"} {
		if executable, err := exec.LookPath(candidate); err == nil {
			return executable, nil
		}
	}
	return "", errors.New("browser operator requires python3 or python on PATH")
}
