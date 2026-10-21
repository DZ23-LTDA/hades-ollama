package agent

// FASE 28.1 item 1: testes do ProjectDetector.
//
// O que estes testes travam:
//
//  1. cada stack e' reconhecida pelo manifesto certo;
//  2. o lockfile DECIDE o gestor de pacotes (pnpm > yarn > bun > npm);
//  3. `expo` nas dependencias promove Node a Expo;
//  4. pastas nativas existentes sao REPORTADAS, nao assumidas ausentes;
//  5. a deteccao e' DETERMINISTICA (mesma entrada, mesma ordem);
//  6. nao se inventam alvos fora do catalogo do HADES;
//  7. o detector NAO escreve nada no projection;
//  8. monorepo: encontra manifests em subdirectorios ate ao limite;
//  9. `node_modules`, `.git`, `vendor`, `_tmp` sao ignorados;
// 10. raiz inexistente/nao-directorio da' erro explicito, nao resultado vazio.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// escrever faz um ficheiro com o conteudo dado, criando pastas.
func escreverFicheiroDeTeste(t *testing.T, raiz string, rel string, conteudo string) {
	t.Helper()
	caminho := filepath.Join(raiz, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatalf("mkdir para %q: %v", rel, err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever %q: %v", rel, err)
	}
}

func TestDetectarGo(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module exemplo\n\ngo 1.24\n")

	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.Stacks, StackGo) {
		t.Fatalf("stack Go nao detectada: %v", det.Stacks)
	}
	if !contains(det.Gestores, GestorGoModules) {
		t.Fatalf("go-modules nao detectado: %v", det.Gestores)
	}
	// Go produz alvos de instalador; todos tem de existir no catalogo.
	if len(det.Alvos) == 0 {
		t.Fatal("projection Go sem alvos")
	}
	for _, alvo := range det.Alvos {
		if !alvoExisteNoCatalogo(alvo) {
			t.Fatalf("alvo %q nao existe no catalogo do HADES", alvo)
		}
	}
}

func TestDetectarNodeComLockfileDecideGestor(t *testing.T) {
	casos := []struct {
		lockfile string
		esperado GestorDePacotes
	}{
		{"pnpm-lock.yaml", GestorPnpm},
		{"yarn.lock", GestorYarn},
		{"bun.lock", GestorBun},
		{"package-lock.json", GestorNpm},
	}
	for _, c := range casos {
		t.Run(string(c.esperado), func(t *testing.T) {
			raiz := t.TempDir()
			escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"x","dependencies":{"react":"19"}}`)
			escreverFicheiroDeTeste(t, raiz, c.lockfile, "")
			det, err := ProjectDetector{Raiz: raiz}.Detectar()
			if err != nil {
				t.Fatalf("detectar: %v", err)
			}
			if !contains(det.Gestores, c.esperado) {
				t.Fatalf("gestor %q nao detectado; veio %v", c.esperado, det.Gestores)
			}
		})
	}
}

// O lockfile especifico manda sobre o generico: um repo com pnpm-lock E package-lock nao e' npm.
func TestDetectarLockfileEspecificoVenceONpm(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"x"}`)
	escreverFicheiroDeTeste(t, raiz, "package-lock.json", "")
	escreverFicheiroDeTeste(t, raiz, "pnpm-lock.yaml", "")

	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.Gestores, GestorPnpm) {
		t.Fatalf("pnpm devia estar: %v", det.Gestores)
	}
	for _, g := range det.Gestores {
		if g == GestorNpm {
			t.Fatalf("npm nao pode constar quando ha' pnpm-lock: %v", det.Gestores)
		}
	}
}

func TestDetectarExpoPorDependencia(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"x","dependencies":{"expo":"~53.0.0","react-native":"0.79.6"}}`)
	escreverFicheiroDeTeste(t, raiz, "pnpm-lock.yaml", "")

	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.Stacks, StackExpo) {
		t.Fatalf("Expo nao promovido: %v", det.Stacks)
	}
	// Expo declara android e ios.
	if !contains(det.Alvos, "android") || !contains(det.Alvos, "ios") {
		t.Fatalf("alvos %v; esperado android e ios", det.Alvos)
	}
}

// A presenca de `android/` MUDA a estrategia e tem de vir nos avisos.
func TestDetectarExpoComAndroidVersionadoAvisa(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"dependencies":{"expo":"~53"}}`)
	if err := os.MkdirAll(filepath.Join(raiz, "android", "app"), 0o700); err != nil {
		t.Fatalf("mkdir android: %v", err)
	}

	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.FicheirosNativos, "android") {
		t.Fatalf("android/ nao reportado em ficheiros_nativos: %v", det.FicheirosNativos)
	}
	achou := false
	for _, a := range det.Avisos {
		if contains([]string{"prebuild"}, "prebuild") && len(a) > 0 &&
			(stringContains(a, "prebuild") || stringContains(a, "regenerar")) {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("nenhum aviso sobre regenerar com android/ presente: %v", det.Avisos)
	}
}

