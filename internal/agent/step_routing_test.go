package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

// E-435 — A PONTE `Step` -> `RouteRequest`, E O MOTOR DE SELECAO ESCOLHENDO POR ELA.
//
// O teste mede DUAS coisas separadas, de proposito:
//  1. o **mapeamento** (*capacidades do passo + da missao, sem duplicata; pin do passo vencendo o da missao*);
//  2. o **motor honrando o mapeamento** - *nao basta traduzir: o pedido tem de produzir a escolha certa quando
//     entregue ao `multillm.Registry`* ✓.
//
// ***Um mapeamento correto entregue a um motor que o ignora nao e' integracao - e' duas coisas certas e uma ligacao
// que nao existe.*** (E e' exactamente por isso que a injecao no Runtime continua declarada como NAO feita.)
func TestPonteDoPassoParaORegistry(t *testing.T) {
	missao := Mission{
		OrganizationID: "org-a",
		Model:          "local/modelo-da-missao",
		Capabilities:   []string{"coding", "vision"},
	}

	// (a) MAPEAMENTO: uniao sem duplicata, com espaco em branco ignorado.
	passo := Step{
		ID:                   "s1",
		RequiredCapabilities: []string{"vision", "  ", "tools"},
	}
	pedido := RouteRequestForStep(missao, passo)
	if len(pedido.RequiredCapabilities) != 3 {
		t.Fatalf("esperava 3 capacidades (coding, vision, tools) sem duplicata e sem vazio, veio %v",
			pedido.RequiredCapabilities)
	}
	vistas := map[string]bool{}
	for _, c := range pedido.RequiredCapabilities {
		if vistas[c] {
			t.Fatalf("capacidade duplicada no pedido: %v", pedido.RequiredCapabilities)
		}
		vistas[c] = true
	}
	for _, esperada := range []string{"coding", "vision", "tools"} {
		if !vistas[esperada] {
			t.Fatalf("faltou %q no pedido: %v", esperada, pedido.RequiredCapabilities)
		}
	}

	// (b) O PIN DO PASSO VENCE O DA MISSAO - *o mais especifico decide.*
	comPin := RouteRequestForStep(missao, Step{ID: "s2", PinnedModelID: "local/modelo-do-passo"})
	if comPin.PinnedModelID != "local/modelo-do-passo" {
		t.Fatalf("o pin do PASSO tem de vencer o da missao, veio %q", comPin.PinnedModelID)
	}
	// E sem pin no passo, o da missao serve de padrao.
	semPin := RouteRequestForStep(missao, Step{ID: "s3"})
	if semPin.PinnedModelID != "local/modelo-da-missao" {
		t.Fatalf("sem pin no passo, o da missao e' o padrao, veio %q", semPin.PinnedModelID)
	}

	// (c) O MOTOR HONRA A PONTE: com pin, a lista de candidatos tem EXATAMENTE o modelo fixado; sem pin, o motor
	// escolhe pela capacidade exigida.
	//
	// ⚠️ E O PIN NAO PODE ATROPELAR A CAPACIDADE: o primeiro modelo que usei aqui (`local/barato`, so' `coding`) deu
	// **`no provider route satisfies the requested constraints`** ✗ - *a ponte propaga as capacidades da MISSAO*
	// (`coding`+`vision`) *e o barato nao as tem* ⇒ ***um pin que nao pode ser honrado FALHA, em vez de trocar de
	// modelo em silencio*** ✓, que e' exactamente o contrato declarado em E-428. *O teste estava pedindo o impossivel
	// e a resposta correta era a recusa.*
	registry := registryDeTeste(t)
	candidatos, err := registry.RouteCandidates(RouteRequestForStep(missao,
		Step{ID: "s4", PinnedModelID: "local/forte"}))
	if err != nil {
		t.Fatalf("candidatos com pin: %v", err)
	}
	if len(candidatos) != 1 || candidatos[0].Model.ID != "local/forte" {
		t.Fatalf("ARMADILHA: o pin da ponte nao chegou ao motor: %+v", candidatos)
	}
	// E o CONTROLE do paragrafo acima: *o mesmo pin num passo que NAO exige visao* passa - *senao o caso anterior
	// estaria medindo "pin sempre falha", que e' um defeito diferente.*
	soCodigo := Mission{Capabilities: []string{"coding"}}
	barato, err := registry.Route(RouteRequestForStep(soCodigo, Step{ID: "s4b", PinnedModelID: "local/barato"}))
	if err != nil {
		t.Fatalf("pin honravel recusado: %v", err)
	}
	if barato.Model.ID != "local/barato" {
		t.Fatalf("com pin em `local/barato` e so' `coding`, a escolha tinha de ser o barato: %q", barato.Model.ID)
	}

	// E a capacidade do PASSO filtra de verdade: pedir "vision" so' pode devolver quem TEM visao.
	//
	// ⚠️ Eu havia asserido o ID `local/visual` e o motor devolveu `local/forte` ✗ - *que **tambem** tem visao* e
	// pontua mais alto ⇒ ***a minha expectativa e' que estava errada, nao o codigo*** ✓. A asserção correta e' a
	// **propriedade** ("tem a capacidade"), nao um ID: *assercao por ID mede a ordenacao de hoje; assercao por
	// propriedade mede a regra.*
	soVisao := RouteRequestForStep(Mission{Capabilities: []string{}}, Step{ID: "s5", RequiredCapabilities: []string{"vision"}})
	decisao, err := registry.Route(soVisao)
	if err != nil {
		t.Fatalf("rota por visao: %v", err)
	}
	temVisao := false
	for _, capacidade := range decisao.Model.Capabilities {
		if capacidade == "vision" {
			temVisao = true
		}
	}
	if !temVisao {
		t.Fatalf("ARMADILHA: a capacidade do PASSO nao filtrou - escolheu %q, que NAO tem `vision` (capacidades: %v)",
			decisao.Model.ID, decisao.Model.Capabilities)
	}

	// E uma exigencia impossivel tem de FALHAR, nao de ser ignorada.
	if _, err := registry.Route(RouteRequestForStep(Mission{},
		Step{ID: "s6", RequiredCapabilities: []string{"capability-que-ninguem-tem"}})); !errors.Is(err, multillm.ErrNoRoute) {
		t.Fatalf("exigencia impossivel tinha de dar ErrNoRoute, veio %v", err)
	}
}

func registryDeTeste(t *testing.T) *multillm.Registry {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	dados := `{"providers":[{"name":"local","type":"openai-compatible","base_url":"https://local.example",` +
		`"api_key_env":"LOCAL_KEY","priority":1,"models":[` +
		`{"id":"barato","capabilities":["coding"],"cost_per_1k_input_cents":1,"cost_per_1k_output_cents":1,"quality_score":10},` +
		`{"id":"visual","capabilities":["vision"],"cost_per_1k_input_cents":9,"cost_per_1k_output_cents":9,"quality_score":50},` +
		`{"id":"forte","capabilities":["coding","vision","tools"],"cost_per_1k_input_cents":50,"cost_per_1k_output_cents":50,"quality_score":90}` +
		`]}]}`
	if err := os.WriteFile(path, []byte(dados), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCAL_KEY", "local")
	registry, err := multillm.Load(path)
	if err != nil {
		t.Fatalf("carregar registry de teste: %v", err)
	}
	return registry
}
