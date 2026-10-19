package agent

// E-1310: o provider do INSTALADOR Windows.
//
// O que estes testes fixam nao e' o caminho feliz -- e' que cada recusa acontece antes de correr o
// Inno Setup, e que o provider NAO adivinha o nome do artefacto.
//
//	*Um artefacto cujo nome o provider adivinha e' um artefacto que o provider pode medir a si mesmo.*

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planoDoInstalador devolve um plano valido para o alvo do instalador.
func planoDoInstalador() CapabilityPlan {
	return CapabilityPlan{
		Alvo: AlvoDoInstaladorWindows,
		Probes: []ToolProbe{
			{
				Requisito: ToolRequirement{Nome: "go", Obrigatorio: true},
				Estado:    ToolUnverified,
			},
		},
	}
}

// TestProviderDeInstaladorSoCobreOInstalador garante que ele nao se declara para os binarios.
func TestProviderDeInstaladorSoCobreOInstalador(t *testing.T) {
	alvos := (ProviderDeInstaladorWindows{}).Alvos()
	if len(alvos) != 1 || alvos[0] != AlvoDoInstaladorWindows {
		t.Fatalf("esperado exactamente [%s], obtido %v", AlvoDoInstaladorWindows, alvos)
	}
	for _, alvo := range alvos {
		if alvo == "windows" || alvo == "linux" || alvo == "macos" {
			t.Fatalf("o provider do instalador nao pode cobrir o alvo de binario %q", alvo)
		}
	}
}

// TestProviderDeInstaladorRecusaBlockers prova que um plano bloqueado nao corre o iscc.
func TestProviderDeInstaladorRecusaBlockers(t *testing.T) {
	plano := planoDoInstalador()
	plano.Blockers = []string{"inno-setup (ABSENT): nao encontrado no PATH resolvido"}

	_, err := (ProviderDeInstaladorWindows{}).Construir(context.Background(), plano,
		OpcoesDeBuild{Raiz: t.TempDir()})
	if !errors.Is(err, ErrPlanoIncompleto) {
		t.Fatalf("esperado ErrPlanoIncompleto, obtido %v", err)
	}
	// A mensagem nomeia a ferramenta que falta, nao diz "0 blockers".
	if !strings.Contains(err.Error(), "inno-setup") {
		t.Fatalf("o erro devia nomear a ferramenta ausente: %v", err)
	}
}

// TestProviderDeInstaladorRecusaAlvoErrado prova que ele nao serve o alvo `windows`.
func TestProviderDeInstaladorRecusaAlvoErrado(t *testing.T) {
	plano := planoDoInstalador()
	plano.Alvo = "windows"
	_, err := (ProviderDeInstaladorWindows{}).Construir(context.Background(), plano,
		OpcoesDeBuild{Raiz: t.TempDir()})
	if !errors.Is(err, ErrSemProviderParaAlvo) {
		t.Fatalf("esperado ErrSemProviderParaAlvo, obtido %v", err)
	}
}

// TestProviderDeInstaladorRecusaSemScriptISS prova que um projecto sem .iss e' recusado.
func TestProviderDeInstaladorRecusaSemScriptISS(t *testing.T) {
	_, err := (ProviderDeInstaladorWindows{}).Construir(context.Background(), planoDoInstalador(),
		OpcoesDeBuild{Raiz: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), ".iss") {
		t.Fatalf("esperado erro a nomear a ausencia de .iss, obtido %v", err)
	}
}

