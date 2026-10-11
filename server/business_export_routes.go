package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Exportação do painel de negócio (estágio 32).
//
// O arquivo sai com `Content-Type` de CSV e `Content-Disposition` de anexo, para o navegador baixar em
// vez de exibir. O nome do arquivo é derivado do escopo, sem dado pessoal. As lacunas (`nao_medido`) e
// a origem de cada número vão DENTRO do arquivo: fora do produto, a tela não está lá para explicar.
func (a *agentAPI) businessExport(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	document, err := a.runtime.BusinessExportCSV(company.OrganizationID, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "hades-business-status.csv"))
	c.String(http.StatusOK, document)
}
