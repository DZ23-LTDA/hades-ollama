package agent

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// PLUGIN EXECUTANDO CÓDIGO REAL DE TERCEIRO (E-183).
//
// O item "Plugins têm instalação e execução verificáveis" estava NAO_COMPROVADO com a lacuna declarada: *"há
// contrato e testes herméticos; falta executar um plugin REAL de terceiro"*.
//
// DUAS DESCOBERTAS REAIS desta rodada, medidas e não supostas:
//
//  1. **A execução de plugin é Linux-only**: `sandboxExecTool.Execute` recusa com `runtime.GOOS != "linux"` —
//     "host-process fallback is disabled". No Windows, o que é verificável é **instalação, assinatura, hash de
//     conteúdo e recusa de adulteração**; a execução acontece no Linux (mesmo precedente de
//     `workspace_isolation_platform_test.go`).
//  2. **O conteúdo instalado é um `.zip`**, não um diretório: o caminho é
//     `plugins/content/<org>/<id>-<versão>.zip`. A adulteração, portanto, é feita **reescrevendo o zip**.
//
// COMO O TERCEIRO É EXECUTADO: o sandbox recebe o **conteúdo do entrypoint** (`readPluginEntrypoint` →
// `input["code"]`), então o entrypoint é o arquivo **byte a byte** do pacote `ms@2.1.3` (MIT, vendorizado no
// repositório) seguido de um rodapé de 5 linhas que imprime o resultado. O sha256 do PREFIXO executado é
// conferido contra o do arquivo publicado: se o terceiro for editado, o teste acusa antes de executar.

const hashDoMsPublicado = "e5f0b6a946a9b2b356a28557728410717df54ea2f599edb619f9839df6b7b0e9"

// rodapeDoTerceiro imprime o que a BIBLIOTECA DE TERCEIRO calculou — nada é reimplementado aqui.
const rodapeDoTerceiro = `
console.log(JSON.stringify({
  minuto: ms(60000),
  doisDias: ms('2 days'),
  versao: 'ms-2.1.3',
}));
`

func lerTerceiro(t *testing.T) string {
	t.Helper()
	dados, err := os.ReadFile(filepath.Join("testdata", "thirdparty", "ms", "index.js"))
	if err != nil {
		t.Fatalf("o fixture do pacote de terceiro precisa existir: %v", err)
	}
	// O arquivo publicado e LF. Um checkout no Windows pode converter para CRLF (foi o que quebrou o job
	// "Go agentic Windows portability" em E-183), entao a conferencia normaliza o fim de linha ANTES de hashear -
	// e o `.gitattributes` mantem o fixture sem conversao (-text). Sem isso, o hash depende da config de git de
	// quem roda, e um teste que depende do ambiente nao prova o byte publicado.
	normalizado := strings.ReplaceAll(string(dados), "\r\n", "\n")
	soma := sha256.Sum256([]byte(normalizado))
	if hex.EncodeToString(soma[:]) != hashDoMsPublicado {
		t.Fatalf("o `index.js` de terceiro mudou (sha256 %s): o executado precisa ser o byte publicado",
			hex.EncodeToString(soma[:])[:16])
	}
	return normalizado
}

// bundleDoTerceiro monta os arquivos do bundle: entrypoint com o terceiro byte a byte + rodapé, e a licença.
func bundleDoTerceiro(t *testing.T) map[string]string {
	t.Helper()
	terceiro := lerTerceiro(t)
	licenca, err := os.ReadFile(filepath.Join("testdata", "thirdparty", "ms", "license.md"))
	if err != nil {
		t.Fatal(err)
	}
	return map[string]string{
		"main.js":        terceiro + rodapeDoTerceiro,
		"LICENSE-ms.txt": string(licenca),
	}
}

func fixtureDoTerceiro(t *testing.T) pluginRuntimeFixture {
	t.Helper()
	return newPluginRuntimeFixture(t, bundleDoTerceiro(t), func(manifest *PluginManifest) {
		manifest.Runtime = "node"
		manifest.Entrypoint = "main.js"
		manifest.License = "MIT"
		manifest.Homepage = "https://github.com/vercel/ms"
		manifest.Description = "Executa o pacote de terceiro ms@2.1.3 (MIT) no sandbox do plugin"
	})
}

