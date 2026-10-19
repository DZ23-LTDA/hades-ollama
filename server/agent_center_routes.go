// Agent Center -- superficie HTTP do nucleo do `internal/agent/agent_persona*`.
//
// # O QUE ESTAS ROTAS NAO FAZEM
//
// Nao executam personas. Nao instalam nada fora do ledger. Nao escrevem no disco do utilizador: o
// `install` RENDERIZA e REGISTA; escrever o ficheiro no destino e' um passo separado, com o caminho
// resolvido. *Uma rota que escreve onde o pedido disser e' uma rota que escreve onde o pedido disser.*
//
// O CATALOGO VEM DO LEDGER, E ISSO E' DELIBERADO
//
// Ainda nao ha catalogo embutido (fatia propria). O que existe e' o que o HADES INSTALOU -- e mostrar o
// ledger como se fosse o catalogo seria dizer que o mundo e' o que ja' se instalou.
//
//	*Uma lista que so' mostra o que ja' se instalou nao e' um catalogo: e' um historico.*
//
// Por isso a resposta separa `agents` (as entradas do ledger) de `catalog_available: false`, e o painel
// tem de mostrar a diferenca.
package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// hashSHA256Hex: o digest dos bytes renderizados. Calculado aqui porque `agent.hashBytes` e' privado e
// exporta-lo so' para isto alargaria a superficie do pacote.
func hashSHA256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// agentCenterLedger devolve o ledger ou falha fechado.
func (a *agentAPI) agentCenterLedger() (*agent.PersonaLedger, bool) {
	return a.personas, a.personas != nil
}

// listAgentPersonas: GET /agents
func (a *agentAPI) listAgentPersonas(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	ledger, ok := a.agentCenterLedger()
	if !ok {
		// Fail-closed: sem ledger nao ha' "nenhum agente", ha' "sem armazenamento".
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent persona ledger is not configured"))
		return
	}
	if err := ledger.Load(organizationID); err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	entradas := ledger.Entries(organizationID)
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"agents":          entradas,
		"count":           len(entradas),
		// O catalogo agora esta' ligado, e a resposta separa as DUAS coisas: o que existe para instalar e o
		// que ja' foi instalado. *Uma lista que so' mostra o que ja' se instalou nao e' um catalogo: e' um
		// historico.*
		"catalog_available": true,
		"catalog_version":   agent.CatalogVersion(),
		"catalog_count":     agent.CatalogCount(),
	})
}

// listAgentCatalog: GET /agents/catalog
//
// Le' so' o catalogo embutido. Nao toca no ledger nem no disco: o catalogo e' o mesmo para toda a gente, e
// mistura-lo com instalacoes faria cada organizacao ver um catalogo diferente.
func (a *agentAPI) listAgentCatalog(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	personas := agent.Catalog()
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"version":         agent.CatalogVersion(),
		"count":           len(personas),
		"personas":        personas,
		// O catalogo embutido nao tem avaliacoes. Dizer isso e' diferente de omitir o campo.
		"evaluations_available": false,
		"note":                  "catálogo embutido: nenhuma destas personas foi avaliada; `evaluation` vazio significa NÃO AVALIADO",
	})
}

// listAgentEvaluation: GET /agents/evaluation
//
// Devolve as suites embutidas e a maquina de estados. NAO corre avaliacao: `Evaluate` precisa de uma
// funcao de inferencia, e ligar o harness ao motor e' um passo separado e provado a parte.
//
//	*Uma suite que declara casos e nao corre nao e' uma avaliacao: e' o roteiro dela.*
func (a *agentAPI) listAgentEvaluation(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	suites := agent.EvalSuites()
	divisoes := make([]string, 0, len(suites))
	for d := range suites {
		divisoes = append(divisoes, d)
	}
	sort.Strings(divisoes)
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"suites":          suites,
		"divisions":       divisoes,
		"states":          agent.EvalStateCatalog(),
		// O harness existe e esta' testado, mas nao esta' ligado a um motor neste runtime.
		"inference_available": false,
		"note":                "as suites estão definidas; a execução contra um modelo é um passo separado e ainda não está ligada",
	})
}

