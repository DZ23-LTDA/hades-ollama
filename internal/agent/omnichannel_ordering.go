package agent

import (
	"strings"
	"time"
)

// ORDENACAO POR CONVERSA (E-298) — estágio 34.
//
// A ordem de CHEGADA não é a ordem do provedor: webhooks se reentregam, redes atrasam e canais paralelos competem.
// Reordenar de verdade exigiria buffer com espera (atrasa a entrega e pode nunca completar) e descartar perderia
// dado. O que se pode fazer com honestidade é MANTER o último instante aceito, MARCAR o evento que chega atrás e
// CONTAR — sem fingir ordem que não existe.

// OutOfOrder marca o evento que chegou ANTES do último já aceito na mesma conversa. Ele NÃO é descartado: o dado
// entra; o que muda é que a ordem fica registrada em vez de presumida.
const omnichannelOrderKeySeparator = "|"

// orderKey identifica a CONVERSA a que a ordem pertence. Sem ConversationID, o contato é a melhor aproximação —
// nunca a organização sozinha, que misturaria conversas distintas na mesma linha do tempo.
func omnichannelOrderKey(organizationID, channel, conversationID, contactID string) string {
	conversa := strings.TrimSpace(conversationID)
	if conversa == "" {
		conversa = strings.TrimSpace(contactID)
	}
	return strings.TrimSpace(organizationID) + omnichannelOrderKeySeparator +
		strings.TrimSpace(channel) + omnichannelOrderKeySeparator + conversa
}

// observeOrderLocked registra o instante aceito e devolve true quando o evento chegou FORA DE ORDEM.
//
// O chamador precisa segurar r.mu. Comparação ESTRITA: instante igual não é fora de ordem (reentrega simultânea,
// empate de relógio ou segundo do provedor não podem virar alarme falso).
func (r *OmnichannelRuntime) observeOrderLocked(organizationID, channel, conversationID, contactID string, at time.Time) bool {
	if r == nil || at.IsZero() {
		return false
	}
	chave := omnichannelOrderKey(organizationID, channel, conversationID, contactID)
	if r.orderHeads == nil {
		r.orderHeads = map[string]int64{}
	}
	instante := at.UTC().Unix()
	if anterior, temos := r.orderHeads[chave]; temos && instante < anterior {
		r.outOfOrder++
		return true
	}
	r.orderHeads[chave] = instante
	return false
}

// OutOfOrderCount informa quantos eventos chegaram fora de ordem DESDE O INÍCIO DO PROCESSO. É visibilidade, não
// correção: o número existe para o status poder mostrar o fenômeno em vez de escondê-lo.
func (r *OmnichannelRuntime) OutOfOrderCount() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.outOfOrder
}
