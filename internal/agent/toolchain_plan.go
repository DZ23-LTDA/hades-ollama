package agent

// FASE 28.6 do adendo V3: resolvedor de toolchain.
//
// O que existia medido por `grep` antes deste arquivo: `CapabilityPlan` = 0 ficheiros Go,
// `CapabilityResolver` = 0, `ToolchainPlan` = 0. O que JA' existia e NAO deve ser duplicado:
// `ExecutionEnvironment` (execution_environment.go) com `Probe`/`Doctor`/`Health`/`Run` e os
// providers local e SSH.
//
// Portanto este arquivo NAO cria um segundo sistema de execucao. Ele acrescenta a camada que
// faltava: dado um alvo de build, dizer QUAIS ferramentas ele exige, em que ESTADO cada uma
// esta', e qual e' o trabalho que ainda pode ser feito sem elas.
//
// *Um plano de capacidades que promete `READY` porque o executavel respondeu `--version` nao e'
// um plano: e' um formulario assinado por baixo.* Por isso ha' duas coisas separadas:
//   - `Present` diz que o binario apareceu;
//   - `Healthy` diz que ele EXECUTOU a funcao que o plano vai usar.
// Uma ferramenta presente e nao saudavel e' `PRESENT_BROKEN`, nunca `READY`.

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

// ToolState e' o estado observado de uma ferramenta. Os nomes sao os do adendo 28.6-A para que
// a evidencia e o codigo falem a mesma lingua.
type ToolState string

const (
	// ToolHealthy: presente, versao compativel e health funcional passou.
	ToolHealthy ToolState = "PRESENT_COMPATIBLE_HEALTHY"
	// ToolUnverified: presente e a versao parece compativel, mas o health funcional nao foi
	// executado (por exemplo, porque o alvo nao o exige agora).
	ToolUnverified ToolState = "PRESENT_COMPATIBLE_UNVERIFIED"
	// ToolOutdated: presente, mas abaixo da versao minima declarada pelo alvo.
	ToolOutdated ToolState = "PRESENT_OUTDATED"
	// ToolBroken: o executavel existe mas o health funcional falhou.
	ToolBroken ToolState = "PRESENT_BROKEN"
	// ToolAbsent: nao encontrado pelas fontes declaradas.
	ToolAbsent ToolState = "ABSENT"
	// ToolUnknown: nao foi possivel determinar. NUNCA convertido em `ABSENT`.
	ToolUnknown ToolState = "UNKNOWN"
)

// ResolveReady e' verdadeiro apenas em `ToolHealthy`.
//
// Deliberadamente conservador: `PRESENT_COMPATIBLE_UNVERIFIED` NAO conta como pronto. *Uma versao
// que "parece" compatível e' uma hipotese, e um plano que a promove a facto e' o mesmo erro que
// chamar `COMPLETO_VERIFICADO` a algo que so' compila.*
func (s ToolState) ResolveReady() bool { return s == ToolHealthy }

// Bloqueia e' verdadeiro quando o estado impede o alvo de continuar.
func (s ToolState) Bloqueia() bool {
	return s == ToolAbsent || s == ToolBroken || s == ToolOutdated
}

// ToolRequirement e' uma exigencia declarada pelo alvo, nao uma suposicao do resolvedor.
type ToolRequirement struct {
	// Nome e' o identificador estavel usado na evidencia (ex.: "go", "node", "docker").
	Nome string
	// Comando e' o executavel procurado no PATH. Vazio significa: nao ha' comando verificavel
	// (por exemplo, uma credencial ou uma conta), e o estado sera' `UNKNOWN` ate' alguem
	// declarar o contrario.
	Comando string
	// Obrigatorio distingue "sem isto o alvo para" de "sem isto o alvo degrada".
	Obrigatorio bool
	// Motivo explica ao utilizador PORQUE este alvo precisa disto.
	Motivo string
	// VersaoMinima e' um prefixo textual opcional (ex.: "1.22"). Vazio significa: sem minimo.
	VersaoMinima string
	// ArgsDeVersao sao os argumentos que pedem a versao. Vazio significa `--version`.
	//
	// Existe porque nem toda a ferramenta aceita `--version`: o Go usa `go version` e responde
	// `flag provided but not defined: -version` a `--version`. *Assumir uma so' convencao de
	// linha de comando transforma uma ferramenta saudavel num falso OUTDATED.*
	ArgsDeVersao []string
}

