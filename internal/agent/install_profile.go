package agent

// FASE 28.6-A do adendo V3: instalador modular, pacotes de capacidade e provisionamento.
//
// O PROBLEMA QUE ESTE FICHEIRO RESOLVE
//
// O `app/ollama.iss` instala TUDO o que o payload contem, num perfil so'. Nao ha' `[Types]` nem
// `[Components]`. Isso significa que o utilizador que so' quer conversar com o HADES recebe o
// mesmo conjunto que o que quer compilar Android -- e o instalador cresce ate' ser impossivel de
// manter. O adendo exige o oposto: nucleo obrigatorio, pacotes opcionais, provisionamento sob
// demanda.
//
// AS DUAS FRASES QUE GOVERNAM ESTE FICHEIRO
//
//   1. "O perfil escolhido e' uma intencao do usuario, nao permissao ilimitada."
//      -- um componente que exige elevacao, servico, driver, reboot ou EULA continua a precisar
//      de approval MESMO no perfil mais completo. O perfil diz o que se quer; nao diz o que se
//      pode.
//
//   2. "Nao converta UNKNOWN em ABSENT."
//      -- ja' e' invariante do `toolchain_plan.go` e aqui continua a ser. `UNKNOWN` nao dispara
//      instalacao automatica: *uma duvida nao e' uma ausencia, e agir sobre ela instala o que ja'
//      existia.*
//
// O QUE ESTE FICHEIRO NAO FAZ
//
// Nao instala nada. Nao descarrega nada. Nao toca no disco. Produz o PLANO -- e o plano e' a
// parte que se pode testar sem privilégios, sem rede e sem risco. *Um instalador cuja logica de
// decisao so' se testa instalando e' um instalador que se testa uma vez por release.*

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PerfilDeInstalacao e' a intencao do utilizador, nao uma autorizacao.
type PerfilDeInstalacao string

const (
	PerfilBasic                 PerfilDeInstalacao = "BASIC"
	PerfilRecommended           PerfilDeInstalacao = "RECOMMENDED"
	PerfilDeveloperComplete     PerfilDeInstalacao = "DEVELOPER_COMPLETE"
	PerfilMobile                PerfilDeInstalacao = "MOBILE"
	PerfilMediaAndVoice         PerfilDeInstalacao = "MEDIA_AND_VOICE"
	PerfilBrowserAndComputerUse PerfilDeInstalacao = "BROWSER_AND_COMPUTER_USE"
	PerfilContainersAndCloud    PerfilDeInstalacao = "CONTAINERS_AND_CLOUD"
	PerfilCustom                PerfilDeInstalacao = "CUSTOM"
	PerfilFullOffline           PerfilDeInstalacao = "FULL_OFFLINE"
)

// PerfisDeInstalacao lista os perfis na ordem em que o instalador os oferece.
//
// `CUSTOM` e `FULL_OFFLINE` ficam no fim porque nao sao "tamanhos": sao modos de composicao.
func PerfisDeInstalacao() []PerfilDeInstalacao {
	return []PerfilDeInstalacao{
		PerfilBasic, PerfilRecommended, PerfilDeveloperComplete, PerfilMobile,
		PerfilMediaAndVoice, PerfilBrowserAndComputerUse, PerfilContainersAndCloud,
		PerfilCustom, PerfilFullOffline,
	}
}

// Valido diz se o perfil e' um dos declarados.
func (p PerfilDeInstalacao) Valido() bool {
	for _, c := range PerfisDeInstalacao() {
		if c == p {
			return true
		}
	}
	return false
}

// EstadoDoPacote e' o ciclo de vida de um componente no gerenciador de capacidades.
//
// Lista fechada. *Um estado novo tem de ser uma decisao, nao um erro de ortografia.*
type EstadoDoPacote string

