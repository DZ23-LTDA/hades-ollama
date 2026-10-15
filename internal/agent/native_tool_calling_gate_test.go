package agent

import (
	"context"
	"errors"
	"testing"
)

// E-663 — O GATE DA CHAMADA NATIVA DE FERRAMENTA NO PASSO.
//
// **O QUE ESTAVA MEDIDO (06:45-03):** `ModelCapabilityRegistry.RequireNativeToolCalling` (`model_capabilities.go:139`)
// existia e **só era chamado em testes**. Um passo que declara a capacidade de MODELO `tools` podia ser roteado para
// um modelo **MEDIDO** como `compatible_text` e **executar assim mesmo** — o "tool call fingido" que o V2.2 proíbe.
// *O registro sabia dizer; ninguém perguntava.*
//
// **AS TRÊS DIREÇÕES QUE O TESTE MEDE** — *e a segunda é a que quase sempre falta*:
//  1. **MEDIDO** sem chamada nativa ⇒ o passo é **RECUSADO** (fail-closed, no laço real);
//  2. **NÃO MEDIDO** ⇒ **não recusa** (ausência de medição **não é** prova de incapacidade), mas **declara** na
//     trilha, porque *resultado ausente é `NAO_MEDIDO`, não aprovação*;
//  3. passo que **não** exige `tools` ⇒ segue intocado, mesmo com o modelo medido sem chamada nativa.
func TestGateDaChamadaNativaDeFerramenta(t *testing.T) {
	passoQueExigeFerramenta := []Step{{
		ID: "precisa-ferramenta", Kind: "workspace.list", Title: "exige tools",
		State: StepPending, PinnedModelID: "local/forte",
		RequiredCapabilities: []string{"tools"},
	}}

	// (1) MEDIDO como sem chamada nativa ⇒ RECUSA no laço real.
	registroSemFerramenta := registroMedidoParaTeste(t, "local/forte", false)
	runtimeRecusa, idRecusa, _ := missaoRoteada(t, &ferramentaQueAnotaModelo{}, passoQueExigeFerramenta, registryDeTeste(t), true)
	runtimeRecusa.modelCapabilities = registroSemFerramenta
	erro := runtimeRecusa.Run(context.Background(), idRecusa)
	if erro == nil {
		t.Fatal("ARMADILHA: o passo exigiu a capacidade `tools` e o modelo foi MEDIDO sem chamada nativa — *o passo " +
			"tem de ser RECUSADO: executar esconderia que a exigencia declarada nao foi honrada*")
	}
	if !errors.Is(erro, ErrModelWithoutNativeToolCalling) {
		t.Fatalf("a recusa tem de ser `ErrModelWithoutNativeToolCalling`, veio %v", erro)
	}

	// (2) NÃO MEDIDO ⇒ NÃO recusa, e DECLARA na trilha. (O registro PADRÃO não conhece `local/forte`.)
	runtimeNaoMedido, idNaoMedido, eventosNaoMedido := missaoRoteada(t, &ferramentaQueAnotaModelo{}, passoQueExigeFerramenta, registryDeTeste(t), true)
	if !ehNaoMedido(runtimeNaoMedido.modelCapabilities, "local/forte") {
		t.Fatal("premissa do teste quebrada: o registro padrao passou a conhecer `local/forte`")
	}
	if erro := runtimeNaoMedido.Run(context.Background(), idNaoMedido); erro != nil {
		t.Fatalf("capacidade NAO MEDIDA nao pode recusar o passo (*ausencia de medicao nao e' prova de "+
			"incapacidade* — o proprio ResolveToolMode diz 'nao medido', nunca 'nao suporta'): %v", erro)
	}
	if rotulo := rotuloDoEvento(t, eventosNaoMedido()); rotulo != "nao_medido" {
		t.Fatalf("a trilha tem de DECLARAR `nao_medido` (veio %q) — *resultado ausente e' NAO_MEDIDO, nao aprovacao*",
			rotulo)
	}

	// (3) Passo que NÃO exige `tools` ⇒ intocado, mesmo com o modelo medido sem chamada nativa.
	passoSemExigencia := []Step{{
		ID: "nao-precisa", Kind: "workspace.list", Title: "sem tools",
		State: StepPending, PinnedModelID: "local/forte",
	}}
	runtimeIntocado, idIntocado, eventosIntocado := missaoRoteada(t, &ferramentaQueAnotaModelo{}, passoSemExigencia, registryDeTeste(t), true)
	runtimeIntocado.modelCapabilities = registroSemFerramenta
	if erro := runtimeIntocado.Run(context.Background(), idIntocado); erro != nil {
		t.Fatalf("passo sem exigencia de `tools` nao pode ser recusado: %v", erro)
	}
	if rotulo := rotuloDoEvento(t, eventosIntocado()); rotulo != "nao_exigido" {
		t.Fatalf("a trilha devia declarar `nao_exigido`, veio %q", rotulo)
	}
}

