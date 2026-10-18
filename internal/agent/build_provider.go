package agent

// FASE 28.1 item 3 do adendo V3: `BuildProvider` -- o adaptador que PRODUZ o artefacto.
//
// A LACUNA QUE ESTA FATIA FECHA
//
// As rondas anteriores construiram a volta de um artefacto que nunca foi produzido: o detector
// (E-1245), o plano (E-1246), a rota (E-1247), o painel (E-1248), o E2E (E-1249/E-1250). O
// `BuildProvider` -- o item 3 dos dez que o adendo 28.1 exige -- continuava a **0 ocorrencias**.
//
//	*Um hub de build que so' sabe dizer o que falta e' um diagnostico, nao um hub.*
//
// O QUE ESTE FICHEIRO E', E O QUE NAO E'
//
// E' a EXECUCAO de uma build declarada. Recebe um `CapabilityPlan` **ja' pronto** e produz um
// artefacto com caminho, tamanho e SHA-256.
//
// Nao decide se deve construir (isso e' `CapabilityPlan.Pronto()`); nao instala ferramentas; nao
// assina; nao copia para destino nenhum. Recebe um plano, executa, mede, devolve.
//
//	*Um provider que decide por si se o plano esta' satisfeito duplica a regra e diverge dela.*
//
// A FRONTEIRA DE PLATAFORMA, DITA
//
// O unico `BuildProvider` implementado e' o de Go, porque e' o unico toolchain que este ambiente
// pode exercitar de ponta a ponta. Nao ha' `BuildProvider` de Android, iOS ou Windows: um
// adaptador que existe mas nunca correu e' pior do que um adaptador ausente, porque o catalogo o
// apresenta como disponivel.
//
//	*Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Artefato e' o resultado de uma build bem-sucedida.
type Artefato struct {
	// Caminho e' o caminho absoluto do ficheiro produzido.
	Caminho string `json:"caminho"`
	// Nome e' o nome do ficheiro, para exibicao.
	Nome string `json:"nome"`
	// TamanhoBytes e' o tamanho medido no disco, nao o que o compilador disse.
	TamanhoBytes int64 `json:"tamanho_bytes"`
	// SHA256 e' o checksum do ficheiro REAL. Sem ele o artefacto nao e' rastreavel.
	SHA256 string `json:"sha256"`
	// Alvo e' o alvo do catalogo que produziu este artefato.
	Alvo string `json:"alvo"`
	// PidDoProcesso e' o pid do processo que construiu. Serve para o auditor saber que houve um
	// processo real, nao um `os.Stat` num ficheiro pre-existente.
	PidDoProcesso int `json:"pid_do_processo"`
	// DuracaoMS e' o tempo real da build.
	DuracaoMS int64 `json:"duracao_ms"`
	// SaidaDoComando e' a saida combinada, TRUNCADA. Nao contem segredo: o comando e' nosso.
	SaidaDoComando string `json:"saida_do_comando,omitempty"`
}

// ResultadoDeBuild agrega os artefatos e o que foi descartado.
type ResultadoDeBuild struct {
	Artefatos []Artefato `json:"artefatos"`
	// NaoConstruidos explica os alvos que o plano declarou e que este provider NAO sabe construir.
	NaoConstruidos []string `json:"nao_construidos,omitempty"`
	// SemArtefacto e' verdadeiro quando nada foi produzido. Nao e' sucesso.
	SemArtefacto bool `json:"sem_artefacto"`
	// Explicacao diz em uma frase o que aconteceu.
	Explicacao string `json:"explicacao"`
}

// ErrPlanoIncompleto e' devolvido quando se tenta construir com um plano que nao esta' pronto.
var ErrPlanoIncompleto = errors.New("plano de capability incompleto: nao ha' build a executar")

// ErrSemProviderParaAlvo e' devolvido quando o alvo declarado nao tem provider.
var ErrSemProviderParaAlvo = errors.New("nenhum provider sabe construir este alvo")

