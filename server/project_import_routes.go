package server

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

type projectImportRequest struct {
	URL       string `json:"url,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Name      string `json:"name,omitempty"`
	UploadID  string `json:"upload_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

func (a *agentAPI) importGitHubProject(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var request projectImportRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	importer := a.scopedRuntime(c).ProjectImporter()
	if importer == nil {
		writeAgentError(c, http.StatusNotImplemented, errors.New("project importer is not configured"))
		return
	}
	importer.GitHubToken = func() string { return os.Getenv("OLLAMA_GITHUB_TOKEN") }
	importer.GitHubTokenForOrganization = func(orgID string) string {
		if token, err := a.scopedRuntime(c).OAuthAccessTokenForOrganization(orgID, "github"); err == nil {
			return token
		}
		if orgID == agent.LocalOrganizationID {
			return os.Getenv("OLLAMA_GITHUB_TOKEN")
		}
		return ""
	}
	result, err := importer.ImportGitHub(c.Request.Context(), organizationID, agent.ProjectImportRequest{URL: request.URL, Ref: request.Ref, Name: request.Name})
	if err != nil {
		if errors.Is(err, agent.ErrGitHubAuthRequired) {
			c.JSON(http.StatusFailedDependency, gin.H{"error": "GitHub authentication is not configured for private repositories", "gate_status": "NOT_CONFIGURED", "action": "connect GitHub on the server with OLLAMA_GITHUB_TOKEN"})
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) importZIPProject(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	var request projectImportRequest
	if err := decodeJSON(c, &request); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(request.UploadID) == "" || strings.TrimSpace(request.ProjectID) == "" {
		writeAgentError(c, http.StatusBadRequest, errors.New("upload_id and project_id are required for ZIP import"))
		return
	}
	project, err := a.projectForRequestWithID(c, request.ProjectID)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	upload, err := a.runtime.Uploads().GetUploadForOrganization(organizationID, request.UploadID)
	if err != nil {
		writeUploadError(c, err)
		return
	}
	if upload.ProjectID != "" && upload.ProjectID != project.ID {
		writeAgentError(c, http.StatusForbidden, errors.New("upload belongs to a different project"))
		return
	}
	importer := a.scopedRuntime(c).ProjectImporter()
	if importer == nil {
		writeAgentError(c, http.StatusNotImplemented, errors.New("project importer is not configured"))
		return
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = project.Name
	}
	result, err := importer.ImportUpload(c.Request.Context(), organizationID, upload, name)
	if err != nil {
		if errors.Is(err, agent.ErrUploadForbidden) {
			writeAgentError(c, http.StatusForbidden, err)
			return
		}
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

func (a *agentAPI) projectForRequestWithID(c *gin.Context, id string) (agent.Project, error) {
	project, err := a.scopedRuntime(c).Context().GetProject(strings.TrimSpace(id))
	if err != nil {
		return agent.Project{}, err
	}
	if !contextRecordOwnedByOrganization(project.OrganizationID, a.organizationID(c)) {
		return agent.Project{}, errAgentForbidden
	}
	return project, nil
}