func TestDetectarPythonComPoetry(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "pyproject.toml", "[project]\nname = \"x\"\n")
	escreverFicheiroDeTeste(t, raiz, "poetry.lock", "")

	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.Stacks, StackPython) {
		t.Fatalf("Python nao detectado: %v", det.Stacks)
	}
	if !contains(det.Gestores, GestorPoetry) {
		t.Fatalf("poetry nao detectado: %v", det.Gestores)
	}
}

func TestDetectarRustEDotNet(t *testing.T) {
	raizRust := t.TempDir()
	escreverFicheiroDeTeste(t, raizRust, "Cargo.toml", "[package]\nname = \"x\"\n")
	det, err := ProjectDetector{Raiz: raizRust}.Detectar()
	if err != nil {
		t.Fatalf("detectar rust: %v", err)
	}
	if !contains(det.Stacks, StackRust) || !contains(det.Gestores, GestorCargo) {
		t.Fatalf("rust/cargo: stacks=%v gestores=%v", det.Stacks, det.Gestores)
	}

	raizNet := t.TempDir()
	escreverFicheiroDeTeste(t, raizNet, "App.csproj", "<Project/>")
	detNet, err := ProjectDetector{Raiz: raizNet}.Detectar()
	if err != nil {
		t.Fatalf("detectar dotnet: %v", err)
	}
	if !contains(detNet.Stacks, StackDotNet) || !contains(detNet.Gestores, GestorNuGet) {
		t.Fatalf("dotnet/nuget: stacks=%v gestores=%v", detNet.Stacks, detNet.Gestores)
	}
}

// Monorepo: encontra manifests em subdirectorios.
func TestDetectarMonorepo(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "apps/web/package.json", `{"name":"web"}`)
	escreverFicheiroDeTeste(t, raiz, "apps/web/pnpm-lock.yaml", "")
	escreverFicheiroDeTeste(t, raiz, "services/api/go.mod", "module api\n")

	det, err := ProjectDetector{Raiz: raiz, MaxProfundidade: 4}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	if !contains(det.Stacks, StackNode) || !contains(det.Stacks, StackGo) {
		t.Fatalf("monorepo: esperado node+go, veio %v", det.Stacks)
	}
}

// Pastas que nunca contem manifests de projection sao ignoradas: sem isto, um node_modules com
// milhares de package.json tornaria a deteccao lenta e errada.
func TestDetectarIgnoraNodeModulesEGitETmp(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module x\n")
	escreverFicheiroDeTeste(t, raiz, "node_modules/pacote/package.json", `{"name":"dep"}`)
	escreverFicheiroDeTeste(t, raiz, "_tmp/vendor/Cargo.toml", "[package]")
	if err := os.MkdirAll(filepath.Join(raiz, ".git"), 0o700); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	escreverFicheiroDeTeste(t, raiz, ".git/algo/package.json", `{"name":"git"}`)

	det, err := ProjectDetector{Raiz: raiz, MaxProfundidade: 5}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	for _, s := range det.Stacks {
		if s == StackRust {
			t.Fatalf("Cargo.toml dentro de _tmp foi detectado: %v", det.Stacks)
		}
	}
	for _, m := range det.Manifests {
		if stringContains(m, "node_modules") || stringContains(m, ".git") {
			t.Fatalf("manifesto de pasta ignorada entrou: %q", m)
		}
	}
}

// Determinismo: duas execucoes tem de dar a MESMA ordem.
func TestDetectarEDeterministico(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module x\n")
	escreverFicheiroDeTeste(t, raiz, "package.json", `{"name":"x"}`)
	escreverFicheiroDeTeste(t, raiz, "pnpm-lock.yaml", "")
	escreverFicheiroDeTeste(t, raiz, "Cargo.toml", "[package]")

	primeiro, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar: %v", err)
	}
	for i := range 3 {
		seguinte, err := ProjectDetector{Raiz: raiz}.Detectar()
		if err != nil {
			t.Fatalf("detectar %d: %v", i, err)
		}
		if len(seguinte.Stacks) != len(primeiro.Stacks) {
			t.Fatalf("execucao %d: stacks=%v vs %v", i, seguinte.Stacks, primeiro.Stacks)
		}
		for j := range seguinte.Stacks {
			if seguinte.Stacks[j] != primeiro.Stacks[j] {
				t.Fatalf("execucao %d: ordem das stacks mudou: %v vs %v",
					i, seguinte.Stacks, primeiro.Stacks)
			}
		}
		for j := range seguinte.Alvos {
			if j >= len(primeiro.Alvos) || seguinte.Alvos[j] != primeiro.Alvos[j] {
				t.Fatalf("execucao %d: ordem dos alvos mudou: %v vs %v",
					i, seguinte.Alvos, primeiro.Alvos)
			}
		}
	}
}

