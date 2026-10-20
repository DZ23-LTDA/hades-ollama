package server

import (
	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// sessaoDeBuild monta o middleware que injecta a organizacao da sessao nos testes de build.
//
// Estes handlers passaram a EXIGIR organizacao (fronteira de tenant medida em
// `audit_tenant_guards`): sem ela respondem 401 e nao chegam a validar o corpo. Um teste que
// quisesse 400/412 sem sessao estaria a pedir o contrato antigo -- e o contrato antigo era o
// defeito: qualquer sessao lia builds de qualquer tenant.
func sessaoDeBuild(organizacaoID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("agent.user", agent.User{ID: "usr_teste"})
		c.Set("agent.organization", agent.Organization{ID: organizacaoID})
		c.Next()
	}
}
