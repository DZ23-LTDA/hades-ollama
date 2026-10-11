package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Superfície HTTP do marketplace (estágio 25).
//
// A capacidade existia apenas no runtime; sem superfície, sincronizar catálogo e estoque não era
// operável por quem usa o produto. Tudo aqui é escopado pela organização da sessão via
// `companyForRequest`, e **nada publica em plataforma externa**: a sincronização roda contra o duble
// de sandbox e o relatório declara isso.
//
// O estado dos canais é derivado apenas da EXISTÊNCIA das variáveis de credencial: os nomes são
// conhecidos, os VALORES nunca são lidos para exibição, registro ou resposta. Credencial presente
// não significa operação — o catálogo devolve `UNAUTHORIZED` até haver handshake real.

func (a *agentAPI) marketplaceChannels(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"channels": agent.MarketplaceChannels(configuredMarketplaceCredentials())})
}

func (a *agentAPI) marketplaceSync(c *gin.Context) {
	company, err := a.companyForRequest(c)
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	var input struct {
		ChannelID string                      `json:"channel_id"`
		Direction string                      `json:"direction"`
		Items     []agent.MarketplaceSyncItem `json:"items"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	report, err := a.runtime.Marketplace().Sync(agent.MarketplaceSyncRequest{
		OrganizationID: company.OrganizationID,
		ChannelID:      input.ChannelID,
		Direction:      input.Direction,
		Items:          input.Items,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, report)
}

// marketplaceMirror mostra o espelho remoto do DUBLE, para inspeção operacional e verificação.
func (a *agentAPI) marketplaceMirror(c *gin.Context) {
	if _, err := a.companyForRequest(c); err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	channelID := strings.TrimSpace(c.Param("channel_id"))
	sku := strings.TrimSpace(c.Param("sku"))
	quantity, present := a.runtime.Marketplace().RemoteStock(channelID, sku)
	c.JSON(http.StatusOK, gin.H{
		"channel_id":       channelID,
		"sku":              sku,
		"present":          present,
		"quantity":         quantity,
		"sandbox":          true,
		"homologacao_live": agent.MarketplaceLiveStatus,
		"motivo":           agent.MarketplaceLiveReason,
	})
}

// configuredMarketplaceCredentials devolve quais variáveis de credencial EXISTEM, apenas para
// derivar o estado do catálogo. Nenhum valor é copiado, comparado por conteúdo ou exposto.
func configuredMarketplaceCredentials() map[string]bool {
	configured := map[string]bool{}
	for _, spec := range agent.MarketplaceChannels(map[string]bool{}) {
		configured[spec.CredentialEnv] = os.Getenv(spec.CredentialEnv) != ""
	}
	return configured
}
