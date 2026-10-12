package agent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// PROVA DO MECANISMO DO FALSO POSITIVO (E-196).
//
// O "vazamento intermitente" da CI era o RELÓGIO: a checagem procurava o fragmento "4111" no arquivo inteiro, e um
// `decided_at` como `...T08:26:43.411114Z` contém "4111". Este teste fixa o mecanismo: com a MESMA trilha (sem
// conteúdo de terceiro) e um timestamp que contém o fragmento, a checagem nova NÃO acusa — e a antiga acusaria.
func TestTimestampComFragmentoNaoEhVazamento(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"
	arquivo := serviceAgentAuditFile{Entries: []ServiceAgentAuditEntry{{
		ID:             "aud_af4240a8-cba",
		OrganizationID: "org-a",
		AgentID:        "support",
		ConversationID: "cv_8c9100c8-651",
		ToolName:       "inbox.reply",
		Effect:         "outbound",
		Status:         "blocked",
		Reason:         "a resposta do turno já é enviada pelo caminho único de resposta",
		DecidedAt:      timeMustParse(t, "2026-10-12T08:26:43.411114Z"),
		RecordedAt:     timeMustParse(t, "2026-10-12T08:26:43.411114Z"),
		PreviousHash:   "genesis",
		Hash:           "a44af3f3e8a606afc2edec7a0512fa49bbfc64e5258c06df48ad7d0c7f618115",
	}}}
	conteudo, err := json.Marshal(arquivo)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(conteudo)

	// A trilha PRECISA conter o fragmento: é isso que torna o caso igual ao da CI.
	if !strings.Contains(texto, "4111") {
		t.Fatalf("o caso de teste precisa conter o fragmento 4111 no timestamp; conteúdo: %s", texto)
	}
	// A checagem antiga (arquivo inteiro) acusaria — e é por isso que ela foi trocada:
	checagemAntiga := strings.Contains(texto, "4111") || strings.Contains(texto, segredo)
	if !checagemAntiga {
		t.Fatal("o caso precisa reproduzir a checagem antiga acusando")
	}
	// A checagem nova NÃO acusa: nenhum campo de conteúdo carrega o texto.
	if campo := campoDaTrilhaComTexto(texto, segredo); campo != "" {
		t.Fatalf("falso positivo voltou: apontou %q numa trilha sem conteúdo de terceiro", campo)
	}
}

func timeMustParse(t *testing.T, valor string) time.Time {
	t.Helper()
	quando, err := time.Parse(time.RFC3339Nano, valor)
	if err != nil {
		t.Fatal(err)
	}
	return quando
}