// BuildProvider produz artefatos reproduziveis a partir de um projection declarado.
//
// O contrato e' deliberadamente pequeno: um `CapabilityPlan` entra, um `ResultadoDeBuild` sai. Tudo
// o que o adendo 28.1 lista a mais (health, capabilities, cancelamento, cleanup) pertence a
// interface de provider generica, que **nao existe ainda** e nao se inventa por simetria.
type BuildProvider interface {
	// Alvos devolve os alvos do catalogo que este provider sabe construir.
	Alvos() []string
	// Construir executa a build. `DryRun` calcula o comando e nao o executa.
	Construir(ctx context.Context, plano CapabilityPlan, opcoes OpcoesDeBuild) (ResultadoDeBuild, error)
}

// OpcoesDeBuild controla a execucao.
type OpcoesDeBuild struct {
	// Raiz e' o directorio do projection.
	Raiz string
	// Destino e' o directorio onde os artefactos sao escritos. Vazio usa `Raiz/dist`.
	Destino string
	// DryRun calcula e devolve o comando sem o executar.
	DryRun bool
	// Timeout limita a build. Zero usa o limite por omissao.
	Timeout time.Duration
}

// TimeoutPorOmissaoDeBuild limita uma build quando o chamador nao define um.
//
//	*Uma build sem tecto e' um processo que fica depois de quem o pediu sair.*
const TimeoutPorOmissaoDeBuild = 10 * time.Minute

// maxSaidaDeComando trunca a saida guardada no artefacto.
//
//	*Guardar a saida inteira de uma build num artefacto transforma um metadado numa copia do log.*
const maxSaidaDeComando = 4096

// ProviderDeBuildGo constroi binarios Go.
//
// E' o unico provider implementado, e a razao esta' escrita no topo do ficheiro.
type ProviderDeBuildGo struct{}

// Alvos declara o que sabe construir: os alvos do catalogo cujo artefacto e' um binario Go.
func (ProviderDeBuildGo) Alvos() []string {
	return []string{"windows", "linux", "macos"}
}