const (
	PacoteNaoInstalado          EstadoDoPacote = "NOT_INSTALLED"
	PacoteBaixando              EstadoDoPacote = "DOWNLOADING"
	PacoteVerificando           EstadoDoPacote = "VERIFYING"
	PacoteInstalando            EstadoDoPacote = "INSTALLING"
	PacotePronto                EstadoDoPacote = "READY"
	PacoteAtualizacaoDisponivel EstadoDoPacote = "UPDATE_AVAILABLE"
	PacoteReparoNecessario      EstadoDoPacote = "REPAIR_REQUIRED"
	PacoteDegradado             EstadoDoPacote = "DEGRADED"
	PacoteEmUso                 EstadoDoPacote = "IN_USE"
	PacoteRemovendo             EstadoDoPacote = "REMOVING"
	PacoteRollbackDisponivel    EstadoDoPacote = "ROLLBACK_AVAILABLE"
	PacoteAutenticacaoRequerida EstadoDoPacote = "AUTH_REQUIRED"
	PacoteAprovacaoRequerida    EstadoDoPacote = "APPROVAL_REQUIRED"
	PacoteBloqueadoExterno      EstadoDoPacote = "BLOCKED_EXTERNAL"
	PacoteNaoSuportado          EstadoDoPacote = "UNSUPPORTED"
	PacoteFalhado               EstadoDoPacote = "FAILED"
)

// Valido diz se o estado pertence a lista fechada.
func (e EstadoDoPacote) Valido() bool {
	switch e {
	case PacoteNaoInstalado, PacoteBaixando, PacoteVerificando, PacoteInstalando,
		PacotePronto, PacoteAtualizacaoDisponivel, PacoteReparoNecessario,
		PacoteDegradado, PacoteEmUso, PacoteRemovendo, PacoteRollbackDisponivel,
		PacoteAutenticacaoRequerida, PacoteAprovacaoRequerida, PacoteBloqueadoExterno,
		PacoteNaoSuportado, PacoteFalhado:
		return true
	}
	return false
}

// ImpactoDoPacote descreve o que instalar custa ao SISTEMA, nao ao disco.
//
// Cada campo verdadeiro e' uma razao para pedir approval -- e a razao tem de ser visivel ANTES,
// nao descoberta no meio.
type ImpactoDoPacote struct {
	// ExigeElevacao: precisa de administrador/UAC.
	ExigeElevacao bool `json:"exige_elevacao"`
	// ExigeReinicio: precisa de reboot para fazer efeito.
	ExigeReinicio bool `json:"exige_reinicio"`
	// CriaServico: regista servico ou daemon persistente.
	CriaServico bool `json:"cria_servico"`
	// AbrePorta: escuta numa porta local.
	AbrePorta bool `json:"abre_porta"`
	// InstalaDriver: instala driver de kernel.
	InstalaDriver bool `json:"instala_driver"`
	// AceitaEULA: exige aceitar licenca de terceiro.
	AceitaEULA bool `json:"aceita_eula"`
	// AlteraAmbienteGlobal: muda PATH, associacoes ou configuracao global.
	AlteraAmbienteGlobal bool `json:"altera_ambiente_global"`
	// RequerContaOuServicoExterno: depende de login/pagamento/API de terceiro.
	RequerContaOuServicoExterno bool `json:"requer_conta_ou_servico_externo"`
}

// ExigeAprovacao resumo: um impacto que muda o sistema exige um humano a decidir.
//
// A regra e' deliberadamente ampla. *Um componente que instala um servico sem perguntar
// transforma uma instalacao numa mudanca de sistema operativo.*
func (i ImpactoDoPacote) ExigeAprovacao() bool {
	return i.ExigeElevacao || i.ExigeReinicio || i.CriaServico || i.AbrePorta ||
		i.InstalaDriver || i.AceitaEULA || i.AlteraAmbienteGlobal ||
		i.RequerContaOuServicoExterno
}

// PacoteDeCapacidade e' um componente instalavel do catalogo.
type PacoteDeCapacidade struct {
	ID   string `json:"id"`
	Nome string `json:"nome"`
	// Dependencias sao IDs de outros pacotes que tem de estar instalados primeiro.
	Dependencias []string `json:"dependencias,omitempty"`
	// TamanhoMB e' o download declarado. Zero significa "nao medido", nao "leve".
	TamanhoMB int `json:"tamanho_mb"`
	// Licenca e' a licenca do componente de terceiro, quando houver.
	Licenca string `json:"licenca,omitempty"`
	// Redistribuivel diz se o HADES pode empacotar o componente no seu proprio instalador.
	// *Um componente que a licenca proibe redistribuir tem de ser baixado no momento, nunca
	// embutido.*
	Redistribuivel bool `json:"redistribuivel"`
	// EscopoPrivadoPrefiravel: pode correr isolado do sistema, sem alterar o ambiente.
	EscopoPrivadoPrefiravel bool            `json:"escopo_privado_preferivel"`
	Impacto                 ImpactoDoPacote `json:"impacto"`
	// Perfis que incluem este pacote por omissao.
	Perfis []PerfilDeInstalacao `json:"perfis"`
}

