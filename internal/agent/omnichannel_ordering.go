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

// SetOrderingStore liga a persistência da ORDENAÇÃO (E-299). Sem store, o estado é em memória e o runtime DIZ isso.
func (r *OmnichannelRuntime) SetOrderingStore(store *OmnichannelOrderingStore) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orderingStore = store
}

// OrderingPersisted informa se há persistência de ordem configurada. Sem store, o estado é em memória — e isso é
// DITO, não escondido.
func (r *OmnichannelRuntime) OrderingPersisted() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.orderingStore != nil && r.orderingStore.Root() != ""
}

// OrderingRoot devolve o diretório usado, para o status poder mostrar ONDE o estado vive.
func (r *OmnichannelRuntime) OrderingRoot() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.orderingStore == nil {
		return ""
	}
	return r.orderingStore.Root()
}

// LastOrderingError devolve a última falha de persistência de ordem. Falha NÃO é engolida: fica registrada para o
// status mostrar que o estado pode não sobreviver a um reinício.
func (r *OmnichannelRuntime) LastOrderingError() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.orderingSaveError != "" {
		return r.orderingSaveError
	}
	return r.orderingLoadError
}

// restoreOrderingLocked carrega as linhas do tempo da organização UMA vez. Erro de leitura é REGISTRADO e o runtime
// segue: perder a carga é pior do que perder o processo, mas não pode passar em silêncio.
func (r *OmnichannelRuntime) restoreOrderingLocked(organizationID string) {
	if r == nil || r.orderingStore == nil || r.orderingStore.Root() == "" {
		return
	}
	if r.loadedOrderOrgs == nil {
		r.loadedOrderOrgs = map[string]bool{}
	}
	if r.loadedOrderOrgs[organizationID] {
		return
	}
	r.loadedOrderOrgs[organizationID] = true
	carregado, err := r.orderingStore.Load(organizationID)
	if err != nil {
		r.orderingLoadError = err.Error()
		return
	}
	if r.orderHeads == nil {
		r.orderHeads = map[string]int64{}
	}
	for chave, instante := range carregado {
		if anterior, temos := r.orderHeads[chave]; !temos || instante > anterior {
			r.orderHeads[chave] = instante
		}
	}
}

// persistOrderingLocked grava as linhas do tempo da organização. Falha é REGISTRADA, nunca engolida.
func (r *OmnichannelRuntime) persistOrderingLocked(organizationID string) {
	if r == nil || r.orderingStore == nil || r.orderingStore.Root() == "" {
		return
	}
	if err := r.orderingStore.Save(organizationID, r.orderHeads); err != nil {
		r.orderingSaveError = err.Error()
		return
	}
	r.orderingSaveError = ""
}
