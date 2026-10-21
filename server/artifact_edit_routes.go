package server

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// statusDoErroDeEdicao mapeia o erro do dominio para o status HTTP certo.
//
// A distincao importa ao cliente: 409 e' "alguem escreveu entretanto" (voltar a ler e tentar de
// novo), 413 e' "grande demais" (nao adianta repetir), 400 e' "pedido invalido" (corrigir o corpo).
// Um 500 para tudo isto obrigaria o cliente a adivinhar.
func statusDoErroDeEdicao(err error) int {
	switch {
	case errors.Is(err, agent.ErrRestauroConflitante):
		return http.StatusConflict
	case errors.Is(err, agent.ErrConteudoArtefactoInalterado):
		return http.StatusConflict
	case errors.Is(err, agent.ErrConteudoArtefactoGrande):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, agent.ErrConteudoArtefactoVazio):
		return http.StatusBadRequest
	case errors.Is(err, agent.ErrCaminhoForaDoWorkspace):
		return http.StatusBadRequest
	default:
		return statusForAgentError(err)
	}
}

// I1b parte 3: PUT /missions/:id/artifacts/:artifact_id
//
// Cria uma VERSAO NOVA com o conteudo enviado (nao substitui: a anterior fica restaurável).
// O caminho vem do MANIFESTO -- o corpo so' traz conteudo e expected_version. Digitar um `path`
// no corpo nao tem efeito nenhum: nao existe campo para isso.
func (a *agentAPI) artifactEdit(c *gin.Context) {
	mission, err := a.missionForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	if !missionVersionMatches(c, mission) {
		return
	}
	if _, ok := artefactoDaMissao(mission.Artifacts, c.Param("artifact_id")); !ok {
		writeAgentError(c, http.StatusNotFound, agent.ErrVersaoInexistente)
		return
	}

	var corpo struct {
		ConteudoBase64 string `json:"content_base64"`
		Esperada       int    `json:"expected_version"`
		// Path NAO existe de proposito. Ver o comentario acima.
	}
	if err := c.ShouldBindJSON(&corpo); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	conteudo, err := base64.StdEncoding.DecodeString(corpo.ConteudoBase64)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, errors.New("content_base64 invalido"))
		return
	}

	novo, err := a.scopedRuntime(c).ArtifactEdit(
		mission.ID, c.Param("artifact_id"), conteudo, corpo.Esperada, time.Now().UTC(),
	)
	if err != nil {
		writeAgentError(c, statusDoErroDeEdicao(err), err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mission_id":  mission.ID,
		"artifact_id": novo.ID,
		"version":     novo.Version,
		"sha256":      novo.SHA256,
		"size":        novo.Size,
	})
}
