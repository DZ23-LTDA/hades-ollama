package agent

import (
	"context"
	"strings"
	"testing"
)

// DEFEITO CORRIGIDO (estágio 15, ALTO): o contexto injetado no modelo filtrava
// memória só por PROJETO. Dentro do mesmo projeto, memória `private` de OUTRO
// usuário entrava no preâmbulo de sistema — e o preâmbulo vai para o modelo, que
// pode ser externo. Era o terceiro caminho da mesma família: a busca já era
// escopada por ator, o export e o delete foram corrigidos, e o contexto não.
//
// O filtro por ator dentro do builder é fail-closed: nem um chamador que entregue
// a lista crua do projeto consegue vazar memória privada alheia.
func bootstrapInputWithMemories(memories []Memory, actor string) ContextBootstrapInput {
	return ContextBootstrapInput{
		OrganizationID:      "org-a",
		ProjectID:           "proj-1",
		MissionID:           "mission-1",
		ActorID:             actor,
		Objective:           "usar o que sei do projeto",
		GrantedCapabilities: []string{"workspace:read"},
		Policy:              DefaultCapabilityPolicy(),
		Memories:            memories,
	}
}

func TestContextBootstrapNeverInjectsOtherActorPrivateMemory(t *testing.T) {
	memories := []Memory{
		{ID: "mem_time", ProjectID: "proj-1", Kind: "note", Content: "conhecimento do time", Visibility: MemoryVisibilityOrganization},
		{ID: "mem_ana", ProjectID: "proj-1", Kind: "note", Content: "segredo da ana", Visibility: MemoryVisibilityPrivate, ActorID: "ana"},
		{ID: "mem_bruno", ProjectID: "proj-1", Kind: "note", Content: "segredo do bruno", Visibility: MemoryVisibilityPrivate, ActorID: "bruno"},
	}
	// O bruno recebe a lista CRUA do projeto (pior caso): ainda assim, a memória
	// privada da ana não pode aparecer.
	bootstrap, err := BuildContextBootstrap(bootstrapInputWithMemories(memories, "bruno"))
	if err != nil {
		t.Fatal(err)
	}
	rendered := bootstrap.SystemMessage()
	if strings.Contains(rendered, "segredo da ana") {
		t.Fatalf("o contexto do modelo VAZOU memória privada de outro usuário: %s", rendered)
	}
	ids := make([]string, 0, len(bootstrap.MemoryRefs))
	for _, ref := range bootstrap.MemoryRefs {
		ids = append(ids, ref.ID)
	}
	if len(ids) != 2 {
		t.Fatalf("o bruno deveria receber o conhecimento do time e a própria memória, veio %v", ids)
	}
	for _, id := range ids {
		if id == "mem_ana" {
			t.Fatalf("memória privada da ana entrou no contexto do bruno: %v", ids)
		}
	}
}

