package agent

// FASE 28.1 do adendo V3, item 1: PROJECT DETECTOR.
//
// A LACUNA MEDIDA
//
// O inventor da FASE 0 encontrou `toolchain_plan.go` com `AlvoDeclarado`, `CapabilityPlan` e
// `ResolverDeToolchain.Plano(ctx, alvo)` -- o preflight funciona, mas **recebe o alvo de fora**.
// Nao existia nada que respondesse a pergunta anterior:
//
//	*De que e' feito este projection, e que alvos ele pode realmente produzir?*
//
// Sem essa pergunta, o `CapabilityPlan` nasce de uma string que alguem escreveu, nao do projection.
//
//	*Um plano de build que comeca com um alvo adivinhado e' um plano para outro projection.*
//
// O QUE ESTE FICHEIRO E', E O QUE NAO E'
//
// E' DETECCAO: le ficheiros de manifesto e devolve stack, gestor de pacotes, targets e ficheiros
// nativos. **Nao instala, nao altera, nao corre nada.** Nao executa `--version` -- isso e' o
// `ResolverDeToolchain`, que continua a ser a autoridade sobre ferramentas presentes.
//
// A SEPARACAO IMPORTA:
//
//	ProjectDetector       "o que este projection PEDE"
//	ResolverDeToolchain   "o que esta maquina TEM"
//	CapabilityPlan        "o que falta para o pedido encontrar o que existe"
//
//	*Tratar um pedido como se fosse uma capacidade instalada e' como tratar uma lista de compras
//	como um armario cheio.*

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StackDeProjecto e' uma stack reconhecida.
type StackDeProjecto string

const (
	StackGo      StackDeProjecto = "go"
	StackNode    StackDeProjecto = "node"
	StackExpo    StackDeProjecto = "expo"
	StackPython  StackDeProjecto = "python"
	StackRust    StackDeProjecto = "rust"
	StackDotNet  StackDeProjecto = "dotnet"
	StackFlutter StackDeProjecto = "flutter"
)

// GestorDePacotes identifica quem manda nas dependencias.
type GestorDePacotes string

const (
	GestorGoModules GestorDePacotes = "go-modules"
	GestorNpm       GestorDePacotes = "npm"
	GestorPnpm      GestorDePacotes = "pnpm"
	GestorYarn      GestorDePacotes = "yarn"
	GestorBun       GestorDePacotes = "bun"
	GestorPoetry    GestorDePacotes = "poetry"
	GestorUv        GestorDePacotes = "uv"
	GestorCargo     GestorDePacotes = "cargo"
	GestorNuGet     GestorDePacotes = "nuget"
	GestorPub       GestorDePacotes = "pub"
)

// ProjectDetector pergunta "de que e' feito este projection".
type ProjectDetector struct {
	// Raiz e' o directorio do projection. Nunca e' escrita.
	Raiz string
	// MaxProfundidade limita a busca de manifests em monorepos.
	MaxProfundidade int
}

// ProjetoDetectado e' o resultado, e e' um FACTOS, nao uma opiniao.
type ProjetoDetectado struct {
	// Raiz e' o caminho absoluto REAL (resolvido), nao o que foi passado.
	Raiz string `json:"raiz"`
	// Stacks estao em ordem deterministica.
	Stacks []StackDeProjecto `json:"stacks"`
	// Gestores estao em ordem deterministica.
	Gestores []GestorDePacotes `json:"gestores"`
	// Alvos sao os targets que o projection DECLARA poder produzir, filtrados pelos que o catalogo
	// do HADES conhece. Ordem deterministica.
	Alvos []string `json:"alvos"`
	// FicheirosNativos sao pastas nativas ja' existentes (android/, ios/). A presenca delas muda a
	// estrategia: *um projection com `android/` versionado nao e' um projection que se regenera com
	// `prebuild` sem destruir trabalho.*
	FicheirosNativos []string `json:"ficheiros_nativos,omitempty"`
	// Manifests sao os ficheiros que justificaram a deteccao, relativos a raiz. Servem para o
	// auditor nao ter de acreditar: *uma deteccao sem prova e' um palpite com formato de resultado*.
	Manifests []string `json:"manifests"`
	// Avisos explicam o que ficou ambiguo.
	Avisos []string `json:"avisos,omitempty"`
}

