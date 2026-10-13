package agent

import (
	"strings"
	"testing"
)

// E-333 — PROVA MECÂNICA DO FALSO POSITIVO DE DLP (e da correção).
//
// Não basta "não reproduzir": aqui eu demonstro o MECANISMO. A CI acusou vazamento no campo `reason`:
//
//	"escrita exige aprovação humana: pedido apr_28741117-50a aguardando decisão"
//
// O segredo varrido é `"meu cartao 4111 1111 1111 1111 e meu pedido"`. O identificador **gerado** contém `4111`. Logo
// a busca por FRAGMENTO acusava vazamento onde não havia — ***o identificador é do próprio sistema, e conteúdo de
// terceiro não vive dentro dele***.
func TestFalsoPositivoDeDLPPorIdentificadorGeradoEstaCorrigido(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"
	reasonDaCI := "escrita exige aprovação humana: pedido apr_28741117-50a aguardando decisão"

	// 1. O mecanismo do falso positivo EXISTE: sem mascarar, o fragmento casa.
	if !contemTexto(reasonDaCI, segredo) {
		t.Skip("o fragmento do segredo não casa com este identificador: o cenário da CI não se reproduz aqui")
	}
	// 2. ARMADILHA: com o mascaramento, o MESMO texto deixa de casar.
	if contemTexto(mascaraIdentificadoresGerados(reasonDaCI), segredo) {
		t.Fatal("o mascaramento não removeu o identificador gerado: o falso positivo continua possível")
	}
	// 3. E o detector completo não acusa mais este caso.
	trilha := `{"entries":[{"id":"aud_1","organization_id":"org-a","agent_id":"receptionist",` +
		`"conversation_id":"cv_938abc6b-fd2","tool_name":"inbox.handoff","effect":"write",` +
		`"status":"approval_required","reason":"` + reasonDaCI + `","previous_hash":"genesis","hash":"d8ff"}]}`
	if campo := campoDaTrilhaComTexto(trilha, segredo); campo != "" {
		t.Fatalf("falso positivo ainda acusado no campo %q", campo)
	}
}

// A INVARIANTE CONTINUA VALENDO: conteúdo real do contato gravado em texto livre É acusado.
func TestVazamentoRealDeConteudoDoContatoContinuaSendoAcusado(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"
	// Vazamento REAL: o texto do contato foi parar no campo de texto livre.
	trilha := `{"entries":[{"id":"aud_1","organization_id":"org-a","agent_id":"receptionist",` +
		`"conversation_id":"cv_1","tool_name":"crm.contact.read","effect":"read",` +
		`"status":"ok","summary":"cliente disse: ` + segredo + `"}]}`
	campo := campoDaTrilhaComTexto(trilha, segredo)
	if campo == "" {
		t.Fatal("ARMADILHA: a correção não pode ter cegado o detector — vazamento real em texto livre precisa ser acusado")
	}
	if !strings.Contains(campo, "summary") {
		t.Fatalf("o campo acusado precisa ser o de texto livre: %q", campo)
	}
	// E o mascaramento não pode apagar texto comum (só identificadores gerados).
	if strings.Contains(mascaraIdentificadoresGerados("cliente disse: "+segredo), "<id>") {
		t.Fatal("o mascaramento não pode alterar conteúdo comum")
	}
}

// Identificadores gerados de vários formatos do sistema são mascarados; texto comum não é.
func TestMascaramentoSoAfetaIdentificadoresGerados(t *testing.T) {
	casos := []string{
		"pedido apr_28741117-50a",
		"id aud_27fe647c-b4d",
		"conversa cv_938abc6b-fd2",
		"live_9f2c1e34-aa11",
		"prod_orga_SKU-1",
	}
	for _, caso := range casos {
		if !strings.Contains(mascaraIdentificadoresGerados(caso), "<id>") {
			t.Fatalf("identificador gerado não foi mascarado: %q", caso)
		}
	}
	comum := "escrita exige aprovação humana e o cliente pediu retorno"
	if mascarado := mascaraIdentificadoresGerados(comum); mascarado != comum {
		t.Fatalf("texto comum foi alterado: %q -> %q", comum, mascarado)
	}
	if vazio := mascaraIdentificadoresGerados(""); vazio != "" {
		t.Fatalf("vazio precisa permanecer vazio: %q", vazio)
	}
}