// Construir executa `go build` e devolve o artefacto medido.
func (p ProviderDeBuildGo) Construir(
	ctx context.Context,
	plano CapabilityPlan,
	opcoes OpcoesDeBuild,
) (ResultadoDeBuild, error) {
	// O plano tem de estar PRONTO. Um provider que constroi com blockers declarados produz um
	// artefacto que ninguem pode reproduzir.
	if !plano.Pronto() {
		return ResultadoDeBuild{}, fmt.Errorf("%w: %d blocker(s): %s",
			ErrPlanoIncompleto, len(plano.Blockers), strings.Join(plano.Blockers, ", "))
	}
	// O alvo tem de constar do catalogo e ser um dos que este provider cobre.
	if !p.cobreAlvo(plano.Alvo) {
		return ResultadoDeBuild{}, fmt.Errorf("%w: %q (provider cobre %s)",
			ErrSemProviderParaAlvo, plano.Alvo, strings.Join(p.Alvos(), ", "))
	}
	raiz := strings.TrimSpace(opcoes.Raiz)
	if raiz == "" {
		return ResultadoDeBuild{}, errors.New("raiz do projection vazia: nao ha' build a executar")
	}
	raizAbs, err := filepath.Abs(raiz)
	if err != nil {
		return ResultadoDeBuild{}, fmt.Errorf("resolver raiz %q: %w", raiz, err)
	}
	if info, errStat := os.Stat(raizAbs); errStat != nil {
		return ResultadoDeBuild{}, fmt.Errorf("raiz %q: %w", raizAbs, errStat)
	} else if !info.IsDir() {
		return ResultadoDeBuild{}, fmt.Errorf("raiz %q nao e' um directorio", raizAbs)
	}
	// Um projection Go minimo tem de ter `go.mod`: sem ele `go build` falha com uma mensagem que
	// ninguem entende a partir do painel.
	if _, errStat := os.Stat(filepath.Join(raizAbs, "go.mod")); errStat != nil {
		return ResultadoDeBuild{}, fmt.Errorf("%s nao tem go.mod: nao e' um projection Go", raizAbs)
	}

	destino := strings.TrimSpace(opcoes.Destino)
	if destino == "" {
		destino = filepath.Join(raizAbs, "dist")
	}
	destinoAbs, err := filepath.Abs(destino)
	if err != nil {
		return ResultadoDeBuild{}, fmt.Errorf("resolver destino %q: %w", destino, err)
	}
	nome := nomeDoArtefacto(raizAbs, plano.Alvo)
	caminhoFinal := filepath.Join(destinoAbs, nome)

	// `-trimpath` para o binario nao conter o caminho da maquina de quem construiu.
	//
	//	*Um binario que carrega o caminho do build e' um binario que nao e' reproduzivel.*
	args := []string{"build", "-trimpath", "-o", caminhoFinal, "."}

	resultado := ResultadoDeBuild{}
	if opcoes.DryRun {
		resultado.NaoConstruidos = nil
		resultado.Explicacao = "dry-run: o comando foi calculado e nao executado"
		resultado.SemArtefacto = true
		return resultado, nil
	}

	if err := os.MkdirAll(destinoAbs, 0o700); err != nil {
		return ResultadoDeBuild{}, fmt.Errorf("criar destino %q: %w", destinoAbs, err)
	}

	timeout := opcoes.Timeout
	if timeout <= 0 {
		timeout = TimeoutPorOmissaoDeBuild
	}
	ctxBuild, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	cmd := exec.CommandContext(ctxBuild, "go", args...)
	cmd.Dir = raizAbs
	// O ambiente herda o do servidor, com DUAS excepcoes deliberadas:
	//
	//   CGO_ENABLED=0  -- o artefacto tem de ser reproduzivel. Com `CGO_ENABLED=1` herdado do `go
	//                     test` do CI, o mesmo commit produziria bytes diferentes conforme o host
	//                     tiver ou nao um compilador C.
	//   GOOS/GOARCH    -- fixados no host do processo, para o teste nao depender do `GOOS` que o
	//                     runner exportou (o CI exporta `GOOS=windows` para o build de
	//                     portabilidade, e herda-lo aqui construiria um .exe no Linux).
	//
	//	*Um build cujo resultado depende do ambiente de quem o chamou nao e' reproduzivel: e'
	//	uma funcao do ambiente.*
	cmd.Env = ambienteDeBuild()
	inicio := time.Now()
	saida, errExec := cmd.CombinedOutput()
	duracao := time.Since(inicio)
	if errExec != nil {
		// A falha inclui a saida REAL do compilador: um erro de build sem a saida do compilador
		// obriga quem le' a correr o comando a mao.
		return ResultadoDeBuild{}, fmt.Errorf("go build falhou (%v): %s", errExec,
			truncarSaida(string(saida)))
	}

	info, errStat := os.Stat(caminhoFinal)
	if errStat != nil {
		// O compilador disse OK mas o ficheiro nao esta' la': isto e' um defeito do provider, nao do
		// projection.
		return ResultadoDeBuild{}, fmt.Errorf("build reportou sucesso mas %q nao existe: %w",
			caminhoFinal, errStat)
	}
	checksum, errHash := sha256DeFicheiro(caminhoFinal)
	if errHash != nil {
		return ResultadoDeBuild{}, fmt.Errorf("checksum de %q: %w", caminhoFinal, errHash)
	}

	artefato := Artefato{
		Caminho:        caminhoFinal,
		Nome:           nome,
		TamanhoBytes:   info.Size(),
		SHA256:         checksum,
		Alvo:           plano.Alvo,
		PidDoProcesso:  cmd.Process.Pid,
		DuracaoMS:      duracao.Milliseconds(),
		SaidaDoComando: truncarSaida(string(saida)),
	}
	resultado.Artefatos = []Artefato{artefato}
	resultado.Explicacao = fmt.Sprintf(
		"1 artefacto produzido (alvo %s, %d bytes, sha256 %s)", plano.Alvo,
		artefato.TamanhoBytes, artefato.SHA256[:12])
	return resultado, nil
}

