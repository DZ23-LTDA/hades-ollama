package agent

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Aviso de aprovação pendente: um webhook OPCIONAL para quem opera saber que existe pedido esperando.
//
// Quatro decisões deliberadas:
//
//  1. DESLIGADO por padrão. Nada sai da máquina sem alguém configurar o destino — e o contrário (avisar
//     "por padrão" para um destino inventado) seria enviar dado de organização para terceiro sem pedido.
//  2. A falha do aviso NÃO bloqueia a aprovação. O controle é a aprovação humana; o aviso é conveniência.
//     Um webhook fora do ar que travasse escrita seria trocar um problema de operação por um incidente.
//  3. PAYLOAD MÍNIMO: identificador, ferramenta, valor e quantas decisões faltam. Sem impressão digital dos
//     parâmetros, sem contato, sem conteúdo de conversa — um webhook vaza o que ele carrega.
//  4. O SEGREDO não aparece em erro, log ou resposta: ele vai só no cabeçalho de assinatura, e o erro do
//     aviso é descrito por CLASSE (status, tempo esgotado, conexão recusada), nunca pela URL.

// ServiceAgentApprovalNotification é o aviso enviado quando um pedido pendente é CRIADO.
type ServiceAgentApprovalNotification struct {
	Event             string    `json:"event"`
	ApprovalID        string    `json:"approval_id"`
	OrganizationID    string    `json:"organization_id"`
	ToolName          string    `json:"tool_name"`
	ValueCents        int64     `json:"value_cents,omitempty"`
	RequiredApprovals int       `json:"required_approvals"`
	RequestedAt       time.Time `json:"requested_at"`
}

// ServiceAgentNotifier avisa que existe pedido de aprovação esperando decisão.
type ServiceAgentNotifier interface {
	NotifyPendingApproval(ctx context.Context, notification ServiceAgentApprovalNotification) error
}

// ServiceAgentWebhookNotifier entrega o aviso por HTTP POST, com assinatura HMAC-SHA256 do corpo.
type ServiceAgentWebhookNotifier struct {
	// URL é o destino. Vazia significa "sem destino": o aviso não é tentado.
	URL string
	// Secret assina o corpo (`X-Hades-Signature: sha256=...`) e NUNCA vai no corpo nem em erro.
	Secret string
	// Client permite injetar transporte em teste; nil usa um cliente com Timeout.
	Client *http.Client
	// Timeout limita a chamada; zero usa DefaultServiceAgentNotifyTimeout.
	Timeout time.Duration
}

// DefaultServiceAgentNotifyTimeout é o tempo máximo do aviso.
//
// Curto de propósito: o aviso é conveniência, e uma espera longa seguraria o turno do agente por causa de
// um destino lento.
const DefaultServiceAgentNotifyTimeout = 5 * time.Second

// NotifyPendingApproval entrega o aviso assinado. A URL vazia não é erro: é "sem destino configurado".
func (n *ServiceAgentWebhookNotifier) NotifyPendingApproval(ctx context.Context, notification ServiceAgentApprovalNotification) error {
	if n == nil || strings.TrimSpace(n.URL) == "" {
		return nil
	}
	if notification.Event == "" {
		notification.Event = "approval.pending"
	}
	corpo, err := json.Marshal(notification)
	if err != nil {
		return errors.New("aviso de aprovação falhou: corpo inválido")
	}
	tempo := n.Timeout
	if tempo <= 0 {
		tempo = DefaultServiceAgentNotifyTimeout
	}
	cliente := n.Client
	if cliente == nil {
		cliente = &http.Client{Timeout: tempo}
	}
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(corpo))
	if err != nil {
		// O erro do net/http carrega a URL: aqui ela é DESCARTADA de propósito.
		return errors.New("aviso de aprovação falhou: destino inválido")
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("X-Hades-Event", notification.Event)
	if assinatura := n.signature(corpo); assinatura != "" {
		requisicao.Header.Set("X-Hades-Signature", assinatura)
	}
	resposta, err := cliente.Do(requisicao)
	if err != nil {
		return errors.New("aviso de aprovação falhou: destino não respondeu")
	}
	defer resposta.Body.Close()
	if resposta.StatusCode < 200 || resposta.StatusCode > 299 {
		return fmt.Errorf("aviso de aprovação falhou: destino respondeu %d", resposta.StatusCode)
	}
	return nil
}

// signature devolve a assinatura HMAC-SHA256 do corpo (vazia quando não há segredo).
func (n *ServiceAgentWebhookNotifier) signature(corpo []byte) string {
	segredo := strings.TrimSpace(n.Secret)
	if segredo == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write(corpo)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
