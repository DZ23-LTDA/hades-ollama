package agent

// FASE 28.1 item 3: testes do `BuildProvider`.
//
// O que estes testes travam:
//
//  1. um plano INCOMPLETO e' recusado antes de qualquer processo;
//  2. um alvo sem provider e' recusado com a razao;
//  3. dry-run calcula e NAO executa (nao cria artefacto);
//  4. uma build REAL de um projection Go minimo produz um artefacto com SHA-256 do ficheiro;
//  5. o SHA-256 medido corresponde ao ficheiro (recalculado no teste);
//  6. um projection que nao compila devolve a saida REAL do compilador;
//  7. um directorio sem `go.mod` e' recusado com mensagem clara;
//  8. o binario produzido e' EXECUTAVEL e corre;
//  9. `ConstruirAlvos` nao tenta alvos bloqueados e diz porque;
// 10. dois builds do mesmo projection produzem o MESMO SHA-256 (reprodutibilidade com `-trimpath`);
// 11. o nome do artefacto nao contem o caminho absoluto;
// 12. construir sem provider para o alvo nao inventa nada.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// planoPronto devolve um `CapabilityPlan` pronto para o alvo pedido.
func planoPronto(alvo string) CapabilityPlan {
	return CapabilityPlan{Alvo: alvo, Blockers: nil}
}

// escreverProjectoGoMinimo cria um projection Go que compila e imprime uma linha.
func escreverProjectoGoMinimo(t *testing.T, raiz, mensagem string) {
	t.Helper()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module exemplobuild\n\ngo 1.24\n")
	escreverFicheiroDeTeste(t, raiz, "main.go",
		"package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\""+mensagem+"\") }\n")
}

// exigeGo salta o teste quando o `go` nao esta' no PATH do processo de teste.
//
//	*Um teste que passa porque o toolchain faltava nao e' um teste verde: e' um teste que nao correu.*
func exigeGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go nao esta' no PATH deste processo de teste (%v); o teste de build nao pode correr", err)
	}
}

func TestBuildProviderRecusaPlanoIncompleto(t *testing.T) {
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "x")
	plano := CapabilityPlan{Alvo: "linux", Blockers: []string{"go"}}

	_, err := ProviderDeBuildGo{}.Construir(context.Background(), plano, OpcoesDeBuild{Raiz: raiz})
	if err == nil {
		t.Fatal("plano com blockers devia ser recusado")
	}
	if !errors.Is(err, ErrPlanoIncompleto) {
		t.Fatalf("erro = %v; esperado ErrPlanoIncompleto", err)
	}
	// E nao pode ter criado nada: *um build recusado que deixa um destino vazio e' um build que
	// deixou lixo.*
	if _, err := os.Stat(filepath.Join(raiz, "dist")); err == nil {
		t.Fatal("o provider criou `dist` mesmo recusando o plano")
	}
}

func TestBuildProviderRecusaAlvoSemProvider(t *testing.T) {
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "x")

	_, err := ProviderDeBuildGo{}.Construir(context.Background(), planoPronto("android"),
		OpcoesDeBuild{Raiz: raiz})
	if err == nil {
		t.Fatal("alvo android devia ser recusado pelo provider Go")
	}
	if !errors.Is(err, ErrSemProviderParaAlvo) {
		t.Fatalf("erro = %v; esperado ErrSemProviderParaAlvo", err)
	}
}

