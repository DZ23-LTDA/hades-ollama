package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Estágio 27 — captura automática de opt-out.
//
// O limite declarado era "o consentimento não captura opt-out automático". As propriedades
// travadas aqui:
//  1. um pedido de descadastro REVOGA o consentimento e o envio passa a ser recusado;
//  2. pedir de novo é IDEMPOTENTE (informa que já estava revogado, sem duplicar histórico);
//  3. mensagens comuns NÃO revogam;
//  4. **dupla negação NÃO revoga**: "não quero parar de receber" significa o oposto;
//  5. a evidência registrada é a FRASE detectada, então a decisão é explicável;
//  6. a captura exige organização, canal e contato.
func TestCaptureOptOutRevokesConsentAndBlocksOutbound(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	organization, channel, contact := LocalOrganizationID, "whatsapp", "contato-optout"

	// Consentimento concedido: o envio é permitido.
	if _, err := runtime.Consent().GrantConsent(organization, channel, contact, "formulario", "aceite registrado", "operador", nil, now); err != nil {
		t.Fatal(err)
	}
	conversation, _, err := runtime.Inbox().AppendInbound(organization, channel, contact, "Cliente", "quero saber do produto", "opt-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := runtime.Inbox().AppendOutbound(organization, conversation.ID, "agent", "vendas", "claro, posso ajudar", runtime.Consent(), now); err != nil {
		t.Fatalf("com consentimento o envio precisa funcionar: %v", err)
	}

	// PEDIDO DE DESCADASTRO: revoga e é explicável.
	outcome, err := runtime.CaptureOptOut(organization, channel, contact, "Não quero mais receber mensagens, por favor", "contato", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Requested || outcome.MatchedPhrase == "" {
		t.Fatalf("o pedido precisa ser detectado com a frase de evidência: %+v", outcome)
	}
	if outcome.AlreadyRevoked {
		t.Fatal("a primeira captura não pode dizer que já estava revogado")
	}
	if outcome.Record.State != ConsentRevoked {
		t.Fatalf("o consentimento precisa ficar revogado: %+v", outcome.Record)
	}
	if outcome.Effect != OptOutEffectBlocked {
		t.Fatalf("o efeito declarado precisa ser o bloqueio do envio: %q", outcome.Effect)
	}
	// A EVIDÊNCIA da revogação é a frase detectada.
	if !strings.Contains(outcome.Record.Reason, outcome.MatchedPhrase) {
		t.Fatalf("o motivo da revogação precisa citar a frase detectada: %q", outcome.Record.Reason)
	}

	// O ENVIO passa a ser RECUSADO: a política vale para o caminho real de saída.
	if _, _, err := runtime.Inbox().AppendOutbound(organization, conversation.ID, "agent", "vendas", "mensagem nova", runtime.Consent(), now.Add(2*time.Minute)); !errors.Is(err, ErrInboxConsentRequired) {
		t.Fatalf("depois do opt-out o envio precisa ser recusado, veio %v", err)
	}

	// PEDIR DE NOVO é idempotente.
	again, err := runtime.CaptureOptOut(organization, channel, contact, "pare de enviar", "contato", now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !again.Requested || !again.AlreadyRevoked {
		t.Fatalf("a segunda captura precisa detectar e informar que já estava revogado: %+v", again)
	}
	records := runtime.Consent().ListConsents(organization)
	revoked := 0
	for _, record := range records {
		if record.ContactID == contact && record.State == ConsentRevoked {
			revoked++
		}
	}
	if revoked != 1 {
		t.Fatalf("não pode duplicar o registro de revogação: %d", revoked)
	}
}

func TestCaptureOptOutDoesNotFireOnOrdinaryMessages(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	organization, channel := LocalOrganizationID, "telegram"

	ordinary := []string{
		"quero comprar duas unidades",
		"não quero cancelar meu pedido, pelo contrário",
		"qual o prazo de entrega?",
		"pode me enviar a proposta, por favor",
		"estou com um problema na entrega",
	}
	for index, body := range ordinary {
		contact := "contato-comum"
		outcome, err := runtime.CaptureOptOut(organization, channel, contact, body, "contato", now.Add(time.Duration(index)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Requested {
			t.Fatalf("mensagem comum NÃO pode revogar consentimento: %q disparou %q", body, outcome.MatchedPhrase)
		}
	}
	// Nenhum registro foi criado por engano.
	for _, record := range runtime.Consent().ListConsents(organization) {
		if record.State == ConsentRevoked {
			t.Fatalf("não pode existir revogação para mensagem comum: %+v", record)
		}
	}
}

// A guarda mais importante: dupla negação. "não quero parar de receber" contém "parar de
// receber" e revogaria o consentimento de quem pediu EXATAMENTE o contrário.
func TestCaptureOptOutIgnoresDoubleNegation(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	organization, channel, contact := LocalOrganizationID, "whatsapp", "contato-negacao"
	if _, err := runtime.Consent().GrantConsent(organization, channel, contact, "formulario", "aceite", "operador", nil, now); err != nil {
		t.Fatal(err)
	}
	for index, body := range []string{
		"não quero parar de receber as novidades",
		"nao quero deixar de receber",
		"não quero me descadastrar agora",
		"não quero cancelar o recebimento, gosto das mensagens",
	} {
		outcome, err := runtime.CaptureOptOut(organization, channel, contact, body, "contato", now.Add(time.Duration(index)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Requested {
			t.Fatalf("dupla negação NÃO pode revogar consentimento: %q disparou %q", body, outcome.MatchedPhrase)
		}
	}
	record, ok := runtime.Consent().ConsentFor(organization, channel, contact)
	if !ok || record.State != ConsentGranted {
		t.Fatalf("o consentimento precisa continuar concedido: %+v", record)
	}
}

func TestCaptureOptOutRequiresScopeAndAcceptsVariants(t *testing.T) {
	runtime := factoryRuntime(t)
	now := time.Now().UTC()
	// Escopo incompleto é erro declarado, não revogação silenciosa.
	if _, err := runtime.CaptureOptOut("", "whatsapp", "contato", "pare de enviar", "contato", now); !errors.Is(err, ErrOptOutInvalid) {
		t.Fatalf("organização ausente precisa falhar, veio %v", err)
	}
	if _, err := runtime.CaptureOptOut(LocalOrganizationID, "", "contato", "pare de enviar", "contato", now); !errors.Is(err, ErrOptOutInvalid) {
		t.Fatalf("canal ausente precisa falhar, veio %v", err)
	}
	if _, err := runtime.CaptureOptOut(LocalOrganizationID, "whatsapp", "", "pare de enviar", "contato", now); !errors.Is(err, ErrOptOutInvalid) {
		t.Fatalf("contato ausente precisa falhar, veio %v", err)
	}
	// Formas reconhecidas: acento, maiúsculas, inglês e pontuação.
	variants := []string{
		"DESCADASTRAR",
		"Please unsubscribe me",
		"pare de mandar mensagens!",
		"Remova meu número, por favor.",
		"Não me contate mais",
	}
	for _, body := range variants {
		requested, phrase := OptOutIntent(body)
		if !requested {
			t.Fatalf("variante precisa ser reconhecida: %q", body)
		}
		if phrase == "" {
			t.Fatalf("a frase de evidência precisa ser informada para %q", body)
		}
	}
}