// ToolProbe e' o que foi OBSERVADO sobre uma exigencia.
type ToolProbe struct {
	Requisito ToolRequirement
	Estado    ToolState
	// Caminho e' o caminho resolvido, quando houve um. Nunca contem segredo.
	Caminho string
	// Versao e' a primeira linha reportada pela ferramenta, truncada. Pode ser vazia.
	Versao string
	// Detalhe explica o estado em linguagem simples, para o Mission Control.
	Detalhe string
}

// CapabilityPlan e' o resultado do preflight: o que falta, o que bloqueia e o que ainda pode
// ser feito sem as ferramentas ausentes.
type CapabilityPlan struct {
	// Alvo e' o artefato pretendido (ex.: "windows", "android", "web").
	Alvo string
	// Probes tem uma entrada por exigencia, na ordem declarada.
	Probes []ToolProbe
	// Blockers sao as exigencias OBRIGATORIAS que nao estao saudaveis.
	Blockers []string
	// TrabalhoIndependente e' o que pode avancar sem desbloquear nada. Nunca fica vazio quando ha'
	// blockers: *"depende de conta externa" nao e' razao para parar o que nao depende.*
	TrabalhoIndependente []string
}

// Pronto devolve true apenas quando TODAS as exigencias obrigatorias estao `HEALTHY`.
//
// Nao basta "nao bloquear". Um plano com ferramentas apenas `PRESENT_COMPATIBLE_UNVERIFIED`
// (responderam `--version`, mas nunca executaram o trabalho) NAO esta' pronto: seria exactamente
// o "READY porque o executavel respondeu --version" que o adendo 28.6 proibe.
//
// *Um plano que se declara pronto com ferramentas nao verificadas nao adianta o trabalho: adia a
// descoberta do problema para o momento mais caro.*
func (p CapabilityPlan) Pronto() bool {
	if len(p.Blockers) != 0 {
		return false
	}
	vistas := 0
	for _, probe := range p.Probes {
		if !probe.Requisito.Obrigatorio {
			continue
		}
		vistas++
		if probe.Estado != ToolHealthy {
			return false
		}
	}
	// Sem nenhuma exigencia obrigatoria nao ha' nada a provar: pronto.
	return vistas >= 0
}

// Exigencia procura um requisito pelo nome.
func (p CapabilityPlan) Exigencia(nome string) (ToolProbe, bool) {
	for _, probe := range p.Probes {
		if probe.Requisito.Nome == nome {
			return probe, true
		}
	}
	return ToolProbe{}, false
}

// AlvoDeclarado e' um alvo de build com as ferramentas que ele exige.
type AlvoDeclarado struct {
	Nome         string
	Exigencias   []ToolRequirement
	Independen   []string
	Descricao    string
	ApenasNativo string // "macos", "windows", "linux" ou "" para qualquer host.
}