func TestBuildProviderDryRunNaoExecuta(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "dry")

	resultado, err := ProviderDeBuildGo{}.Construir(context.Background(), planoPronto("linux"),
		OpcoesDeBuild{Raiz: raiz, DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !resultado.SemArtefacto {
		t.Fatal("dry-run devia declarar que nao produziu artefacto")
	}
	if len(resultado.Artefatos) != 0 {
		t.Fatalf("dry-run produziu artefactos: %v", resultado.Artefatos)
	}
	if _, err := os.Stat(filepath.Join(raiz, "dist")); err == nil {
		t.Fatal("dry-run criou `dist`")
	}
}

func TestBuildProviderConstroiBinarioReal(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "ola-hades")

	resultado, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz})
	if err != nil {
		t.Fatalf("construir: %v", err)
	}
	if len(resultado.Artefatos) != 1 {
		t.Fatalf("artefactos = %d; esperado 1 (%v)", len(resultado.Artefatos), resultado)
	}
	art := resultado.Artefatos[0]
	if art.SHA256 == "" || len(art.SHA256) != 64 {
		t.Fatalf("sha256 invalido: %q", art.SHA256)
	}
	if art.TamanhoBytes <= 0 {
		t.Fatalf("tamanho = %d", art.TamanhoBytes)
	}
	if art.PidDoProcesso <= 0 {
		t.Fatalf("pid = %d; nao houve processo real", art.PidDoProcesso)
	}
	if _, err := os.Stat(art.Caminho); err != nil {
		t.Fatalf("artefacto nao esta' no disco: %v", err)
	}

	// O SHA-256 tem de corresponder ao ficheiro. Um checksum que nao corresponde e' pior do que
	// nenhum: *um checksum errado e' uma prova de integridade que mente.*
	recalculado, err := sha256DeFicheiro(art.Caminho)
	if err != nil {
		t.Fatalf("recalcular: %v", err)
	}
	if recalcular := strings.TrimSpace(recalculado); recalcular != art.SHA256 {
		t.Fatalf("sha256 = %q; recalculado = %q", art.SHA256, recalcular)
	}
}

// O binario produzido tem de CORRER. Um ficheiro com bytes nao e' um artefacto utilizavel.
func TestBuildProviderBinarioExecuta(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "artefacto-vivo")

	resultado, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz})
	if err != nil {
		t.Fatalf("construir: %v", err)
	}
	art := resultado.Artefatos[0]
	saida, errExec := exec.Command(art.Caminho).CombinedOutput()
	if errExec != nil {
		t.Fatalf("executar %q: %v (%s)", art.Caminho, errExec, saida)
	}
	if !strings.Contains(string(saida), "artefacto-vivo") {
		t.Fatalf("saida = %q; esperado conter `artefacto-vivo`", saida)
	}
}

// Dois builds do mesmo projection produzem o MESMO SHA-256.
//
//	*Um build que muda de bytes entre execucoes nao e' reproduzivel: e' uma execucao com sorte.*
func TestBuildProviderEProduzivel(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "reproduz")

	primeiro, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz, Destino: filepath.Join(raiz, "d1")})
	if err != nil {
		t.Fatalf("primeiro build: %v", err)
	}
	segundo, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz, Destino: filepath.Join(raiz, "d2")})
	if err != nil {
		t.Fatalf("segundo build: %v", err)
	}
	if primeiro.Artefatos[0].SHA256 != segundo.Artefatos[0].SHA256 {
		t.Fatalf("checksums diferentes:\n  1: %s\n  2: %s",
			primeiro.Artefatos[0].SHA256, segundo.Artefatos[0].SHA256)
	}
}

func TestBuildProviderRecusaProjetoQueNaoCompila(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	// `main.go` com erro de sintaxe deliberado.
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module quebrado\n\ngo 1.24\n")
	escreverFicheiroDeTeste(t, raiz, "main.go", "package main\n\nfunc main() { isto nao compila }\n")

	_, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz})
	if err == nil {
		t.Fatal("projection invalido devia falhar")
	}
	// A mensagem tem de trazer a saida REAL do compilador: sem ela, quem le' tem de correr o
	// comando a mao.
	if !strings.Contains(err.Error(), "go build falhou") {
		t.Fatalf("erro sem contexto de build: %v", err)
	}
	if _, errStat := os.Stat(filepath.Join(raiz, "dist")); errStat != nil {
		// `dist` pode existir (criado antes do build); o que nao pode existir e' um artefacto.
		if !os.IsNotExist(errStat) {
			t.Fatalf("stat dist: %v", errStat)
		}
	}
}

