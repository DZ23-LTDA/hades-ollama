package server

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

const (
	maxMissionAttachmentCount       = 10
	maxMissionAttachmentFileBytes   = 10 << 20
	maxMissionAttachmentTotalBytes  = 32 << 20
	maxMissionAttachmentRequestSize = maxMissionAttachmentTotalBytes + (1 << 20)
)

func createMissionAttachmentArchive(c *gin.Context) (path, projectName string, size int64, err error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMissionAttachmentRequestSize)
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		return "", "", 0, errors.New("anexos inválidos ou maiores que o limite permitido")
	}
	form := c.Request.MultipartForm
	if form == nil {
		return "", "", 0, errors.New("formulário de anexos ausente")
	}
	defer form.RemoveAll()
	for key := range form.Value {
		if key != "name" {
			return "", "", 0, errors.New("campo multipart não reconhecido")
		}
	}
	for key := range form.File {
		if key != "files" {
			return "", "", 0, errors.New("campo de arquivo multipart não reconhecido")
		}
	}
	if len(form.Value["name"]) > 1 {
		return "", "", 0, errors.New("o nome do projeto deve ser informado uma única vez")
	}
	projectName = "Anexos da missão"
	if len(form.Value["name"]) == 1 {
		projectName = strings.TrimSpace(form.Value["name"][0])
		if len(projectName) > 120 {
			return "", "", 0, errors.New("o nome do projeto não pode exceder 120 caracteres")
		}
		if projectName == "" {
			projectName = "Anexos da missão"
		}
	}
	files := form.File["files"]
	if len(files) == 0 || len(files) > maxMissionAttachmentCount {
		return "", "", 0, fmt.Errorf("selecione entre 1 e %d arquivos", maxMissionAttachmentCount)
	}

	archive, err := os.CreateTemp("", "ollama-agent-attachments-*.zip")
	if err != nil {
		return "", "", 0, errors.New("não foi possível preparar os anexos")
	}
	path = archive.Name()
	cleanup := func() {
		_ = archive.Close()
		_ = os.Remove(path)
	}
	zipWriter := zip.NewWriter(archive)
	var total int64
	for index, header := range files {
		filename := strings.TrimSpace(header.Filename)
		if filename == "" || filename == "." || filename == ".." || filepath.Base(filename) != filename || strings.ContainsAny(filename, `/\\`) || strings.Contains(filename, "..") {
			cleanup()
			return "", "", 0, errors.New("um anexo tem nome de arquivo inválido")
		}
		if !agent.SupportedProjectDocumentFilename(filename) {
			cleanup()
			return "", "", 0, fmt.Errorf("formato não suportado para %s", filename)
		}
		if header.Size < 0 || header.Size > maxMissionAttachmentFileBytes || total+header.Size > maxMissionAttachmentTotalBytes {
			cleanup()
			return "", "", 0, errors.New("os anexos excedem o limite de tamanho permitido")
		}
		input, openErr := header.Open()
		if openErr != nil {
			cleanup()
			return "", "", 0, errors.New("não foi possível ler um dos anexos")
		}
		entryName := fmt.Sprintf("attachments/%02d-%s", index+1, filename)
		entry, createErr := zipWriter.CreateHeader(&zip.FileHeader{Name: entryName, Method: zip.Deflate})
		if createErr != nil {
			_ = input.Close()
			cleanup()
			return "", "", 0, errors.New("não foi possível empacotar os anexos")
		}
		written, copyErr := io.Copy(entry, io.LimitReader(input, maxMissionAttachmentFileBytes+1))
		closeErr := input.Close()
		if copyErr != nil || closeErr != nil || written > maxMissionAttachmentFileBytes || written != header.Size || total+written > maxMissionAttachmentTotalBytes {
			cleanup()
			return "", "", 0, errors.New("um anexo excedeu o limite de tamanho ou foi enviado incompleto")
		}
		total += written
	}
	if err := zipWriter.Close(); err != nil {
		cleanup()
		return "", "", 0, errors.New("não foi possível finalizar os anexos")
	}
	if err := archive.Sync(); err != nil {
		cleanup()
		return "", "", 0, errors.New("não foi possível gravar os anexos")
	}
	if err := archive.Close(); err != nil {
		_ = os.Remove(path)
		return "", "", 0, errors.New("não foi possível finalizar os anexos")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() <= 0 || info.Size() > maxMissionAttachmentRequestSize {
		_ = os.Remove(path)
		return "", "", 0, errors.New("o pacote de anexos excede o limite permitido")
	}
	return path, projectName, info.Size(), nil
}

func (a *agentAPI) importMissionAttachments(c *gin.Context) {
	organizationID, ok := a.requireContextOrganization(c)
	if !ok {
		return
	}
	importer := a.scopedRuntime(c).ProjectImporter()
	if importer == nil {
		writeAgentError(c, http.StatusNotImplemented, errors.New("project importer is not configured"))
		return
	}
	archivePath, name, size, err := createMissionAttachmentArchive(c)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	defer os.Remove(archivePath)
	upload := agent.UploadSession{
		Filename:       "attachments.zip",
		OrganizationID: organizationID,
		TotalSize:      size,
		State:          agent.UploadCompleted,
		FinalPath:      archivePath,
	}
	result, err := importer.ImportUpload(c.Request.Context(), organizationID, upload, name)
	if err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	result.Source = "attachments"
	c.JSON(http.StatusCreated, result)
}