// ConstruirAlvos executa a build de todos os alvos PRONTOS de um plano de projection.
//
// Um alvo bloqueado NAO e' tentado, e a razao aparece em `NaoConstruidos`: *um build tentado contra
// um plano que declara blockers produz uma falha que ja' se sabia.*
func (p ProviderDeBuildGo) ConstruirAlvos(
	ctx context.Context,
	plano PlanoDoProjeto,
	opcoes OpcoesDeBuild,
) (ResultadoDeBuild, error) {
	agregado := ResultadoDeBuild{}
	if len(plano.Alvos) == 0 {
		agregado.SemArtefacto = true
		agregado.Explicacao = "o plano do projection nao tem alvos"
		return agregado, nil
	}
	// Ordem deterministica: o resultado nao pode depender da ordem do mapa do plano.
	alvos := make([]PlanoPorAlvo, len(plano.Alvos))
	copy(alvos, plano.Alvos)
	sort.Slice(alvos, func(i, j int) bool { return alvos[i].Alvo < alvos[j].Alvo })

	var razoes []string
	for _, entrada := range alvos {
		if entrada.ErroDePlano != "" {
			razoes = append(razoes, entrada.Alvo+": plano nao construido ("+entrada.ErroDePlano+")")
			continue
		}
		if !entrada.Plano.Pronto() {
			razoes = append(razoes, entrada.Alvo+": plano com blockers")
			continue
		}
		if !p.cobreAlvo(entrada.Alvo) {
			razoes = append(razoes, entrada.Alvo+": nenhum provider cobre este alvo")
			continue
		}
		um, err := p.Construir(ctx, entrada.Plano, opcoes)
		if err != nil {
			return ResultadoDeBuild{}, fmt.Errorf("alvo %s: %w", entrada.Alvo, err)
		}
		agregado.Artefatos = append(agregado.Artefatos, um.Artefatos...)
	}
	agregado.NaoConstruidos = razoes
	agregado.SemArtefacto = len(agregado.Artefatos) == 0
	switch {
	case len(agregado.Artefatos) > 0 && len(razoes) == 0:
		agregado.Explicacao = fmt.Sprintf("%d artefacto(s) produzido(s)", len(agregado.Artefatos))
	case len(agregado.Artefatos) > 0:
		agregado.Explicacao = fmt.Sprintf("%d artefacto(s) produzido(s), %d alvo(s) nao construido(s)",
			len(agregado.Artefatos), len(razoes))
	default:
		agregado.Explicacao = fmt.Sprintf(
			"nenhum artefacto produzido: %d alvo(s) nao construido(s)", len(razoes))
	}
	return agregado, nil
}

// cobreAlvo diz se este provider constroi o alvo.
func (p ProviderDeBuildGo) cobreAlvo(alvo string) bool {
	for _, a := range p.Alvos() {
		if strings.EqualFold(a, strings.TrimSpace(alvo)) {
			return true
		}
	}
	return false
}

// nomeDoArtefacto deriva um nome estavel do directorio do projection e do alvo.
//
//	*Um nome que inclui o caminho absoluto da maquina de quem construiu nao e' um nome: e' um
//	identificador de sessao.*
func nomeDoArtefacto(raiz, alvo string) string {
	base := filepath.Base(raiz)
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "artefacto"
	}
	ext := ""
	if strings.EqualFold(alvo, "windows") {
		ext = ".exe"
	}
	return base + "-" + strings.ToLower(alvo) + ext
}

