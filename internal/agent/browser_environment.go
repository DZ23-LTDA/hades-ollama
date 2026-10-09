package agent

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// BrowserEnvironmentStatus reports whether the local machine can run the
// Playwright-backed browser operator, and — when it cannot — exactly what the
// user needs to do. The goal is plug-and-play: never a cryptic Python error,
// always a clear next step in pt-BR.
type BrowserEnvironmentStatus struct {
	PythonPath        string `json:"python_path,omitempty"`
	PythonOK          bool   `json:"python_ok"`
	PlaywrightOK      bool   `json:"playwright_ok"`
	PlaywrightVersion string `json:"playwright_version,omitempty"`
	ChromiumOK        bool   `json:"chromium_ok"`
	Ready             bool   `json:"ready"`
	// Guidance lists the exact pt-BR steps (and commands) to close each gap.
	Guidance []string `json:"guidance"`
	// CanAutoSetup is true when a python interpreter exists, so the app can run
	// the dependency install on the user's behalf.
	CanAutoSetup bool `json:"can_auto_setup"`
}

// pythonProbe runs `python -c code` and returns trimmed stdout. Injected so the
// readiness logic is unit-testable without a real interpreter.
type pythonProbe func(pythonPath, code string) (string, error)

const (
	probePlaywrightVersion = "import playwright,sys;sys.stdout.write(getattr(playwright,'__version__',''))"
	probeChromiumPath      = "from playwright.sync_api import sync_playwright\nimport os,sys\n" +
		"with sync_playwright() as p:\n    sys.stdout.write('ok' if os.path.exists(p.chromium.executable_path) else 'missing')"
)

// checkBrowserEnvironment is the pure core: given a way to locate python and a
// probe runner, it derives the readiness status and guidance.
func checkBrowserEnvironment(lookPath func(string) (string, error), probe pythonProbe) BrowserEnvironmentStatus {
	status := BrowserEnvironmentStatus{}

	var pythonPath string
	for _, candidate := range []string{"python3", "python"} {
		if p, err := lookPath(candidate); err == nil && p != "" {
			pythonPath = p
			break
		}
	}
	if pythonPath == "" {
		status.Guidance = []string{
			"Instale o Python 3 (python.org ou a loja do sistema) e marque \"Add to PATH\".",
		}
		return status
	}
	status.PythonPath = pythonPath
	status.PythonOK = true
	status.CanAutoSetup = true

	if version, err := probe(pythonPath, probePlaywrightVersion); err == nil {
		status.PlaywrightOK = true
		status.PlaywrightVersion = strings.TrimSpace(version)
	} else {
		status.Guidance = append(status.Guidance,
			"Instale o Playwright: python -m pip install playwright")
	}

	if status.PlaywrightOK {
		if out, err := probe(pythonPath, probeChromiumPath); err == nil && strings.TrimSpace(out) == "ok" {
			status.ChromiumOK = true
		} else {
			status.Guidance = append(status.Guidance,
				"Baixe o navegador Chromium do Playwright: python -m playwright install chromium")
		}
	}

	status.Ready = status.PythonOK && status.PlaywrightOK && status.ChromiumOK
	if status.Ready {
		status.Guidance = nil
	}
	return status
}

// BrowserEnvironment reports the live readiness of the browser operator on this
// machine.
func BrowserEnvironment(ctx context.Context) BrowserEnvironmentStatus {
	return checkBrowserEnvironment(exec.LookPath, func(pythonPath, code string) (string, error) {
		probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out, err := exec.CommandContext(probeCtx, pythonPath, "-c", code).Output()
		return string(out), err
	})
}

// SetupBrowserEnvironment installs the missing browser-operator dependencies on
// this machine using the real environment. It is side-effectful (runs pip and
// downloads Chromium) and must only be invoked on explicit user request.
func SetupBrowserEnvironment(ctx context.Context) BrowserSetupResult {
	status := BrowserEnvironment(ctx)
	run := func(runCtx context.Context, name string, args ...string) (string, error) {
		stepCtx, cancel := context.WithTimeout(runCtx, 10*time.Minute)
		defer cancel()
		out, err := exec.CommandContext(stepCtx, name, args...).CombinedOutput()
		return string(out), err
	}
	return setupBrowserEnvironment(ctx, status, run, func() BrowserEnvironmentStatus {
		return BrowserEnvironment(ctx)
	})
}

// BrowserSetupStep is one command run while provisioning the browser operator.
type BrowserSetupStep struct {
	Description string `json:"description"`
	OK          bool   `json:"ok"`
	Output      string `json:"output,omitempty"`
}

// BrowserSetupResult is the outcome of an auto-setup attempt.
type BrowserSetupResult struct {
	Steps  []BrowserSetupStep       `json:"steps"`
	Status BrowserEnvironmentStatus `json:"status"`
}

// commandRunner runs an external command and returns combined output. Injected
// for tests so setup can be exercised without touching the real environment.
type commandRunner func(ctx context.Context, name string, args ...string) (string, error)

// setupBrowserEnvironment installs the missing dependencies (Playwright + its
// Chromium) using the given python interpreter. It is side-effectful and must
// only run on explicit user request. Returns the per-step results and the
// recomputed status.
func setupBrowserEnvironment(ctx context.Context, status BrowserEnvironmentStatus, run commandRunner, recheck func() BrowserEnvironmentStatus) BrowserSetupResult {
	result := BrowserSetupResult{}
	if !status.PythonOK {
		result.Status = status
		return result
	}
	steps := []struct {
		desc string
		args []string
		skip bool
	}{
		{desc: "Instalando Playwright (pip)", args: []string{"-m", "pip", "install", "--user", "playwright"}, skip: status.PlaywrightOK},
		{desc: "Baixando o Chromium do Playwright", args: []string{"-m", "playwright", "install", "chromium"}, skip: status.ChromiumOK && status.PlaywrightOK},
	}
	for _, step := range steps {
		if step.skip {
			continue
		}
		out, err := run(ctx, status.PythonPath, step.args...)
		result.Steps = append(result.Steps, BrowserSetupStep{
			Description: step.desc,
			OK:          err == nil,
			Output:      strings.TrimSpace(out),
		})
		if err != nil {
			break
		}
	}
	result.Status = recheck()
	return result
}
