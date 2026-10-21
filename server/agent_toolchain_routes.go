package server

// FASE 28.7 do adendo V3: expor o resolvedor de toolchain por API.
//
// O resolvedor (`internal/agent/toolchain_plan.go`) e a CLI (`cmd/agent_toolchain.go`) ja'
// existiam. Faltava o caminho que o Mission Control consome: uma rota HTTP. *Uma capacidade que
// nao chega a UI nao aparece no produto -- fica no codigo.*
//
// O servidor sonda o PATH do HOST onde corre. Isso so' esta' certo quando o build acontece no
// mesmo host do servidor (o caso do HADES desktop e do `hades serve` local). Quando o build corre
// num runner remoto, o plano corre no runner -- e a resposta diz isso pelo campo `host`, para o
// consumidor nao confundir as duas maquinas.
//
// Nenhuma rota aqui instala, altera ou executa trabalho de utilizador. Todas sao LEITURA.
// *Um preflight que muda o sistema nao e' preflight: e' instalacao disfarcada.*

import (
	"errors"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// toolchainProbeResposta e' a projecao minima de uma sonda para o consumidor.
//
// Explicitamente NAO devolve ponteiros internos nem o `ToolRequirement` inteiro: o contrato e'
// o que o Mission Control precisa, nao a estrutura de implementacao.
type toolchainProbeResposta struct {
	Nome        string `json:"nome"`
	Estado      string `json:"estado"`
	Obrigatorio bool   `json:"obrigatorio"`
	Caminho     string `json:"caminho,omitempty"`
	Versao      string `json:"versao,omitempty"`
	Motivo      string `json:"motivo"`
	Detalhe     string `json:"detalhe"`
}

type toolchainPlanoResposta struct {
	Alvo      string                   `json:"alvo"`
	Pronto    bool                     `json:"pronto"`
	Veredicto string                   `json:"veredicto"`
	Probes    []toolchainProbeResposta `json:"probes"`
	Blockers  []string                 `json:"blockers"`
	Avanca    []string                 `json:"trabalho_independente"`
}

// toolchainPlanoComHost acrescenta o contexto da maquina onde o plano correu.
//
// *Um plano sem o host deixa o leitor supor que foi a maquina dele.* O build pode correr num
// runner remoto, e entao o plano corre no runner -- a resposta tem de dizer onde.
type toolchainPlanoComHost struct {
	toolchainPlanoResposta
	Host string `json:"host"`
	GOOS string `json:"goos"`
}

// toolchainAlvoResposta e' a projecao minima de um alvo na listagem.
type toolchainAlvoResposta struct {
	Nome           string   `json:"nome"`
	Descricao      string   `json:"descricao"`
	HostNecessario string   `json:"host_necessario,omitempty"`
	Exigencias     []string `json:"exigencias"`
	Independe      []string `json:"trabalho_independente,omitempty"`
}

// toolchainTargets responde `toolchain.detect`: quais alvos de build este HADES conhece.
//
// Devolve tambem o `host` e o `goos` para o consumidor saber em que maquina o catalogo foi lido.
func (a *agentAPI) toolchainTargets(c *gin.Context) {
	alvos := agent.CatalogoDeAlvos()
	resposta := make([]toolchainAlvoResposta, 0, len(alvos))
	for _, alvo := range alvos {
		exigencias := make([]string, 0, len(alvo.Exigencias))
		for _, req := range alvo.Exigencias {
			marca := ""
			if req.Obrigatorio {
				marca = " (obrigatoria)"
			}
			exigencias = append(exigencias, req.Nome+marca+" -- "+req.Motivo)
		}
		resposta = append(resposta, toolchainAlvoResposta{
			Nome:           alvo.Nome,
			Descricao:      alvo.Descricao,
			HostNecessario: alvo.ApenasNativo,
			Exigencias:     exigencias,
			Independe:      alvo.Independen,
		})
	}
	host, _ := os.Hostname()
	c.JSON(http.StatusOK, gin.H{
		"host":  host,
		"goos":  runtime.GOOS,
		"alvos": resposta,
		"count": len(resposta),
	})
}

// toolchainPlan responde `toolchain.doctor` para UM alvo.
//
// Alvo desconhecido devolve **404**, nao 200 com plano vazio: *um plano vazio para um alvo que
// nao existe pareceria "tudo pronto" a quem le so' o `pronto`.*
func (a *agentAPI) toolchainPlan(c *gin.Context) {
	alvo := strings.TrimSpace(c.Param("alvo"))
	if alvo == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("alvo vazio; use GET /toolchain/plan/:alvo"))
		return
	}
	resolver := agent.NovoResolverDeToolchain()
	plano, err := resolver.Plano(c.Request.Context(), alvo)
	if err != nil {
		// O erro do resolvedor ja' lista os alvos conhecidos.
		writeAgentError(c, http.StatusNotFound, err)
		return
	}
	host, _ := os.Hostname()
	// Usa o tipo declarado em vez de um `gin.H` inline: *um contrato tipado e' o que garante que
	// a forma do JSON nao muda quando alguem edita o handler.* `host`/`goos` vem por composicao
	// porque dizem ONDE o plano correu -- um plano sem host deixa o leitor supor que foi a
	// maquina dele, e o build pode correr num runner remoto.
	resposta := toolchainPlanoComHost{
		toolchainPlanoResposta: toolchainPlanoResposta{
			Alvo:      plano.Alvo,
			Pronto:    plano.Pronto(),
			Veredicto: veredictoDoPlano(plano),
			Probes:    projetarProbes(plano.Probes),
			Blockers:  listaNaoNula(plano.Blockers),
			// O trabalho independente NUNCA fica vazio quando ha' blockers: "depende de conta
			// externa" nao e' razao para parar o que nao depende.
			Avanca: listaNaoNula(plano.TrabalhoIndependente),
		},
	}
	resposta.Host = host
	resposta.GOOS = runtime.GOOS
	c.JSON(http.StatusOK, resposta)
}