// O detector NAO escreve nada.
func TestDetectarNaoEscreveNoProjecto(t *testing.T) {
	raiz := t.TempDir()
	escreverFicheiroDeTeste(t, raiz, "go.mod", "module x\n")
	listar := func() []string {
		var nomes []string
		_ = filepath.WalkDir(raiz, func(c string, e os.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				rel, _ := filepath.Rel(raiz, c)
				nomes = append(nomes, rel)
			}
			return nil
		})
		return nomes
	}
	antes := listar()
	if _, err := (ProjectDetector{Raiz: raiz}).Detectar(); err != nil {
		t.Fatalf("detectar: %v", err)
	}
	depois := listar()
	if len(antes) != len(depois) {
		t.Fatalf("o detector alterou o projection: antes=%v depois=%v", antes, depois)
	}
}

// Raiz inexistente ou nao-directorio da' erro, nao resultado vazio.
func TestDetectarRaizInvalidaDaErro(t *testing.T) {
	if _, err := (ProjectDetector{Raiz: filepath.Join(t.TempDir(), "nao-existe")}).Detectar(); err == nil {
		t.Fatal("raiz inexistente devia dar erro")
	}
	raiz := t.TempDir()
	ficheiro := filepath.Join(raiz, "x.txt")
	if err := os.WriteFile(ficheiro, []byte("x"), 0o600); err != nil {
		t.Fatalf("criar ficheiro: %v", err)
	}
	if _, err := (ProjectDetector{Raiz: ficheiro}).Detectar(); err == nil {
		t.Fatal("raiz que e' ficheiro devia dar erro")
	}
	if _, err := (ProjectDetector{Raiz: ""}).Detectar(); err == nil {
		t.Fatal("raiz vazia devia dar erro")
	}
}

// Projection sem nada reconhecivel: erro proprio, nao resultado vazio.
func TestDetectarProjectoVazioDaErroProprio(t *testing.T) {
	raiz := t.TempDir()
	_, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err == nil {
		t.Fatal("projection sem manifests devia dar erro")
	}
	if !errors.Is(err, ErrProjectoNaoDetectado) {
		t.Fatalf("erro = %v; esperado ErrProjectoNaoDetectado", err)
	}
}

// Os alvos detectados tem de existir no catalogo.
func TestDetectarNaoInventaAlvos(t *testing.T) {
	for _, stack := range []struct {
		nome string
		arq  string
		cont string
	}{
		{"go", "go.mod", "module x\n"},
		{"node", "package.json", `{"name":"x"}`},
		{"rust", "Cargo.toml", "[package]"},
		{"dotnet", "App.csproj", "<Project/>"},
	} {
		t.Run(stack.nome, func(t *testing.T) {
			raiz := t.TempDir()
			escreverFicheiroDeTeste(t, raiz, stack.arq, stack.cont)
			det, err := ProjectDetector{Raiz: raiz}.Detectar()
			if err != nil {
				t.Fatalf("detectar: %v", err)
			}
			for _, alvo := range det.Alvos {
				if !alvoExisteNoCatalogo(alvo) {
					t.Fatalf("alvo %q inventado (nao esta' no catalogo)", alvo)
				}
			}
		})
	}
}

// O repo HADES real: detectar a si mesmo e' o teste que nenhum fixture da'.
func TestDetectarORepoHades(t *testing.T) {
	raiz, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(raiz, "go.mod")); err != nil {
		t.Skipf("nao estou dentro do repo HADES (%v); teste de fixture cobre o resto", err)
	}
	det, err := ProjectDetector{Raiz: raiz}.Detectar()
	if err != nil {
		t.Fatalf("detectar o proprio repo: %v", err)
	}
	if !contains(det.Stacks, StackGo) {
		t.Fatalf("o repo HADES nao foi detectado como Go: %v", det.Stacks)
	}
	if len(det.Alvos) == 0 {
		t.Fatal("o repo HADES nao produziu alvos")
	}
	t.Logf("auto-deteccao: stacks=%v gestores=%v alvos=%v nativos=%v",
		det.Stacks, det.Gestores, det.Alvos, det.FicheirosNativos)
}

// === helpers ===

func alvoExisteNoCatalogo(nome string) bool {
	for _, a := range CatalogoDeAlvos() {
		if a.Nome == nome {
			return true
		}
	}
	return false
}

func stringContains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
