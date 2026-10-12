package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TESTE DO INSTRUMENTO DE DIAGNÓSTICO (E-193).
//
// `campoDaTrilhaComTexto` existe para responder, no log do CI, "qual campo vazou". Um diagnóstico que não acha o
// campo é pior que nenhum, porque transforma uma falha real em "não sei onde": estes casos o alimentam com vazamentos
// SINTÉTICOS em cada campo e exigem que ele aponte o campo certo.
func TestCampoDaTrilhaComTextoApontaOCampo(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"

	casos := []struct {
		nome     string
		entrada  ServiceAgentAuditEntry
		esperado string
	}{
		{"summary", ServiceAgentAuditEntry{Summary: "atendeu: " + segredo}, "entries[0].summary"},
		{"reason", ServiceAgentAuditEntry{Reason: "motivo informado: " + segredo}, "entries[0].reason"},
		{"tool_name", ServiceAgentAuditEntry{ToolName: segredo}, "entries[0].tool_name"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			arquivo := serviceAgentAuditFile{Entries: []ServiceAgentAuditEntry{caso.entrada}}
			conteudo, err := json.Marshal(arquivo)
			if err != nil {
				t.Fatal(err)
			}
			if campo := campoDaTrilhaComTexto(string(conteudo), segredo); campo != caso.esperado {
				t.Fatalf("o diagnóstico precisa apontar %q, apontou %q", caso.esperado, campo)
			}
		})
	}

	// E o negativo: trilha limpa não pode ser acusada.
	limpo := serviceAgentAuditFile{Entries: []ServiceAgentAuditEntry{{
		ToolName: "crm.contact.read",
		Status:   "ok",
		Summary:  "leu o contato autorizado",
	}}}
	conteudo, err := json.Marshal(limpo)
	if err != nil {
		t.Fatal(err)
	}
	if campo := campoDaTrilhaComTexto(string(conteudo), segredo); campo != "" {
		t.Fatalf("trilha limpa não pode ser acusada, apontou %q", campo)
	}
}

// A máscara não pode revelar o segredo: o log do CI é tão público quanto o do produto.
func TestMascararTextoNaoRevelaOTexto(t *testing.T) {
	segredo := "meu cartao 4111 1111 1111 1111 e meu pedido"
	mascarado := mascararTexto(segredo)
	if strings.Contains(mascarado, "4111") || strings.Contains(mascarado, segredo) {
		t.Fatalf("a máscara revelou o texto: %q", mascarado)
	}
	// A máscara precisa dizer ALGO: comprimento e começo são úteis para localizar o campo.
	if !strings.Contains(mascarado, "43") {
		t.Fatalf("a máscara precisa informar o comprimento do texto: %q", mascarado)
	}
}