// TestProviderDeInstaladorRecusaVariosScripts prova que ele nao escolhe por nos.
//
// Escolher o primeiro por ordem alfabetica seria uma decisao do provider sobre o conteudo do
// utilizador -- e um provider que escolhe o script tambem escolheria o instalador.
func TestProviderDeInstaladorRecusaVariosScripts(t *testing.T) {
	dir := t.TempDir()
	for _, nome := range []string{"a.iss", "b.iss"} {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte("[Setup]\n"), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	_, err := (ProviderDeInstaladorWindows{}).Construir(context.Background(), planoDoInstalador(),
		OpcoesDeBuild{Raiz: dir})
	if err == nil || !strings.Contains(err.Error(), "2 ficheiros .iss") {
		t.Fatalf("esperado erro a nomear os dois scripts, obtido %v", err)
	}
}

// TestProviderDeInstaladorDryRunNaoExecuta prova que o dry-run devolve o comando sem correr nada.
//
// Tambem prova que o dry-run nao cria o destino: um dry-run que escreve no disco nao e' um dry-run.
func TestProviderDeInstaladorDryRunNaoExecuta(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "teste.iss"), []byte("[Setup]\n"), 0o600); err != nil {
		t.Fatalf("escrever .iss: %v", err)
	}
	destino := filepath.Join(dir, "dist-nao-deve-existir")

	resultado, err := (ProviderDeInstaladorWindows{}).Construir(context.Background(), planoDoInstalador(),
		OpcoesDeBuild{Raiz: dir, Destino: destino, DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !strings.Contains(resultado.Explicacao, "iscc") {
		t.Fatalf("a explicacao devia conter o comando: %q", resultado.Explicacao)
	}
	if len(resultado.Artefatos) != 0 {
		t.Fatalf("um dry-run nao produz artefactos, obtidos %d", len(resultado.Artefatos))
	}
	if _, errStat := os.Stat(destino); !errors.Is(errStat, os.ErrNotExist) {
		t.Fatal("um dry-run nao pode criar o directorio de destino")
	}
}

// TestDiferencaDeListas isola a funcao que decide qual artefacto e' NOVO.
//
//	*Sem esta funcao, o provider devolveria o primeiro .exe do dist -- que pode ser de uma build
//
// anterior.*
func TestDiferencaDeListas(t *testing.T) {
	casos := []struct {
		nome          string
		antes, depois []string
		esperado      int
		explicacao    string
	}{
		{"nada antes, um depois", nil, []string{"novo.exe"}, 1, "o caso normal"},
		{"um antes, o mesmo depois", []string{"velho.exe"}, []string{"velho.exe"}, 0, "nada novo"},
		{"um antes, dois depois", []string{"velho.exe"}, []string{"velho.exe", "novo.exe"}, 1, "so' o novo"},
		{"dois antes, nenhum depois", []string{"a.exe", "b.exe"}, nil, 0, "tudo desapareceu"},
	}
	for _, caso := range casos {
		obtido := diferencaDeListas(caso.depois, caso.antes)
		if len(obtido) != caso.esperado {
			t.Fatalf("%s: esperado %d novo(s), obtidos %d (%v) -- %s",
				caso.nome, caso.esperado, len(obtido), obtido, caso.explicacao)
		}
	}
}

// TestScriptsInnoSoVeissNaRaiz prova que a descoberta nao desce a subdirectorios nem aceita
// extensoes parecidas.
func TestScriptsInnoSoVeissNaRaiz(t *testing.T) {
	dir := t.TempDir()
	ficheiros := map[string]string{
		"bom.iss":       "[Setup]\n",
		"outro.ISS":     "[Setup]\n", // case-insensitive, como o Windows
		"nao-e.txt":     "x",
		"parecido.iss2": "x",
	}
	for nome, conteudo := range ficheiros {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte(conteudo), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "ignorado.iss"), []byte("[Setup]\n"), 0o600); err != nil {
		t.Fatalf("escrever sub: %v", err)
	}

	scripts, err := scriptsInno(dir)
	if err != nil {
		t.Fatalf("scriptsInno: %v", err)
	}
	if len(scripts) != 2 {
		t.Fatalf("esperados 2 scripts (.iss e .ISS), obtidos %d: %v", len(scripts), scripts)
	}
	for _, nome := range scripts {
		if strings.EqualFold(nome, "ignorado.iss") {
			t.Fatal("um .iss em subdirectorio nao pode ser descoberto na raiz")
		}
	}
}