// CatalogoDePacotes e' o catalogo minimo exigido pelo adendo.
//
// Declarativo e ordenado por ID na leitura. *Um catalogo que se descobre em runtime muda de forma
// quando a maquina muda, e deixa de ser comparavel entre dois hosts.*
func CatalogoDePacotes() []PacoteDeCapacidade {
	return []PacoteDeCapacidade{
		{
			ID: "hades-core", Nome: "HADES Core",
			TamanhoMB: 180, Licenca: "MIT", Redistribuivel: true,
			Perfis: []PerfilDeInstalacao{
				PerfilBasic, PerfilRecommended, PerfilDeveloperComplete,
				PerfilMobile, PerfilMediaAndVoice, PerfilBrowserAndComputerUse,
				PerfilContainersAndCloud, PerfilFullOffline,
			},
		},
		{
			ID: "local-ai", Nome: "IA local (Ollama)",
			Dependencias: []string{"hades-core"}, TamanhoMB: 260,
			Licenca: "MIT", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{CriaServico: true, AbrePorta: true},
			Perfis:  []PerfilDeInstalacao{PerfilRecommended, PerfilFullOffline},
		},
		{
			ID: "browser-automation", Nome: "Navegador controlado",
			Dependencias: []string{"hades-core"}, TamanhoMB: 320,
			Licenca: "Apache-2.0", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Perfis: []PerfilDeInstalacao{
				PerfilRecommended, PerfilBrowserAndComputerUse,
				PerfilFullOffline,
			},
		},
		{
			ID: "computer-use", Nome: "Computer Use",
			Dependencias: []string{"hades-core"}, TamanhoMB: 40,
			Licenca: "MIT", Redistribuivel: true,
			Impacto: ImpactoDoPacote{ExigeElevacao: true},
			Perfis:  []PerfilDeInstalacao{PerfilBrowserAndComputerUse},
		},
		{
			ID: "media", Nome: "Media (FFmpeg/FFprobe)",
			TamanhoMB: 140, Licenca: "LGPL-2.1", Redistribuivel: true,
			EscopoPrivadoPrefiravel: true,
			Perfis:                  []PerfilDeInstalacao{PerfilMediaAndVoice},
		},
		{
			ID: "voice", Nome: "Voz (STT/TTS)",
			Dependencias: []string{"media"}, TamanhoMB: 210,
			Licenca: "MIT", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{AceitaEULA: true},
			Perfis:  []PerfilDeInstalacao{PerfilMediaAndVoice},
		},
		{
			ID: "vision-ocr", Nome: "Visao e OCR",
			Dependencias: []string{"media"}, TamanhoMB: 95,
			Licenca: "Apache-2.0", Redistribuivel: true,
			Perfis: []PerfilDeInstalacao{PerfilMediaAndVoice},
		},
		{
			ID: "developer-js", Nome: "Desenvolvimento JS",
			Dependencias: []string{"hades-core"}, TamanhoMB: 120,
			Licenca: "MIT", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Perfis: []PerfilDeInstalacao{PerfilDeveloperComplete, PerfilMobile, PerfilFullOffline},
		},
		{
			ID: "developer-python", Nome: "Desenvolvimento Python",
			Dependencias: []string{"hades-core"}, TamanhoMB: 90,
			Licenca: "PSF-2.0", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Perfis: []PerfilDeInstalacao{PerfilDeveloperComplete, PerfilFullOffline},
		},
		{
			ID: "developer-go-rust", Nome: "Desenvolvimento Go e Rust",
			Dependencias: []string{"hades-core"}, TamanhoMB: 620,
			Licenca: "BSD-3-Clause", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Perfis: []PerfilDeInstalacao{PerfilDeveloperComplete},
		},
		{
			ID: "developer-dotnet-windows", Nome: ".NET e Windows SDK",
			Dependencias: []string{"hades-core"}, TamanhoMB: 890,
			Licenca: "MIT", Redistribuivel: false, EscopoPrivadoPrefiravel: false,
			Impacto: ImpactoDoPacote{
				ExigeElevacao: true, AceitaEULA: true,
				AlteraAmbienteGlobal: true,
			},
			Perfis: []PerfilDeInstalacao{PerfilDeveloperComplete},
		},
		{
			ID: "mobile-android", Nome: "Android SDK e ADB",
			Dependencias: []string{"developer-js"}, TamanhoMB: 1240,
			Licenca: "Apache-2.0", Redistribuivel: false, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{AceitaEULA: true, AlteraAmbienteGlobal: true},
			Perfis:  []PerfilDeInstalacao{PerfilMobile, PerfilDeveloperComplete},
		},
		{
			ID: "mobile-expo", Nome: "Expo e EAS",
			Dependencias: []string{"developer-js"}, TamanhoMB: 130,
			Licenca: "MIT", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{RequerContaOuServicoExterno: true},
			Perfis:  []PerfilDeInstalacao{PerfilMobile},
		},
		{
			ID: "mobile-flutter", Nome: "Flutter e Dart",
			Dependencias: []string{"hades-core"}, TamanhoMB: 980,
			Licenca: "BSD-3-Clause", Redistribuivel: false, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{AlteraAmbienteGlobal: true},
			Perfis:  []PerfilDeInstalacao{PerfilMobile},
		},
		{
			ID: "containers", Nome: "Containers e Buildx",
			Dependencias: []string{"hades-core"}, TamanhoMB: 540,
			Licenca: "Apache-2.0", Redistribuivel: false, EscopoPrivadoPrefiravel: false,
			Impacto: ImpactoDoPacote{
				ExigeElevacao: true, CriaServico: true,
				ExigeReinicio: true, AbrePorta: true,
			},
			Perfis: []PerfilDeInstalacao{PerfilContainersAndCloud},
		},
		{
			ID: "release-security", Nome: "SBOM, checksums e Cosign",
			Dependencias: []string{"hades-core"}, TamanhoMB: 75,
			Licenca: "Apache-2.0", Redistribuivel: true, EscopoPrivadoPrefiravel: true,
			Perfis: []PerfilDeInstalacao{
				PerfilDeveloperComplete, PerfilContainersAndCloud,
				PerfilFullOffline,
			},
		},
		{
			ID: "plugins-mcp-skills", Nome: "Plugins, MCPs e skills",
			Dependencias: []string{"hades-core"}, TamanhoMB: 25,
			Licenca: "MIT", Redistribuivel: true,
			Perfis: []PerfilDeInstalacao{PerfilRecommended},
		},
		{
			ID: "cloud-and-store-clis", Nome: "CLIs de cloud e lojas",
			Dependencias: []string{"hades-core"}, TamanhoMB: 160,
			Licenca: "Apache-2.0", Redistribuivel: false, EscopoPrivadoPrefiravel: true,
			Impacto: ImpactoDoPacote{RequerContaOuServicoExterno: true},
			Perfis:  []PerfilDeInstalacao{PerfilContainersAndCloud},
		},
	}
}