// ErrProjectoNaoDetectado e' devolvido quando nada foi reconhecido.
var ErrProjectoNaoDetectado = errors.New("nenhuma stack reconhecida no projection")

// manifestsPorStack mapeia ficheiro -> stack. A ordem de verificacao e' a ordem das chaves, e por
// isso nao ha' mapa: um mapa em Go nao tem ordem, e a deteccao tem de ser deterministica.
var manifestsPorStack = []struct {
	ficheiro string
	stack    StackDeProjecto
	gestor   GestorDePacotes
}{
	{"go.mod", StackGo, GestorGoModules},
	{"package.json", StackNode, ""},
	{"pyproject.toml", StackPython, ""},
	{"requirements.txt", StackPython, ""},
	{"Cargo.toml", StackRust, GestorCargo},
	{"pubspec.yaml", StackFlutter, GestorPub},
}

// gestoresNode sao os lockfiles que decidem o gestor. Ordem = prioridade.
var gestoresNode = []struct {
	ficheiro string
	gestor   GestorDePacotes
}{
	{"pnpm-lock.yaml", GestorPnpm},
	{"yarn.lock", GestorYarn},
	{"bun.lockb", GestorBun},
	{"bun.lock", GestorBun},
	{"package-lock.json", GestorNpm},
}

// manifestsPython decidem o gestor Python.
var gestoresPython = []struct {
	ficheiro string
	gestor   GestorDePacotes
}{
	{"poetry.lock", GestorPoetry},
	{"uv.lock", GestorUv},
	{"requirements.txt", ""},
}

// pastasNativas sao as pastas que, se existirem, mudam a estrategia de build.
var pastasNativas = []string{"android", "ios", "macos", "windows", "linux"}

