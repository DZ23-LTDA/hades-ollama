package agent

import (
	"archive/zip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runtimeBundleFixture cria um bundle zip real (é assim que o conteúdo é
// guardado) com os arquivos pedidos e o assina.
func runtimeBundleFixture(t *testing.T, files map[string]string) (string, SignedArtifact, ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plugin.zip")
	archive, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	signed, err := SignArtifact(path, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	return path, signed, publicKey, privateKey
}

type pluginRuntimeFixture struct {
	registry     *PluginRegistry
	policy       CapabilityPolicy
	installation PluginInstallation
}

func newPluginRuntimeFixture(t *testing.T, files map[string]string, mutate func(*PluginManifest)) pluginRuntimeFixture {
	t.Helper()
	return installPluginRuntimeFixture(t, files, mutate, true)
}

// installPluginRuntimeFixture instala um bundle assinado; `promote` decide se o
// plugin também é promovido a confiável. A promoção é persistida, então um
// plugin instalado e não promovido é o estado real de "não confiável" — mutar
// `Trusted` só em memória não serve, porque o runtime recarrega do disco.
func installPluginRuntimeFixture(t *testing.T, files map[string]string, mutate func(*PluginManifest), promote bool) pluginRuntimeFixture {
	t.Helper()
	path, signed, publicKey, privateKey := runtimeBundleFixture(t, files)
	policy := DefaultCapabilityPolicy().WithTrustedSkillKey("publisher-1", publicKey)
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manifest := PluginManifest{
		ID:           "runtime-plugin",
		Version:      "1.0.0",
		Name:         "Plugin de runtime",
		Kind:         PluginKindBundle,
		Runtime:      "python",
		Entrypoint:   "main.py",
		Scopes:       []string{"sandbox:execute"},
		SigningKeyID: "publisher-1",
	}
	if mutate != nil {
		mutate(&manifest)
	}
	signedManifest, err := SignPluginManifest(manifest, privateKey, "publisher-1")
	if err != nil {
		t.Fatal(err)
	}
	installation, err := registry.InstallBundleForOrganization("org-runtime", signedManifest, path, signed, policy)
	if err != nil {
		t.Fatalf("install bundle: %v", err)
	}
	if _, err := registry.PromoteTrustedForOrganization("org-runtime", installation.Manifest.ID, policy); err != nil {
		t.Fatalf("promote trusted: %v", err)
	}
	installation, err = registry.GetForOrganization("org-runtime", installation.Manifest.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !installation.Trusted || !installation.ContentVerified {
		t.Fatalf("instalação deveria estar confiável e verificada: %+v", installation)
	}
	return pluginRuntimeFixture{registry: registry, policy: policy, installation: installation}
}

func pluginRuntimeRequest(pluginID, stepID string) PluginExecutionRequest {
	return PluginExecutionRequest{
		OrganizationID: "org-runtime",
		PluginID:       pluginID,
		StepID:         stepID,
		SandboxMode:    "strict",
	}
}

func TestPluginRuntimeRefusesDisabledPlugin(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	if err := fixture.registry.SetEnabledForOrganization("org-runtime", fixture.installation.Manifest.ID, false); err != nil {
		t.Fatal(err)
	}
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)
	_, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_disabled"))
	if !errors.Is(err, ErrPluginExecutionDisabled) {
		t.Fatalf("plugin desabilitado deveria ser recusado, got %v", err)
	}
	if len(runtime.AuditTrail()) != 0 {
		t.Fatal("execução recusada não deve gerar registro de execução")
	}
}

func TestPluginRuntimeRefusesUntrustedPlugin(t *testing.T) {
	// Semântica real do servidor: um bundle com assinatura válida já entra
	// confiável. Já a instalação pelo caminho de manifesto (conector/MCP/skill)
	// começa NÃO confiável e só executa após `PromoteTrustedForOrganization`.
	// Este teste cobre exatamente esse estado.
	registry, err := NewPluginRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultCapabilityPolicy()
	manifest := PluginManifest{
		ID:         "manifest-plugin",
		Version:    "1.0.0",
		Name:       "Plugin por manifesto",
		Kind:       PluginKindConnector,
		Runtime:    "python",
		Entrypoint: "main.py",
		Scopes:     []string{"sandbox:execute"},
	}
	installation, err := registry.InstallForOrganization("org-runtime", manifest, policy)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if installation.Trusted {
		t.Fatal("pré-condição: instalação por manifesto não deveria nascer confiável")
	}
	runtime := NewPluginRuntime(registry, policy)
	_, err = runtime.Execute(context.Background(), pluginRuntimeRequest(installation.Manifest.ID, "step_plugin_untrusted"))
	if !errors.Is(err, ErrPluginExecutionUntrusted) {
		t.Fatalf("plugin não confiável deveria ser recusado, got %v", err)
	}
}

func TestPluginRuntimeRefusesTamperedContent(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	// Alterar o bundle guardado depois da verificação precisa ser detectado.
	if err := os.WriteFile(fixture.installation.ContentPath, []byte("bundle adulterado"), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)
	_, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_tampered"))
	if !errors.Is(err, ErrPluginExecutionUnverified) {
		t.Fatalf("conteúdo adulterado deveria ser recusado, got %v", err)
	}
}

func TestPluginRuntimeRefusesEntrypointOutsideManifest(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*PluginManifest)
		request  func(PluginExecutionRequest) PluginExecutionRequest
		wantErrs error
	}{
		{
			name:   "traversal no entrypoint",
			mutate: func(m *PluginManifest) { m.Entrypoint = "../escape.py" },
		},
		{
			name:   "extensão incompatível com o runtime",
			mutate: func(m *PluginManifest) { m.Entrypoint = "main.js" },
		},
		{
			name:    "confirmação divergente",
			mutate:  nil,
			request: func(r PluginExecutionRequest) PluginExecutionRequest { r.EntrypointConfirmation = "outro.py"; return r },
		},
		{
			name:    "runtime divergente",
			mutate:  nil,
			request: func(r PluginExecutionRequest) PluginExecutionRequest { r.RuntimeConfirmation = "node"; return r },
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, testCase.mutate)
			runtime := NewPluginRuntime(fixture.registry, fixture.policy)
			request := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_entry")
			if testCase.request != nil {
				request = testCase.request(request)
			}
			_, err := runtime.Execute(context.Background(), request)
			if !errors.Is(err, ErrPluginExecutionEntrypoint) {
				t.Fatalf("entrypoint inválido deveria ser recusado, got %v", err)
			}
		})
	}
}

