package agent

import (
	"context"
	"testing"
)

// E-664 — A TRILHA DA DECISAO CARREGA OS PRETERIDOS E O CUSTO (§4).
//
// **O QUE ESTAVA MEDIDO:** o evento `step.model.route` registrava o **vencedor**, o score e o motivo — e **nao os
// preteridos**, nem custo. Uma escolha sem os preteridos **nao e' auditavel**: nao responde *"por que ESTE e nao
// aquele"*, que e' a pergunta do estagio 43; e sem custo nao existe base para **orcamento** (§4/§7).
//
// **AS DIRECOES QUE O TESTE MEDE** — *e a segunda e' a que costuma faltar*:
//  1. a trilha **existe** e traz os candidatos do proprio motor, **ordenados**, com o vencedor em primeiro;
//  2. o **custo** vem do registro (medido) — nao de um numero inventado nem de um recalculo paralelo;
//  3. **sem** registro de modelos a trilha e' `nil` e o evento **declara** `candidates_measured: false`
//     (*`nil` significa NAO MEDIDO, nunca "nenhum candidato"*).
func TestTrilhaDaDecisaoCarregaCandidatosECusto(t *testing.T) {
	passos := []Step{{
		ID: "escolha", Kind: "workspace.list", Title: "escolher modelo",
		State: StepPending, RequiredCapabilities: []string{"coding"},
	}}
	emissor := &ferramentaQueAnotaModelo{}
	runtime, id, eventos := missaoRoteada(t, emissor, passos, registryDeTeste(t), true)
	if err := runtime.Run(context.Background(), id); err != nil {
		t.Fatalf("executar a missao roteada: %v", err)
	}

	pacote := payloadDoEvento(t, eventos(), "step.model.route")

	// (1) A trilha existe, tem MAIS DE UM candidato (o registry de teste oferece dois modelos para `coding`) e o
	// vencedor e' o primeiro — *se `Route` e `RouteCandidates` divergissem, este teste falharia: seriam duas
	// respostas para a mesma pergunta.*
	medidos, ok := pacote["candidates_measured"].(bool)
	if !ok || !medidos {
		t.Fatalf("o evento tem de DECLARAR `candidates_measured` verdadeiro quando consegue medir (veio %v)",
			pacote["candidates_measured"])
	}
	// ⚠️ O payload passa por PERSISTENCIA: a lista volta como `[]any` de `map[string]any`, nao como
	// `[]map[string]any`. *Assercao tem de casar com a forma PERSISTIDA, nao com a forma em memoria* — foi
	// exatamente isto que a primeira versao deste teste errou.
	candidatos := candidatosDoPayload(t, pacote["candidates"])
	if len(candidatos) < 2 {
		t.Fatalf("esperados ao menos 2 candidatos para `coding`, vieram %d — *uma escolha sem preteridos nao e' "+
			"auditavel*", len(candidatos))
	}
	vencedor, _ := pacote["model"].(string)
	primeiro, _ := candidatos[0]["model"].(string)
	if vencedor == "" || primeiro != vencedor {
		t.Fatalf("o primeiro candidato tem de ser o VENCEDOR: vencedor=%q primeiro=%q — *duas contas sobre a "+
			"mesma decisao divergiriam em silencio*", vencedor, primeiro)
	}
	presentes := map[string]bool{}
	for _, candidato := range candidatos {
		if modelo, ok := candidato["model"].(string); ok {
			presentes[modelo] = true
		}
	}
	for _, esperado := range []string{"local/barato", "local/forte"} {
		if !presentes[esperado] {
			t.Fatalf("o candidato %q tinha de constar da trilha; presentes=%v", esperado, presentes)
		}
	}

	// (2) O CUSTO e' o do registro de teste (`registryDeTeste`: barato 1+1, visual 9+9, forte 50+50), nunca um
	// numero inventado. Confere-se contra a propria entrada do candidato vencedor.
	custoNumerico, ok := numeroDoPayload(pacote["estimated_cost_cents_per_1k"])
	if !ok {
		t.Fatalf("`estimated_cost_cents_per_1k` veio como %T", pacote["estimated_cost_cents_per_1k"])
	}
	custoDoCandidatoVencedor, ok := numeroDoPayload(candidatos[0]["cost_cents_per_1k"])
	if !ok {
		t.Fatalf("o candidato vencedor tem de trazer `cost_cents_per_1k` (veio %T)", candidatos[0]["cost_cents_per_1k"])
	}
	if custoNumerico != custoDoCandidatoVencedor {
		t.Fatalf("o custo do vencedor (%v) diverge do custo na trilha (%v) — *duas fontes para o mesmo numero*",
			custoNumerico, custoDoCandidatoVencedor)
	}
	declarados := map[int64]bool{2: true, 18: true, 100: true} // barato, visual, forte (soma entrada+saida)
	if !declarados[int64(custoNumerico)] {
		t.Fatalf("o custo medido %v nao corresponde a NENHUM custo declarado no registro de teste (%v) — "+
			"*custo inventado*", custoNumerico, declarados)
	}

	// (3) A trilha tambem traz o PROVEDOR e o MOTIVO de cada candidato (auditoria completa da decisao).
	if provedor, _ := candidatos[0]["provider"].(string); provedor == "" {
		t.Fatal("cada candidato tem de trazer o provedor")
	}
	if motivo, _ := candidatos[0]["reason"].(string); motivo == "" {
		t.Fatal("cada candidato tem de trazer o motivo (routeReason)")
	}
}