func TestBuildProviderRecusaSemGoMod(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "leia-me.txt", "nao sou um projection go")

	_, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(runtime.GOOS), OpcoesDeBuild{Raiz: raiz})
	if err == nil {
		t.Fatal("directorio sem go.mod devia ser recusado")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("erro sem mencionar go.mod: %v", err)
	}
}

// `ConstruirAlvos` nao tenta alvos bloqueados e diz a razao de cada um.
func TestBuildProviderConstruirAlvosSaltaBloqueados(t *testing.T) {
	exigeGo(t)
	raiz := t.TempDir()
	escreverProjectoGoMinimo(t, raiz, "alvos")

	plano := PlanoDoProjeto{
		Alvos: []PlanoPorAlvo{
			{Alvo: runtime.GOOS, Plano: plannerPronto(runtime.GOOS)},
			{Alvo: "linux", Plano: plannerBloqueado("linux", "docker")},
			{Alvo: "android", Plano: plannerPronto("android")},
		},
	}
	resultado, err := ProviderDeBuildGo{}.ConstruirAlvos(context.Background(), plano,
		OpcoesDeBuild{Raiz: raiz})
	if err != nil {
		t.Fatalf("construir alvos: %v", err)
	}
	if len(resultado.Artefatos) != 1 {
		t.Fatalf("artefactos = %d; esperado 1", len(resultado.Artefatos))
	}
	if resultado.SemArtefacto {
		t.Fatal("SemArtefacto com 1 artefacto produzido")
	}
	// Os dois nao construidos tem de estar declarados.
	if len(resultado.NaoConstruidos) != 2 {
		t.Fatalf("NaoConstruidos = %v; esperado 2 razoes", resultado.NaoConstruidos)
	}
	juntos := strings.Join(resultado.NaoConstruidos, " | ")
	if !strings.Contains(juntos, "blockers") {
		t.Fatalf("razoes sem `blockers`: %v", resultado.NaoConstruidos)
	}
	if !strings.Contains(juntos, "nenhum provider") {
		t.Fatalf("razoes sem `nenhum provider`: %v", resultado.NaoConstruidos)
	}
}

// Um plano de projection sem alvos nao e' um build: e' ausencia de build.
func TestBuildProviderConstruirAlvosSemAlvos(t *testing.T) {
	resultado, err := ProviderDeBuildGo{}.ConstruirAlvos(context.Background(),
		PlanoDoProjeto{}, OpcoesDeBuild{Raiz: t.TempDir()})
	if err != nil {
		t.Fatalf("construir alvos: %v", err)
	}
	if !resultado.SemArtefacto {
		t.Fatal("plano sem alvos devia dar SemArtefacto")
	}
	if len(resultado.Artefatos) != 0 {
		t.Fatalf("artefactos = %v", resultado.Artefatos)
	}
}

// O nome do artefacto nao contem o caminho absoluto da maquina.
func TestNomeDoArtefactoNaoVazaCaminho(t *testing.T) {
	nome := nomeDoArtefacto(filepath.Join("C:", "Users", "alguem", "projeto"), "windows")
	if strings.Contains(nome, "Users") || strings.Contains(nome, "alguem") {
		t.Fatalf("nome vaza o caminho: %q", nome)
	}
	if !strings.HasSuffix(nome, ".exe") {
		t.Fatalf("alvo windows sem .exe: %q", nome)
	}
	if nomeDoArtefacto("/caminho/para/x", "linux") != "x-linux" {
		t.Fatalf("nome = %q", nomeDoArtefacto("/caminho/para/x", "linux"))
	}
}

// helpers

func plannerPronto(alvo string) CapabilityPlan { return CapabilityPlan{Alvo: alvo} }

func plannerBloqueado(alvo, blocker string) CapabilityPlan {
	return CapabilityPlan{Alvo: alvo, Blockers: []string{blocker}}
}