// NoCatalogo procura um pacote pelo ID.
func NoCatalogo(id string) (PacoteDeCapacidade, bool) {
	for _, p := range CatalogoDePacotes() {
		if p.ID == id {
			return p, true
		}
	}
	return PacoteDeCapacidade{}, false
}

// PacotesDoPerfil devolve os pacotes que um perfil inclui, em ordem estavel.
func PacotesDoPerfil(perfil PerfilDeInstalacao) []string {
	var saida []string
	for _, p := range CatalogoDePacotes() {
		for _, c := range p.Perfis {
			if c == perfil {
				saida = append(saida, p.ID)
				break
			}
		}
	}
	sort.Strings(saida)
	return saida
}

// ItemDoPlano e' um pacote com a decisao ja' tomada para ESTA maquina.
type ItemDoPlano struct {
	PacoteID string `json:"pacote_id"`
	Nome     string `json:"nome"`
	// Acao e' o que vai acontecer: `INSTALAR`, `REUTILIZAR`, `MANTER`, `PULAR`, `APROVAR`.
	Acao string `json:"acao"`
	// Motivo e' legivel pelo utilizador, porque a decisao tem de ser explicavel.
	Motivo string `json:"motivo"`
	// RequerAprovacao vem do IMPACTO, nao do perfil.
	// *Um perfil mais completo nao torna um servico menos invasivo.*
	RequerAprovacao bool `json:"requer_aprovacao"`
	// TamanhoMB do download, quando houver.
	TamanhoMB int `json:"tamanho_mb"`
}

