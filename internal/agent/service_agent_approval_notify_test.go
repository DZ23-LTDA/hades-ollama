package agent

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// AVISO de aprovação pendente (estágio 35) — o limite declarado era "sem notificação ao aprovador".
//
// O que estes testes travam:
//
//  1. DESLIGADO por padrão: sem destino configurado, ZERO requisição sai da máquina;
//  2. ligado, o pedido NOVO gera exatamente UM aviso, com payload MÍNIMO e assinatura válida;
//  3. repetir o turno (mesmo pedido reusado) NÃO reenvia — aviso duplicado vira spam para quem decide;
//  4. a falha do destino NÃO bloqueia a aprovação: o pedido continua pendente e decidível;
//  5. o SEGREDO não aparece em corpo, erro nem resposta — e a URL não vaza no erro.
type avisoGravado struct {
	cabecalhos http.Header
	corpo      []byte
}

type avisoColetor struct {
	mu       sync.Mutex
	pedidos  []avisoGravado
	status   int
	servidor *httptest.Server
}

func novoAvisoColetor(t *testing.T, status int) *avisoColetor {
	t.Helper()
	coletor := &avisoColetor{status: status}
	coletor.servidor = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		coletor.mu.Lock()
		coletor.pedidos = append(coletor.pedidos, avisoGravado{cabecalhos: r.Header.Clone(), corpo: corpo})
		coletor.mu.Unlock()
		if coletor.status == 0 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(coletor.status)
	}))
	t.Cleanup(coletor.servidor.Close)
	return coletor
}

func (c *avisoColetor) total() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pedidos)
}

func (c *avisoColetor) primeiro(t *testing.T) avisoGravado {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pedidos) == 0 {
		t.Fatal("nenhum aviso recebido")
	}
	return c.pedidos[0]
}

func TestServiceAgentApprovalNotificationIsOffByDefault(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	// O store vem SEM notificador: nenhum destino é inventado por padrão.
	if runtime.ApprovalStore().Notifier != nil {
		t.Fatal("nenhum destino deve existir por padrão")
	}
	if _, err := runtime.RunTurn(salesWriteTurn(conversationID, "Plano avançado", 79900), now); err != nil {
		t.Fatal(err)
	}
	if avisos := runtime.ApprovalStore().NotificationsSent(); avisos != 0 {
		t.Fatalf("sem destino configurado, nenhum aviso pode ser enviado: %d", avisos)
	}
}

