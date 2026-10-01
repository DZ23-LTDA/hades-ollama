package launch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ollama/ollama/envconfig"
)

// Gemini implements Runner for Google Gemini CLI integration.
type Gemini struct{}

func (g *Gemini) String() string { return "Gemini CLI" }

func (g *Gemini) args(model string, extra []string) []string {
	var args []string
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, extra...)
	return args
}

func (g *Gemini) findPath() (string, error) {
	if p, err := exec.LookPath("gemini"); err == nil {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	name := "gemini"
	if runtime.GOOS == "windows" {
		name = "gemini.exe"
	}
	fallback := filepath.Join(home, ".local", "bin", name)
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", fmt.Errorf("gemini binary not found")
}

func (g *Gemini) Run(model string, _ []LaunchModel, args []string) error {
	geminiPath, err := g.findPath()
	if err != nil {
		return fmt.Errorf("gemini is not installed, install from https://github.com/google/gemini-cli")
	}

	cmd := exec.Command(geminiPath, g.args(model, args)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = append(os.Environ(), g.envVars(model)...)

	return cmd.Run()
}

func (g *Gemini) envVars(model string) []string {
	env := []string{
		"GEMINI_BASE_URL=" + envconfig.Host().String() + "/v1",
		"GEMINI_API_KEY=ollama",
	}

	if model != "" {
		env = append(env, "GEMINI_MODEL="+model)
	}

	return env
}
