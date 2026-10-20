package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// requireOrganizacaoDaSessao exige que a sessao tenha organizacao e devolve-a.
//
// Existe para que a guarda de tenant tenha UM nome que o auditor reconhece, em vez de cada handler
// inventar a sua forma. Devolve 401 e `false` quando nao ha sessao -- quem chama apenas faz `return`.
func requireOrganizacaoDaSessao(c *gin.Context) (string, bool) {
	organizacao, ok := OrganizacaoDaSessao(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"erro": "sessao sem organizacao"})
		return "", false
	}
	return organizacao, true
}