func TestContextBootstrapWithoutActorInjectsNoPrivateMemory(t *testing.T) {
	memories := []Memory{
		{ID: "mem_time", ProjectID: "proj-1", Kind: "note", Content: "conhecimento do time", Visibility: MemoryVisibilityOrganization},
		{ID: "mem_ana", ProjectID: "proj-1", Kind: "note", Content: "segredo da ana", Visibility: MemoryVisibilityPrivate, ActorID: "ana"},
	}
	// Sem ator identificado, nada privado entra — na dúvida, não vaza.
	bootstrap, err := BuildContextBootstrap(bootstrapInputWithMemories(memories, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(bootstrap.MemoryRefs) != 1 || bootstrap.MemoryRefs[0].ID != "mem_time" {
		t.Fatalf("sem ator, só o conhecimento do projeto pode entrar: %+v", bootstrap.MemoryRefs)
	}
	if strings.Contains(bootstrap.SystemMessage(), "segredo da ana") {
		t.Fatal("contexto sem ator não pode carregar memória privada")
	}
}

// Critério do estágio 15: TROCA DE MODELO preservando contexto e RETOMADA.
//
// A troca de modelo no produto é a seleção por missão (`mission.Model` sobrepõe o
// modelo do planner). O teste mede as duas coisas: o modelo que chega ao provedor
// MUDA, e o contexto injetado NÃO muda. É a diferença entre "trocar o motor" e
// "perder o contexto".
func TestContextBootstrapSurvivesModelSwitchAndRestart(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto da missão", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "decisão registrada do time", Visibility: MemoryVisibilityOrganization,
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	planWith := func(model string, store *ContextStore) (string, string) {
		t.Helper()
		stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
		planner := OllamaPlanner{Client: stub, Model: "modelo-padrão"}
		memories, err := store.SearchMemoriesForActor(context.Background(), project.ID, "ana", "decisão", MaxContextBootstrapMemories)
		if err != nil {
			t.Fatal(err)
		}
		bootstrap, err := BuildContextBootstrap(ContextBootstrapInput{
			OrganizationID:      "org-a",
			ProjectID:           project.ID,
			MissionID:           "mission-1",
			ActorID:             "ana",
			Objective:           "continuar a missão",
			GrantedCapabilities: []string{"workspace:read"},
			Policy:              DefaultCapabilityPolicy(),
			Memories:            memories,
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = planner.Plan(context.Background(), Mission{
			ID:             "mission-1",
			Objective:      "continuar a missão",
			OrganizationID: "org-a",
			ProjectID:      project.ID,
			Model:          model,
			Capabilities:   []string{"workspace:read"},
		})
		if err != nil {
			t.Fatal(err)
		}
		_ = stub
		return bootstrap.SystemMessage(), memories[0].ID
	}

	preambleA, memoryA := planWith("modelo-local", store)
	// Ainda o MESMO store: troca de modelo não pode mexer no contexto.
	preambleB, memoryB := planWith("modelo-na-nuvem", store)
	// REINÍCIO: store novo sobre o mesmo diretório (retomada).
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	preambleC, memoryC := planWith("modelo-local", restarted)

	if preambleA != preambleB {
		t.Fatalf("a troca de modelo mudou o contexto injetado:\nA=%q\nB=%q", preambleA, preambleB)
	}
	if preambleA != preambleC {
		t.Fatalf("a retomada após reinício mudou o contexto injetado:\nA=%q\nC=%q", preambleA, preambleC)
	}
	if memoryA != memoryB || memoryA != memoryC || memoryA == "" {
		t.Fatalf("a memória do projeto precisa sobreviver à troca e ao reinício: %q %q %q", memoryA, memoryB, memoryC)
	}
	if !strings.Contains(preambleA, "Hades") {
		t.Fatalf("o preâmbulo precisa manter a identidade: %q", preambleA)
	}
}

// A troca de modelo precisa ser REAL: o modelo selecionado chega ao provedor.
func TestModelSwitchChangesTheModelReachingTheProvider(t *testing.T) {
	plan := func(model string) string {
		t.Helper()
		stub := &plannerChatStub{response: `{"steps":[{"kind":"workspace.read","title":"inspect","risk":"read","input":{"path":"."}}]}`}
		planner := OllamaPlanner{Client: stub, Model: "modelo-padrão"}
		if _, err := planner.Plan(context.Background(), Mission{
			ID: "mission-1", Objective: "ir", OrganizationID: "org-a", ProjectID: "proj-1", Model: model,
		}); err != nil {
			t.Fatal(err)
		}
		return stub.model
	}
	if got := plan("modelo-na-nuvem"); got != "modelo-na-nuvem" {
		t.Fatalf("o modelo da missão precisa vencer o padrão, veio %q", got)
	}
	if got := plan(""); got != "modelo-padrão" {
		t.Fatalf("sem seleção, vale o modelo do planner, veio %q", got)
	}
}
