package agent

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Estágio 29 — caixa de entrada unificada.
//
// As propriedades travadas aqui:
//  1. UMA conversa por contato e canal: mensagem nova reaproveita a conversa aberta,
//     porque criar uma por mensagem destruiria histórico e contexto;
//  2. DEDUPLICAÇÃO de webhook por `ExternalID`, inclusive sob concorrência: entrega
//     repetida NÃO pode virar mensagem duplicada (§11.7);
//  3. SLA MEDIDO contra a primeira resposta real, não presumido;
//  4. handoff é ESTADO com motivo e responsável, e a passagem fica registrada;
//  5. responder respeita o CONSENTIMENTO do contato;
//  6. isolamento por organização e persistência.
func inboxStore(t *testing.T) (*InboxStore, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "inbox")
	store, err := NewInboxStore(root)
	if err != nil {
		t.Fatal(err)
	}
	return store, root
}

func TestInboxReusesConversationPerContactAndChannel(t *testing.T) {
	store, _ := inboxStore(t)
	now := time.Now().UTC()
	first, message, err := store.AppendInbound("org-a", "whatsapp", "+5511999998888", "Ana", "primeira mensagem", "wamid.1", now)
	if err != nil {
		t.Fatal(err)
	}
	if message.Deduplicated {
		t.Fatal("a primeira mensagem não é deduplicada")
	}
	second, _, err := store.AppendInbound("org-a", "whatsapp", "+5511999998888", "Ana", "segunda mensagem", "wamid.2", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("o mesmo contato e canal precisa REAPROVEITAR a conversa: %s vs %s", second.ID, first.ID)
	}
	if second.MessageCount != 2 {
		t.Fatalf("a conversa precisa acumular as mensagens: %d", second.MessageCount)
	}
	// CANAL diferente é outra conversa: o histórico de cada canal é próprio.
	telegram, _, err := store.AppendInbound("org-a", "telegram", "+5511999998888", "Ana", "mensagem no telegram", "tg.1", now)
	if err != nil {
		t.Fatal(err)
	}
	if telegram.ID == first.ID {
		t.Fatal("canais diferentes precisam de conversas diferentes")
	}
	// ORGANIZAÇÃO diferente também.
	outra, _, err := store.AppendInbound("org-b", "whatsapp", "+5511999998888", "Ana", "na outra org", "wamid.1", now)
	if err != nil {
		t.Fatal(err)
	}
	if outra.ID == first.ID {
		t.Fatal("organizações diferentes precisam de conversas diferentes")
	}
	// Conversa encerrada NÃO é reaproveitada: a próxima mensagem abre uma nova.
	if _, err := store.Close("org-a", first.ID, "ana", "resolvido", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	reaberta, _, err := store.AppendInbound("org-a", "whatsapp", "+5511999998888", "Ana", "voltou a falar", "wamid.3", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if reaberta.ID == first.ID {
		t.Fatal("depois de encerrada, a conversa nova precisa ser outra")
	}
	if reaberta.Status != InboxStatusOpen {
		t.Fatalf("a conversa nova precisa nascer aberta: %+v", reaberta)
	}
}

func TestInboxDeduplicatesRepeatedWebhookDelivery(t *testing.T) {
	store, _ := inboxStore(t)
	now := time.Now().UTC()
	_, original, err := store.AppendInbound("org-a", "whatsapp", "+5511977776666", "Bruno", "mensagem do webhook", "wamid.repetido", now)
	if err != nil {
		t.Fatal(err)
	}
	// Reentrega do MESMO evento: devolve a original marcada, sem criar outra.
	_, repeated, err := store.AppendInbound("org-a", "whatsapp", "+5511977776666", "Bruno", "mensagem do webhook", "wamid.repetido", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !repeated.Deduplicated {
		t.Fatal("a reentrega precisa ser marcada como deduplicada")
	}
	if repeated.ID != original.ID {
		t.Fatalf("a reentrega precisa devolver a mensagem ORIGINAL: %s vs %s", repeated.ID, original.ID)
	}
	_, messages, err := store.Conversation("org-a", original.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("webhook duplicado não pode duplicar mensagem: %d", len(messages))
	}

	// CONCORRÊNCIA: dez entregas simultâneas do mesmo evento.
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = store.AppendInbound("org-a", "telegram", "chat-9", "Carla", "evento concorrente", "tg.repetido", now)
		}()
	}
	wg.Wait()
	conversations := store.ListConversations("org-a", "")
	var target string
	for _, conversation := range conversations {
		if conversation.Channel == InboxChannelTelegram {
			target = conversation.ID
		}
	}
	if target == "" {
		t.Fatal("a conversa do telegram precisa existir")
	}
	_, telegramMessages, err := store.Conversation("org-a", target)
	if err != nil {
		t.Fatal(err)
	}
	if len(telegramMessages) != 1 {
		t.Fatalf("entregas concorrentes do mesmo evento precisam gerar UMA mensagem: %d", len(telegramMessages))
	}
	// Mensagem SEM external id não é deduplicada (não há chave confiável).
	if _, first, err := store.AppendInbound("org-a", "webchat", "sessao-1", "Dora", "sem id externo", "", now); err != nil {
		t.Fatal(err)
	} else if _, second, err := store.AppendInbound("org-a", "webchat", "sessao-1", "Dora", "sem id externo", "", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	} else if first.ID == second.ID {
		t.Fatal("sem chave externa não há como deduplicar: as mensagens são distintas")
	}
}

func TestInboxSLAIsMeasuredNotPresumed(t *testing.T) {
	store, _ := inboxStore(t)
	now := time.Now().UTC()
	store.SetFirstResponseSLA("org-a", 10*time.Minute)
	conversation, _, err := store.AppendInbound("org-a", "whatsapp", "+5511966665555", "Eva", "preciso de ajuda", "wamid.sla", now)
	if err != nil {
		t.Fatal(err)
	}
	if conversation.ReplyDueAt == nil {
		t.Fatal("a conversa precisa nascer com prazo de primeira resposta")
	}
	metrics := store.Metrics("org-a", now.Add(time.Minute))
	if metrics.AwaitingFirstReply != 1 || metrics.SLABreached != 0 {
		t.Fatalf("antes do prazo não há violação: %+v", metrics)
	}
	// Depois do prazo, SEM resposta: violação medida.
	metrics = store.Metrics("org-a", now.Add(11*time.Minute))
	if metrics.SLABreached != 1 {
		t.Fatalf("passado o prazo sem resposta, o SLA precisa aparecer violado: %+v", metrics)
	}
	// Com resposta, o relógio para e a média é real.
	if _, _, err := store.AppendOutbound("org-a", conversation.ID, "agent", "bot", "já vou ajudar", nil, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	metrics = store.Metrics("org-a", now.Add(20*time.Minute))
	if metrics.SLABreached != 0 {
		t.Fatalf("respondida no prazo não pode contar violação: %+v", metrics)
	}
	if metrics.FirstResponseAvgSecs != 120 {
		t.Fatalf("a média de primeira resposta precisa ser 120s: %v", metrics.FirstResponseAvgSecs)
	}
	if metrics.MessagesInbound != 1 || metrics.MessagesOutbound != 1 {
		t.Fatalf("volume de mensagens incorreto: %+v", metrics)
	}
}

func TestInboxHandoffRequiresReasonAndMovesToHuman(t *testing.T) {
	store, _ := inboxStore(t)
	now := time.Now().UTC()
	conversation, _, err := store.AppendInbound("org-a", "whatsapp", "+5511955554444", "Fábio", "quero falar com uma pessoa", "wamid.handoff", now)
	if err != nil {
		t.Fatal(err)
	}
	// Handoff sem motivo é recusado: transferência sem justificativa não é auditável.
	if _, _, err := store.HandoffToHuman("org-a", conversation.ID, "bot", "ana", "   ", now); !errors.Is(err, ErrInboxInvalid) {
		t.Fatalf("handoff sem motivo precisa ser recusado, veio %v", err)
	}
	updated, message, err := store.HandoffToHuman("org-a", conversation.ID, "bot", "ana", "cliente pediu atendimento humano", now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != InboxStatusPendingHuman {
		t.Fatalf("a conversa precisa ir para atendimento humano: %+v", updated)
	}
	if updated.AssignedTo != "ana" || updated.AssignedKind != "human" {
		t.Fatalf("o responsável humano precisa ficar registrado: %+v", updated)
	}
	if !strings.Contains(message.Body, "cliente pediu atendimento humano") {
		t.Fatalf("a passagem precisa ficar na conversa: %+v", message)
	}
	// A conversa transferida aparece no indicador e na lista filtrada.
	metrics := store.Metrics("org-a", now)
	if metrics.PendingHuman != 1 {
		t.Fatalf("o indicador precisa contar a transferência: %+v", metrics)
	}
	if got := store.ListConversations("org-a", InboxStatusPendingHuman); len(got) != 1 {
		t.Fatalf("o filtro por status precisa encontrar a conversa: %+v", got)
	}
}

func TestInboxReplyRespectsConsent(t *testing.T) {
	store, _ := inboxStore(t)
	registry, err := NewConsentRegistry(filepath.Join(t.TempDir(), "consent"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	phone := "+5511944443333"
	conversation, _, err := store.AppendInbound("org-a", "whatsapp", phone, "Gina", "oi", "wamid.consent", now)
	if err != nil {
		t.Fatal(err)
	}
	// Sem consentimento registrado, responder é RECUSADO — a caixa de entrada não é
	// uma porta dos fundos para contatar quem não autorizou.
	if _, _, err := store.AppendOutbound("org-a", conversation.ID, "agent", "bot", "resposta", registry, now); !errors.Is(err, ErrInboxConsentRequired) {
		t.Fatalf("sem consentimento a resposta precisa ser recusada, veio %v", err)
	}
	if _, err := registry.GrantConsent("org-a", ConsentChannelWhatsApp, phone, "site", "checkbox", "ana", nil, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AppendOutbound("org-a", conversation.ID, "agent", "bot", "resposta autorizada", registry, now); err != nil {
		t.Fatalf("com consentimento a resposta precisa funcionar: %v", err)
	}
	// Opt-out volta a bloquear.
	if _, err := registry.RevokeConsent("org-a", ConsentChannelWhatsApp, phone, "pedido do titular", "bruno", now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AppendOutbound("org-a", conversation.ID, "agent", "bot", "insistência", registry, now); !errors.Is(err, ErrInboxConsentRequired) {
		t.Fatalf("após opt-out a resposta precisa ser recusada, veio %v", err)
	}
}

func TestInboxSurvivesRestartAndIsScopedByOrganization(t *testing.T) {
	store, root := inboxStore(t)
	now := time.Now().UTC()
	conversation, _, err := store.AppendInbound("org-a", "whatsapp", "+5511933332222", "Hugo", "mensagem persistente", "wamid.persist", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.AppendInbound("org-b", "whatsapp", "+5511922221111", "Iara", "da outra org", "wamid.outra", now); err != nil {
		t.Fatal(err)
	}
	// Leitura cruzada não enxerga a conversa da outra organização.
	if _, _, err := store.Conversation("org-b", conversation.ID); !errors.Is(err, ErrInboxConversationGone) {
		t.Fatalf("conversa de outra organização não pode ser lida, veio %v", err)
	}
	if got := len(store.ListConversations("org-b", "")); got != 1 {
		t.Fatalf("a listagem precisa ser escopada: %d", got)
	}

	restarted, err := NewInboxStore(root)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, messages, err := restarted.Conversation("org-a", conversation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.MessageCount != 1 || len(messages) != 1 {
		t.Fatalf("conversa e mensagens precisam sobreviver ao reinício: %+v", reloaded)
	}
	// E a deduplicação continua valendo depois do reinício: o índice foi reconstruído.
	if _, repeated, err := restarted.AppendInbound("org-a", "whatsapp", "+5511933332222", "Hugo", "mensagem persistente", "wamid.persist", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	} else if !repeated.Deduplicated {
		t.Fatal("a deduplicação precisa valer após o reinício")
	}
}
