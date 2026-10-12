package agent

import (
	"context"
	"strings"
	"testing"
)

// DIAGNÓSTICO DE FALHA DE EXECUÇÃO DE PLUGIN (E-185, limite B-14).
//
// O executor de sandbox SEMPRE devolve `stdout`/`stderr`/`exit_code` no `Value`, inclusive quando o processo falha.
// O runtime de plugin, no caminho de erro, devolvia apenas o registro de auditoria — então QUALQUER falha chegava ao
// operador como "exit status 1" sem uma linha de explicação. Este teste trava o comportamento correto: a falha
// precisa CARREGAR o diagnóstico do sandbox.
//
// O teste roda em qualquer plataforma porque não exige que a execução funcione: ele exige que a FALHA venha
// explicada. Onde o próprio executor recusa antes de rodar (Windows), a checagem de diagnóstico é declarada.
func TestPluginExecutionFailureCarriesSandboxDiagnostics(t *testing.T) {
	fixture := newPluginRuntimeFixture(t, map[string]string{
		// `process.exit(3)` faz o interpretador falhar de propósito: é a falha do CÓDIGO, não do ambiente.
		"main.js": "process.exit(3);",
	}, func(manifest *PluginManifest) {
		manifest.Runtime = "node"
		manifest.Entrypoint = "main.js"
		manifest.License = "MIT"
	})

	modo, aviso := modoDeSandboxParaTeste()
	pedido := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_falha_diagnostico")
	pedido.SandboxMode = modo
	pedido.ApprovalNotice = aviso

	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	resultado, err := pluginRuntime.Execute(context.Background(), pedido)

	// O registro de auditoria precisa dizer a VERDADE sobre o resultado.
	if resultado.Record.Succeeded {
		t.Fatal("uma execução que falhou não pode ser registrada como bem-sucedida")
	}
	if err == nil {
		t.Fatal("a falha do plugin precisa virar erro")
	}
	// O diagnóstico do sandbox precisa chegar junto da falha.
	if resultado.Output == nil {
		// O executor RECUSA antes de rodar quando falta plataforma, privilégio de montagem, interpretador confiável
		// ou cgroup: nesses casos não existe diagnóstico de processo para propagar, e o teste DECLARA em vez de
		// acusar um defeito que não existe. Medido em CI: `sandbox interpreter unavailable for node` acontece
		// ANTES de qualquer processo.
		for _, motivo := range []string{"namespace executor", "host-process fallback", "sandbox interpreter unavailable", "CGROUP_ROOT", "unshare", "Operation not permitted"} {
			if strings.Contains(err.Error(), motivo) {
				t.Skipf("executor de sandbox indisponível neste ambiente, sem diagnóstico a propagar (%s): %v", motivo, err)
			}
		}
		t.Fatalf("a falha precisa carregar o diagnóstico do sandbox (`stdout`/`stderr`/`exit_code`), veio Output nulo: %v", err)
	}
	mapa, ok := resultado.Output.(map[string]any)
	if !ok {
		t.Fatalf("o diagnóstico precisa ser o mapa do sandbox, veio %T", resultado.Output)
	}
	if _, existe := mapa["stderr"]; !existe {
		t.Fatalf("o diagnóstico precisa conter `stderr`; chaves: %v", chavesDoMapa(mapa))
	}
	// O stderr do sandbox é REGISTRADO: é ele que diz ao operador POR QUE a execução não rodou (medido: no container
	// não privilegiado, `unshare: unshare failed: Operation not permitted`).
	t.Logf("diagnóstico do sandbox: exit_code=%v stderr=%q stdout=%q",
		mapa["exit_code"], textoDoCampo(resultado.Output, "stderr"), textoDoCampo(resultado.Output, "stdout"))
	if strings.TrimSpace(textoDoCampo(resultado.Output, "stderr")) == "" && mapa["exit_code"] == nil {
		t.Fatal("a falha precisa trazer stderr OU exit_code: diagnóstico vazio não serve para operar")
	}
	if len(pluginRuntime.AuditTrail()) == 0 {
		t.Fatal("uma execução que falhou também precisa deixar registro de auditoria")
	}
}

func chavesDoMapa(mapa map[string]any) []string {
	chaves := make([]string, 0, len(mapa))
	for chave := range mapa {
		chaves = append(chaves, chave)
	}
	return chaves
}