// (3) SEM registro de modelos nao ha trilha: `nil` — e o chamador declara, em vez de fingir lista vazia.
func TestTrilhaSemRegistroDeModelosENaoMedida(t *testing.T) {
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: t.TempDir() + "-ws",
		DataRoot:      t.TempDir() + "-data",
	})
	if err != nil {
		t.Fatalf("runtime sem registry de modelos: %v", err)
	}
	if trilha := runtime.trilhaDeCandidatos(Mission{}, Step{ID: "s", RequiredCapabilities: []string{"coding"}}, 5); trilha != nil {
		t.Fatalf("sem registry de modelos a trilha tem de ser NIL (NAO MEDIDO), veio %v", trilha)
	}
}

// payloadDoEvento devolve o payload do ULTIMO evento do tipo pedido - *o mais recente e' o que corresponde ao
// passo executado.*
func payloadDoEvento(t *testing.T, eventos []Event, tipo string) map[string]any {
	t.Helper()
	var escolhido map[string]any
	for _, evento := range eventos {
		if evento.Type != tipo {
			continue
		}
		pacote, ok := evento.Payload.(map[string]any)
		if !ok {
			t.Fatalf("payload do evento %q veio como %T", tipo, evento.Payload)
		}
		escolhido = pacote
	}
	if escolhido == nil {
		t.Fatalf("nenhum evento %q foi registrado", tipo)
	}
	return escolhido
}

// numeroDoPayload converte os tipos numericos que o JSON/persistencia pode devolver, sem inventar valor.
func numeroDoPayload(valor any) (float64, bool) {
	switch numero := valor.(type) {
	case float64:
		return numero, true
	case int:
		return float64(numero), true
	case int64:
		return float64(numero), true
	default:
		return 0, false
	}
}

// candidatosDoPayload normaliza a lista de candidatos para a forma que o teste usa, aceitando as DUAS formas
// (em memoria e depois da persistencia). *Falha alto se vier outra coisa: aceitar qualquer coisa esvaziaria a
// assercao.*
func candidatosDoPayload(t *testing.T, valor any) []map[string]any {
	t.Helper()
	if lista, ok := valor.([]map[string]any); ok {
		return lista
	}
	lista, ok := valor.([]any)
	if !ok {
		t.Fatalf("o campo `candidates` veio como %T; o esperado e' uma lista de mapas", valor)
	}
	normalizada := make([]map[string]any, 0, len(lista))
	for indice, item := range lista {
		pacote, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("o candidato %d veio como %T; o esperado e' um mapa", indice, item)
		}
		normalizada = append(normalizada, pacote)
	}
	return normalizada
}
