package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// Rotas do motor de comércio (estágio 24). O checkout exige a chave de idempotência:
// sem ela, um reenvio criaria um segundo pedido.
func (a *agentAPI) commerce() *agent.CommerceStore {
	if a.runtime == nil {
		return nil
	}
	return a.runtime.Commerce()
}

var errCommerceUnavailable = agent.ErrCommerceInvalid

func (a *agentAPI) upsertCommerceProduct(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		SKU        string `json:"sku"`
		Name       string `json:"name"`
		PriceCents int64  `json:"price_cents"`
		Currency   string `json:"currency"`
		Active     *bool  `json:"active"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	product, err := store.UpsertProduct(agent.CommerceProduct{
		OrganizationID: agentOrganizationID(c), SKU: input.SKU, Name: input.Name,
		PriceCents: input.PriceCents, Currency: input.Currency, Active: active,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, product)
}

func (a *agentAPI) listCommerceProducts(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	products := store.Products(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"products": products, "count": len(products)})
}

// recordCommerceStock lança movimento de estoque (entrada, ajuste ou saída).
func (a *agentAPI) recordCommerceStock(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		SKU       string `json:"sku"`
		Delta     int64  `json:"delta"`
		Kind      string `json:"kind"`
		Reference string `json:"reference"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	movement, err := store.RecordStock(agentOrganizationID(c), input.SKU, input.Delta, input.Kind, input.Reference, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"movement": movement, "balance": store.Stock(agentOrganizationID(c), input.SKU)})
}

func (a *agentAPI) commerceStock(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	sku := c.Param("sku")
	if sku == "" {
		sku = c.Query("sku")
	}
	c.JSON(http.StatusOK, gin.H{
		"balance":   store.Stock(agentOrganizationID(c), sku),
		"movements": store.Movements(agentOrganizationID(c), sku),
	})
}

// createCommerceOrder usa a chave de idempotência do CABEÇALHO (ou do corpo). Criar
// de novo com a mesma chave devolve 200 com o MESMO pedido, marcado como duplicado.
func (a *agentAPI) createCommerceOrder(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		ContactID      string `json:"contact_id"`
		Currency       string `json:"currency"`
		IdempotencyKey string `json:"idempotency_key"`
		Items          []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		} `json:"items"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		key = input.IdempotencyKey
	}
	items := make([]agent.CommerceOrderItem, 0, len(input.Items))
	for _, item := range input.Items {
		items = append(items, agent.CommerceOrderItem{SKU: item.SKU, Quantity: item.Quantity})
	}
	order, err := store.CreateOrder(agent.CommerceOrderRequest{
		OrganizationID: agentOrganizationID(c),
		ContactID:      input.ContactID,
		Currency:       input.Currency,
		IdempotencyKey: key,
		Items:          items,
	}, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	status := http.StatusCreated
	if order.Duplicated {
		status = http.StatusOK
	}
	c.JSON(status, gin.H{"order": order, "duplicated": order.Duplicated})
}

func (a *agentAPI) listCommerceOrders(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	orders := store.Orders(agentOrganizationID(c))
	c.JSON(http.StatusOK, gin.H{"orders": orders, "count": len(orders)})
}

func (a *agentAPI) getCommerceOrder(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	order, err := store.Order(agentOrganizationID(c), c.Param("id"))
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, order)
}

// confirmCommercePayment aceita SOMENTE referência de pagamento — nunca dado de cartão.
func (a *agentAPI) confirmCommercePayment(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		Method      string `json:"method"`
		Reference   string `json:"reference"`
		AmountCents int64  `json:"amount_cents"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	order, err := store.ConfirmPayment(agentOrganizationID(c), c.Param("id"), agent.CommercePayment{
		Method: input.Method, Reference: input.Reference, AmountCents: input.AmountCents,
	}, agentActorID(c), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (a *agentAPI) reconcileCommerceOrder(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		Reference string `json:"reference"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	order, err := store.Reconcile(agentOrganizationID(c), c.Param("id"), input.Reference, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (a *agentAPI) fulfillCommerceOrder(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	order, err := store.Fulfill(agentOrganizationID(c), c.Param("id"), time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (a *agentAPI) cancelCommerceOrder(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(c, &input); err != nil {
		writeAgentError(c, http.StatusBadRequest, err)
		return
	}
	order, err := store.Cancel(agentOrganizationID(c), c.Param("id"), input.Reason, time.Now().UTC())
	if err != nil {
		writeAgentError(c, statusForAgentError(err), err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (a *agentAPI) commerceSummary(c *gin.Context) {
	store := a.commerce()
	if store == nil {
		writeAgentError(c, http.StatusServiceUnavailable, errCommerceUnavailable)
		return
	}
	c.JSON(http.StatusOK, store.Summary(agentOrganizationID(c)))
}
