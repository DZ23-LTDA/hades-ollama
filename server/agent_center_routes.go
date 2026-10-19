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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		// `catalog_note`, e nao `note`: e o nome que o painel LE. Duas chaves para a mesma coisa sao
		// duas verdades que um dia divergem -- e o painel ja tinha ficado com a que ninguem emite.
		"catalog_note": "catálogo embutido: nenhuma destas personas foi avaliada; `evaluation` vazio significa NÃO AVALIADO",
	})
}

// listAgentEvaluation: GET /agents/evaluation
//
// Devolve as suites embutidas e a maquina de estados. NAO corre avaliacao -- para isso existe
// `POST /agents/evaluation/run`. Aqui `inference_available` e' MEDIDO ao vivo contra o motor local
// (E-1376), nao presumido: a avaliacao corre, mas so' quando alguem a pede.
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
	// Perguntado ao motor AGORA, nao presumido.
	//
	//	*Um estado que so o programador pode nomear valido nao e um estado: e um campo de texto.*
	motor := detectarMotor(c.Request.Context(), "")
	c.JSON(http.StatusOK, gin.H{
		"organization_id":     organizationID,
		"suites":              suites,
		"divisions":           divisoes,
		"states":              agent.EvalStateCatalog(),
		"inference_available": motor.Disponivel,
		"inference_model":     motor.Modelo,
		"inference_endpoint":  motor.Endpoint,
		"inference_reason":    motor.Motivo,
		"note":                "as suites estao definidas; a disponibilidade de inferencia e' MEDIDA ao vivo contra o motor local, nao presumida",
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

// backupAgentPersonas: POST /agents/backup
//
// Guarda ledger E conteudo dos destinos. Idempotente: o mesmo estado devolve o mesmo id.
//
//	*Um rollback que restaura metade nao e' um rollback: e' um segundo defeito.*
func (a *agentAPI) backupAgentPersonas(c *gin.Context) {
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
	bk, err := ledger.Backup(organizationID, time.Now())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"organization_id": organizationID,
		"backup":          bk,
		"files":           len(bk.Files),
		"entries":         len(bk.Entries),
	})
}

// listAgentBackups: GET /agents/backups
func (a *agentAPI) listAgentBackups(c *gin.Context) {
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
	ids, err := ledger.Backups(organizationID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"count":           len(ids),
		"backups":         ids,
	})
}

// restoreAgentPersonas: POST /agents/restore
//
// Repoe ficheiros E ledger, ou nada. O corpo leva o `backup_id`; nao ha' "restaurar o mais recente" por
// omissao -- *o ultimo a escrever nao ganha por ser o ultimo, e o mais recente tambem nao.*
func (a *agentAPI) restoreAgentPersonas(c *gin.Context) {
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
		BackupID string `json:"backup_id"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	id := strings.TrimSpace(pedido.BackupID)
	if id == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("backup_id is required"))
		return
	}
	if err := ledger.Restore(organizationID, id); err != nil {
		if errors.Is(err, agent.ErrBackupNaoEncontrado) {
			writeAgentError(c, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, agent.ErrBackupCorrompido) {
			// 409: o backup existe mas nao serve. Nao e' "nao encontrado" nem "pedido invalido".
			writeAgentError(c, http.StatusConflict, err)
			return
		}
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"organization_id": organizationID, "restored": id})
}

// runAgentEvaluation: POST /agents/evaluation/run
//
// CORRE a suite contra o motor local. E' POST porque gasta tempo e CPU: nao acontece por abrir um painel.
//
//	*Aprovar uma acao sem ver o custo e aprovar um numero que nao se leu.*
//
// Recusa antes de gastar quando o motor nao esta' la'. E o estado devolvido NUNCA e' PRODUCTION_APPROVED.
func (a *agentAPI) runAgentEvaluation(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	var pedido struct {
		PersonaID string `json:"persona_id"`
		Division  string `json:"division"`
		Model     string `json:"model"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	// Teto de tempo da requisicao inteira: uma avaliacao que o utilizador nao ve terminar e' um timeout.
	ctx, cancelar := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancelar()

	rel, motor, err := executarAvaliacao(ctx, pedido.PersonaID, pedido.Division, pedido.Model)
	if err != nil {
		if errors.Is(err, errMotorIndisponivel) {
			// 503: o produto esta' bem, o motor e' que nao esta' la'. Nao e' defeito do pedido.
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"organization_id": organizationID,
				"error":           err.Error(),
				"engine":          motor,
				"note":            "nenhuma inferência foi executada; o motor local não está disponível",
			})
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"engine":          motor,
		"report":          rel,
		// Repetir no topo o que o relatorio ja' diz: o painel nao deve ter de o inferir.
		"state":         string(rel.State),
		"inference_ran": true,
		"note":          "avaliação real contra o motor local; o estado não é aprovação de produção",
	})
}