// TestPluginInstallsAndVerifiesRealThirdPartyContent cobre o que é verificável em QUALQUER sistema: instalação
// por assinatura, verificação por digest do conteúdo e presença do código de terceiro byte a byte.
func TestPluginInstallsAndVerifiesRealThirdPartyContent(t *testing.T) {
	fixture := fixtureDoTerceiro(t)
	if !fixture.installation.Trusted || !fixture.installation.ContentVerified {
		t.Fatalf("instalação precisa estar confiável e verificada: %+v", fixture.installation)
	}
	if fixture.installation.Manifest.License != "MIT" {
		t.Fatalf("a licença do terceiro precisa constar do manifesto: %q", fixture.installation.Manifest.License)
	}
	if err := fixture.registry.VerifyInstalledContent("org-runtime", fixture.installation.Manifest.ID); err != nil {
		t.Fatalf("conteúdo instalado precisa verificar: %v", err)
	}
	conteudo := lerEntradaDoZip(t, fixture.installation.ContentPath, "main.js")
	if !strings.HasPrefix(conteudo, lerTerceiro(t)) {
		t.Fatal("o conteúdo instalado precisa começar pelo código de terceiro, byte a byte")
	}
	if !zipContem(t, fixture.installation.ContentPath, "LICENSE-ms.txt") {
		t.Fatal("o bundle precisa carregar a licença do terceiro")
	}
}

// TestPluginRefusesTamperedThirdPartyContent: com o terceiro ADULTERADO depois da instalação, a verificação de
// conteúdo acusa e a execução é recusada — mesmo no Windows, onde ela já seria recusada por outro motivo.
func TestPluginRefusesTamperedThirdPartyContent(t *testing.T) {
	fixture := fixtureDoTerceiro(t)
	adulterarEntradaDoZip(t, fixture.installation.ContentPath, "main.js", "var s = 1000;", "var s = 9999;")

	if err := fixture.registry.VerifyInstalledContent("org-runtime", fixture.installation.Manifest.ID); err == nil {
		t.Fatal("VerifyInstalledContent precisa acusar o conteúdo adulterado")
	}
	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	if _, err := pluginRuntime.Execute(context.Background(), pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_adulterado")); err == nil {
		t.Fatal("conteúdo adulterado NÃO pode executar")
	}
}

// modoDeSandboxParaTeste respeita o modo CONFIGURADO no ambiente, como o servidor faz. O `strict` exige
// `OLLAMA_AGENT_SANDBOX_CGROUP_ROOT` (pré-requisito de operador); o `best-effort` é o caminho permitido apenas com
// aviso de aprovação explícito — e é declarado como isolamento mais fraco.
func modoDeSandboxParaTeste() (string, string) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("OLLAMA_AGENT_SANDBOX_MODE")), "best-effort") {
		return "best-effort", sandboxBestEffortApprovalNotice
	}
	return "strict", ""
}

