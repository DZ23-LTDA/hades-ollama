package server

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// maxUploadChunkBytes limita o tamanho de um unico chunk lido em memoria.
const maxUploadChunkBytes = 64 << 20 // 64 MiB

func uploadStatus(err error) int {
	switch {
	case errors.Is(err, agent.ErrUploadNotFound):
		return http.StatusNotFound
	case errors.Is(err, agent.ErrUploadForbidden):
		return http.StatusForbidden
	case errors.Is(err, agent.ErrUploadQuotaExceeded):
		return http.StatusInsufficientStorage
	case errors.Is(err, agent.ErrUploadNotReceiving), errors.Is(err, agent.ErrUploadInvalidChunk),
		errors.Is(err, agent.ErrUploadSizeInvalid), errors.Is(err, agent.ErrUploadIncomplete),
		errors.Is(err, agent.ErrUploadHashMismatch), errors.Is(err, agent.ErrUploadFilename):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func writeUploadError(c *gin.Context, err error) {
	status := uploadStatus(err)
	message := "upload operation failed"
	switch {
	case errors.Is(err, agent.ErrUploadNotFound):
		message = "upload session not found"
	case errors.Is(err, agent.ErrUploadForbidden):
		message = "upload session is outside the active organization"
	case errors.Is(err, agent.ErrUploadQuotaExceeded):
		message = "upload exceeds organization quota"
	case errors.Is(err, agent.ErrUploadNotReceiving):
		message = "upload session is not receiving"
	case errors.Is(err, agent.ErrUploadInvalidChunk):
		message = "upload chunk is invalid or out of bounds"
	case errors.Is(err, agent.ErrUploadSizeInvalid):
		message = "upload total size is invalid"
	case errors.Is(err, agent.ErrUploadIncomplete):
		message = "upload is not complete"
	case errors.Is(err, agent.ErrUploadHashMismatch):
		message = "upload hash does not match"
	case errors.Is(err, agent.ErrUploadFilename):
		message = "upload filename is invalid"
	case status == http.StatusInternalServerError:
		message = "upload storage operation failed"
	}
	writeAgentError(c, status, errors.New(message))
}

type startUploadRequest struct {
	ProjectID      string `json:"project_id"`
	Filename       string `json:"filename"`
	TotalSize      int64  `json:"total_size"`
	ChunkSize      int64  `json:"chunk_size"`
	ExpectedSHA256 string `json:"sha256"`
}

func (a *agentAPI) startUpload(c *gin.Context) {
	var req startUploadRequest
	if err := decodeJSON(c, &req); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	org := agentOrganizationID(c)
	session, err := a.runtime.Uploads().StartUpload(org, req.ProjectID, req.Filename, req.TotalSize, req.ChunkSize, req.ExpectedSHA256)
	if err != nil {
		writeUploadError(c, err)
		return
	}
	c.JSON(http.StatusCreated, session)
}

func (a *agentAPI) uploadChunk(c *gin.Context) {
	offset, err := strconv.ParseInt(c.Query("offset"), 10, 64)
	if err != nil || offset < 0 {
		writeAgentError(c, http.StatusBadRequest, errors.New("invalid or missing offset"))
		return
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, maxUploadChunkBytes+1))
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if len(data) > maxUploadChunkBytes {
		writeAgentError(c, http.StatusRequestEntityTooLarge, errors.New("chunk exceeds per-request limit"))
		return
	}
	org := agentOrganizationID(c)
	session, err := a.runtime.Uploads().AppendChunk(org, c.Param("id"), offset, data)
	if err != nil {
		writeUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

func (a *agentAPI) finalizeUpload(c *gin.Context) {
	org := agentOrganizationID(c)
	session, err := a.runtime.Uploads().FinalizeUpload(org, c.Param("id"))
	if err != nil {
		writeUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

func (a *agentAPI) cancelUpload(c *gin.Context) {
	org := agentOrganizationID(c)
	session, err := a.runtime.Uploads().CancelUpload(org, c.Param("id"))
	if err != nil {
		writeUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}

func (a *agentAPI) getUpload(c *gin.Context) {
	org := agentOrganizationID(c)
	session, err := a.runtime.Uploads().GetUploadForOrganization(org, c.Param("id"))
	if err != nil {
		writeUploadError(c, err)
		return
	}
	c.JSON(http.StatusOK, session)
}
