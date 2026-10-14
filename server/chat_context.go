package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/internal/agent"
	"github.com/ollama/ollama/version"
)

// Hades Context Bootstrap no caminho de chat — estágio 5.
//
// O preâmbulo canônico já era injetado no planejador e na rota de missão, mas o
// chat continuava conversando "cru": o modelo não recebia identidade, modo,
// permissões declaradas nem os limites da sessão. Aqui o preâmbulo passa a
// acompanhar o chat da superfície Hades (`/api/agent/v1/chat`), SEM duplicar a
// inferência: a rota reusa o mesmo `ChatHandler` e apenas injeta a primeira
// mensagem de sistema antes dele.
//
// Compatibilidade preservada de propósito: `/api/chat` (o contrato de sempre,
// usado pelo CLI e por clientes OpenAI-compatible) NÃO é alterado.
const (
	// Marca a presença do preâmbulo para a injeção ser idempotente.
	chatContextPreambleMarker = "Você opera dentro do Hades"
	// Limite defensivo do corpo lido pelo middleware (streams grandes passam no
	// handler de chat, não aqui).
	maxChatContextBodyBytes = 32 << 20
	// O objetivo vem da última mensagem do usuário, truncado para não inflar o
	// contexto nem vazar um texto gigante para o modelo.
	maxChatObjectiveRunes = 400
)

// buildChatContextPreamble monta o preâmbulo do Hades para uma requisição de chat.
func buildChatContextPreamble(organizationID, projectID string, request api.ChatRequest, granted []string, policy agent.CapabilityPolicy) (string, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return "", fmt.Errorf("%w: organização é obrigatória para montar o contexto", agent.ErrContextBootstrapEmpty)
	}
	// Escopos concedidos passam pela política: escopo desconhecido não vira
	// permissão declarada (a validação falha e o preâmbulo sai sem esses escopos).
	validated := granted
	if len(granted) > 0 {
		checked, err := policy.ValidateMissionCapabilities(granted)
		if err != nil {
			return "", err
		}
		validated = checked
	}
	bootstrap, err := agent.BuildContextBootstrap(agent.ContextBootstrapInput{
		Version:             version.Version,
		OrganizationID:      organizationID,
		ProjectID:           strings.TrimSpace(projectID),
		Objective:           chatObjective(request.Messages),
		GrantedCapabilities: validated,
		Policy:              policy,
		UntrustedSources:    []string{"web", "email", "whatsapp", "telegram", "mcp", "arquivos"},
	})
	if err != nil {
		return "", err
	}
	return bootstrap.SystemMessage(), nil
}

// chatObjective usa a última mensagem do usuário como objetivo da sessão.
func chatObjective(messages []api.Message) string {
	for index := len(messages) - 1; index >= 0; index-- {
		if !strings.EqualFold(strings.TrimSpace(messages[index].Role), "user") {
			continue
		}
		content := strings.TrimSpace(messages[index].Content)
		runes := []rune(content)
		if len(runes) > maxChatObjectiveRunes {
			return string(runes[:maxChatObjectiveRunes]) + "…"
		}
		return content
	}
	return ""
}

// injectContextBootstrap devolve as mensagens com o preâmbulo como primeira
// mensagem de sistema. É idempotente (não duplica se já houver um preâmbulo) e
// não altera o slice recebido.
func injectContextBootstrap(messages []api.Message, preamble string) []api.Message {
	preamble = strings.TrimSpace(preamble)
	if preamble == "" {
		return messages
	}
	for _, message := range messages {
		content := strings.TrimSpace(message.Content)
		// Duas formas de já haver contexto: o preâmbulo canônico (marcador) ou
		// exatamente o preâmbulo que seria injetado agora. Sem isso, reenviar a
		// conversa duplicaria o contexto a cada turno.
		if strings.Contains(content, chatContextPreambleMarker) || content == preamble {
			return messages
		}
	}
	injected := make([]api.Message, 0, len(messages)+1)
	injected = append(injected, api.Message{Role: "system", Content: preamble})
	injected = append(injected, messages...)
	return injected
}

// withAgentChatContext injeta o preâmbulo no corpo da requisição de chat antes do
// handler. Qualquer falha na leitura do corpo devolve o pedido original: a
// injeção de contexto não pode impedir uma conversa legítima, e o erro de corpo
// inválido continua sendo do handler de chat.
func withAgentChatContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c == nil || c.Request == nil || c.Request.Body == nil {
			c.Next()
			return
		}
		organizationID := agentOrganizationFromContext(c)
		if organizationID == "" {
			c.Next()
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxChatContextBodyBytes+1))
		if err != nil || len(body) > maxChatContextBodyBytes {
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			c.Next()
			return
		}
		// Repõe o corpo para o handler seguinte, com ou sem injeção.
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		var request api.ChatRequest
		if err := json.Unmarshal(body, &request); err != nil || len(request.Messages) == 0 {
			c.Next()
			return
		}
		projectID := strings.TrimSpace(c.GetHeader("X-Hades-Project"))
		preamble, err := buildChatContextPreamble(organizationID, projectID, request, agentChatGrantedCapabilities(c), agent.DefaultCapabilityPolicy())
		if err != nil {
			// Contexto indisponível não bloqueia a conversa: o modelo responde sem
			// o preâmbulo, e o motivo fica no log do servidor.
			c.Next()
			return
		}
		// ORÇAMENTO DE CONTEXTO (§2 do complemento universal de memória): medir e
		// DECIDIR por orçamento ANTES de enviar ao modelo — e não apenas depois de o
		// modelo recusar o contexto, que era o único gatilho existente. Sem
		// `OLLAMA_AGENT_CHAT_CONTEXT_TOKENS` declarado o teto é 0, a decisão é "não
		// declarado" e NADA muda: esta ligação não altera comportamento por omissão.
		// A decisão fica registada no contexto do pedido (não só num log) para que o
		// handler, uma tela de Cowork ou um teste a possam LER.
		RegistarDecisaoDeOrcamento(c, DecidirOrcamentoDoChat(preamble, request.Messages, OrcamentoDeContextoDoChat()))
		request.Messages = injectContextBootstrap(request.Messages, preamble)
		encoded, err := json.Marshal(request)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(encoded))
		c.Request.ContentLength = int64(len(encoded))
		c.Next()
	}
}

// agentOrganizationFromContext lê o escopo já resolvido pelo middleware agentic.
func agentOrganizationFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get("agent.organization"); ok {
		if organization, ok := value.(agent.Organization); ok {
			return strings.TrimSpace(organization.ID)
		}
	}
	return ""
}

// agentChatGrantedCapabilities devolve os escopos declarados para esta sessão.
// Por padrão o chat recebe leitura de workspace; nada além disso é concedido sem
// pedido explícito — e o pedido passa pela política.
func agentChatGrantedCapabilities(c *gin.Context) []string {
	if c == nil {
		return []string{"workspace:read"}
	}
	raw := strings.TrimSpace(c.GetHeader("X-Hades-Capabilities"))
	if raw == "" {
		return []string{"workspace:read"}
	}
	parts := strings.Split(raw, ",")
	granted := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			granted = append(granted, trimmed)
		}
	}
	if len(granted) == 0 {
		return []string{"workspace:read"}
	}
	return granted
}