// listAgentTeams: GET /agents/teams
//
// Devolve os presets embutidos JA' VALIDADOS contra o catalogo, com o que falta declarado.
//
// Um preset invalido nao e' escondido: vai na resposta com `ok: false` e os problemas. Esconder um preset
// quebrado faria o painel parecer que a equipa existe.
//
//	*Um estado que mente e' pior do que nenhum.*
func (a *agentAPI) listAgentTeams(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	equipas := agent.Teams()
	type item struct {
		Team       agent.AgentTeam      `json:"team"`
		Validation agent.TeamValidation `json:"validation"`
		Covered    []string             `json:"covered_divisions"`
		Missing    []string             `json:"missing_divisions"`
		Summary    string               `json:"summary"`
	}
	itens := make([]item, 0, len(equipas))
	funcionais := 0
	for _, eq := range equipas {
		v := agent.ValidateTeam(eq, agent.CatalogLookup)
		cob, falt := agent.TeamCoverage(eq, agent.CatalogLookup)
		if eq.Functional {
			funcionais++
		}
		itens = append(itens, item{
			Team: eq, Validation: v, Covered: cob, Missing: falt,
			Summary: agent.DescribeTeam(eq, v, cob, falt),
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"version":         agent.TeamPresetVersion(),
		"count":           len(itens),
		"teams":           itens,
		// Nenhum preset correu uma missao real: dizer isto e' diferente de omitir o campo.
		"functional_count": funcionais,
		"note":             "nenhum preset completou uma missão representativa com ferramentas reais; `functional:false` significa NÃO PROVADO",
	})
}

// agentPersonaDrift: GET /agents/drift
func (a *agentAPI) agentPersonaDrift(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	ledger, ok := a.agentCenterLedger()
	if !ok {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent persona ledger is not configured"))
		return
	}
	if err := ledger.Load(organizationID); err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}

	// Ler o disco de cada destino registado. Um destino que nao existe e' `REMOVED`, nao erro.
	noDisco := map[string][]byte{}
	for _, entrada := range ledger.Entries(organizationID) {
		conteudo, err := os.ReadFile(entrada.Destination)
		if err != nil {
			continue
		}
		noDisco[entrada.Destination] = conteudo
	}
	itens := ledger.Reconcile(organizationID, noDisco, nil)
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"items":           itens,
		"summary":         agent.DriftSummary(itens),
	})
}

// installAgentPersona: POST /agents/install
//
// Renderiza a persona para o formato pedido, calcula os dois hashes e REGISTA. Nao escreve no destino.
func (a *agentAPI) installAgentPersona(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	ledger, ok := a.agentCenterLedger()
	if !ok {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent persona ledger is not configured"))
		return
	}

	var pedido struct {
		Persona     agent.AgentPersona `json:"persona"`
		Target      string             `json:"target"`
		Destination string             `json:"destination"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}

	alvo := agent.TargetFormat(strings.TrimSpace(pedido.Target))
	if agent.FormatSupport(alvo) != agent.SupportInstallable {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("target is not installable: "+string(agent.FormatSupport(alvo))))
		return
	}

	// O destino tem de ser RELATIVO e ficar dentro de algo que o servidor conhece. Recusar `..` e caminho
	// absoluto e' o minimo: *um destino que o cliente escreve e' um destino que o cliente escolhe.*
	destino := strings.TrimSpace(pedido.Destination)
	if destino == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("destination is required"))
		return
	}
	if filepath.IsAbs(destino) || strings.Contains(destino, "..") {
		writeAgentError(c, http.StatusBadRequest,
			errors.New("destination must be a relative path inside the workspace"))
		return
	}

	renderizado, err := agent.Render(pedido.Persona, alvo)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	hashOrigem, err := pedido.Persona.ContentHash()
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}

	entrada := agent.PersonaLedgerEntry{
		PersonaID:    pedido.Persona.ID,
		Version:      pedido.Persona.Version,
		Target:       alvo,
		Destination:  destino,
		Scope:        organizationID,
		SourceHash:   hashOrigem,
		RenderedHash: hashSHA256Hex(renderizado),
	}
	efetiva, criada, err := ledger.Record(organizationID, entrada)
	if err != nil {
		if errors.Is(err, agent.ErrPluginVersionConflict) {
			writeAgentError(c, http.StatusConflict, err)
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}

	// 201 quando criou, 200 quando o mesmo pedido ja' estava registado (retry).
	estado := http.StatusOK
	if criada {
		estado = http.StatusCreated
	}
	c.JSON(estado, gin.H{
		"organization_id": organizationID,
		"entry":           efetiva,
		"created":         criada,
		// O conteudo NAO e' devolvido: o pedido para instalar nao e' o pedido para ler.
		"bytes": len(renderizado),
	})
}