func TestServiceAgentApprovalNotificationSendsOnceWithMinimalPayload(t *testing.T) {
	runtime, _, conversationID, now := approvalFixture(t)
	coletor := novoAvisoColetor(t, http.StatusOK)
	const segredo = "segredo-de-assinatura-do-hades"
	runtime.ApprovalStore().Notifier = &ServiceAgentWebhookNotifier{
		URL: coletor.servidor.URL, Secret: segredo, Client: coletor.servidor.Client(),
	}

	// Valor alto: o aviso precisa dizer que faltam DUAS decisões.
	pedido := salesWriteTurn(conversationID, "Plano corporativo", 250000)
	if _, err := runtime.RunTurn(pedido, now); err != nil {
		t.Fatal(err)
	}
	if avisos := runtime.ApprovalStore().NotificationsSent(); avisos != 1 {
		t.Fatalf("o pedido novo precisa gerar exatamente UM aviso: %d", avisos)
	}
	gravado := coletor.primeiro(t)

	// O corpo é MÍNIMO e não carrega impressão digital de parâmetros nem contato.
	var aviso ServiceAgentApprovalNotification
	if err := json.Unmarshal(gravado.corpo, &aviso); err != nil {
		t.Fatalf("corpo do aviso precisa ser JSON: %v (%s)", err, gravado.corpo)
	}
	if aviso.Event != "approval.pending" || aviso.ApprovalID == "" || aviso.ToolName != "crm.opportunity.write" {
		t.Fatalf("aviso incompleto: %+v", aviso)
	}
	if aviso.ValueCents != 250000 || aviso.RequiredApprovals != 2 {
		t.Fatalf("o aviso precisa dizer o valor e o que falta: %+v", aviso)
	}
	for _, proibido := range []string{"fingerprint", "conversation_id", "contact", segredo, "title"} {
		if strings.Contains(strings.ToLower(string(gravado.corpo)), strings.ToLower(proibido)) {
			t.Fatalf("o corpo do aviso não pode conter %q: %s", proibido, gravado.corpo)
		}
	}
	// ASSINATURA: o segredo vai só no cabeçalho, e a assinatura confere com o corpo.
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write(gravado.corpo)
	esperada := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if recebida := gravado.cabecalhos.Get("X-Hades-Signature"); recebida != esperada {
		t.Fatalf("assinatura divergente: recebida=%q esperada=%q", recebida, esperada)
	}
	if gravado.cabecalhos.Get("X-Hades-Event") != "approval.pending" {
		t.Fatalf("evento precisa ir no cabeçalho: %v", gravado.cabecalhos)
	}

	// REPETIR o turno reusa o mesmo pedido: NÃO reenvia (aviso duplicado vira spam).
	if _, err := runtime.RunTurn(pedido, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if total := coletor.total(); total != 1 {
		t.Fatalf("repetir o turno não pode reenviar o aviso: %d", total)
	}
	if avisos := runtime.ApprovalStore().NotificationsSent(); avisos != 1 {
		t.Fatalf("contagem de avisos precisa seguir em um: %d", avisos)
	}
}

func TestServiceAgentApprovalNotificationFailureDoesNotBlock(t *testing.T) {
	runtime, crm, conversationID, now := approvalFixture(t)
	coletor := novoAvisoColetor(t, http.StatusInternalServerError)
	runtime.ApprovalStore().Notifier = &ServiceAgentWebhookNotifier{
		URL: coletor.servidor.URL, Secret: "segredo", Client: coletor.servidor.Client(),
	}
	antes := len(crm.ListOpportunities("org-a", ""))
	pedido := salesWriteTurn(conversationID, "Plano avançado", 79900)

	turn, err := runtime.RunTurn(pedido, now)
	if err != nil {
		t.Fatal(err)
	}
	// O aviso falhou, mas o PEDIDO existe e o turno diz o que falta — a aprovação é o controle.
	call := callByName(turn.ToolCalls, "crm.opportunity.write")
	if call == nil || call.Status != ServiceAgentToolApprovalRequired || call.ApprovalID == "" {
		t.Fatalf("a falha do aviso não pode impedir o pedido de aprovação: %+v", turn.ToolCalls)
	}
	if avisos := runtime.ApprovalStore().NotificationsSent(); avisos != 0 {
		t.Fatalf("aviso com erro não conta como entregue: %d", avisos)
	}
	falha := runtime.ApprovalStore().LastNotificationError()
	if !strings.Contains(falha, "500") {
		t.Fatalf("a falha precisa ser declarada por CLASSE (status): %q", falha)
	}
	// E a decisão humana continua funcionando: nada foi travado pelo destino fora do ar.
	if _, err := runtime.ApprovalStore().Decide("org-a", call.ApprovalID, true, "gerente-ana", "ok", now); err != nil {
		t.Fatal(err)
	}
	turn, err = runtime.RunTurn(pedido, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if call := callByName(turn.ToolCalls, "crm.opportunity.write"); call == nil || call.Status != ServiceAgentToolExecuted {
		t.Fatalf("depois da aprovação a escrita precisa executar: %+v", turn.ToolCalls)
	}
	if depois := len(crm.ListOpportunities("org-a", "")); depois != antes+1 {
		t.Fatalf("a escrita precisa acontecer uma vez: antes=%d depois=%d", antes, depois)
	}
}

func TestServiceAgentWebhookNotifierNeverLeaksDestinationOrSecret(t *testing.T) {
	const segredo = "segredo-que-nao-pode-vazar"
	notificador := &ServiceAgentWebhookNotifier{
		// Destino inválido de propósito: o erro do net/http carrega a URL se alguém repassar o erro cru.
		URL: "http://127.0.0.1:1/aprovar?token=" + segredo, Secret: segredo,
	}
	err := notificador.NotifyPendingApproval(t.Context(), ServiceAgentApprovalNotification{ApprovalID: "apr_1", ToolName: "pricing.write"})
	if err == nil {
		t.Fatal("destino inválido precisa falhar")
	}
	if strings.Contains(err.Error(), segredo) || strings.Contains(err.Error(), "token=") || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("o erro do aviso não pode carregar destino nem segredo: %q", err.Error())
	}
	// Sem destino configurado não é erro: é ausência de configuração (e nada é enviado).
	if err := (&ServiceAgentWebhookNotifier{}).NotifyPendingApproval(t.Context(), ServiceAgentApprovalNotification{}); err != nil {
		t.Fatalf("sem destino não há falha de aviso: %v", err)
	}
}
