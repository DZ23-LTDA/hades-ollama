package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estagio 29 — SLA DE RESOLUCAO (E-266). O limite declarado era "SLA apenas de primeira resposta".
//
// O SLA de primeira resposta NAO pode afrouxar: medir a resolucao e' medir MAIS, nao medir menos. As assercoes de
// primeira resposta abaixo existem para travar isso.
func TestInboxSLAdeResolucaoNaoAfrouxaPrimeiraResposta(t *testing.T) {
	store, err := NewInboxStore(filepath.Join(t.TempDir(), "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	org := LocalOrganizationID
	store.SetResolutionSLA(org, 2*time.Hour)

	criar := func(canal, contato string, recebidoEm time.Time, responderEm *time.Time, fecharEm *time.Time) string {
		conversa, _, err := store.AppendInbound(org, canal, contato, contato, "ola", canal+"-"+contato, recebidoEm)
		if err != nil {
			t.Fatalf("inbound %s: %v", contato, err)
		}
		if responderEm != nil {
			if _, _, err := store.AppendOutbound(org, conversa.ID, "agent", "ana", "respondo ja", nil, *responderEm); err != nil {
				t.Fatalf("outbound %s: %v", contato, err)
			}
		}
		if fecharEm != nil {
			if _, err := store.Close(org, conversa.ID, "ana", "resolvido", *fecharEm); err != nil {
				t.Fatalf("close %s: %v", contato, err)
			}
		}
		return conversa.ID
	}

	respondidaDentro := agora.Add(-50 * time.Minute)
	respondidaFora := agora.Add(-4 * time.Hour)
	fechadaDentro := agora.Add(-90 * time.Minute) // resolvida em 1h30 (< 2h)
	fechadaFora := agora.Add(-60 * time.Minute)   // resolvida em 4h  (> 2h)
	criar("whatsapp", "dentro", agora.Add(-3*time.Hour), &respondidaDentro, &fechadaDentro)
	criar("telegram", "fora", agora.Add(-5*time.Hour), &respondidaFora, &fechadaFora)
	// Duas conversas ABERTAS: uma vencida, uma no prazo.
	criar("email", "aberta-vencida", agora.Add(-4*time.Hour), nil, nil)
	criar("webchat", "aberta-no-prazo", agora.Add(-30*time.Minute), nil, nil)

	metricas := store.Metrics(org, agora)
	if metricas.ResolutionSLABreached != 1 {
		t.Fatalf("uma conversa fechada passou da janela: %+v", metricas)
	}
	if metricas.ResolutionOverdue != 1 {
		t.Fatalf("uma conversa ABERTA ja venceu a janela: %+v", metricas)
	}
	// Media das duas resolvidas: (1h30 + 4h) / 2 = 2h45 = 9900s.
	if media := metricas.ResolutionAvgSecs; media < 9800 || media > 10000 {
		t.Fatalf("media de resolucao errada: %.1f", media)
	}
	// PRIMEIRA RESPOSTA INTACTA: as duas conversas sem resposta contam como aguardando e estouraram a janela de 15min.
	if metricas.AwaitingFirstReply != 2 || metricas.SLABreached != 2 {
		t.Fatalf("o SLA de primeira resposta foi afetado: aguardando=%d estourado=%d", metricas.AwaitingFirstReply, metricas.SLABreached)
	}
	if metricas.MessagesInbound != 4 {
		t.Fatalf("mensagens inbound: %d", metricas.MessagesInbound)
	}

	// ISOLAMENTO: outra organizacao nao herda a janela nem as metricas.
	outra := store.Metrics("org-b", agora)
	if outra.TotalConversations != 0 || outra.ResolutionOverdue != 0 || outra.ResolutionSLABreached != 0 {
		t.Fatalf("metricas nao podem vazar entre organizacoes: %+v", outra)
	}
	// A janela padrao (24h) DECLARADA vale para quem nao definiu: 4h de atraso NAO vence 24h.
	if _, _, err := store.AppendInbound("org-b", "whatsapp", "cliente-b", "Cliente B", "oi", "b-1", agora.Add(-4*time.Hour)); err != nil {
		t.Fatal(err)
	}
	outra = store.Metrics("org-b", agora)
	if outra.ResolutionOverdue != 0 {
		t.Fatalf("a janela padrao de 24h nao pode marcar 4h como vencida: %+v", outra)
	}
	if outra.TotalConversations != 1 {
		t.Fatalf("a conversa da outra organizacao precisa aparecer nas metricas dela: %+v", outra)
	}
}
