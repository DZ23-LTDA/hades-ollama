package server

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// FATIA 1: publicar o artefacto de uma missao e servir o resultado por URL.
//
// NAO reusa o `POST /builders/:id/publish`: aquele publica PROJECTOS de builder; o painel mostra
// ARTEFACTOS de missao. Ligar um ao outro daria a ilusao de publicar.
func (a *agentAPI) publishArtifact(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	// NEGATIVO: sem artefacto nao ha o que publicar.
	alvo, ok := artefactoDaMissao(mission.Artifacts, c.Param("artifact_id"))
	if !ok {
		writeAgentError(c, http.StatusNotFound, agent.ErrVersaoInexistente)
		return
	}
	_, url, err := a.scopedRuntime(c).ArtifactPublish(mission.ID, c.Param("artifact_id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"mission_id":  mission.ID,
		"artifact_id": alvo.ID,
		"version":     alvo.Version,
		"url":         url,
		"scope":       "missao do chamador",
	})
}

// publishedFile serve o artefacto publicado. O caminho e' reconstruido a partir do URL e verificado
// pelo runtime (nunca vem do disco), e o content-type vem do sufixo.
func (a *agentAPI) publishedFile(c *gin.Context) {
	caminho, err := a.scopedRuntime(c).PublishedPath(c.Param("chave"), c.Param("nome"))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if _, err := os.Stat(caminho); err != nil {
		writeAgentError(c, http.StatusNotFound, err)
		return
	}
	c.Header("Content-Type", inlineContentType(c.Param("nome")))
	c.Header("Cache-Control", "no-store")
	c.File(caminho)
}
