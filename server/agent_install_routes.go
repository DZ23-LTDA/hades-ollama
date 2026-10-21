package server

// FASE 28.7 do adendo V3: superficie canonica do instalador modular.
//
// O QUE ESTAS ROTAS EXPONHEM
//
//	GET /api/agent/v1/install/profiles      -- os 9 perfis e o catalogo de pacotes
//	GET /api/agent/v1/install/plan/:perfil  -- o plano para UM perfil, com descoberta real
//
// O `toolchain.install_plan` do adendo e' isto. O motor esta' em
// `internal/agent/install_profile.go`; aqui so' se traduz o resultado para JSON.
//
// AS TRES DECISOES QUE ESTAS ROTAS TORNAM VISIVEIS
//
//  1. O plano corre a SONDA REAL (`agent.SondarFerramentaNoPATH`), nao dados de exemplo. Uma rota
//     de planeamento que devolve o catalogo sem olhar para a maquina diria "falta tudo" ou "esta'
//     tudo" -- e ambas as respostas seriam inventadas.
//
//  2. `UNKNOWN` NAO vira `ABSENT`. Pacote que a sonda nao conseguiu avaliar sai como `PULAR` +
//     bloqueio, nunca como "instalar". *Nao saber se existe e' diferente de saber que nao existe.*
//
//  3. O perfil NAO autoriza. `aprovacoes_necessarias` conta os componentes cujo IMPACTO pede um
//     humano -- elevacao, servico, reinicio, porta, driver, EULA, ambiente global, conta externa.
//     *Um perfil mais completo nao torna um servico menos invasivo.*
//
// O QUE ESTAS ROTAS NAO FAZEM
//
// Nao instalam. Nao baixam. Nao tocam no disco. Sao **so' leitura** -- e o teste de POST recusado
// e' o que impede que alguem transfer o planeador em instalador sem uma decisao explicita.

