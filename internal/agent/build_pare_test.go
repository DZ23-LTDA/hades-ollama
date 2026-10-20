package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NEGATIVO com a prova mais importante: com o pare acionado, a build recusa **e nao cria nada**.
//
//	*Uma recusa que ja' escreveu no disco e' uma recusa que nao protegeu nada.*
func TestBuildRecusaComPareAcionadoENaoCriaFicheiros(t *testing.T) {
	raiz := t.TempDir()
	destino := filepath.Join(raiz, "dist-que-nao-pode-existir")

	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("vazamento investigado", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}

	provider := ProviderDeBuildGo{}
	_, err := provider.Construir(t.Context(), CapabilityPlan{Alvo: "linux"}, OpcoesDeBuild{
		Raiz:    raiz,
		Destino: destino,
		DryRun:  true,
		Pare:    pare,
	})
	if !errors.Is(err, ErrExecucaoParada) {
		t.Fatalf("esperava ErrExecucaoParada, veio %v", err)
	}
	// A mensagem tem de dizer PORQUE parou, nao so' que parou.
	if !strings.Contains(err.Error(), "vazamento investigado") {
		t.Fatalf("o erro tem de citar o motivo do pare: %q", err.Error())
	}
	// E o filesystem tem de estar intocado.
	if _, errStat := os.Stat(destino); !os.IsNotExist(errStat) {
		t.Fatalf("o destino NAO pode ter sido criado com o pare acionado (stat: %v)", errStat)
	}
}

// POSITIVO: pare presente mas NAO acionado nao atrapalha -- o build segue o caminho normal (dry-run,
// para nao compilar de verdade no teste).
func TestBuildSegueComPareNaoAcionado(t *testing.T) {
	raiz := t.TempDir()
	escreverProjectoMinimo(t, raiz)

	pare := NovoPareDeEmergencia() // desarmado
	provider := ProviderDeBuildGo{}
	resultado, err := provider.Construir(t.Context(), CapabilityPlan{Alvo: "linux"}, OpcoesDeBuild{
		Raiz:   raiz,
		DryRun: true,
		Pare:   pare,
	})
	if err != nil {
		t.Fatalf("com o pare desarmado a build devia seguir: %v", err)
	}
	if !resultado.SemArtefacto {
		t.Fatalf("dry-run nao pode produzir artefacto: %+v", resultado)
	}
}

// COMPATIBILIDADE: `Pare` nil mantem o comportamento anterior (sem travao).
func TestBuildSemPareContinuaAFuncionar(t *testing.T) {
	raiz := t.TempDir()
	escreverProjectoMinimo(t, raiz)

	provider := ProviderDeBuildGo{}
	if _, err := provider.Construir(t.Context(), CapabilityPlan{Alvo: "linux"}, OpcoesDeBuild{Raiz: raiz, DryRun: true}); err != nil {
		t.Fatalf("sem pare a build devia seguir: %v", err)
	}
}

// E o rearme devolve a capacidade de executar -- a transicao inversa tambem tem de ser verdadeira.
func TestRearmarDevolveACapacidadeDeExecutar(t *testing.T) {
	raiz := t.TempDir()
	escreverProjectoMinimo(t, raiz)

	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("teste", "operador"); err != nil {
		t.Fatal(err)
	}
	provider := ProviderDeBuildGo{}
	if _, err := provider.Construir(t.Context(), CapabilityPlan{Alvo: "linux"}, OpcoesDeBuild{Raiz: raiz, DryRun: true, Pare: pare}); !errors.Is(err, ErrExecucaoParada) {
		t.Fatalf("acionado devia recusar, veio %v", err)
	}
	if err := pare.Rearmar("operador", "causa fechada"); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Construir(t.Context(), CapabilityPlan{Alvo: "linux"}, OpcoesDeBuild{Raiz: raiz, DryRun: true, Pare: pare}); err != nil {
		t.Fatalf("depois do rearme devia executar: %v", err)
	}
}

func escreverProjectoMinimo(t *testing.T, directorio string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directorio, "go.mod"),
		[]byte("module exemplo.pt/hades\n\ngo 1.24\n"), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directorio, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatalf("escrever main.go: %v", err)
	}
}

// O caminho MULTI-ALVO tem o mesmo travao, e para antes do primeiro alvo.
func TestConstruirAlvosRecusaComPareAcionado(t *testing.T) {
	raiz := t.TempDir()
	destino := filepath.Join(raiz, "dist-multi-que-nao-pode-existir")
	escreverProjectoMinimo(t, raiz)

	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("paragem multi-alvo", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}

	plano := PlanoDoProjeto{Alvos: []PlanoPorAlvo{{Alvo: "linux"}}}
	_, err := ProviderDeBuildGo{}.ConstruirAlvos(t.Context(), plano, OpcoesDeBuild{
		Raiz:    raiz,
		Destino: destino,
		DryRun:  true,
		Pare:    pare,
	})
	if !errors.Is(err, ErrExecucaoParada) {
		t.Fatalf("esperava ErrExecucaoParada no caminho multi-alvo, veio %v", err)
	}
	if !strings.Contains(err.Error(), "paragem multi-alvo") {
		t.Fatalf("a recusa tem de citar o motivo: %q", err.Error())
	}
	if _, errStat := os.Stat(destino); !os.IsNotExist(errStat) {
		t.Fatalf("o destino NAO pode ter sido criado (stat: %v)", errStat)
	}
}

// O STAGING tambem respeita o pare, e a recusa vem antes de qualquer validacao.
func TestStagingRecusaComPareAcionado(t *testing.T) {
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatalf("criar servico: %v", err)
	}
	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("paragem do staging", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	servico.ComPareDeEmergencia(pare)

	// Sem projecto, sem senha e sem versao publicada -- a recusa tem de ser a do PARE, nao uma
	// validacao qualquer: se fosse a validacao, o travao nao estaria a ser consultado.
	_, errStage := servico.StageFullstack(t.Context(), "projecto-inexistente", "")
	if !errors.Is(errStage, ErrExecucaoParada) {
		t.Fatalf("esperava ErrExecucaoParada (e nao uma validacao), veio %v", errStage)
	}
	if !strings.Contains(errStage.Error(), "paragem do staging") {
		t.Fatalf("a recusa tem de citar o motivo: %q", errStage.Error())
	}
}

// E sem pare ligado o servico continua a validar como antes (compatibilidade).
func TestStagingSemPareContinuaAValidar(t *testing.T) {
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatalf("criar servico: %v", err)
	}
	_, errStage := servico.StageFullstack(t.Context(), "projecto-inexistente", "")
	if errStage == nil {
		t.Fatal("sem pare e sem projecto, o staging tem de falhar por validacao")
	}
	if errors.Is(errStage, ErrExecucaoParada) {
		t.Fatalf("nao pode ser recusa do pare: %v", errStage)
	}
}
