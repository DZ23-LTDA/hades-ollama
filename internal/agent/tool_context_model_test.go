package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/internal/multillm"
)

// E-438 — **A DECISÃO APLICADA**: o modelo roteado para o passo CHEGA À FERRAMENTA.
//
// **O QUE ESTAVA MEDIDO**: a rota por passo era tomada e **registrada** no evento `step.model.route` (E-437), mas
// **não chegava à ferramenta** ✗ — o `ToolContext` (`types.go:202`) não tinha campo de modelo ⇒ *a decisão ficava na
// cadeia de evidências e **fora do caminho de execução*** ✓. ***Decisão registrada e não aplicada é documentação, não
// é controle.***
//
// ⇒ acrescentado `ToolContext.ModelID` e o laço passa o modelo escolhido ✓.
//
// **AS TRÊS DIREÇÕES QUE O TESTE MEDE** — *e é a terceira que quase sempre falta*:
//  1. **com** registry e **pin** no passo ⇒ a ferramenta recebe **aquele** modelo (`local/barato`);
//  2. **sem** registry ⇒ a ferramenta recebe **vazio** — *e vazio significa "não houve roteamento por passo", **não**
//     "escolha um modelo por conta própria"* ✓;
//  3. o **registro** (`step.model.route`) e a **entrega** (`ToolContext.ModelID`) contam a **mesma história** ⇒
//     *é isto que impede o evento de virar evidência de algo que não aconteceu* ✓.
func TestModeloRoteadoChegaNaFerramenta(t *testing.T) {
	emissor := &ferramentaQueAnotaModelo{}

	// (1) COM registry e pin no passo.
	missaoComPin := []Step{{ID: "passo-pinado", Kind: "workspace.list", Title: "anotar",
		State: StepPending, PinnedModelID: "local/barato"}}
	runtime, id, eventos := missaoRoteada(t, emissor, missaoComPin, registryDeTeste(t), true)
	if err := runtime.Run(context.Background(), id); err != nil {
		t.Fatalf("executar missao roteada: %v", err)
	}
	if emissor.modeloVisto != "local/barato" {
		t.Fatalf("ARMADILHA: a ferramenta nao recebeu o modelo roteado (recebeu %q) — *a decisao foi tomada e "+
			"REGISTRADA, mas nao APLICADA*", emissor.modeloVisto)
	}
	// (3) E a evidencia registrada aponta o MESMO modelo que a ferramenta recebeu.
	var doEvento string
	for _, evento := range eventos() {
		if evento.Type != "step.model.route" {
			continue
		}
		// `Payload` e' `any` (medido ao compilar): *assercao explicita, nao acesso direto.*
		pacote, ok := evento.Payload.(map[string]any)
		if !ok {
			t.Fatalf("o payload do evento `step.model.route` veio como %T, e o teste espera um mapa - *o registro "+
				"precisa ser legivel por quem audita*", evento.Payload)
		}
		if valor, ok := pacote["model"].(string); ok {
			doEvento = valor
		}
	}
	if doEvento == "" {
		t.Fatal("o evento `step.model.route` tem de existir: *sem ele a entrega nao e' auditavel*")
	}
	if doEvento != emissor.modeloVisto {
		t.Fatalf("o REGISTRO e a ENTREGA contam historias diferentes: evento=%q ferramenta=%q — *um evento que nao "+
			"corresponde ao que a ferramenta recebeu e' evidencia de algo que nao aconteceu*", doEvento, emissor.modeloVisto)
	}

	// (2) SEM registry: nada muda no comportamento, e o campo fica VAZIO (*nao inventa modelo*).
	semRegistry := &ferramentaQueAnotaModelo{}
	runtime2, id2, _ := missaoRoteada(t, semRegistry, missaoComPin, nil, true)
	if err := runtime2.Run(context.Background(), id2); err != nil {
		t.Fatalf("executar missao sem registry: %v", err)
	}
	if semRegistry.modeloVisto != "" {
		t.Fatalf("sem registry o `ModelID` tem de ficar VAZIO (recebeu %q) — *vazio significa 'nao houve roteamento "+
			"por passo', nao 'use este modelo'*", semRegistry.modeloVisto)
	}
}