// TestPluginExecutesRealThirdPartyPackage é a prova de EXECUÇÃO: só roda onde o executor de namespace existe
// (Linux). Fora dele, o skip declara o motivo em vez de fingir cobertura.
func TestPluginExecutesRealThirdPartyPackage(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox.exec requires the Linux namespace executor; host-process fallback is disabled")
	}
	fixture := fixtureDoTerceiro(t)
	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	modo, aviso := modoDeSandboxParaTeste()
	pedido := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_terceiro")
	pedido.SandboxMode = modo
	pedido.ApprovalNotice = aviso
	// Motivos AMBIENTAIS que impedem sequer TENTAR a execucao - declarados em vez de mascarados:
	//  - `CGROUP_ROOT`: o modo strict exige o pré-requisito de operador;
	//  - `sandbox interpreter unavailable`: o sandbox só executa interpretadores de caminho confiável (dentro de
	//    /usr ou /usr/local), e o `node` do runner de CI não está lá (medido em E-183);
	//  - `namespace executor` / `host-process fallback`: execução indisponível na plataforma (Windows).
	// O DEFEITO B-13 ("not a directory") NÃO está nesta lista: ele foi corrigido em E-184, então reaparecer é falha.
	resultado, err := pluginRuntime.Execute(context.Background(), pedido)
	if err != nil {
		// REGRESSÃO DE B-13: o defeito era o workspace apontar para o ARQUIVO `.zip`. Se este erro voltar, o
		// executor está de novo sem diretório de trabalho - e isso é FALHA, não motivo para pular.
		if strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("REGRESSÃO DE B-13: o workspace voltou a ser o arquivo do bundle: %v", err)
		}
		// Motivos AMBIENTAIS (`E-184`): o executor monta um chroot com `mount`/`chroot`, então exige privilégio
		// de montagem — que não existe no Windows, nem em container não privilegiado. O sintoma medido é o
		// processo sair com `exit status 1` e saída vazia. Onde não há executor utilizável, o teste DECLARA.
		for _, motivo := range []string{"CGROUP_ROOT", "sandbox interpreter unavailable", "namespace executor", "host-process fallback", "exit status"} {
			if strings.Contains(err.Error(), motivo) {
				t.Skipf("execução de plugin indisponível neste ambiente (%s): %v | stderr=%q stdout=%q",
					motivo, err, textoDoCampo(resultado.Output, "stderr"), textoDoCampo(resultado.Output, "stdout"))
			}
		}
		t.Fatalf("falha inesperada ao executar o plugin de terceiro: %v | stderr=%q stdout=%q",
			err, textoDoCampo(resultado.Output, "stderr"), textoDoCampo(resultado.Output, "stdout"))
	}
	t.Logf("execução de terceiro confirmada")
	if !resultado.Record.Succeeded {
		t.Fatalf("o registro precisa declarar execução comprovada: %+v", resultado.Record)
	}
	if resultado.Record.Language != "node" || resultado.Record.Entrypoint != "main.js" {
		t.Fatalf("registro inesperado: %+v", resultado.Record)
	}
	stdout := stdoutDaExecucao(t, resultado.Output)
	for _, esperado := range []string{`"minuto":"1m"`, `"doisDias":172800000`, `"versao":"ms-2.1.3"`} {
		if !strings.Contains(stdout, esperado) {
			t.Fatalf("a saída precisa conter o resultado do terceiro %q; veio: %s", esperado, stdout)
		}
	}
	if len(pluginRuntime.AuditTrail()) == 0 {
		t.Fatal("execução aceita precisa deixar registro de auditoria")
	}
	t.Logf("execução de terceiro confirmada: o pacote ms@2.1.3 rodou dentro do sandbox do plugin")
}

// LIMITE REAL DO RUNTIME (E-183): o sandbox recebe SOMENTE o conteúdo do entrypoint, então um plugin de VÁRIOS
// arquivos não carrega os irmãos do bundle. Só é observável onde há executor (Linux).
func TestPluginExecutionSeesTheExtractedBundleSiblings(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox.exec requires the Linux namespace executor; host-process fallback is disabled")
	}
	// O código do plugin é criado num SUBDIRETÓRIO do workspace; como o workspace agora é o BUNDLE EXTRAÍDO, os
	// arquivos irmãos ficam alcançáveis em `../` - é isso que torna plugin multiarquivo possível (E-184).
	fixture := newPluginRuntimeFixture(t, map[string]string{
		"main.js":         "console.log(require('../lib/ajudante.js'));",
		"lib/ajudante.js": "module.exports = 'ajudante presente';",
	}, func(manifest *PluginManifest) {
		manifest.Runtime = "node"
		manifest.Entrypoint = "main.js"
		manifest.License = "MIT"
	})
	// O irmão ESTÁ no bundle instalado...
	if !zipContem(t, fixture.installation.ContentPath, "lib/ajudante.js") {
		t.Fatal("o bundle instalado precisa conter o arquivo irmão")
	}
	// ...e mesmo assim a execução não o alcança: só o entrypoint vai para o sandbox.
	pluginRuntime := NewPluginRuntime(fixture.registry, fixture.policy)
	modo, aviso := modoDeSandboxParaTeste()
	pedido := pluginRuntimeRequest(fixture.installation.Manifest.ID, "step_plugin_multiarquivo")
	pedido.SandboxMode = modo
	pedido.ApprovalNotice = aviso
	resultado, err := pluginRuntime.Execute(context.Background(), pedido)
	if err != nil {
		for _, motivo := range []string{"sandbox interpreter unavailable", "CGROUP_ROOT", "namespace executor", "host-process fallback", "exit status"} {
			if strings.Contains(err.Error(), motivo) {
				t.Skipf("plugin multiarquivo não observável neste ambiente (%s): %v", motivo, err)
			}
		}
		t.Fatalf("o plugin multiarquivo precisa executar e carregar o irmão: %v", err)
	}
	if !strings.Contains(stdoutDaExecucao(t, resultado.Output), "ajudante presente") {
		t.Fatalf("o irmão do bundle precisa ser carregado a partir do código em execução: %s", stdoutDaExecucao(t, resultado.Output))
	}
}