import (
	"context"
	"errors"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// installPerfilResposta descreve um perfil selecionavel.
type installPerfilResposta struct {
	ID        string   `json:"id"`
	Descricao string   `json:"descricao"`
	Pacotes   []string `json:"pacotes"`
	TamanhoMB int      `json:"tamanho_mb"`
	// EscolhePacotes distingue CUSTOM: e' o unico em que o utilizador compoe a mao.
	EscolhePacotes bool `json:"escolhe_pacotes"`
}

// installPacoteResposta descreve um componente do catalogo sem revelar decisao.
type installPacoteResposta struct {
	ID             string   `json:"id"`
	Nome           string   `json:"nome"`
	Dependencias   []string `json:"dependencias"`
	TamanhoMB      int      `json:"tamanho_mb"`
	Licenca        string   `json:"licenca"`
	Redistribuivel bool     `json:"redistribuivel"`
	ExigeAprovacao bool     `json:"exige_aprovacao"`
	Impacto        []string `json:"impacto"`
}

// installItemResposta e' um pacote com a decisao ja' tomada para esta maquina.
type installItemResposta struct {
	PacoteID        string `json:"pacote_id"`
	Nome            string `json:"nome"`
	Acao            string `json:"acao"`
	Motivo          string `json:"motivo"`
	RequerAprovacao bool   `json:"requer_aprovacao"`
	TamanhoMB       int    `json:"tamanho_mb"`
}

// installPlanoResposta e' a resposta de `install/plan/:perfil`.
type installPlanoResposta struct {
	Perfil                string                `json:"perfil"`
	Host                  string                `json:"host"`
	Goos                  string                `json:"goos"`
	Itens                 []installItemResposta `json:"itens"`
	TamanhoTotalMB        int                   `json:"tamanho_total_mb"`
	DownloadsEvitados     int                   `json:"downloads_evitados"`
	AprovacoesNecessarias int                   `json:"aprovacoes_necessarias"`
	Bloqueios             []string              `json:"bloqueios"`
	Pronto                bool                  `json:"pronto"`
	Resumo                string                `json:"resumo"`
	// ExecutaAlgo e' sempre false e existe para o consumidor nao ter de adivinhar.
	ExecutaAlgo bool `json:"executa_algo"`
}

// descricaoDoPerfil explica, em portuguese, o que cada perfil inclui.
//
// A descricao vem do PERFIL, nao do frontend: *uma UI que inventa a descricao de um perfil
// descreve um produto que pode nao existir.*
func descricaoDoPerfil(p agent.PerfilDeInstalacao) string {
	switch p {
	case agent.PerfilBasic:
		return "HADES Core, Mission Control, chat e /goal, updater e diagnostico"
	case agent.PerfilRecommended:
		return "Core + IA local, navegador controlado e plugins/MCPs/skills"
	case agent.PerfilDeveloperComplete:
		return "Core + toolchains de desenvolvimento e release-security"
	case agent.PerfilMobile:
		return "Core + toolchains para Android, Expo e Flutter"
	case agent.PerfilMediaAndVoice:
		return "Core + media, voz e visao/OCR"
	case agent.PerfilBrowserAndComputerUse:
		return "Core + navegador controlado e Computer Use"
	case agent.PerfilContainersAndCloud:
		return "Core + containers e CLIs de cloud/lojas"
	case agent.PerfilCustom:
		return "escolha componente a componente; nenhum pacote por omissao"
	case agent.PerfilFullOffline:
		return "Core + IA local, JS, Python, release-security para instalacao sem rede"
	}
	return ""
}

// impactoEmPalavras traduz o impacto de um pacote para o consumidor.
func impactoEmPalavras(i agent.ImpactoDoPacote) []string {
	partes := make([]string, 0, 8)
	if i.ExigeElevacao {
		partes = append(partes, "exige elevacao administrativa")
	}
	if i.ExigeReinicio {
		partes = append(partes, "exige reinicio")
	}
	if i.CriaServico {
		partes = append(partes, "cria servico persistente")
	}
	if i.AbrePorta {
		partes = append(partes, "abre porta local")
	}
	if i.InstalaDriver {
		partes = append(partes, "instala driver")
	}
	if i.AceitaEULA {
		partes = append(partes, "exige aceitar licenca de terceiro")
	}
	if i.AlteraAmbienteGlobal {
		partes = append(partes, "altera ambiente global (PATH/associacoes)")
	}
	if i.RequerContaOuServicoExterno {
		partes = append(partes, "depende de conta ou servico externo")
	}
	return partes
}

// installProfiles responde o catalogo e os perfis.
//
// Devolve tambem `problemas` da auto-auditoria do catalogo: *um catalogo que se entrega sem se
// auditar transporta os seus proprios defeitos para a UI.*
func (a *agentAPI) installProfiles(c *gin.Context) {
	perfis := agent.PerfisDeInstalacao()
	respostaPerfis := make([]installPerfilResposta, 0, len(perfis))
	for _, p := range perfis {
		pacotes := agent.PacotesDoPerfil(p)
		tamanho := 0
		for _, id := range pacotes {
			if pkg, existe := agent.NoCatalogo(id); existe {
				tamanho += pkg.TamanhoMB
			}
		}
		respostaPerfis = append(respostaPerfis, installPerfilResposta{
			ID:             string(p),
			Descricao:      descricaoDoPerfil(p),
			Pacotes:        pacotes,
			TamanhoMB:      tamanho,
			EscolhePacotes: p == agent.PerfilCustom,
		})
	}

	catalogo := agent.CatalogoDePacotes()
	respostaCatalogo := make([]installPacoteResposta, 0, len(catalogo))
	for _, pkg := range catalogo {
		respostaCatalogo = append(respostaCatalogo, installPacoteResposta{
			ID:             pkg.ID,
			Nome:           pkg.Nome,
			Dependencias:   pkg.Dependencias,
			TamanhoMB:      pkg.TamanhoMB,
			Licenca:        pkg.Licenca,
			Redistribuivel: pkg.Redistribuivel,
			ExigeAprovacao: pkg.Impacto.ExigeAprovacao(),
			Impacto:        impactoEmPalavras(pkg.Impacto),
		})
	}

	problemas := agent.AuditarCatalogo()
	avisos := make([]string, 0, len(problemas))
	for _, p := range problemas {
		avisos = append(avisos, p.PacoteID+": "+p.Mensagem)
	}

	host, _ := os.Hostname()
	c.JSON(http.StatusOK, gin.H{
		"host":    host,
		"goos":    runtime.GOOS,
		"perfis":  respostaPerfis,
		"pacotes": respostaCatalogo,
		// `avisos` existe para o consumidor poder distinguir "catalogo saudavel" de "catalogo
		// com defeito conhecido". Lista, nunca null.
		"avisos": avisos,
		"count":  len(respostaPerfis),
	})
}

// installPlan responde o plano para UM perfil.
//
// Perfil desconhecido devolve **400**, nao 200 com plano vazio: *um plano vazio para um perfil que
// nao existe diria "nada a instalar" a quem le so' o tamanho.*
func (a *agentAPI) installPlan(c *gin.Context) {
	pedido := strings.TrimSpace(c.Param("perfil"))
	if pedido == "" {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("perfil vazio; use GET /install/plan/:perfil"))
		return
	}
	perfil := agent.PerfilDeInstalacao(strings.ToUpper(pedido))
	if !perfil.Valido() {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("perfil desconhecido; use GET /install/profiles para a lista"))
		return
	}

	// A sonda corre com prazo: *uma descoberta sem limite transforma um preflight numa espera.*
	ctx, cancelar := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancelar()

	estado, descobertos := agent.ProjetarPacotesComProber(ctx, agent.SondarFerramentaNoPATH)
	plano, err := agent.MontarPlanoDeInstalacao(perfil, estado, descobertos)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}

	itens := make([]installItemResposta, 0, len(plano.Itens))
	for _, item := range plano.Itens {
		itens = append(itens, installItemResposta{
			PacoteID:        item.PacoteID,
			Nome:            item.Nome,
			Acao:            item.Acao,
			Motivo:          item.Motivo,
			RequerAprovacao: item.RequerAprovacao,
			TamanhoMB:       item.TamanhoMB,
		})
	}
	bloqueios := plano.Bloqueios
	if bloqueios == nil {
		// Lista vazia, nunca `null`: *um `null` faz o consumidor tratar "sem bloqueios" como
		// "campo ausente" e cair no ramo de erro.*
		bloqueios = []string{}
	}

	host, _ := os.Hostname()
	c.JSON(http.StatusOK, installPlanoResposta{
		Perfil:                string(plano.Perfil),
		Host:                  host,
		Goos:                  runtime.GOOS,
		Itens:                 itens,
		TamanhoTotalMB:        plano.TamanhoTotalMB,
		DownloadsEvitados:     plano.DownloadsEvitados,
		AprovacoesNecessarias: plano.AprovacoesNecessarias,
		Bloqueios:             bloqueios,
		Pronto:                plano.Pronto(),
		Resumo:                plano.ResumoDoPlano(),
		// A rota e' de leitura e di-lo no payload. *Uma rota que planear e executar na mesma
		// chamada poupa um clique e gasta uma decisao.*
		ExecutaAlgo: false,
	})
}
