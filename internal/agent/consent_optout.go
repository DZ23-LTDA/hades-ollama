package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Captura automática de OPT-OUT (pedido de descadastro) — estágio 27.
//
// LIMITE DECLARADO QUE ISTO FECHA: o consentimento tinha concessão e revogação explícitas,
// mas **nenhuma captura automática**: se um contato escrevesse "não quero mais receber", o
// sistema só registraria a revogação se um humano fizesse isso à mão. Este arquivo detecta o
// pedido no texto recebido e **revoga o consentimento**, que é o que bloqueia o envio.
//
// Decisões deliberadas:
//
//   - **DIREÇÃO SEGURA**: a detecção só serve para REVOGAR, nunca para conceder. Um falso
//     positivo bloqueia uma mensagem (custo baixo, reversível pelo operador); um falso negativo
//     continuaria enviando para quem pediu para parar (custo alto e irreversível). Por isso a
//     heurística é ASSIMÉTRICA de propósito;
//   - **DETERMINÍSTICA E AUDITÁVEL**: nada de modelo aqui. A frase que disparou a revogação é
//     registrada como evidência da revogação, então a decisão é explicável;
//   - **GUARDA CONTRA DUPLA NEGAÇÃO**: "não quero PARAR de receber" contém "parar de receber",
//     mas significa o oposto. Frases negadas são explicitamente descartadas antes de qualquer
//     revogação;
//   - **NÃO SE RESPONDE** a quem pediu para parar: a confirmação de descadastro seria a própria
//     mensagem que o contato pediu para não receber. O turno é marcado como não despachado, e
//     isso é declarado em vez de contornado com um caminho que ignora a política.
//
// A limitação honesta: a lista de frases cobre pt-BR e inglês comuns, não todas as formas de
// dizer "pare". Frase não reconhecida continua dependendo de ação humana — por isso o estado da
// capacidade é PARCIAL, e não "completo".

var ErrOptOutInvalid = errors.New("opt-out request invalid")

// Frases que PEDEM para parar o contato. Todas em minúsculas e sem acento, porque o texto é
// normalizado antes da comparação.
var optOutPhrases = []string{
	"nao quero mais receber",
	"nao quero receber mais",
	"nao quero receber",
	"pare de me enviar",
	"pare de enviar",
	"pare de me mandar",
	"pare de mandar",
	"para de me enviar",
	"para de enviar",
	"nao me envie mais",
	"nao me mande mais",
	"nao me contate",
	"nao me contacte",
	"nao me ligue mais",
	"nao me chame mais",
	"remova meu numero",
	"remover meu numero",
	"remova meu contato",
	"quero ser removido",
	"quero descadastrar",
	"descadastrar",
	"descadastre",
	"cancelar o recebimento",
	"cancele o recebimento",
	"sair da lista",
	"me tire da lista",
	"me tira da lista",
	"quero sair da lista",
	"opt out",
	"opt-out",
	"unsubscribe",
	"stop",
	"do not contact me",
	"do not message me",
	"stop messaging me",
	"stop sending",
}

// optOutNegations são marcadores de NEGAÇÃO da própria frase de descadastro: "não quero
// parar", "não quero deixar de receber", "não quero me descadastrar". Se um deles aparece
// ANTES da frase detectada, o pedido não é de descadastro.
var optOutNegations = []string{
	"nao quero parar",
	"nao quero deixar de",
	"nao quero cancelar o recebimento",
	"nao quero me descadastrar",
	"nao quero ser removido",
	"nao quero sair da lista",
	"don't want to stop",
	"do not want to stop",
}

// OptOutIntent responde se o texto recebido é um pedido de descadastro e qual frase o
// caracterizou. O texto é dado de terceiro e tratado apenas como dado.
func OptOutIntent(body string) (bool, string) {
	normalized := normalizeOptOutText(body)
	if normalized == "" {
		return false, ""
	}
	// A negação tem precedência: sem isso, "não quero parar de receber" revogaria o
	// consentimento de quem pediu justamente o contrário.
	for _, negation := range optOutNegations {
		if index := strings.Index(normalized, negation); index >= 0 {
			matched := matchOptOutPhrase(normalized)
			// Só descarta quando a frase detectada vem DEPOIS da negação, que é o caso de
			// "não quero parar de <receber>".
			if matched != "" && strings.Index(normalized, matched) > index {
				return false, ""
			}
		}
	}
	matched := matchOptOutPhrase(normalized)
	if matched == "" {
		return false, ""
	}
	return true, matched
}