// ferramentaQueAnotaModelo registra o `ToolContext.ModelID` que RECEBEU - *o unico jeito de medir a entrega.*
//
// ⚠️ E-438: o `Descriptor().Name` tem de ser uma ferramenta do **CATALOGO** ✗ - *eu usei `teste.anota.modelo` e o
// proprio motor recusou o plano com "planner returned unsupported tool"* ✓. *A recusa esta' certa: um plano so'
// executa ferramentas conhecidas.* Uso `workspace.list`, que existe, e o teste mede o **laco**, nao o catalogo.
type ferramentaQueAnotaModelo struct {
	modeloVisto string
}

func (f *ferramentaQueAnotaModelo) Descriptor() ToolDescriptor {
	return ToolDescriptor{Name: "workspace.list", Description: "anota o modelo recebido",
		Scopes: []string{"workspace:read"}}
}

func (f *ferramentaQueAnotaModelo) Execute(_ context.Context, contexto ToolContext, _ map[string]any) (ToolResult, error) {
	f.modeloVisto = contexto.ModelID
	return ToolResult{Value: map[string]any{"modelo": contexto.ModelID}}, nil
}

// planejadorFixo devolve um plano pronto - *sem ele o plano viria do RulePlanner e eu nao poderia fixar o `DependsOn`/
// `PinnedModelID` do passo.*
type planejadorFixo struct{ passos []Step }

func (p planejadorFixo) Plan(context.Context, Mission) ([]Step, error) {
	return append([]Step(nil), p.passos...), nil
}

func missaoRoteada(t *testing.T, ferramenta Tool, passos []Step, registry *multillm.Registry, comProjeto bool) (*Runtime, string, func() []Event) {
	t.Helper()
	raiz := t.TempDir()
	runtime, contexto := runtimeSobre(t, raiz)
	projetoID := ""
	if comProjeto {
		// ⚠️ E-438, DOIS enganos meus de setup, ambos recusados pelo proprio motor - *e as duas recusas estao certas*:
		//   1. "project root is outside the runtime workspace" -> eu usei outro `t.TempDir()`;
		//   2. "organization-scoped missions require a project owned by the active organization" -> *faltava ligar a
		//      missao ao projeto* (`ProjectID`). ***Missao com escopo de organizacao sem projeto e' recusada - e tem
		//      de ser: sem projeto nao ha a quem pertencer.***
		projeto, err := contexto.CreateProject("projeto do roteamento", filepath.Join(raiz, "workspace"), "org-a")
		if err != nil {
			t.Fatal(err)
		}
		projetoID = projeto.ID
	}
	// 🚨 E-438 — **A ORDEM IMPORTA, E O DIAGNOSTICO PROVOU**: o planejador tem de ser trocado **ANTES** do
	// `CreateMission`, porque e' `CreateMission` que **planeja e PERSISTE** o plano. Eu trocava **depois** ⇒ *o plano
	// persistido era o do `RulePlanner` (`kind: "workspace.list"`), a **minha** ferramenta nunca era executada, e o
	// `ModelID` chegava vazio na ferramenta errada* ✗✓.
	//
	// **E o diagnostico mostrou que a FEATURE funciona**: o evento `step.model.route` **disparou** com
	// `model: local/forte`. *O defeito era do teste, nao do produto - e foi preciso medir para saber qual dos dois.*
	runtime.planner = planejadorFixo{passos: passos}
	runtime.models = registry
	runtime.tools.Register(ferramenta)

	missao, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "entregar o modelo roteado a ferramenta", OrganizationID: "org-a", ProjectID: projetoID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime, missao.ID, func() []Event {
		eventos, err := runtime.Events(missao.ID)
		if err != nil {
			t.Fatalf("eventos: %v", err)
		}
		return eventos
	}
}

var _ = os.Getenv