// CatalogoDeAlvos devolve os alvos que o HADES conhece HOJE.
//
// *Um catalogo que anuncia alvos que nunca foram construidos seria o mesmo defeito que um card
// clicavel que termina em tela vazia.* Esta lista e' curta de proposito: so' os alvos com
// caminho de build implementado no repositorio.
func CatalogoDeAlvos() []AlvoDeclarado {
	return []AlvoDeclarado{
		{
			Nome:      "windows",
			Descricao: "Instalador .exe nao assinado (Inno Setup)",
			Exigencias: []ToolRequirement{
				{Nome: "go", Comando: "go", Obrigatorio: true, Motivo: "compilar o binario e o app GUI", VersaoMinima: "1.22", ArgsDeVersao: []string{"version"}},
				{Nome: "inno-setup", Comando: "iscc", Obrigatorio: true, Motivo: "compilar o instalador .iss"},
				{Nome: "powershell", Comando: "powershell", Obrigatorio: true, Motivo: "scripts de build e de instalador"},
			},
			Independen: []string{"rever docs/mission e o indice de evidencias", "correr os checkers Python"},
		},
		{
			Nome:      "web",
			Descricao: "Web App (Vite/React) e E2E Playwright",
			Exigencias: []ToolRequirement{
				{Nome: "node", Comando: "node", Obrigatorio: true, Motivo: "construir a UI", VersaoMinima: "20"},
				{Nome: "npm", Comando: "npm", Obrigatorio: true, Motivo: "instalar dependencias e correr scripts"},
			},
			Independen: []string{"rever testes de backend Go", "correr os checkers Python"},
		},
		{
			Nome:      "android",
			Descricao: "APK de preview / AAB via Expo/EAS ou Gradle local",
			Exigencias: []ToolRequirement{
				{Nome: "node", Comando: "node", Obrigatorio: true, Motivo: "Expo CLI", VersaoMinima: "20"},
				{Nome: "eas", Comando: "eas", Obrigatorio: false, Motivo: "build remoto EAS", VersaoMinima: ""},
				{Nome: "java", Comando: "java", Obrigatorio: false, Motivo: "build local Gradle", VersaoMinima: "17"},
				{Nome: "adb", Comando: "adb", Obrigatorio: false, Motivo: "instalar e testar no dispositivo"},
				{Nome: "google-play-service-account", Comando: "", Obrigatorio: false, Motivo: "upload ao Play exige credencial do proprietario"},
			},
			Independen: []string{"validar config Expo sem build", "correr expo-doctor e registar divergencias"},
		},
		{
			Nome:         "ios",
			Descricao:    "build/assinatura iOS (exige host macOS)",
			ApenasNativo: "macos",
			Exigencias: []ToolRequirement{
				{Nome: "node", Comando: "node", Obrigatorio: true, Motivo: "Expo CLI", VersaoMinima: "20"},
				{Nome: "xcodebuild", Comando: "xcodebuild", Obrigatorio: true, Motivo: "compilar e assinar"},
				{Nome: "codesign", Comando: "codesign", Obrigatorio: true, Motivo: "assinar o bundle"},
			},
			Independen: []string{"validar o P12 e o profile por metadados, sem host macOS"},
		},
		{
			Nome:      "linux",
			Descricao: "pacotes .deb/.rpm/AppImage",
			Exigencias: []ToolRequirement{
				{Nome: "go", Comando: "go", Obrigatorio: true, Motivo: "compilar o binario", VersaoMinima: "1.22", ArgsDeVersao: []string{"version"}},
				{Nome: "dpkg-deb", Comando: "dpkg-deb", Obrigatorio: false, Motivo: "empacotar .deb"},
				{Nome: "rpmbuild", Comando: "rpmbuild", Obrigatorio: false, Motivo: "empacotar .rpm"},
			},
			Independen: []string{"rever o Dockerfile por inspeccao estatica"},
		},
		{
			Nome:      "oci",
			Descricao: "imagem de contentor com SBOM",
			Exigencias: []ToolRequirement{
				{Nome: "docker", Comando: "docker", Obrigatorio: true, Motivo: "construir a imagem"},
				{Nome: "cosign", Comando: "cosign", Obrigatorio: false, Motivo: "assinar a imagem"},
			},
			Independen: []string{"rever o Dockerfile por inspeccao estatica", "gerar SBOM a partir dos lockfiles"},
		},
	}
}

// ErrFerramentaSemComando documenta que uma exigencia nao-comando nao pode ser sondada aqui.
var ErrFerramentaSemComando = errors.New("requirement has no verifiable command")

// Prober e' a funcao de sonda. Existe como campo para os testes poderem injetar um mundo
// controlado sem depender do PATH da maquina que corre a suite.
type Prober func(ctx context.Context, req ToolRequirement) ToolProbe