func matchOptOutPhrase(normalized string) string {
	for _, phrase := range optOutPhrases {
		if strings.Contains(normalized, phrase) {
			return phrase
		}
	}
	return ""
}

// normalizeOptOutText baixa para minúsculas, remove acentos e colapsa espaços, para que
// "Não quero mais RECEBER!" e "nao quero mais receber" sejam a mesma coisa.
func normalizeOptOutText(body string) string {
	lowered := strings.ToLower(strings.TrimSpace(body))
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "ê", "e", "è", "e",
		"í", "i", "ì", "i", "î", "i",
		"ó", "o", "ô", "o", "õ", "o", "ò", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"’", "'", "“", "\"", "”", "\"",
	)
	normalized := replacer.Replace(lowered)
	// Colapsa pontuação em espaço para que a frase não seja quebrada por vírgula ou ponto.
	normalized = strings.NewReplacer(",", " ", ".", " ", ";", " ", "!", " ", "?", " ", "\n", " ", "\t", " ").Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

// OptOutOutcome é o resultado da captura, com a evidência da decisão.
type OptOutOutcome struct {
	OrganizationID string `json:"organization_id"`
	Channel        string `json:"channel"`
	ContactID      string `json:"contact_id"`
	Requested      bool   `json:"requested"`
	MatchedPhrase  string `json:"matched_phrase,omitempty"`
	// AlreadyRevoked evita tratar um pedido repetido como novidade.
	AlreadyRevoked bool          `json:"already_revoked"`
	Record         ConsentRecord `json:"record"`
	// Effect declara o efeito REAL aplicado, em vez de sugerir que algo foi enviado.
	Effect string `json:"effect"`
}

// OptOutEffectBlocked é o efeito declarado quando o envio passa a ser autorizado apenas com
// novo consentimento explícito.
const OptOutEffectBlocked = "envio_bloqueado_ate_novo_consentimento_explicito"

// CaptureOptOut detecta o pedido de descadastro no texto recebido e revoga o consentimento.
//
// Idempotente por natureza: revogar de novo apenas informa `AlreadyRevoked`, sem duplicar
// histórico nem falhar — pedir para parar duas vezes é comum.
func (r *Runtime) CaptureOptOut(organizationID, channel, contactID, body, actor string, now time.Time) (OptOutOutcome, error) {
	outcome := OptOutOutcome{
		OrganizationID: consentScope(organizationID),
		Channel:        strings.ToLower(strings.TrimSpace(channel)),
		ContactID:      strings.TrimSpace(contactID),
		Effect:         OptOutEffectBlocked,
	}
	if r == nil || r.Consent() == nil {
		return outcome, fmt.Errorf("%w: registro de consentimento indisponível", ErrOptOutInvalid)
	}
	// O escopo é validado na ENTRADA, antes de qualquer normalização: `consentScope` mapeia
	// vazio para a organização local (convenção do produto), e aceitar isso aqui engoliria um
	// escopo ausente — quem revoga consentimento precisa DIZER de qual organização fala.
	if strings.TrimSpace(organizationID) == "" || outcome.Channel == "" || outcome.ContactID == "" {
		return outcome, fmt.Errorf("%w: organização, canal e contato são obrigatórios", ErrOptOutInvalid)
	}
	requested, phrase := OptOutIntent(body)
	outcome.Requested = requested
	outcome.MatchedPhrase = phrase
	if !requested {
		return outcome, nil
	}
	// A evidência da revogação é a FRASE detectada: a decisão fica explicável.
	reason := "opt-out capturado automaticamente"
	if phrase != "" {
		reason = "opt-out capturado automaticamente: \"" + phrase + "\""
	}
	if existing, ok := r.Consent().ConsentFor(outcome.OrganizationID, outcome.Channel, outcome.ContactID); ok &&
		existing.State == ConsentRevoked {
		outcome.AlreadyRevoked = true
		outcome.Record = existing
		return outcome, nil
	}
	record, err := r.Consent().RevokeConsent(outcome.OrganizationID, outcome.Channel, outcome.ContactID, reason, actor, now)
	if err != nil {
		return outcome, err
	}
	outcome.Record = record
	return outcome, nil
}
