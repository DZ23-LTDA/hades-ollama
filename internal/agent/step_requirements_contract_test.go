package agent

import (
	"encoding/json"
	"testing"
)

// E-429 — O NO DO PLANO CARREGA AS EXIGENCIAS (Estagio 43, ADR-002). **CONTRATO + ROUND-TRIP, e so' isso.**
//
// **O QUE ESTAVA MEDIDO**: o `Step` (`types.go:117`) era uma **lista plana** - `ID/Kind/Title/Input/Risk/
// RequiresApproval/*SHA256/State/Attempts/Result/Error` - sem **modelo**, sem **capability** e sem **dependencia**
// ⇒ *nao havia como dizer "este passo precisa de visao" nem "quero ESTE modelo aqui": a escolha era do **turno**
// inteiro, nao do **passo*** ✗. E sem `DependsOn` o plano **nao e' um DAG** - *a ordem da lista e' a unica
// expressao de dependencia, e ordem de lista nao distingue "depois" de "por causa de".*
//
// **O QUE ESTE TESTE PROVA, e o que ele NAO prova** - *a distincao importa mais que o teste*:
//   - PROVA: os tres campos existem, **sobrevivem ao JSON** (*ida e volta*) e ficam **vazios quando nao usados**
//     (*`omitempty` - um plano antigo continua legivel* ✓);
//   - NAO PROVA: que o **runtime** consulta o `Registry` com essas exigencias por passo. **Isso NAO esta feito** ✗
//     e esta declarado no LIMITE de E-429.
//
// ***Declarar o contrato como "implementado" quando so' o contrato existe seria exatamente a classe que esta missao
// vem derrubando: afirmacao cuja cor nao depende do comportamento alegado.***
func TestNoDoPlanoCarregaExigenciasEPin(t *testing.T) {
	passo := Step{
		ID:                   "spec-calc",
		Kind:                 "agent",
		Title:                "Calcular o orcamento",
		RequiredCapabilities: []string{"vision", "coding"},
		PinnedModelID:        "local/modelo-barato",
		DependsOn:            []string{"pesquisa", "planta"},
		State:                StepPending,
	}

	// (a) IDA E VOLTA pelo JSON: *o plano e' PERSISTIDO, entao o campo que nao sobrevive a serializacao nao existe
	// de verdade.*
	codificado, err := json.Marshal(passo)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	var voltou Step
	if err := json.Unmarshal(codificado, &voltou); err != nil {
		t.Fatalf("desserializar: %v", err)
	}
	if len(voltou.RequiredCapabilities) != 2 || voltou.RequiredCapabilities[0] != "vision" {
		t.Fatalf("as capacidades exigidas nao sobreviveram ao JSON: %+v", voltou.RequiredCapabilities)
	}
	if voltou.PinnedModelID != "local/modelo-barato" {
		t.Fatalf("o PIN nao sobreviveu ao JSON: %q", voltou.PinnedModelID)
	}
	if len(voltou.DependsOn) != 2 || voltou.DependsOn[1] != "planta" {
		t.Fatalf("as DEPENDENCIAS nao sobreviveram ao JSON: %+v", voltou.DependsOn)
	}

	// (b) E os nomes JSON estao declarados como o resto do arquivo usa (*snake_case*), para o plano persistido
	// continuar legivel por quem ja' o le.
	for campo, esperado := range map[string]string{
		"required_capabilities": "requiredCapabilities",
		"pinned_model_id":       "pinnedModelId",
		"depends_on":            "dependsOn",
	} {
		if !contemChave(codificado, campo) {
			t.Fatalf("o JSON do passo nao traz a chave %q (esperada para %s): %s", campo, esperado, codificado)
		}
	}

	// (c) PLANO ANTIGO CONTINUA LEGIVEL: sem os campos novos, o `omitempty` nao os escreve - *compatibilidade nao e'
	// detalhe, e' o que permite o campo entrar sem migracao.*
	antigo, err := json.Marshal(Step{ID: "s1", Kind: "agent", Title: "antigo", State: StepPending})
	if err != nil {
		t.Fatal(err)
	}
	if contemChave(antigo, "required_capabilities") || contemChave(antigo, "pinned_model_id") || contemChave(antigo, "depends_on") {
		t.Fatalf("um passo SEM as exigencias nao pode escrever as chaves novas (`omitempty`): %s", antigo)
	}
}

func contemChave(codificado []byte, chave string) bool {
	var bruto map[string]json.RawMessage
	if err := json.Unmarshal(codificado, &bruto); err != nil {
		return false
	}
	_, ok := bruto[chave]
	return ok
}