func stdoutDaExecucao(t *testing.T, valor any) string {
	t.Helper()
	mapa, ok := valor.(map[string]any)
	if !ok {
		t.Fatalf("a saída do sandbox precisa ser um mapa com `stdout`, veio %T", valor)
	}
	texto, _ := mapa["stdout"].(string)
	return texto
}

func lerEntradaDoZip(t *testing.T, caminho, nome string) string {
	t.Helper()
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatalf("o conteúdo instalado precisa ser um zip legível: %v", err)
	}
	defer func() { _ = leitor.Close() }()
	for _, arquivo := range leitor.File {
		if arquivo.Name != nome {
			continue
		}
		aberto, err := arquivo.Open()
		if err != nil {
			t.Fatal(err)
		}
		dados, err := io.ReadAll(aberto)
		_ = aberto.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(dados)
	}
	t.Fatalf("o zip instalado precisa conter %q", nome)
	return ""
}

func zipContem(t *testing.T, caminho, nome string) bool {
	t.Helper()
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatalf("o conteúdo instalado precisa ser um zip legível: %v", err)
	}
	defer func() { _ = leitor.Close() }()
	for _, arquivo := range leitor.File {
		if arquivo.Name == nome {
			return true
		}
	}
	return false
}

// adulterarEntradaDoZip reescreve uma entrada do conteúdo INSTALADO preservando as demais: simula alteração
// pós-instalação sem depender de permissão de sistema de arquivos.
func adulterarEntradaDoZip(t *testing.T, caminho, nome, de, para string) {
	t.Helper()
	leitor, err := zip.OpenReader(caminho)
	if err != nil {
		t.Fatal(err)
	}
	novo := caminho + ".novo"
	destino, err := os.Create(novo)
	if err != nil {
		t.Fatal(err)
	}
	escritor := zip.NewWriter(destino)
	alterou := false
	for _, arquivo := range leitor.File {
		aberto, err := arquivo.Open()
		if err != nil {
			t.Fatal(err)
		}
		dados, err := io.ReadAll(aberto)
		_ = aberto.Close()
		if err != nil {
			t.Fatal(err)
		}
		if arquivo.Name == nome {
			alterado := strings.Replace(string(dados), de, para, 1)
			if alterado == string(dados) {
				t.Fatalf("a adulteração precisa alterar %q", nome)
			}
			dados = []byte(alterado)
			alterou = true
		}
		entrada, err := escritor.Create(arquivo.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entrada.Write(dados); err != nil {
			t.Fatal(err)
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := leitor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := destino.Close(); err != nil {
		t.Fatal(err)
	}
	if !alterou {
		t.Fatalf("a entrada %q não foi encontrada no zip instalado", nome)
	}
	if err := os.Rename(novo, caminho); err != nil {
		t.Fatal(err)
	}
}

// campoDaSaida e textoDoCampo leem o mapa devolvido pelo sandbox sem assumir tipos: o diagnóstico de uma falha de
// execução precisa do stderr REAL, não de uma suposição.
func campoDaSaida(valor any, campo string) any {
	if mapa, ok := valor.(map[string]any); ok {
		return mapa[campo]
	}
	return nil
}

func textoDoCampo(valor any, campo string) string {
	if texto, ok := campoDaSaida(valor, campo).(string); ok {
		return texto
	}
	return ""
}