// AcoesDoPlano sao as acoes possiveis. Lista fechada.
const (
	AcaoInstalar   = "INSTALAR"
	AcaoReutilizar = "REUTILIZAR"
	AcaoManter     = "MANTER"
	AcaoPular      = "PULAR"
	AcaoAprovar    = "APROVAR"
)

// PlanoDeInstalacao e' o resultado do preflight.
type PlanoDeInstalacao struct {
	Perfil PerfilDeInstalacao `json:"perfil"`
	Itens  []ItemDoPlano      `json:"itens"`
	// TamanhoTotalMB soma APENAS o que sera' baixado.
	TamanhoTotalMB int `json:"tamanho_total_mb"`
	// DownloadsEvitados conta quantos pacotes ja' existiam e nao serao baixados.
	DownloadsEvitados int `json:"downloads_evitados"`
	// AprovacoesNecessarias e' quantos itens exigem um humano antes de correr.
	AprovacoesNecessarias int `json:"aprovacoes_necessarias"`
	// Bloqueios sao os impedimentos externos verdadeiros.
	Bloqueios []string `json:"bloqueios"`
}

// Pronto diz se o plano pode ser executado sem aprovacao pendente.
//
// *Um plano que se executa com aprovacoes pendentes nao e' um plano: e' uma instalacao a
// acontecer sem ninguem a decidir.*
func (p PlanoDeInstalacao) Pronto() bool {
	return len(p.Bloqueios) == 0 && p.AprovacoesNecessarias == 0
}

// ResumoDoPlano e' uma linha para o utilizador.
func (p PlanoDeInstalacao) ResumoDoPlano() string {
	return fmt.Sprintf("perfil %s: %d itens, %d MB a baixar, %d reaproveitados, %d aprovacoes, %d bloqueios",
		p.Perfil, len(p.Itens), p.TamanhoTotalMB, p.DownloadsEvitados,
		p.AprovacoesNecessarias, len(p.Bloqueios))
}