// importAgentPersona: POST /agents/import
//
// TRIAGEM, NAO INSTALACAO. Recebe o manifesto bruto, valida a forma e escaneia o conteudo. Nao escreve nada,
// nao activa nada: devolve a decisao.
//
//	*Importar sem activar e o unico jeito de o utilizador ver o que esta a instalar antes de o instalar.*
//
// O estado devolvido nunca e' PRODUCTION_APPROVED: uma importacao nao aprova ninguem.
func (a *agentAPI) importAgentPersona(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	var pedido struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(pedido.Content) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("content is required"))
		return
	}
	// Limite de 256 KiB: um manifesto de persona que nao cabe nisto nao e um manifesto.
	const limiteBytes = 256 * 1024
	persona, estado, relatorio, err := agent.ImportDecision([]byte(pedido.Content), limiteBytes)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"state":           string(estado),
		"report":          relatorio,
		// `installed` e' sempre false: esta rota nao instala. Dizer isto e' diferente de omitir.
		"installed": false,
		"note":      "triagem apenas: nenhum ficheiro foi escrito e nada foi activado",
		"persona": gin.H{
			"id":      persona.ID,
			"name":    persona.Name,
			"version": persona.Version,
			"source":  persona.Source,
			"license": persona.License,
		},
	})
}

// agentQuarantineStore devolve o store, ou nil quando nao ha raiz configurada.
func (a *agentAPI) agentQuarantineStore() (*agent.QuarantineStore, bool) {
	return a.quarantine, a.quarantine != nil
}

// holdAgentPersona: POST /agents/quarantine
//
// Valida, triagem e RETE. Escreve: e' o que faltava na ronda anterior, em que a triagem era stateless.
//
//	*Quarentena e' um estado, nao um log.*
func (a *agentAPI) holdAgentPersona(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	store, ok := a.agentQuarantineStore()
	if !ok {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent quarantine store is not configured"))
		return
	}
	var pedido struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(pedido.Content) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("content is required"))
		return
	}
	const limiteBytes = 256 * 1024
	persona, estado, relatorio, err := agent.ImportDecision([]byte(pedido.Content), limiteBytes)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if estado != agent.EvalQuarantined {
		// Limpa: nao ha' nada para reter. Devolver 200 com `quarantined:false` e' diferente de criar uma
		// entrada vazia para o utilizador achar que ha' algo retido.
		c.JSON(http.StatusOK, gin.H{
			"organization_id": organizationID,
			"state":           string(estado),
			"quarantined":     false,
			"report":          relatorio,
			"note":            "sem achados: nada foi retido",
		})
		return
	}
	reg, criada, err := store.Enter(organizationID, persona, relatorio, time.Now())
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	codigo := http.StatusOK
	if criada {
		codigo = http.StatusCreated
	}
	c.JSON(codigo, gin.H{
		"organization_id": organizationID,
		"state":           string(reg.State),
		"quarantined":     true,
		"created":         criada,
		"record":          reg,
	})
}

// listAgentQuarantine: GET /agents/quarantine
func (a *agentAPI) listAgentQuarantine(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	store, ok := a.agentQuarantineStore()
	if !ok {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent quarantine store is not configured"))
		return
	}
	lista, err := store.List(organizationID)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"count":           len(lista),
		"entries":         lista,
	})
}

