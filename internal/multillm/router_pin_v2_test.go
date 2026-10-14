package multillm

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// E-428 — PIN DE MODELO POR TAREFA (estratégia `MANUAL` do Estágio 43): RESTRIÇÃO RÍGIDA, NÃO PREFERÊNCIA.
//
// **O DIAGNÓSTICO, CORRIGIDO POR MEDIÇÃO — DUAS VEZES.** Eu escrevi na matriz de lacunas que `MIN_COST` estava
// *ausente*: **falso** — `RoutePreferenceCheapest` existe (`router.go:25`). *A capacidade existia com **outro nome**,
// que é exatamente a armadilha que o complemento avisa ("não suponha ausência com base somente no nome").* E eu
// também suporia que `PreferredProvider` era *"um bônus que deixa os outros concorrerem"*: **medido, também falso** —
// ele **filtra** (`:67`).
//
// ⇒ **A lacuna real era outra**: não havia como fixar o **MODELO**. `PreferredProvider` fixa o **provedor**; faltava
// *"quero **este** modelo nesta tarefa"*.
//
// ***Uma tarefa que fixa um modelo e recebe outro silenciosamente é a mesma traição de confiar numa aprovação que
// não foi dada — só muda o objeto.***
func TestPinDeModeloERestricaoRigidaNaoPreferencia(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "providers.json")
	// UM provedor, DOIS modelos: é este o caso que `PreferredProvider` não consegue separar — *e é o caso real de
	// "quero o barato deste provedor, não o forte dele".*
	data := `{"providers":[{"name":"local","type":"openai-compatible","base_url":"https://local.example",` +
		`"api_key_env":"LOCAL_KEY","priority":1,"models":[` +
		`{"id":"modelo-forte","capabilities":["coding"],"cost_per_1k_input_cents":50,"cost_per_1k_output_cents":50,"quality_score":90},` +
		`{"id":"modelo-barato","capabilities":["coding"],"cost_per_1k_input_cents":1,"cost_per_1k_output_cents":1,"quality_score":20}` +
		`]}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOCAL_KEY", "local")
	registry, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	// (a) SEM pin, a preferência por qualidade escolhe o forte — *o controle: o motor continua funcionando.*
	semPin, err := registry.Route(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		Preference:           RoutePreferenceBestQuality,
	})
	if err != nil {
		t.Fatalf("sem pin: %v", err)
	}
	if semPin.Model.ID != "local/modelo-forte" {
		t.Fatalf("sem pin, a qualidade alta deveria vencer: %+v", semPin)
	}

	// (b) COM pin no modelo BARATO, ele tem de vencer — *mesmo com a preferência pedindo qualidade.*
	// ***É aqui que um pin implementado como bônus de pontuação falharia.***
	comPin, err := registry.Route(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		Preference:           RoutePreferenceBestQuality,
		PinnedModelID:        "local/modelo-barato",
	})
	if err != nil {
		t.Fatalf("com pin: %v", err)
	}
	if comPin.Model.ID != "local/modelo-barato" {
		t.Fatalf("ARMADILHA: com PIN em `modelo-barato`, o motor escolheu %q — *o pin virou preferência e a tarefa "+
			"recebeu outro modelo silenciosamente*", comPin.Model.ID)
	}

	// (c) E o pin RESTRINGE A LISTA inteira: nenhum outro candidato pode aparecer —
	// *é a diferença entre "respeitei sua preferência" e "respeitei sua decisão".*
	candidatos, err := registry.RouteCandidates(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		PinnedModelID:        "local/modelo-barato",
	})
	if err != nil {
		t.Fatalf("candidatos com pin: %v", err)
	}
	if len(candidatos) != 1 || candidatos[0].Model.ID != "local/modelo-barato" {
		t.Fatalf("ARMADILHA: com PIN, a lista de candidatos tinha de ter EXATAMENTE o modelo fixado, veio %+v "+
			"(*um pin que deixa outros concorrerem não é pin*)", candidatos)
	}

	// (d) PIN QUE NÃO PODE SER HONRADO ⇒ `ErrNoRoute`, **nunca** um substituto silencioso.
	if _, err := registry.Route(RouteRequest{
		RequiredCapabilities: []string{"coding"},
		PinnedModelID:        "modelo-que-nao-existe",
	}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("ARMADILHA: pin impossível devolveu %v; *um pin não honrado tem de FALHAR, não escolher outro*", err)
	}

	// (e) Pin + capability que o modelo não tem ⇒ falha, em vez de o motor ignorar uma das duas restrições.
	if _, err := registry.Route(RouteRequest{
		RequiredCapabilities: []string{"vision"},
		PinnedModelID:        "local/modelo-barato",
	}); !errors.Is(err, ErrNoRoute) {
		t.Fatalf("pin + capability incompatível tinha de falhar: %v", err)
	}
}