// Detectar le o projection e devolve o que reconheceu.
//
// Nunca escreve, nunca executa. Se nao reconhecer nada devolve `ErrProjectoNaoDetectado` em vez de
// um resultado vazio -- *um projection vazio e um projection irreconhecivel nao sao a mesma coisa, e
// quem chama tem de poder distinguir*.
func (d ProjectDetector) Detectar() (ProjetoDetectado, error) {
	raizBruta := strings.TrimSpace(d.Raiz)
	if raizBruta == "" {
		return ProjetoDetectado{}, fmt.Errorf("%w: raiz vazia", ErrProjectoNaoDetectado)
	}
	// Caminho REAL: um caminho absoluto nao e' o mesmo que um caminho real.
	abs, err := filepath.Abs(raizBruta)
	if err != nil {
		return ProjetoDetectado{}, fmt.Errorf("resolver %q: %w", raizBruta, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return ProjetoDetectado{}, fmt.Errorf("resolver symlinks de %q: %w", abs, err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return ProjetoDetectado{}, fmt.Errorf("estat %q: %w", real, err)
	}
	if !info.IsDir() {
		return ProjetoDetectado{}, fmt.Errorf("%w: %q nao e' um directorio", ErrProjectoNaoDetectado, real)
	}

	profundidade := d.MaxProfundidade
	if profundidade <= 0 {
		profundidade = 3
	}

	det := ProjetoDetectado{Raiz: real}
	stacks := map[StackDeProjecto]bool{}
	gestores := map[GestorDePacotes]bool{}
	manifests := map[string]bool{}
	// problemas acumula o que nao pode ser lido. Nao aborta a deteccao do resto, mas TAMBEM nao
	// desaparece: *um erro que nao aborta nao e' um erro que nao aconteceu.*
	var problemas []string

	// Caminhar com profundidade limitada. *Uma varredura sem limite num monorepo e' uma pausa que
	// ninguem pediu.*
	_ = filepath.WalkDir(real, func(caminho string, entrada os.DirEntry, errCaminhar error) error {
		if errCaminhar != nil {
			// Um ficheiro ilegivel nao aborta a deteccao dos outros -- mas fica registado em
			// `problemas` e sobe para os avisos. Devolver `nil` aqui e' o contrato do
			// `filepath.WalkDir`: o retorno diz "continuar", nao diz "nao houve erro".
			//
			//	*Um erro que nao aborta nao e' um erro que nao aconteceu.*
			problemas = append(problemas, "nao foi possivel ler "+caminho)
			return nil //nolint:nilerr // o nil e' "continuar a caminhada"; o erro foi registado acima.
		}
		rel, errRel := filepath.Rel(real, caminho)
		if errRel != nil {
			problemas = append(problemas, "nao foi possivel calcular o caminho relativo de "+caminho)
			return nil //nolint:nilerr // idem: continuar, com o erro em `problemas`.
		}
		profundidadeAtual := 0
		if rel != "." {
			profundidadeAtual = len(strings.Split(filepath.ToSlash(rel), "/"))
		}
		if entrada.IsDir() {
			// Nao entrar em pastas que nunca contem manifests de projection, e nao descer alem do
			// limite.
			nome := entrada.Name()
			if nome == ".git" || nome == "node_modules" || nome == "vendor" || nome == "_tmp" ||
				nome == "dist" || nome == "build" {
				return filepath.SkipDir
			}
			if profundidadeAtual >= profundidade {
				return filepath.SkipDir
			}
			return nil
		}
		base := entrada.Name()
		for _, m := range manifestsPorStack {
			if base == m.ficheiro {
				stacks[m.stack] = true
				if m.gestor != "" {
					gestores[m.gestor] = true
				}
				manifests[rel] = true
			}
		}
		return nil
	})

	// Node: o lockfile decide o gestor, e o `expo` nas dependencias promove a stack.
	if stacks[StackNode] {
		for _, g := range gestoresNode {
			if _, errStat := os.Stat(filepath.Join(real, g.ficheiro)); errStat == nil {
				// O lockfile ESPECIFICO vence o generico -- mas so' se for OUTRO gestor.
				//
				//	*Uma regra de precedencia que apaga o proprio vencedor nao e' precedencia:
				//	e' um delete com um comentario a dizer que e' regra.*
				//
				// (A primeira versao deste codigo fazia `delete(gestores, GestorNpm)` sem
				// condicao e apagava o npm tambem quando era o npm o lockfile encontrado. O teste
				// `TestDetectarNodeComLockfileDecideGestor` apanhou-o.)
				if g.gestor != GestorNpm {
					delete(gestores, GestorNpm)
				}
				gestores[g.gestor] = true
				manifests[g.ficheiro] = true
				break
			}
		}
		if dados, errLeitura := os.ReadFile(filepath.Join(real, "package.json")); errLeitura == nil {
			texto := string(dados)
			if strings.Contains(texto, `"expo"`) {
				stacks[StackExpo] = true
			}
			if strings.Contains(texto, `"@expo/`) || strings.Contains(texto, `"react-native"`) {
				stacks[StackExpo] = true
			}
		}
	}

	// Python: o lockfile decide o gestor.
	if stacks[StackPython] {
		for _, g := range gestoresPython {
			if _, errStat := os.Stat(filepath.Join(real, g.ficheiro)); errStat == nil {
				if g.gestor != "" {
					gestores[g.gestor] = true
				}
				manifests[g.ficheiro] = true
			}
		}
	}

	// .NET nao tem um ficheiro unico: qualquer *.csproj/*.sln conta.
	entries, errLeituraRaiz := os.ReadDir(real)
	if errLeituraRaiz == nil {
		for _, e := range entries {
			nome := e.Name()
			if strings.HasSuffix(nome, ".csproj") || strings.HasSuffix(nome, ".sln") ||
				strings.HasSuffix(nome, ".fsproj") {
				stacks[StackDotNet] = true
				gestores[GestorNuGet] = true
				manifests[nome] = true
			}
		}
	}

	if len(stacks) == 0 {
		return ProjetoDetectado{}, fmt.Errorf("%w em %q", ErrProjectoNaoDetectado, real)
	}

	// Pastas nativas: a sua presenca muda a estrategia e tem de ser dita.
	for _, pasta := range pastasNativas {
		if info, errStat := os.Stat(filepath.Join(real, pasta)); errStat == nil && info.IsDir() {
			det.FicheirosNativos = append(det.FicheirosNativos, pasta)
		}
	}

	det.Stacks = chavesOrdenadas(stacks)
	det.Gestores = chavesOrdenadas(gestores)
	det.Manifests = chavesOrdenadas(manifests)
	det.Alvos = alvosParaStacks(det.Stacks, det.FicheirosNativos, &det)

	// Problemas de leitura entram nos avisos ANTES dos avisos de estrategia: quem le' precisa de
	// saber primeiro o que a deteccao NAO conseguiu ver.
	det.Avisos = append(det.Avisos, problemas...)

	// Avisos que o chamador precisa de ler ANTES de planear.
	if contains(det.Stacks, StackExpo) && contains(det.FicheirosNativos, "android") {
		det.Avisos = append(det.Avisos,
			"projection Expo com pasta android/ presente: NAO regenerar com prebuild sem revisar o WIP")
	}
	if contains(det.Stacks, StackNode) && !contains(det.Gestores, GestorPnpm) &&
		!contains(det.Gestores, GestorYarn) && !contains(det.Gestores, GestorBun) &&
		!contains(det.Gestores, GestorNpm) {
		det.Avisos = append(det.Avisos,
			"package.json sem lockfile reconhecido: instalacao nao reproduzivel ate haver lockfile")
	}
	return det, nil
}

// alvosParaStacks traduz stacks em alvos do catalogo do HADES.
//
// Nao inventa alvos: um alvo so' entra se o `CatalogoDeAlvos` o conhecer. Um detector que anuncia
// alvos fora do catalogo cria um plano que o resolver nao consegue satisfazer.
func alvosParaStacks(stacks []StackDeProjecto, nativas []string, det *ProjetoDetectado) []string {
	conhecidos := map[string]bool{}
	for _, a := range CatalogoDeAlvos() {
		conhecidos[a.Nome] = true
	}
	candidatos := map[string]bool{}
	for _, s := range stacks {
		switch s {
		case StackGo:
			candidatos["windows"] = true
			candidatos["linux"] = true
			candidatos["macos"] = true
		case StackNode, StackExpo:
			candidatos["web"] = true
			if s == StackExpo || contains(nativas, "android") {
				candidatos["android"] = true
			}
			if s == StackExpo || contains(nativas, "ios") {
				candidatos["ios"] = true
			}
		case StackRust, StackDotNet:
			candidatos["windows"] = true
			candidatos["linux"] = true
			candidatos["macos"] = true
		case StackPython:
			// Python nao gera um alvo de instalador no catalogo actual: nao inventamos um.
			if det != nil {
				det.Avisos = append(det.Avisos,
					"stack Python detectada sem alvo de build no catalogo: nao ha' alvo a planear")
			}
		case StackFlutter:
			candidatos["android"] = true
			candidatos["ios"] = true
		}
	}
	saida := make([]string, 0, len(candidatos))
	for c := range candidatos {
		if conhecidos[c] {
			saida = append(saida, c)
		}
	}
	sort.Strings(saida)
	return saida
}

// chavesOrdenadas devolve as chaves de um mapa em ordem, para o resultado ser determinista.
//
//	*Um resultado cuja ordem muda entre execucoes nao e' o mesmo resultado.*
func chavesOrdenadas[K ~string](m map[K]bool) []K {
	saida := make([]K, 0, len(m))
	for k := range m {
		saida = append(saida, k)
	}
	sort.Slice(saida, func(i, j int) bool { return saida[i] < saida[j] })
	return saida
}

// contains diz se uma lista contem um valor.
func contains[K ~string](lista []K, alvo K) bool {
	for _, v := range lista {
		if v == alvo {
			return true
		}
	}
	return false
}