// releaseAgentPersona: POST /agents/quarantine/release
//
// Exige `reviewer`. Leva a `SECURITY_REVIEWED`, NUNCA a `PRODUCTION_APPROVED`.
func (a *agentAPI) releaseAgentPersona(c *gin.Context) {
	organizationID := a.organizationID(c)
	if organizationID == "" {
		writeAgentError(c, http.StatusForbidden, errAgentForbidden)
		return
	}
	store, ok := a.agentQuarantineStore()
	if !ok {
		writeAgentError(c, http.StatusServiceUnavailable, errors.New("agent quarantine store is not configured"))
		return
	}
	var pedido struct {
		ID       string `json:"id"`
		Reviewer string `json:"reviewer"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	reg, err := store.Release(organizationID, pedido.ID, pedido.Reviewer, time.Now())
	if err != nil {
		switch {
		case errors.Is(err, agent.ErrQuarentenaNaoEncontrada):
			writeAgentError(c, http.StatusNotFound, err)
		case errors.Is(err, agent.ErrLibertacaoRecusada):
			writeAgentError(c, http.StatusBadRequest, err)
		default:
			writeAgentError(c, http.StatusBadRequest, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organization_id": organizationID,
		"record":          reg,
		"note":            "revisão de segurança registada; isto NÃO aprova para produção",
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
// removeAgentPersona desinstala uma persona do ledger -- mas NAO apaga trabalho local nao integrado.
//
// A REGRA, E O QUE ELA PROTEGE
//
//	*Nunca sobrescreva modificacao local sem diff, backup e autorizacao. Nunca remova arquivo estrangeiro.*
//
// Se o ficheiro no destino ja' nao corresponde ao que o HADES escreveu (`LOCALLY_MODIFIED`) ou se o hash
// esperado conflita (`CONFLICTED`), o pedido e' RECUSADO com 409 e o motivo. Remover nesse estado apagaria
// trabalho que nao e' do HADES. Nos restantes estados -- `CURRENT`, `OUTDATED`, `REMOVED` -- a entrada sai do
// ledger e o ficheiro do destino NAO e' tocado: o ledger e' um registo do que o HADES escreveu, nao um
// gerenciador de ficheiros do utilizador.
// updateAgentPersona actualiza uma instalacao -- com BACKUP e com a mesma recusa do `remove`.
//
// A ORDEM NAO E' NEGOCIAVEL
//
// 1. reconciliar: se o destino tem trabalho local (`LOCALLY_MODIFIED`/`CONFLICTED`), 409 e nada acontece;
// 2. BACKUP antes de escrever -- *backup antes de actualizacao* e' o requisito, nao uma cortesia;
// 3. so' entao registar a entrada nova.
//
// Uma actualizacao que falha a meio deixa o backup, e o restauro e' uma chamada. Uma actualizacao que
// sobrescreve sem backup nao tem caminho de volta.
func (a *agentAPI) updateAgentPersona(c *gin.Context) {
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

	var pedido struct {
		Persona     agent.AgentPersona `json:"persona"`
		Target      string             `json:"target"`
		Destination string             `json:"destination"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
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
	alvo := agent.TargetFormat(strings.TrimSpace(pedido.Target))
	if alvo == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("target is required"))
		return
	}

	// 1. Reconciliar ANTES de escrever.
	noDisco := map[string][]byte{}
	for _, entrada := range ledger.Entries(organizationID) {
		if entrada.PersonaID != pedido.Persona.ID {
			continue
		}
		if conteudo, err := os.ReadFile(entrada.Destination); err == nil {
			noDisco[entrada.Destination] = conteudo
		}
	}
	for _, item := range ledger.Reconcile(organizationID, noDisco, nil) {
		if item.Entry.PersonaID != pedido.Persona.ID {
			continue
		}
		if item.State == agent.DriftLocallyModified || item.State == agent.DriftConflicted {
			writeAgentError(c, http.StatusConflict, errors.New(
				"recusado: o destino tem trabalho local nao integrado ("+string(item.State)+
					"); faca backup e reconcilie antes de actualizar"))
			return
		}
	}

	// 2. BACKUP antes de escrever. Se o backup falhar, nao se actualiza nada.
	backup, err := ledger.Backup(organizationID, time.Now())
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError,
			errors.New("backup antes da actualizacao falhou: "+err.Error()))
		return
	}

	// 3. Render + registar (o `Record` actualiza a entrada da mesma chave).
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
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"updated":    true,
		"backup_id":  backup.ID,
		"bytes":      len(renderizado),
		"entry":      efetiva,
		"created":    criada,
		"note":       "backup feito antes da escrita; restaurar por /agents/restore se necessario",
	})
}

func (a *agentAPI) removeAgentPersona(c *gin.Context) {
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

	var pedido struct {
		PersonaID string `json:"persona_id"`
	}
	if err := decodeJSON(c, &pedido); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(pedido.PersonaID) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("falta persona_id"))
		return
	}

	// Reconciliar ANTES de remover: e' a unica forma de saber se o destino tem trabalho local.
	noDisco := map[string][]byte{}
	for _, entrada := range ledger.Entries(organizationID) {
		if entrada.PersonaID != pedido.PersonaID {
			continue
		}
		if conteudo, err := os.ReadFile(entrada.Destination); err == nil {
			noDisco[entrada.Destination] = conteudo
		}
	}
	for _, item := range ledger.Reconcile(organizationID, noDisco, nil) {
		if item.Entry.PersonaID != pedido.PersonaID {
			continue
		}
		if item.State == agent.DriftLocallyModified || item.State == agent.DriftConflicted {
			writeAgentError(c, http.StatusConflict, errors.New(
				"recusado: o destino tem trabalho local nao integrado ("+string(item.State)+
					"); faca backup e reconcilie antes de remover"))
			return
		}
	}

	removido, err := ledger.Remove(organizationID, pedido.PersonaID)
	if err != nil {
		writeAgentError(c, http.StatusInternalServerError, err)
		return
	}
	if !removido {
		writeAgentError(c, http.StatusNotFound, errors.New("persona nao registada neste ledger"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"removed":    pedido.PersonaID,
		"file_kept":  true,
		"note":       "entrada retirada do ledger; o ficheiro no destino nao foi apagado",
	})
}

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