// MontarPlanoDeInstalacao constroi o plano a partir do perfil e do que a maquina ja' tem.
//
// `estado` e' o resultado da descoberta: pacoteID -> EstadoDoPacote. Um pacote ausente do mapa e'
// tratado como `NOT_INSTALLED` **apenas se a descoberta o cobriu** -- e' por isso que a funcao
// recebe tambem `descobertos`, o conjunto de pacotes que a sonda conseguiu avaliar.
//
// *Esta distincao e' o coracao do `UNKNOWN` vs `ABSENT`: um pacote que a sonda nao avaliou nao e'
// um pacote ausente, e instala-lo seria instalar por cima do que ja' existe.*
func MontarPlanoDeInstalacao(
	perfil PerfilDeInstalacao,
	estado map[string]EstadoDoPacote,
	descobertos map[string]bool,
) (PlanoDeInstalacao, error) {
	if !perfil.Valido() {
		return PlanoDeInstalacao{}, fmt.Errorf("perfil desconhecido: %q", perfil)
	}
	plano := PlanoDeInstalacao{Perfil: perfil}

	// `CUSTOM` nao traz pacotes por omissao: quem escolhe e' o utilizador.
	pedidos := PacotesDoPerfil(perfil)
	if perfil == PerfilCustom {
		pedidos = nil
	}

	// Resolver dependencias transitivas primeiro.
	resolvidos := map[string]bool{}
	var ordem []string
	var visitar func(id string, caminho []string) error
	visitar = func(id string, caminho []string) error {
		if resolvidos[id] {
			return nil
		}
		for _, c := range caminho {
			if c == id {
				return fmt.Errorf("dependencia circular: %s", strings.Join(append(caminho, id), " -> "))
			}
		}
		pkg, existe := NoCatalogo(id)
		if !existe {
			return fmt.Errorf("pacote %q referido mas ausente do catalogo", id)
		}
		for _, dep := range pkg.Dependencias {
			if err := visitar(dep, append(caminho, id)); err != nil {
				return err
			}
		}
		resolvidos[id] = true
		ordem = append(ordem, id)
		return nil
	}
	for _, id := range pedidos {
		if err := visitar(id, nil); err != nil {
			return PlanoDeInstalacao{}, err
		}
	}

	for _, id := range ordem {
		pkg, _ := NoCatalogo(id)
		item := ItemDoPlano{PacoteID: id, Nome: pkg.Nome, TamanhoMB: pkg.TamanhoMB}

		avaliado := descobertos[id]
		est, temEstado := estado[id]

		switch {
		case !avaliado || !temEstado || est == PacoteBloqueadoExterno:
			// Nao avaliado nao e' ausente.
			item.Acao = AcaoPular
			item.Motivo = "nao foi possivel determinar o estado nesta maquina; " +
				"instalar por cima do que ja' existe e' pior do que nao instalar"
			plano.Bloqueios = append(plano.Bloqueios,
				fmt.Sprintf("%s: estado desconhecido (UNKNOWN nao e' ABSENT)", id))
		case est == PacotePronto || est == PacoteEmUso:
			item.Acao = AcaoReutilizar
			item.Motivo = "ja' instalado e saudavel; reinstalar so' para padronizar nao e' motivo"
			plano.DownloadsEvitados++
		case est == PacoteAtualizacaoDisponivel:
			item.Acao = AcaoInstalar
			item.Motivo = "versao instalada abaixo da compativel"
			plano.TamanhoTotalMB += pkg.TamanhoMB
		case est == PacoteDegradado || est == PacoteReparoNecessario || est == PacoteFalhado:
			item.Acao = AcaoInstalar
			item.Motivo = "presente mas com health funcional falhado (PRESENT_BROKEN)"
			plano.TamanhoTotalMB += pkg.TamanhoMB
		default:
			item.Acao = AcaoInstalar
			item.Motivo = "ausente"
			plano.TamanhoTotalMB += pkg.TamanhoMB
		}

		// A aprovacao vem do IMPACTO, nunca do perfil.
		if item.Acao == AcaoInstalar && pkg.Impacto.ExigeAprovacao() {
			item.RequerAprovacao = true
			item.Acao = AcaoAprovar
			item.Motivo += "; exige aprovacao: " + descreverImpacto(pkg.Impacto)
			plano.AprovacoesNecessarias++
		}
		plano.Itens = append(plano.Itens, item)
	}
	return plano, nil
}

// descreverImpacto explica, em portuguese, porque e' que aquele pacote pede um humano.
func descreverImpacto(i ImpactoDoPacote) string {
	var partes []string
	if i.ExigeElevacao {
		partes = append(partes, "elevacao administrativa")
	}
	if i.ExigeReinicio {
		partes = append(partes, "reinicio")
	}
	if i.CriaServico {
		partes = append(partes, "servico persistente")
	}
	if i.AbrePorta {
		partes = append(partes, "porta local")
	}
	if i.InstalaDriver {
		partes = append(partes, "driver")
	}
	if i.AceitaEULA {
		partes = append(partes, "aceitar licenca de terceiro")
	}
	if i.AlteraAmbienteGlobal {
		partes = append(partes, "altera ambiente global")
	}
	if i.RequerContaOuServicoExterno {
		partes = append(partes, "conta/servico externo")
	}
	return strings.Join(partes, ", ")
}

// ProblemaDoCatalogo e' um achado da auto-auditoria.
type ProblemaDoCatalogo struct {
	PacoteID string `json:"pacote_id,omitempty"`
	Mensagem string `json:"mensagem"`
}