func TestPluginRuntimeRequiresScopeGrantedByPolicy(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	// Política vazia: nenhum escopo conhecido, então `sandbox:execute` não é
	// autorizado. Catálogo não concede capacidade.
	runtime := NewPluginRuntime(fixture.registry, CapabilityPolicy{})
	_, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_scope"))
	if !errors.Is(err, ErrPluginExecutionScopeDenied) {
		t.Fatalf("escopo não concedido deveria recusar, got %v", err)
	}
}

func TestPluginRuntimeRequiresApprovedSandboxMode(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)

	semModo := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_mode")
	semModo.SandboxMode = ""
	if _, err := runtime.Execute(context.Background(), semModo); !errors.Is(err, ErrPluginExecutionApproval) {
		t.Fatalf("modo ausente deveria recusar, got %v", err)
	}

	bestEffort := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_mode")
	bestEffort.SandboxMode = "best-effort"
	if _, err := runtime.Execute(context.Background(), bestEffort); !errors.Is(err, ErrPluginExecutionApproval) {
		t.Fatalf("best-effort sem aviso deveria recusar, got %v", err)
	}
}

func TestPluginRuntimeEntrypointMissingFromBundle(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"outro.py": "print('ok')"}, nil)
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)
	_, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_missing"))
	if !errors.Is(err, ErrPluginExecutionEntrypoint) {
		t.Fatalf("entrypoint ausente no bundle deveria recusar, got %v", err)
	}
}

func TestPluginRuntimeRejectsBundleWithSuspiciousPath(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"../evil.py": "print('evil')"}, nil)
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)
	_, err := runtime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_suspicious"))
	if !errors.Is(err, ErrPluginExecutionEntrypoint) {
		t.Fatalf("bundle com caminho suspeito deveria recusar, got %v", err)
	}
}

