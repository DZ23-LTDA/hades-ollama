package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// O FILHO GOVERNADO RECEBE UM TEMP GRAVÁVEL — E NÃO O TEMP DO OPERADOR (E-208).
//
// MOTIVO MEDIDO: em ambiente confinado o TEMP do operador pode ser SOMENTE LEITURA. A prova veio do próprio
// toolchain, sem envolver a CLI: `go test` sem redirecionamento de TEMP falhou com
//
//	error writing long arguments to response file: open C:\Users\DZ23\AppData\Local\Temp\args1454007406: Acesso negado
//
// Repassar esse caminho a um CLI filho o faz morrer no próprio bootstrap ao criar artefatos temporários (aliases de
// PATH, arquivos de resposta). O teste é HERMÉTICO: usa o próprio binário de teste como CLI falsa (nada de rede, de
// conta ou de binário externo) e verifica o contrato — o ambiente do filho aponta TEMP/TMP/TMPDIR para um diretório
// DENTRO do diretório da sessão.
func TestFilhoGovernadoRecebeTempGravavel(t *testing.T) {
	binario, err := os.Executable()
	if err != nil {
		t.Skipf("sem caminho do binário de teste: %v", err)
	}
	workdir := t.TempDir()

	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()

	resultado, err := runGovernedCLI(ctx, GovernedCLIRequest{
		Harness:    "codex",
		Prompt:     "apenas prepare o ambiente",
		Workdir:    workdir,
		Timeout:    30 * time.Second,
		MaxOutput:  4096,
		BinaryPath: binario,
		// O `--` encerra o parsing de flags do pacote testing; o que vem depois chega ao helper como argumento dele.
		ArgsOverride: []string{governedCLIHelperRunFlag, "--", "--temp-probe"},
	})
	if err != nil {
		t.Fatalf("a execução governada falhou: %v", err)
	}
	if resultado.ExitCode != 0 {
		t.Fatalf("exit=%d; stderr=%q", resultado.ExitCode, resultado.Stderr)
	}

	saida := strings.TrimSpace(resultado.Stdout)
	if saida == "" {
		t.Fatal("o filho não devolveu o ambiente: não há como provar o contrato")
	}

	raiz, err := filepath.Abs(workdir)
	if err != nil {
		t.Fatal(err)
	}
	prefixo := strings.ToLower(raiz) + strings.ToLower(string(os.PathSeparator))

	vistos := 0
	for _, par := range strings.Split(saida, "|") {
		nome, valor, achou := strings.Cut(par, "=")
		if !achou || strings.TrimSpace(valor) == "" {
			continue
		}
		vistos++
		absoluto, absErr := filepath.Abs(strings.TrimSpace(valor))
		if absErr != nil {
			t.Fatalf("%s inválido (%q): %v", nome, valor, absErr)
		}
		if !strings.HasPrefix(strings.ToLower(absoluto), prefixo) {
			t.Fatalf("%s do filho (%q) precisa ficar DENTRO do diretório da sessão (%q): o TEMP do operador pode ser somente leitura", nome, valor, raiz)
		}
	}
	if vistos == 0 {
		t.Fatalf("nenhuma variável temporária chegou ao filho; saída=%q", saida)
	}
	t.Logf("filho recebeu %d variável(is) temporária(s) dentro da sessão: %q", vistos, saida)

	// O diretório da sessão é temporário: não pode sobrar lixo no diretório de trabalho.
	sobras, err := filepath.Glob(filepath.Join(workdir, ".hades-cli-tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sobras) != 0 {
		t.Fatalf("o diretório temporário da sessão deveria ser removido ao final, mas sobrou: %v", sobras)
	}
}