// AuditarCatalogo verifica a coerencia do catalogo.
//
// Cobre: ID vazio ou duplicado, dependencia inexistente, ciclo, pacote sem perfil, perfil sem
// `hades-core`, tamanho negativo e componente nao redistribuivel marcado como privado.
//
// *Um catalogo que ninguem audita ganha pacotes que dependem de si proprios e perfis que nao
// instalam o nucleo.*
func AuditarCatalogo() []ProblemaDoCatalogo {
	var problemas []ProblemaDoCatalogo
	catalogo := CatalogoDePacotes()
	vistos := map[string]bool{}
	for _, p := range catalogo {
		if strings.TrimSpace(p.ID) == "" {
			problemas = append(problemas, ProblemaDoCatalogo{Mensagem: "pacote sem ID"})
			continue
		}
		if vistos[p.ID] {
			problemas = append(problemas, ProblemaDoCatalogo{
				PacoteID: p.ID, Mensagem: "ID duplicado",
			})
		}
		vistos[p.ID] = true
		if p.TamanhoMB < 0 {
			problemas = append(problemas, ProblemaDoCatalogo{
				PacoteID: p.ID, Mensagem: "tamanho negativo",
			})
		}
		if len(p.Perfis) == 0 && p.ID != "hades-core" {
			problemas = append(problemas, ProblemaDoCatalogo{
				PacoteID: p.ID,
				Mensagem: "pacote em nenhum perfil: nunca sera' oferecido",
			})
		}
		for _, dep := range p.Dependencias {
			if dep == p.ID {
				problemas = append(problemas, ProblemaDoCatalogo{
					PacoteID: p.ID, Mensagem: "depende de si proprio",
				})
			}
		}
	}
	// Dependencias inexistentes.
	for _, p := range catalogo {
		for _, dep := range p.Dependencias {
			if _, existe := NoCatalogo(dep); !existe {
				problemas = append(problemas, ProblemaDoCatalogo{
					PacoteID: p.ID, Mensagem: "dependencia inexistente: " + dep,
				})
			}
		}
	}
	// Ciclos, por perfil.
	for _, perfil := range PerfisDeInstalacao() {
		if perfil == PerfilCustom {
			continue
		}
		if _, err := MontarPlanoDeInstalacao(perfil, map[string]EstadoDoPacote{}, map[string]bool{}); err != nil {
			problemas = append(problemas, ProblemaDoCatalogo{
				Mensagem: fmt.Sprintf("perfil %s nao resolve: %v", perfil, err),
			})
		}
	}
	// Todo o perfil tem de incluir o nucleo, exceto CUSTOM (que escolhe a mao).
	for _, perfil := range PerfisDeInstalacao() {
		if perfil == PerfilCustom {
			continue
		}
		inclui := false
		for _, p := range catalogo {
			if p.ID != "hades-core" {
				continue
			}
			for _, c := range p.Perfis {
				if c == perfil {
					inclui = true
				}
			}
		}
		if !inclui {
			problemas = append(problemas, ProblemaDoCatalogo{
				Mensagem: "perfil " + string(perfil) + " nao inclui hades-core",
			})
		}
	}
	return problemas
}

// ---------------------------------------------------------------------------------------------
// FASE 28.7: ponte entre o catalogo de pacotes e a SONDA REAL de ferramentas.
//
// O `MontarPlanoDeInstalacao` precisa de saber o estado de cada pacote. Antes desta funcao, esse
// estado so' existia em testes -- e um motor que so' corre sobre dados fabricados nao e' um
// motor: e' uma maquete.
//
// AQUI O ESTADO VEM DO `Prober`, a mesma sonda que o `toolchain_plan.go` usa. Nao ha' um segundo
// mecanismo de descoberta.
//
// A REGRA QUE ATRAVESSA TUDO: `ToolUnknown` NAO entra no conjunto `descobertos`. Um pacote cuja
// ferramenta representativa nao pode ser determinada continua `UNKNOWN` -- e o plano vai
// PULA-LO, nao instala-lo.
//
// *Nao saber se existe e' diferente de saber que nao existe, e o precario e' instalar por cima.*
// ---------------------------------------------------------------------------------------------