// toolchainDoctor responde `toolchain.doctor` para TODOS os alvos, com o resumo por alvo.
//
// Nao devolve 503 quando tudo esta' bloqueado: bloqueio nao e' indisponibilidade do servidor.
// *Um 503 diria "o servidor esta' quebrado" quando o que falta e' uma ferramenta na maquina.*
func (a *agentAPI) toolchainDoctor(c *gin.Context) {
	resolver := agent.NovoResolverDeToolchain()
	nomes := make([]string, 0, len(agent.CatalogoDeAlvos()))
	for _, alvo := range agent.CatalogoDeAlvos() {
		nomes = append(nomes, alvo.Nome)
	}
	sort.Strings(nomes)

	type resumo struct {
		Alvo      string `json:"alvo"`
		Pronto    bool   `json:"pronto"`
		Veredicto string `json:"veredicto"`
		Falta     int    `json:"exigencias_bloqueantes"`
	}
	saida := make([]resumo, 0, len(nomes))
	prontos, bloqueados, incompletos := 0, 0, 0
	for _, nome := range nomes {
		plano, err := resolver.Plano(c.Request.Context(), nome)
		if err != nil {
			// Um alvo do proprio catalogo que falha ao resolver e' defeito interno, nao do
			// utilizador: registar e continuar, para um alvo nao esconder os outros.
			saida = append(saida, resumo{Alvo: nome, Veredicto: "erro"})
			continue
		}
		veredicto := veredictoDoPlano(plano)
		switch veredicto {
		case "pronto":
			prontos++
		case "bloqueado":
			bloqueados++
		default:
			incompletos++
		}
		saida = append(saida, resumo{
			Alvo: nome, Pronto: plano.Pronto(), Veredicto: veredicto, Falta: len(plano.Blockers),
		})
	}
	host, _ := os.Hostname()
	c.JSON(http.StatusOK, gin.H{
		"host":        host,
		"goos":        runtime.GOOS,
		"alvos":       saida,
		"prontos":     prontos,
		"bloqueados":  bloqueados,
		"incompletos": incompletos,
		"total":       len(saida),
		// Mensagem pronta para o Mission Control, na mesma semantica da CLI.
		"resumo": resumoDoDoctor(prontos, bloqueados, incompletos, len(saida)),
	})
}

// veredictoDoPlano traduz o plano em um de tres estados.
//
// Tres, nao dois. *Falta de prova nao e' prova de bloqueio:* um alvo com ferramentas presentes
// mas sem health funcional esta' `incompleto`, e chama-lo `bloqueado` faria o utilizador procurar
// o que ja' tem.
func veredictoDoPlano(plano agent.CapabilityPlan) string {
	switch {
	case plano.Pronto():
		return "pronto"
	case len(plano.Blockers) > 0:
		return "bloqueado"
	default:
		return "incompleto"
	}
}

func projetarProbes(probes []agent.ToolProbe) []toolchainProbeResposta {
	saida := make([]toolchainProbeResposta, 0, len(probes))
	for _, probe := range probes {
		saida = append(saida, toolchainProbeResposta{
			Nome:        probe.Requisito.Nome,
			Estado:      string(probe.Estado),
			Obrigatorio: probe.Requisito.Obrigatorio,
			Caminho:     probe.Caminho,
			Versao:      probe.Versao,
			Motivo:      probe.Requisito.Motivo,
			Detalhe:     probe.Detalhe,
		})
	}
	return saida
}

// listaNaoNula garante `[]` no JSON em vez de `null`.
//
// *Um consumidor que recebe `null` tem de tratar dois casos para a mesma pergunta; um contrato
// que promete lista devolve lista.*
func listaNaoNula(itens []string) []string {
	if itens == nil {
		return []string{}
	}
	return itens
}

func resumoDoDoctor(prontos, bloqueados, incompletos, total int) string {
	switch {
	case bloqueados > 0 && incompletos > 0:
		return itoa(bloqueados) + " de " + itoa(total) + " alvos bloqueados, " +
			itoa(incompletos) + " incompletos (ferramentas presentes, falta health funcional)"
	case bloqueados > 0:
		return itoa(bloqueados) + " de " + itoa(total) + " alvos bloqueados"
	case incompletos > 0:
		return "nenhum alvo bloqueado; " + itoa(incompletos) + " de " + itoa(total) +
			" incompletos (falta health funcional)"
	default:
		return "todos os " + itoa(total) + " alvos prontos"
	}
}

// itoa evita trazer `strconv` so' para tres chamadas neste arquivo.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negativo := n < 0
	if negativo {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if negativo {
		return "-" + string(buf)
	}
	return string(buf)
}
