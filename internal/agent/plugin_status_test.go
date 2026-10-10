package agent

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestPluginStatusRequiresTrustBeforeAnythingElse(t *testing.T) {
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy()
	installation, err := registry.InstallForOrganization("org-status", PluginManifest{
		ID:         "manifest-plugin",
		Version:    "1.0.0",
		Name:       "Plugin por manifesto",
		Kind:       PluginKindConnector,
		Runtime:    "python",
		Entrypoint: "main.py",
		Scopes:     []string{"sandbox:execute"},
	}, policy)
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewPluginRuntime(registry, policy)
	statuses, err := runtime.StatusForOrganization("org-status")
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("status = %+v", statuses)
	}
	status := statuses[0]
	if status.State != PluginStateUnauthorized {
		t.Fatalf("estado = %q, quer %q (%v)", status.State, PluginStateUnauthorized, status.Reasons)
	}
	if status.ReadyToUse {
		t.Fatal("plugin não confiável não pode ser anunciado como pronto")
	}
	if status.PluginID != installation.Manifest.ID {
		t.Fatalf("plugin id = %q", status.PluginID)
	}
}

func TestPluginStatusDistinguishesUnsupportedUnhealthyAndInstalled(t *testing.T) {
	t.Run("sem runtime declarado é UNSUPPORTED", func(t *testing.T) {
		fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, func(m *PluginManifest) {
			m.Runtime = ""
			m.Entrypoint = ""
		})
		statuses, err := NewPluginRuntime(fixture.registry, fixture.policy).StatusForOrganization("org-runtime")
		if err != nil {
			t.Fatal(err)
		}
		if len(statuses) != 1 || statuses[0].State != PluginStateUnsupported {
			t.Fatalf("status = %+v", statuses)
		}
		if statuses[0].ReadyToUse {
			t.Fatal("sem runtime não pode ser pronto")
		}
	})

	t.Run("conteúdo adulterado é UNHEALTHY", func(t *testing.T) {
		fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
		if err := os.WriteFile(fixture.installation.ContentPath, []byte("bundle adulterado"), 0o600); err != nil {
			t.Fatal(err)
		}
		statuses, err := NewPluginRuntime(fixture.registry, fixture.policy).StatusForOrganization("org-runtime")
		if err != nil {
			t.Fatal(err)
		}
		if len(statuses) != 1 || statuses[0].State != PluginStateUnhealthy {
			t.Fatalf("status = %+v", statuses)
		}
	})

	t.Run("desabilitado é CONFIG_REQUIRED", func(t *testing.T) {
		fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
		if err := fixture.registry.SetEnabledForOrganization("org-runtime", fixture.installation.Manifest.ID, false); err != nil {
			t.Fatal(err)
		}
		statuses, err := NewPluginRuntime(fixture.registry, fixture.policy).StatusForOrganization("org-runtime")
		if err != nil {
			t.Fatal(err)
		}
		if len(statuses) != 1 || statuses[0].State != PluginStateConfigRequired {
			t.Fatalf("status = %+v", statuses)
		}
	})

	t.Run("instalado sem execução é INSTALLED e NÃO é pronto", func(t *testing.T) {
		fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
		statuses, err := NewPluginRuntime(fixture.registry, fixture.policy).StatusForOrganization("org-runtime")
		if err != nil {
			t.Fatal(err)
		}
		if len(statuses) != 1 {
			t.Fatalf("status = %+v", statuses)
		}
		status := statuses[0]
		if status.State != PluginStateInstalled {
			t.Fatalf("estado = %q, quer %q (%v)", status.State, PluginStateInstalled, status.Reasons)
		}
		if status.ReadyToUse {
			t.Fatal("instalado sem execução comprovada não pode ser 'pronto para usar' (§11.3)")
		}
		if !status.Enabled || !status.Trusted || !status.ContentVerified {
			t.Fatalf("status deveria refletir instalação sadia: %+v", status)
		}
		if len(status.Capabilities) == 0 || status.Capabilities[0] != "sandbox:execute" {
			t.Fatalf("capacidades = %v", status.Capabilities)
		}
	})
}

func TestPluginStatusIsTestedLiveOnlyWithRealExecution(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)

	// Tentativa recusada pelo executor NÃO promove o plugin.
	if _, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_status_run")); err != nil {
		if !errors.Is(err, ErrPluginExecutionSandbox) {
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	statuses, err := runtime.StatusForOrganization("org-runtime")
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State == PluginStateTestedLive {
		t.Fatal("execução recusada não pode virar TESTED_LIVE")
	}
	if statuses[0].LastExecution == nil || statuses[0].LastExecution.Succeeded {
		t.Fatalf("registro da tentativa deveria existir e constar como não bem-sucedida: %+v", statuses[0].LastExecution)
	}

	// Execução comprovada (registro real bem-sucedido) promove para TESTED_LIVE.
	runtime.record(PluginExecutionRecord{
		PluginID:    fixture.installation.Manifest.ID,
		Version:     fixture.installation.Manifest.Version,
		Digest:      fixture.installation.ContentSHA256,
		Entrypoint:  "main.py",
		Language:    "python",
		SandboxMode: "strict",
		StepID:      "step_status_ok",
		ExecutedAt:  time.Now().UTC(),
		Isolation:   "sandbox.exec",
		Succeeded:   true,
	})
	statuses, err = runtime.StatusForOrganization("org-runtime")
	if err != nil {
		t.Fatal(err)
	}
	if statuses[0].State != PluginStateTestedLive || !statuses[0].ReadyToUse {
		t.Fatalf("status após execução comprovada = %+v", statuses[0])
	}
}

func TestPluginStatusNeverClaimsConnected(t *testing.T) {
	// A taxonomia tem CONNECTED para vínculo vivo; sem sonda de conexão real o
	// resolvedor não pode produzi-lo — declarar aqui seria fingir integração.
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	statuses, err := NewPluginRuntime(fixture.registry, fixture.policy).StatusForOrganization("org-runtime")
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range statuses {
		if status.State == PluginStateConnected {
			t.Fatalf("CONNECTED não pode ser declarado sem verificação de vínculo: %+v", status)
		}
	}
}