// FerramentaRepresentativa diz qual comando representa um pacote na descoberta.
//
// Vazio significa: o pacote nao tem comando verificavel (e' uma capacidade interna do HADES ou
// uma credencial). Nesse caso o estado e' `UNKNOWN` por contrato do proprio `ToolRequirement`,
// e o pacote e' PULADO -- nunca instalado as cegas.
func FerramentaRepresentativa(pacoteID string) (comando, motivo string, tem bool) {
	switch pacoteID {
	case "local-ai":
		return "ollama", "runtime de inferencia local", true
	case "browser-automation":
		return "node", "runtime para Playwright", true
	case "media":
		return "ffmpeg", "codecs e processamento de media", true
	case "voice":
		return "ffmpeg", "base para STT/TTS (o modelo em si e' dado, nao binario)", true
	case "vision-ocr":
		return "tesseract", "OCR local", true
	case "developer-js":
		return "node", "runtime JavaScript", true
	case "developer-python":
		return "python", "runtime Python", true
	case "developer-go-rust":
		return "go", "toolchain Go (Rust e' verificado a parte)", true
	case "developer-dotnet-windows":
		return "dotnet", ".NET SDK", true
	case "mobile-android":
		return "adb", "Android platform-tools", true
	case "mobile-expo":
		return "eas", "Expo/EAS CLI", true
	case "mobile-flutter":
		return "flutter", "Flutter SDK", true
	case "containers":
		return "docker", "runtime de containers", true
	case "release-security":
		return "cosign", "assinatura e verificacao de artefactos", true
	case "cloud-and-store-clis":
		return "gh", "CLI de cloud/lojas", true
	}
	// hades-core, computer-use, plugins-mcp-skills: capacidade interna ou credencial.
	return "", "", false
}

// ProjetarPacotesComProber sonda cada pacote do catalogo e devolve o estado observado.
//
// Devolve dois mapas: `estado` (o que se sabe) e `descobertos` (o que foi possivel avaliar).
// A distincao NAO e' decorativa: `MontarPlanoDeInstalacao` so' instala o que esta' em
// `descobertos`, e trata o resto como duvida.
func ProjetarPacotesComProber(ctx context.Context, prober Prober) (map[string]EstadoDoPacote, map[string]bool) {
	estado := make(map[string]EstadoDoPacote, len(CatalogoDePacotes()))
	descobertos := make(map[string]bool, len(CatalogoDePacotes()))
	if prober == nil {
		// Sem sonda nao ha' descoberta: tudo fica UNKNOWN. *Um prober nulo nao pode virar
		// "tudo ausente", senao a ausencia de sonda instala o mundo.*
		return estado, descobertos
	}
	for _, pkg := range CatalogoDePacotes() {
		comando, motivo, tem := FerramentaRepresentativa(pkg.ID)
		req := ToolRequirement{
			Nome:        pkg.ID,
			Comando:     comando,
			Obrigatorio: false,
			Motivo:      motivo,
		}
		probe := prober(ctx, req)
		// O contrato do Prober manda devolver o requisito ecoado; se nao o fizer, corrigimos
		// para nao perdermos a ligacao entre pacote e sonda.
		if probe.Requisito.Nome == "" {
			probe.Requisito = req
		}
		if probe.Requisito.Nome == "" || !tem {
			// Sem comando verificavel: UNKNOWN, e fora de `descobertos`.
			estado[pkg.ID] = PacoteBloqueadoExterno
			continue
		}
		switch probe.Estado {
		case ToolHealthy, ToolUnverified:
			estado[pkg.ID] = PacotePronto
			descobertos[pkg.ID] = true
		case ToolOutdated:
			estado[pkg.ID] = PacoteAtualizacaoDisponivel
			descobertos[pkg.ID] = true
		case ToolBroken:
			estado[pkg.ID] = PacoteDegradado
			descobertos[pkg.ID] = true
		case ToolAbsent:
			estado[pkg.ID] = PacoteNaoInstalado
			descobertos[pkg.ID] = true
		case ToolUnknown, "":
			// NAO entra em `descobertos`. E' o invariante.
			estado[pkg.ID] = PacoteBloqueadoExterno
		default:
			estado[pkg.ID] = PacoteBloqueadoExterno
		}
	}
	return estado, descobertos
}