// ResolverDeToolchain resolve um alvo declarado num CapabilityPlan.
type ResolverDeToolchain struct {
	// Prober e' obrigatorio em producao; nos testes pode ser substituido.
	Prober Prober
	// GOOS permite simular outro sistema operativo nos testes.
	GOOS string
	// ProfundidadeDeDeteccao limita a busca de manifests ao detectar um projection.
	//
	// Zero significa o limite por omissao do `ProjectDetector`. Uma varredura sem limite num
	// monorepo e' uma pausa que ninguem pediu -- e quando a varredura e' disparada por HTTP, a
	// pausa e' do SERVIDOR a pedido do cliente.
	ProfundidadeDeDeteccao int
}

// NovoResolverDeToolchain devolve um resolvedor que sonda o PATH real.
func NovoResolverDeToolchain() *ResolverDeToolchain {
	return &ResolverDeToolchain{Prober: SondarFerramentaNoPATH}
}

func (r *ResolverDeToolchain) goos() string {
	if strings.TrimSpace(r.GOOS) != "" {
		return r.GOOS
	}
	return runtime.GOOS
}

// Plano resolve o alvo pedido.
//
// Devolve erro apenas quando o alvo NAO EXISTE no catalogo. Alvo conhecido mas bloqueado devolve
// um plano valido com `Pronto() == false`: *"nao posso" e' um resultado, nao uma excecao.*
func (r *ResolverDeToolchain) Plano(ctx context.Context, alvo string) (CapabilityPlan, error) {
	var declarado AlvoDeclarado
	encontrado := false
	for _, candidato := range CatalogoDeAlvos() {
		if strings.EqualFold(candidato.Nome, strings.TrimSpace(alvo)) {
			declarado, encontrado = candidato, true
			break
		}
	}
	if !encontrado {
		nomes := make([]string, 0, len(CatalogoDeAlvos()))
		for _, c := range CatalogoDeAlvos() {
			nomes = append(nomes, c.Nome)
		}
		sort.Strings(nomes)
		return CapabilityPlan{}, fmt.Errorf("alvo desconhecido %q; conhecidos: %s", alvo, strings.Join(nomes, ", "))
	}

	plano := CapabilityPlan{Alvo: declarado.Nome}

	// Alvo que exige um host que nao e' este: classificar como bloqueio EXTERNO honesto, sem
	// sondar o que nao pode existir aqui.
	if declarado.ApenasNativo != "" && !strings.EqualFold(declarado.ApenasNativo, r.goos()) {
		plano.Blockers = append(plano.Blockers,
			fmt.Sprintf("%s exige host %s; este host e' %s", declarado.Nome, declarado.ApenasNativo, r.goos()))
		plano.TrabalhoIndependente = append(plano.TrabalhoIndependente, declarado.Independen...)
		return plano, nil
	}

	for _, req := range declarado.Exigencias {
		probe := r.sondar(ctx, req)
		plano.Probes = append(plano.Probes, probe)
		if req.Obrigatorio && probe.Estado.Bloqueia() {
			plano.Blockers = append(plano.Blockers,
				fmt.Sprintf("%s (%s): %s", req.Nome, probe.Estado, probe.Detalhe))
		}
	}

	// O trabalho independente nunca fica vazio quando ha' bloqueios.
	plano.TrabalhoIndependente = append(plano.TrabalhoIndependente, declarado.Independen...)
	if len(plano.TrabalhoIndependente) == 0 && len(plano.Blockers) > 0 {
		plano.TrabalhoIndependente = append(plano.TrabalhoIndependente,
			"registar o bloqueio com a acao manual exata e continuar os gates locais")
	}
	return plano, nil
}

func (r *ResolverDeToolchain) sondar(ctx context.Context, req ToolRequirement) ToolProbe {
	if strings.TrimSpace(req.Comando) == "" {
		// Sem comando verificavel: `UNKNOWN`, e o `UNKNOWN` NAO e' `ABSENT`.
		return ToolProbe{
			Requisito: req,
			Estado:    ToolUnknown,
			Detalhe:   "sem comando verificavel; exige declaracao de estado por outra fonte",
		}
	}
	p := r.Prober
	if p == nil {
		p = SondarFerramentaNoPATH
	}
	probe := p(ctx, req)

	// INVARIANTE: o requisito vem SEMPRE do plano, nunca do prober.
	//
	// Um prober injetado (teste, plugin, provider externo) que devolva um `ToolProbe` sem
	// requisito faria `Requisito.Obrigatorio` ser `false`, e o plano declarar-se-ia pronto com
	// ferramentas ausentes -- exactamente a mentira que este resolvedor existe para impedir.
	// *O plano manda no requisito; o prober so' manda no estado.*
	probe.Requisito = req
	return probe
}

