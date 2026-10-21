package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// ESTAGIO 47 — UNIVERSAL ARTIFACT DRIVE: A SUPERFICIE HTTP QUE FALTAVA.
//
// **MEDIDO ANTES DE ESCREVER:** o dominio de versionamento (`ArtefactoHistorico`,
// `ArtefactoCompararVersoes`, `ArtefactoRestaurarVersao`) esta' implementado e testado em
// `internal/agent`, *e nao tinha UMA UNICA rota* — a unica superficie de artefacto era o
// download (`GET /missions/:id/artifacts/:artifact_id`). A capacidade existia e ninguem a
// alcancava: e' a mesma classe de defeito do Companion, medida agora na camada de API.
//
// O indice de versoes que o dominio recebe e' `mission.Artifacts` - *o dominio nao tem store
// de versoes; o chamador e' que fornece a fatia.* Portanto o versionamento e' **por missao**,
// e a resposta di-lo (`scope`) em vez de prometer um historico global que nao existe.

// artefactoDaMissao localiza o manifesto pelo ID dentro da missao do chamador.
func artefactoDaMissao(artefactos []agent.ArtifactManifest, id string) (agent.ArtifactManifest, bool) {
	for _, manifesto := range artefactos {
		if manifesto.ID == id {
			return manifesto, true
		}
	}
	return agent.ArtifactManifest{}, false
}

// artifactVersions devolve o historico de versoes de um artefacto da missao.
//
// LEITURA: escopo pelo chamador (`missionForRequest`) - *sem CAS, porque nao muda nada.*
func (a *agentAPI) artifactVersions(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	alvo, ok := artefactoDaMissao(mission.Artifacts, c.Param("artifact_id"))
	if !ok {
		// artefacto inexistente NESTA missao: 404 pelo mesmo erro de versao inexistente
		// (`ErrVersaoInexistente`), que o mapa de erros ja' converte - *uma so' verdade.*
		writeAgentError(c, http.StatusNotFound, agent.ErrVersaoInexistente)
		return
	}
	historico := agent.ArtefactoHistorico(mission.Artifacts, alvo.TenantID, alvo.ProjectID, alvo.Path)
	c.JSON(http.StatusOK, gin.H{
		"mission_id":  mission.ID,
		"artifact_id": alvo.ID,
		"path":        alvo.Path,
		"versions":    historico,
		"scope":       "missao do chamador",
	})
}

// artifactRestore restaura uma versao anterior, produzindo uma versao NOVA.
//
// ESCRITA: usa o CAS em dois niveis - *na versao da MISSAO* (`missionVersionMatches`, como
// todas as escritas desta API) e *na versao corrente do artefacto* (`expected_version`, pelo
// dominio). Um conflito devolve 409 e nao 500: o cliente tem de distinguir «alguem escreveu
// entretanto» de «o servidor avariou».
func (a *agentAPI) artifactRestore(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	alvo, ok := artefactoDaMissao(mission.Artifacts, c.Param("artifact_id"))
	if !ok {
		writeAgentError(c, http.StatusNotFound, agent.ErrVersaoInexistente)
		return
	}
	var corpo struct {
		Versao   int `json:"version"`
		Esperada int `json:"expected_version"`
	}
	if err := c.ShouldBindJSON(&corpo); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	restaurado, err := agent.ArtefactoRestaurarVersao(
		mission.Artifacts, alvo.TenantID, alvo.ProjectID, alvo.Path,
		corpo.Versao, corpo.Esperada, time.Now().UTC(),
	)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mission_id": mission.ID,
		"restored":   restaurado,
		"scope":      "missao do chamador",
	})
}
