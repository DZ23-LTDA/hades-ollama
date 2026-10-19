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

// alvoDoHost devolve o alvo do CATALOGO correspondente ao sistema onde o teste corre.
//
// *`runtime.GOOS` no macOS e' `darwin`, e o catalogo de alvos diz `macos`.* A primeira versao
// destes testes passava `runtime.GOOS` directamente ao provider -- passava no Windows, onde os dois
// nomes coincidem, e falhava no macOS com `ErrSemProviderParaAlvo`. O CI apanhou-o em tres jobs.
//
//	*Um teste que passa na maquina onde foi escrito e falha nas outras tres nao testa o produto:
//	testa o sistema de quem o escreveu.*
func alvoDoHost() string { return alvoDoHostPara(runtime.GOOS) }

// alvoDoHostPara e' a tabela, separada para o teste poder cobrir os TRES sistemas sem correr neles.
func alvoDoHostPara(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "windows", "linux":
		return goos
	default:
		// Devolve o nome do sistema tal e qual: o provider recusa-o com `ErrSemProviderParaAlvo`, e
		// o teste seguinte documenta que e' isso que acontece. Inventar um alvo aqui esconderia a
		// ausencia de provider atras de um nome plausivel.
		return goos
	}
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
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz})
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
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz})
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
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz, Destino: filepath.Join(raiz, "d1")})
	if err != nil {
		t.Fatalf("primeiro build: %v", err)
	}
	segundo, err := ProviderDeBuildGo{}.Construir(context.Background(),
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz, Destino: filepath.Join(raiz, "d2")})
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
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz})
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
		planoPronto(alvoDoHost()), OpcoesDeBuild{Raiz: raiz})
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
			// *O `Alvo` do plano e o `Alvo` do `CapabilityPlan` tem de ser o MESMO nome do
			// catalogo.* A E-1253 corrigiu o segundo (`plannerPronto(...)`) por substituicao de
			// texto e deixou este a `runtime.GOOS`: no macOS `darwin` != `macos`, o alvo era pulado
			// e o teste falhava com `artefactos = 0`.
			//
			//	*Uma correccao aplicada por substituicao de texto corrige as ocorrencias que a
			//	string descreve, nao as que o defeito tem.*
			{Alvo: alvoDoHost(), Plano: plannerPronto(alvoDoHost())},
			{Alvo: "linux", Plano: plannerBloqueado("linux", "docker")},
			{Alvo: "android", Plano: plannerPronto("android")},
		},
	}
	resultado, err := ProviderDeBuildGo{}.ConstruirAlvos(context.Background(), plano,
		OpcoesDeBuild{Raiz: raiz})
	if err != nil {
		t.Fatalf("construir alvos: %v", err)
	}

	// GUARDA INDEPENDENTE DE PLATAFORMA (E-1257): o `Alvo` que este teste usa tem de ser um alvo do
	// CATALOGO. Sem esta assercao, o teste so' falharia no macOS -- e eu escrevo-o no Windows.
	//
	//	*Um teste cuja falha depende do sistema onde corre e' um teste que so' se descobre na CI
	//	mais lenta.*
	if !(ProviderDeBuildGo{}).cobreAlvo(alvoDoHost()) {
		t.Fatalf("alvoDoHost() = %q nao e' um alvo do catalogo %v: o teste falharia noutro SO",
			alvoDoHost(), ProviderDeBuildGo{}.Alvos())
	}
	if alvoDoHost() == "darwin" {
		t.Fatal("alvoDoHost() devolveu `darwin`; o catalogo diz `macos`")
	}
	if len(resultado.Artefatos) != 1 {
		t.Fatalf("artefactos = %d; esperado 1 (nao construidos: %v)",
			len(resultado.Artefatos), resultado.NaoConstruidos)
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

// O ambiente do `go build` sobrepoe CGO_ENABLED/GOOS/GOARCH e preserva o resto.
//
// *Um build cujo resultado depende do ambiente de quem o chamou nao e' reproduzivel: e' uma funcao
// do ambiente.* Este teste prova a transformacao, e falharia se alguem removesse a sobreposicao.
func TestAmbienteDeBuildSobrepoeODeterministico(t *testing.T) {
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOPATH", "/gopath-de-teste")

	// Alvo vazio: usa o GOOS do HOST. Este teste mede a sobreposicao de CGO/GOPATH, nao a do GOOS
	// (essa tem teste proprio, `TestAmbienteDeBuildUsaOAlvoComoGOOS`).
	ambiente := ambienteDeBuild("")
	juntos := strings.Join(ambiente, "\n")

	if strings.Contains(juntos, "CGO_ENABLED=1") {
		t.Fatalf("CGO_ENABLED=1 do chamador sobreviveu:\n%s", juntos)
	}
	if !strings.Contains(juntos, "CGO_ENABLED=0") {
		t.Fatalf("CGO_ENABLED=0 ausente:\n%s", juntos)
	}
	if strings.Contains(juntos, "GOOS=plan9") {
		t.Fatalf("GOOS=plan9 do chamador sobreviveu -- o CI exporta GOOS e o build herdaria:\n%s", juntos)
	}
	if !strings.Contains(juntos, "GOOS="+runtime.GOOS) {
		t.Fatalf("GOOS do host ausente:\n%s", juntos)
	}
	// O que NAO foi sobreposto tem de ser preservado -- sem isto o GOROOT privado do checkout
	// desapareceria e o build falharia.
	if !strings.Contains(juntos, "GOPATH=/gopath-de-teste") {
		t.Fatalf("GOPATH do chamador foi perdido:\n%s", juntos)
	}
	// Ordem estavel: dois ambientes iguais tem de sair iguais.
	if strings.Join(ambienteDeBuild(""), "\n") != juntos {
		t.Fatal("ambienteDeBuild nao e' deterministico entre chamadas")
	}
}

// A TABELA de mapeamento sistema->alvo, provada sem correr em macOS.
//
// *Um teste que so' passa no sistema onde foi escrito nao prova o mapeamento: prova a coincidencia.*
func TestAlvoDoHostPara(t *testing.T) {
	casos := []struct{ goos, esperado, porque string }{
		{"windows", "windows", "coincide com o catalogo"},
		{"linux", "linux", "coincide com o catalogo"},
		{"darwin", "macos", "NAO coincide: runtime.GOOS diz darwin, o catalogo diz macos"},
		{"freebsd", "freebsd", "sem provider: o nome passa tal e qual para ser recusado"},
	}
	for _, c := range casos {
		if obtido := alvoDoHostPara(c.goos); obtido != c.esperado {
			t.Errorf("alvoDoHostPara(%q) = %q; esperado %q (%s)", c.goos, obtido, c.esperado, c.porque)
		}
	}
	// E `darwin` tem de ser RECUSADO pelo provider: se alguem "corrigir" a tabela devolvendo
	// `darwin` para `macos`, o alvo passaria a existir no provider e o teste acima passaria a
	// mentir. Aqui prova-se o contrario.
	if _, err := (ProviderDeBuildGo{}).Construir(context.Background(),
		planoPronto("darwin"), OpcoesDeBuild{Raiz: t.TempDir()}); !errors.Is(err, ErrSemProviderParaAlvo) {
		t.Fatalf("alvo `darwin` devia ser recusado com ErrSemProviderParaAlvo; obtido %v", err)
	}
}

// helpers

func plannerPronto(alvo string) CapabilityPlan { return CapabilityPlan{Alvo: alvo} }

func plannerBloqueado(alvo, blocker string) CapabilityPlan {
	return CapabilityPlan{Alvo: alvo, Blockers: []string{blocker}}
}

// TestAmbienteDeBuildUsaOAlvoComoGOOS fixa o defeito medido no E-1307.
//
// # O DEFEITO
//
// A versao anterior fixava `GOOS = runtime.GOOS`. Pedir o alvo `linux` num Windows produzia um
// ficheiro chamado `build-alvo-linux` cujos primeiros bytes eram `4D 5A 90 00` (`MZ`, Windows PE).
// Medido no disco, nao inferido.
//
//	*Um artefacto que diz `-linux` e contem `MZ` nao e' um artefacto Linux: e' um artefacto Windows
//	com um nome errado -- e um nome errado viaja mais depressa do que o binario.*
func TestAmbienteDeBuildUsaOAlvoComoGOOS(t *testing.T) {
	casos := map[string]string{
		"linux":   "linux",
		"windows": "windows",
		"macos":   "darwin",
		"LINUX":   "linux", // o catalogo e' case-insensitive em `Plano`
	}
	for alvo, goosEsperado := range casos {
		ambiente := ambienteDeBuild(alvo)
		juntos := strings.Join(ambiente, "\n")
		if !strings.Contains(juntos, "GOOS="+goosEsperado) {
			t.Fatalf("alvo %q: GOOS esperado %q, ambiente:\n%s", alvo, goosEsperado, juntos)
		}
		// O ambiente herdado nao pode sobreviver: e' o que tornava o alvo decorativo.
		if strings.Contains(juntos, "GOOS=plan9") {
			t.Fatalf("alvo %q: o GOOS herdado sobreviveu", alvo)
		}
	}
}

// TestAmbienteDeBuildSemAlvoUsaOHost garante que o smoke nao compila para o alvo errado.
func TestAmbienteDeBuildSemAlvoUsaOHost(t *testing.T) {
	ambiente := ambienteDeBuild("")
	if !strings.Contains(strings.Join(ambiente, "\n"), "GOOS="+runtime.GOOS) {
		t.Fatalf("sem alvo o GOOS devia ser o do host (%s)", runtime.GOOS)
	}
}

// TestNomeDoArtefactoExtensaoSegueOAlvo prova que so' o alvo Windows ganha `.exe`.
//
// # O DEFEITO QUE ISTO PREVINE
//
// No E-1307 o binario Windows foi escrito sem `.exe`. O `cmd` recusou executa-lo: *"nao e'
// reconhecido como um comando interno ou externo"*. Um executavel Windows sem extensao nao e'
// executavel Windows.
func TestNomeDoArtefactoExtensaoSegueOAlvo(t *testing.T) {
	casos := map[string]string{
		"windows": "proj-windows.exe",
		"linux":   "proj-linux",
		"macos":   "proj-macos",
	}
	for alvo, esperado := range casos {
		if obtido := nomeDoArtefacto("/tmp/proj", alvo); obtido != esperado {
			t.Fatalf("alvo %q: esperado %q, obtido %q", alvo, esperado, obtido)
		}
	}
}