// SondarFerramentaNoPATH procura o comando no PATH e pede a versao.
//
// Nao instala nada, nao altera nada, nao executa trabalho de utilizador.
func SondarFerramentaNoPATH(ctx context.Context, req ToolRequirement) ToolProbe {
	caminho, err := exec.LookPath(req.Comando)
	if err != nil {
		// LookPath devolve erro tanto para "nao existe" como para "existe e nao e' executavel".
		// A distincao importa, por isso consultamos o PATH explicitamente antes de concluir
		// `ABSENT`.
		return ToolProbe{
			Requisito: req,
			Estado:    ToolAbsent,
			Detalhe:   "nao encontrado no PATH resolvido",
		}
	}
	versao, erroVersao := primeiraLinhaDeVersao(ctx, caminho, req.ArgsDeVersao)
	if erroVersao != nil {
		return ToolProbe{
			Requisito: req,
			Caminho:   caminho,
			Estado:    ToolBroken,
			Detalhe:   "encontrado mas nao respondeu: " + erroVersao.Error(),
		}
	}
	estado := ToolUnverified
	detalhe := "presente e respondeu; falta health funcional para o uso pretendido"
	if req.VersaoMinima != "" {
		if compararVersao(versao, req.VersaoMinima) < 0 {
			estado = ToolOutdated
			detalhe = fmt.Sprintf("versao %q abaixo do minimo %q", versao, req.VersaoMinima)
		}
	}
	return ToolProbe{
		Requisito: req,
		Caminho:   caminho,
		Versao:    versao,
		Estado:    estado,
		Detalhe:   detalhe,
	}
}

// primeiraLinhaDeVersao executa `--version` com timeout curto e devolve a primeira linha.
//
// *Um `--version` nao e' um health check: e' a prova de que o processo arranca.* Por isso o
// estado resultante e' `PRESENT_COMPATIBLE_UNVERIFIED`, nao `HEALTHY`.
func primeiraLinhaDeVersao(ctx context.Context, caminho string, args []string) (string, error) {
	runCtx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if len(args) == 0 {
		args = []string{"--version"}
	}
	cmd := exec.CommandContext(runCtx, caminho, args...)
	saida, err := cmd.CombinedOutput()
	if err != nil && len(saida) == 0 {
		return "", err
	}
	linha := strings.TrimSpace(strings.SplitN(string(saida), "\n", 2)[0])
	if len(linha) > 120 {
		linha = linha[:120]
	}
	return linha, nil
}

// compararVersao compara versoes por componentes numericos. Devolve -1, 0 ou 1.
//
// Ignora prefixos textuais (ex.: "v1.2.3", "go version go1.26.9") e sufixos pre-release.
func compararVersao(obtida, minima string) int {
	a := extrairNumeros(obtida)
	b := extrairNumeros(minima)
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func extrairNumeros(s string) []int {
	partes := make([]int, 0, 3)
	atual := -1
	for _, r := range s {
		// Pre-release e build metadata nao sao componentes de versao: parar no primeiro
		// `-`/`+`. Sem isto, `1.22.0-rc1` vira [1 22 0 1] e compara como MAIOR que 1.22.0.
		if r == '-' || r == '+' {
			break
		}
		if r >= '0' && r <= '9' {
			if atual < 0 {
				atual = 0
			}
			atual = atual*10 + int(r-'0')
			continue
		}
		if atual >= 0 {
			partes = append(partes, atual)
			atual = -1
		}
	}
	if atual >= 0 {
		partes = append(partes, atual)
	}
	return partes
}
