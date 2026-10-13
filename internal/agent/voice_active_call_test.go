package agent

import (
	"errors"
	"testing"
	"time"
)

// ESTÁGIO 37 — CHAMADA ATIVA SÓ COM FINALIDADE AUTORIZADA, CONTATO ELEGÍVEL E APROVAÇÃO (E-303).
//
// O risco nomeado pelo estágio é "chamada ativa sem consentimento". Antes deste teste não existia NENHUM conceito de
// chamada ativa no código: o Hades ligando para alguém — o efeito externo de maior risco do estágio — não tinha
// portão nenhum. Aqui cada portão é medido separadamente, e todos recusam EXPLICITAMENTE.
func TestChamadaAtivaExigeFinalidadeConsentimentoEAprovacao(t *testing.T) {
	store := NewVoiceActiveCallStore(t.TempDir())
	consent, err := NewConsentRegistry(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	const org = "org-a"
	const contato = "+5511955554444"
	const canal = "phone"

	pedido := func(purpose, organizationID, contactID string) (VoiceActiveCall, error) {
		return store.RequestActiveCall(organizationID, VoiceActiveCall{
			ContactID:   contactID,
			Channel:     canal,
			Purpose:     purpose,
			RequestedBy: "agente-comercial",
		}, consent, agora)
	}

	// 1. FINALIDADE NÃO AUTORIZADA: recusa antes de qualquer outra coisa. "O modelo pediu" não é finalidade.
	if _, err := pedido("vender_mais", org, contato); !errors.Is(err, ErrVoiceActiveCallPurposeNotAllowed) {
		t.Fatalf("finalidade fora da lista precisa ser recusada, veio: %v", err)
	}
	chamadas, err := store.Calls(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(chamadas) != 0 {
		t.Fatalf("pedido recusado não pode deixar registro: %+v", chamadas)
	}

	// 2. SEM CONSENTIMENTO: contato não é elegível — recusa, e nada é registrado.
	if _, err := pedido("support_followup", org, contato); err == nil {
		t.Fatal("sem consentimento concedido, a chamada ativa precisa ser recusada")
	}
	if chamadas, _ = store.Calls(org); len(chamadas) != 0 {
		t.Fatalf("chamada sem consentimento não pode deixar registro: %+v", chamadas)
	}

	// 3. COM CONSENTIMENTO: o pedido é aceito, mas nasce PENDENTE — aprovação humana é obrigatória.
	if _, err := consent.GrantConsent(org, canal, contato, "site", "checkbox", "ana", nil, agora); err != nil {
		t.Fatal(err)
	}
	pendente, err := pedido("support_followup", org, contato)
	if err != nil {
		t.Fatalf("com consentimento o pedido precisa ser aceito: %v", err)
	}
	if pendente.Status != VoiceActiveCallPending {
		t.Fatalf("o pedido precisa nascer PENDENTE de aprovação, veio %q", pendente.Status)
	}
	if pendente.ProviderExecutedAt != nil {
		t.Fatal("nada foi telefonado aqui: o registro não pode fingir execução do provedor")
	}

	// 4. APROVAÇÃO: libera; aprovar DE NOVO é recusado (efeito externo uma vez).
	autorizada, err := store.ApproveActiveCall(org, pendente.ID, "supervisora-bia", agora)
	if err != nil {
		t.Fatalf("aprovação precisa liberar a chamada: %v", err)
	}
	if autorizada.Status != VoiceActiveCallAuthorized || autorizada.ApprovedBy != "supervisora-bia" {
		t.Fatalf("a autorização precisa ficar registrada com quem aprovou: %+v", autorizada)
	}
	if _, err := store.ApproveActiveCall(org, pendente.ID, "supervisora-bia", agora); err == nil {
		t.Fatal("aprovar a mesma chamada duas vezes precisa ser recusado")
	}
	if _, err := store.ApproveActiveCall(org, pendente.ID, "", agora); !errors.Is(err, ErrVoiceActiveCallRequiresApproval) {
		t.Fatalf("aprovar sem aprovador identificado precisa ser recusado: %v", err)
	}

	// 5. TETO POR JANELA: com uma autorizada, novas autorizações param no teto — com recusa EXPLÍCITA.
	autorizadas := 1
	for range VoiceActiveCallLimitPerWindow + 2 {
		novo, err := pedido("delivery_notice", org, contato)
		if err != nil {
			// O teto é aplicado já no PEDIDO: ao alcançá-lo, a recusa explícita é o desfecho ESPERADO.
			if errors.Is(err, ErrVoiceActiveCallThrottled) {
				break
			}
			t.Fatalf("recusa inesperada antes do teto: %v", err)
		}
		if _, err := store.ApproveActiveCall(org, novo.ID, "supervisora-bia", agora); err != nil {
			if errors.Is(err, ErrVoiceActiveCallThrottled) {
				continue
			}
			t.Fatalf("recusa inesperada ao aprovar: %v", err)
		}
		autorizadas++
	}
	if autorizadas != VoiceActiveCallLimitPerWindow {
		t.Fatalf("o teto por janela precisa ser exatamente %d (mediu %d)", VoiceActiveCallLimitPerWindow, autorizadas)
	}
	_, errQuarto := pedido("delivery_notice", org, contato)
	if !errors.Is(errQuarto, ErrVoiceActiveCallThrottled) {
		t.Fatalf("passado o teto, o pedido precisa recusar EXPLICITAMENTE com ErrVoiceActiveCallThrottled, veio: %v", errQuarto)
	}

	// 6. ISOLAMENTO: outra organização não herda o teto nem o consentimento da primeira.
	const outra = "org-b"
	if _, err := pedido("support_followup", outra, contato); err == nil {
		t.Fatal("outra organização não pode herdar o consentimento da primeira")
	}
	if _, err := consent.GrantConsent(outra, canal, contato, "site", "checkbox", "ana", nil, agora); err != nil {
		t.Fatal(err)
	}
	if _, err := pedido("support_followup", outra, contato); err != nil {
		t.Fatalf("a segunda organização tem a própria janela e o próprio consentimento: %v", err)
	}

	// 7. REVOGAÇÃO DEPOIS: a elegibilidade é REMEDIDA no pedido, não presumida da concessão antiga.
	if _, err := consent.RevokeConsent(org, canal, contato, "pedido do titular", "ana", agora); err != nil {
		t.Fatal(err)
	}
	if _, err := pedido("support_followup", org, contato); err == nil {
		t.Fatal("consentimento revogado precisa voltar a recusar a chamada ativa")
	}
}