// TestPluginRuntimeDelegatesToGovernedSandbox prova que, passando todas as
// portas, a execução é DELEGADA ao executor governado e registrada. Onde não há
// executor (host não Linux) ou o sandbox não está configurado, o resultado é
// falha explícita — nunca execução simulada.
// INVARIANTES DE DELEGAÇÃO (E-293): valem nos DOIS desfechos — o plugin executou ou o sandbox recusou. O que este
// teste NÃO faz é aceitar os dois sem distinguir: ele verifica o que precisa ser verdade sempre, e nada mais.
func TestPluginRuntimeRegistraDelegacaoAoSandboxGovernado(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('plugin ok')"}, nil)
	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	result, err := pluginRuntime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_run"))
	// INVARIANTE CENTRAL: sucesso e saída andam JUNTOS. Sucesso silencioso sem saída é exatamente o que não pode passar.
	if (err == nil) != (result.Output != nil) {
		t.Fatalf("sucesso e saída precisam andar juntos: err=%v saída=%v", err, result.Output)
	}
	if result.Record.Isolation != "sandbox.exec" {
		t.Fatalf("registro deveria apontar o executor governado, got %q", result.Record.Isolation)
	}
	if result.Record.PluginID != fixture.installation.Manifest.ID || result.Record.Digest == "" {
		t.Fatalf("registro incompleto: %+v", result.Record)
	}
	trail := pluginRuntime.AuditTrail()
	if len(trail) != 1 {
		t.Fatalf("trilha de auditoria = %d, quer 1", len(trail))
	}
	if trail[0].SandboxMode != "strict" || trail[0].StepID != "step_plugin_run" {
		t.Fatalf("trilha não reflete o pedido: %+v", trail[0])
	}
}

// DESFECHO REAL DA EXECUÇÃO (E-293) — asserção ESTRITA e PORTÁTIL.
//
// ERRO CORRIGIDO AQUI: a primeira versão afirmava "em Linux o executor de namespace existe" e quebrou o CI (o
// container não tem privilégio/kernel para o executor). *Afirmar por SISTEMA OPERACIONAL é afirmar por proxy: o que
// importa é a CAPACIDADE, e ela é observável pelo próprio desfecho da execução.*
//
// O que este teste exige, em QUALQUER host, sem tolerar sucesso silencioso:
//  1. **sucesso e saída andam juntos** — sem erro TEM de haver saída; com erro NÃO pode haver saída;
//  2. em falha, o erro é de **sandbox governado** e **NOMEIA o executor ausente** (o desbloqueio fica explícito);
//  3. em sucesso, a saída vem do **próprio plugin** (execução real de ponta a ponta).
func TestPluginRuntimeAfirmaODesfechoRealDaExecucao(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('plugin ok')"}, nil)
	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	result, err := pluginRuntime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_run"))

	// 1. O invariante central, nos dois desfechos: não existe sucesso silencioso sem saída.
	if (err == nil) != (result.Output != nil) {
		t.Fatalf("sucesso e saída precisam andar juntos: err=%v saída=%v", err, result.Output)
	}
	if err == nil {
		// 3. Execução REAL: a saída precisa vir do plugin.
		if !strings.Contains(strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", result.Output))), "plugin") {
			t.Fatalf("execução bem-sucedida precisa devolver a saída do próprio plugin: %+v", result.Output)
		}
		return
	}
	// 2. Recusa do sandbox governado: precisa ser explícita e explicar o desbloqueio.
	if !errors.Is(err, ErrPluginExecutionSandbox) {
		t.Fatalf("a recusa precisa vir do sandbox governado: %v", err)
	}
	if !strings.Contains(err.Error(), "namespace executor") {
		t.Fatalf("a recusa precisa NOMEAR o executor ausente (procedimento de desbloqueio): %v", err)
	}
	if result.Output != nil {
		t.Fatalf("recusa do sandbox não pode devolver saída: %+v", result.Output)
	}
}

func TestPluginRuntimeRejectsEmptyOrganizationOrPlugin(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{"main.py": "print('ok')"}, nil)
	runtime := NewPluginRuntime(fixture.registry, fixture.policy)
	if _, err := runtime.Execute(context.Background(), PluginExecutionRequest{StepID: "step_plugin_vazio"}); !errors.Is(err, ErrPluginExecutionNotInstalled) {
		t.Fatalf("organização/plugin vazios deveriam recusar, got %v", err)
	}
	// Step id inválido é recusado antes de qualquer leitura de plugin.
	request := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step id inválido")
	if _, err := runtime.Execute(context.Background(), request); err == nil {
		t.Fatal("step id inválido deveria recusar")
	}
	// Plugin inexistente na organização também é recusado.
	request = pluginRuntimeRequest("plugin-que-nao-existe", "step_plugin_ausente")
	if _, err := runtime.Execute(context.Background(), request); !errors.Is(err, ErrPluginExecutionNotInstalled) {
		t.Fatalf("plugin inexistente deveria recusar, got %v", err)
	}
}