// (4) A contraposta que impede o gate de ser um "recusa tudo": MEDIDO **com** chamada nativa ⇒ aprova e declara.
func TestGateDaChamadaNativaAprovaModeloMedidoComFerramentas(t *testing.T) {
	registroComFerramenta := registroMedidoParaTeste(t, "local/forte", true)
	passos := []Step{{
		ID: "precisa-ferramenta", Kind: "workspace.list", Title: "exige tools",
		State: StepPending, PinnedModelID: "local/forte",
		RequiredCapabilities: []string{"tools"},
	}}
	emissor := &ferramentaQueAnotaModelo{}
	runtime, id, eventos := missaoRoteada(t, emissor, passos, registryDeTeste(t), true)
	runtime.modelCapabilities = registroComFerramenta
	if erro := runtime.Run(context.Background(), id); erro != nil {
		t.Fatalf("modelo MEDIDO com tool call nativo nao pode ser recusado: %v", erro)
	}
	if rotulo := rotuloDoEvento(t, eventos()); rotulo != "nativo" {
		t.Fatalf("a trilha devia declarar `nativo`, veio %q", rotulo)
	}
	if emissor.modeloVisto != "local/forte" {
		t.Fatalf("o passo aprovado tinha de executar com o modelo roteado, veio %q", emissor.modeloVisto)
	}
}

// (5) O VOCABULÁRIO: a exigência é a capacidade de **MODELO** `tools` — nunca o escopo de **FERRAMENTA**.
//
// *E-437 deixou registrado que misturar os dois vocabulários foi um defeito real deste repositório: um passo de
// missão passou a exigir uma capacidade-ferramenta do MODELO e falhava fechado por um motivo que não existia.*
func TestExigeFerramentaNativaReconheceSomenteACapacidadeDeModelo(t *testing.T) {
	casos := []struct {
		nome     string
		caps     []string
		esperado bool
	}{
		{"capacidade de modelo `tools`", []string{"tools"}, true},
		{"`tools` com espacos e caixa alta", []string{"  Tools "}, true},
		{"`coding` nao exige ferramenta", []string{"coding"}, false},
		{"escopo de FERRAMENTA nao conta", []string{"workspace:write"}, false},
		{"misto: tools entre outras", []string{"coding", "tools"}, true},
		{"vazio", nil, false},
	}
	for _, caso := range casos {
		if obtido := exigeFerramentaNativa(Step{RequiredCapabilities: caso.caps}); obtido != caso.esperado {
			t.Fatalf("caso %q: esperado %v, veio %v", caso.nome, caso.esperado, obtido)
		}
	}
}

// registroMedidoParaTeste devolve o registro padrão ACRESCENTADO da medição do modelo do teste - assim o teste
// prova o comportamento de um modelo MEDIDO sem depender de qual modelo real está no ambiente do operador.
func registroMedidoParaTeste(t *testing.T, modelo string, toolCalling bool) *ModelCapabilityRegistry {
	t.Helper()
	registro := DefaultModelCapabilityRegistry()
	if err := registro.WithCapabilities(ModelCapabilities{
		Model: modelo, ToolCalling: toolCalling, StructuredOutput: true,
		Source: CapabilitySourceMeasured,
		Note:   "medicao injetada pelo teste do E-663",
	}); err != nil {
		t.Fatalf("injetar a medicao de %q no registro: %v", modelo, err)
	}
	return registro
}

func ehNaoMedido(registro *ModelCapabilityRegistry, modelo string) bool {
	_, conhecido := registro.Lookup(modelo)
	return !conhecido
}

// rotuloDoEvento lê o campo `native_tool_calling` do evento `step.model.route` — *a trilha é o que torna a
// decisão auditável; um rótulo que não chega ao evento não serve para auditar.*
func rotuloDoEvento(t *testing.T, eventos []Event) string {
	t.Helper()
	for _, evento := range eventos {
		if evento.Type != "step.model.route" {
			continue
		}
		pacote, ok := evento.Payload.(map[string]any)
		if !ok {
			t.Fatalf("o payload do evento `step.model.route` veio como %T; o teste espera um mapa", evento.Payload)
		}
		if valor, ok := pacote["native_tool_calling"].(string); ok {
			return valor
		}
		t.Fatal("o evento `step.model.route` tem de trazer o rotulo `native_tool_calling` (E-663)")
	}
	t.Fatal("nenhum evento `step.model.route` foi registrado")
	return ""
}