// sha256DeFicheiro calcula o SHA-256 lendo o ficheiro em blocos.
func sha256DeFicheiro(caminho string) (string, error) {
	ficheiro, err := os.Open(caminho)
	if err != nil {
		return "", err
	}
	defer ficheiro.Close()
	h := sha256.New()
	if _, err := io.Copy(h, ficheiro); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ambienteDeBuild monta o ambiente do subprocesso `go build` a partir do ambiente do servidor.
func ambienteDeBuild() []string {
	// Mantem tudo o que o servidor tem (PATH, HOME, GOROOT, GOPATH, caches privados do checkout) e
	// sobrepoe apenas o que precisa de ser deterministico.
	forcadas := map[string]string{
		"CGO_ENABLED": "0",
		"GOOS":        runtime.GOOS,
		"GOARCH":      runtime.GOARCH,
	}
	ambiente := make([]string, 0, len(os.Environ())+len(forcadas))
	for _, entrada := range os.Environ() {
		nome, _, _ := strings.Cut(entrada, "=")
		if _, sobreposta := forcadas[nome]; sobreposta {
			continue
		}
		ambiente = append(ambiente, entrada)
	}
	for nome, valor := range forcadas {
		ambiente = append(ambiente, nome+"="+valor)
	}
	sort.Strings(ambiente)
	return ambiente
}

// truncarSaida limita a saida guardada.
func truncarSaida(s string) string {
	if len(s) <= maxSaidaDeComando {
		return s
	}
	return s[:maxSaidaDeComando] + "\n...[truncado]"
}

// ---------------------------------------------------------------------------
// A LIGACAO QUE FALTAVA: construir -> VERIFICAR
// ---------------------------------------------------------------------------
//
// A E-1251 construiu um artefacto real e mediu o seu SHA-256. A E-1256 implementou o verificador. A
// E-1258 expo-os por HTTP. Mas **o build nunca verificava o que produziu** -- as duas pecas existiam
// e nao se falavam:
//
//	*Tres pecas provadas em separado nao sao uma integracao provada.*
//
// O QUE ESTA FUNCAO VERIFICA, E O QUE ISSO VALE
//
// O `SHA256` de cada `Artefato` foi medido no instante em que o build o produziu. `VerificarConstrucao`
// volta a ler o ficheiro do disco e compara-o com essa referencia:
//
//   - iguais -> o ficheiro nao mudou desde o build;
//   - diferentes -> alguem (ou algo) o alterou depois.
//
// **Isto NAO e' uma verificacao de assinatura**, e o campo `Assinatura` de cada resultado diz
// exactamente o que se conseguiu olhar. *Aprovado significa "passou os gates que consegui aplicar",
// nao "todos os gates existiram".*
//
// A FRONTEIRA QUE A FUNCAO RECUSA PASSAR
//
// Um build que nao produziu nada nao tem nada para verificar. Devolver "tudo verificado" sobre uma
// lista vazia seria a mentira mais barata do sistema:
//
//	*Um plano vazio apresentado como sucesso e' a mesma mentira que um card clicavel que termina em
//	tela vazia.*

// ErrSemArtefactosParaVerificar e' devolvido quando se tenta verificar um build que nao produziu
// nada.
var ErrSemArtefactosParaVerificar = errors.New(
	"build sem artefactos: nao ha' nada para verificar",
)

// VerificacaoDaConstrucao agrega os resultados de verificar todos os artefactos de um build.
type VerificacaoDaConstrucao struct {
	// Resultados tem uma entrada por artefacto, na mesma ordem do `ResultadoDeBuild`.
	Resultados []ResultadoDeVerificacao `json:"resultados"`
	// Total e' quantos artefactos foram verificados.
	Total int `json:"total"`
	// Integros e' quantos passaram a comparacao de checksum.
	Integros int `json:"integros"`
	// Adulterados e' quantos falharam. Um unico adulterado reprova a construcao inteira.
	Adulterados int `json:"adulterados"`
	// SemAssinatura e' quantos tinham `AUSENTE` -- o adaptador olhou e nao havia assinatura.
	SemAssinatura int `json:"sem_assinatura"`
	// NaoVerificadas e' quantos ficaram em `NAO_VERIFICADA` -- nao houve adaptador para olhar.
	//
	// Separado de `SemAssinatura` de proposito: "olhei e nao ha'" e "nao consegui olhar" sao factos
	// diferentes. Conta-los juntos produzia a frase "N com assinatura presente" para artefactos que
	// ninguem inspeccionou -- e *um estado que mente e' pior do que nenhum*.
	NaoVerificadas int `json:"nao_verificadas"`
	// Aprovada e' verdadeira apenas se TODOS os artefactos passaram e nenhum foi adulterado.
	Aprovada bool `json:"aprovada"`
	// Explicacao diz, em uma frase, o que aconteceu -- e porque nao foi aprovada quando nao foi.
	Explicacao string `json:"explicacao"`
}

// VerificarConstrucao verifica cada artefacto de um build contra o SHA-256 que o build mediu.
//
// Devolve `ErrSemArtefactosParaVerificar` para um build vazio: *um relatorio que obriga a somar a mao
// e' um relatorio para quem ja' sabia a resposta*, e um "0 de 0 aprovado" nao e' uma aprovacao.
func VerificarConstrucao(
	ctx context.Context,
	resultado ResultadoDeBuild,
	verificador VerificadorDeArtefacto,
) (VerificacaoDaConstrucao, error) {
	if len(resultado.Artefatos) == 0 {
		return VerificacaoDaConstrucao{
			Aprovada:   false,
			Explicacao: "o build nao produziu artefactos; nao ha' o que verificar",
		}, ErrSemArtefactosParaVerificar
	}

	verificacao := VerificacaoDaConstrucao{
		Resultados: make([]ResultadoDeVerificacao, 0, len(resultado.Artefatos)),
		Total:      len(resultado.Artefatos),
		Aprovada:   true,
	}

	for _, artefato := range resultado.Artefatos {
		if err := ctx.Err(); err != nil {
			return verificacao, err
		}
		// O SHA do build e' a REFERENCIA; o verificador mede o ficheiro AGORA.
		one, err := verificador.Verificar(ctx, artefato.Caminho, artefato.SHA256)
		if err != nil {
			// Um erro de infraestrutura nao e' um veredito: propaga-o em vez de o converter em
			// "adulterado". *Um erro que nao aborta nao e' um erro que nao aconteceu.*
			return verificacao, fmt.Errorf("verificar %q: %w", artefato.Caminho, err)
		}
		verificacao.Resultados = append(verificacao.Resultados, one)
		if one.Integro {
			verificacao.Integros++
		} else {
			verificacao.Adulterados++
			verificacao.Aprovada = false
		}
		switch one.Assinatura {
		case AssinaturaAusente:
			verificacao.SemAssinatura++
		case AssinaturaNaoVerificada:
			verificacao.NaoVerificadas++
		}
	}

	verificacao.Explicacao = explicarVerificacaoDaConstrucao(verificacao)
	return verificacao, nil
}

// explicarVerificacaoDaConstrucao compoe a frase final. Conta os factos; nao os arredonda.
//
// A ORDEM DOS RAMOS IMPORTA. Primeiro a adulteracao (o facto mais grave), depois a ausencia de
// instrumento, e so' no fim a contagem do que foi realmente visto. Inverter os dois ultimos produziria
// "N com assinatura presente" para artefactos que ninguem inspeccionou.
func explicarVerificacaoDaConstrucao(v VerificacaoDaConstrucao) string {
	if v.Adulterados > 0 {
		return fmt.Sprintf(
			"%d de %d artefactos ADULTERADOS: o ficheiro mudou depois do build",
			v.Adulterados, v.Total,
		)
	}
	if v.NaoVerificadas == v.Total {
		return fmt.Sprintf(
			"%d de %d artefactos integros; assinatura NAO verificada neste host (nenhum adaptador)",
			v.Integros, v.Total,
		)
	}
	if v.SemAssinatura == v.Total {
		return fmt.Sprintf(
			"%d de %d artefactos integros; nenhum tem assinatura (esperado nesta RC)",
			v.Integros, v.Total,
		)
	}
	comAssinatura := v.Total - v.SemAssinatura - v.NaoVerificadas
	return fmt.Sprintf(
		"%d de %d artefactos integros; %d com assinatura presente, %d nao verificadas",
		v.Integros, v.Total, comAssinatura, v.NaoVerificadas,
	)
}
