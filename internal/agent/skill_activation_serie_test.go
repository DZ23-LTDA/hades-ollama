package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSkillSelectionVaziaSerializaComoLista trava o contrato que o painel consome: uma selecao sem
// skills tem de sair como `"selected": []`, nunca `"selected": null`.
//
// *Por que isto existe:* em 2026-10-21 a captura de ecra da pagina /skills falhou com
// `TypeError: Cannot read properties of null (reading 'map')` -- o servidor devolvia `null` porque o
// slice era nil. O cliente nao pode ser obrigado a adivinhar: ou o contrato e' lista, ou nao e'.
func TestSkillSelectionVaziaSerializaComoLista(t *testing.T) {
	dados, err := json.Marshal(SkillSelection{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	texto := string(dados)
	if strings.Contains(texto, `"selected":null`) {
		t.Fatalf("contrato quebrado: selected saiu null (%s)", texto)
	}
	if !strings.Contains(texto, `"selected":[]`) {
		t.Fatalf("contrato esperado: selected como lista vazia (%s)", texto)
	}
}
