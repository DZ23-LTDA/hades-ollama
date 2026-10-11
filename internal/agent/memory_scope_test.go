package agent

import (
	"context"
	"testing"
)

// Atividade do estágio 15: ESCOPOS de missão, agente e equipe.
//
// A distinção que estes testes travam:
//   - DIVULGAÇÃO: memória presa à missão X não aparece em busca de outro contexto;
//   - PROPRIEDADE: o dono continua podendo corrigir e apagar a memória dele, mesmo
//     presa a uma missão antiga — senão a memória vira órfã e imortal, o oposto da
//     exclusão e da retenção que o estágio pede.
func scopeStore(t *testing.T) (*ContextStore, Project) {
	t.Helper()
	store, err := NewContextStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto com escopos", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	return store, project
}

func TestMissionScopedMemoryIsNotDisclosedElsewhere(t *testing.T) {
	store, project := scopeStore(t)
	daMissao, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "detalhe da missão um", MissionID: "missao-1",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "conhecimento do projeto",
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	// Sem missão no escopo: só o conhecimento do projeto aparece.
	semMissao, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "detalhe", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, memory := range semMissao {
		if memory.ID == daMissao.ID {
			t.Fatalf("memória da missão um vazou para busca sem missão: %+v", memory)
		}
	}
	// Na missão CERTA, aparece.
	naMissao, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana", MissionID: "missao-1"}, "detalhe", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(naMissao) != 1 || naMissao[0].ID != daMissao.ID {
		t.Fatalf("a missão dona precisa encontrar a memória: %+v", naMissao)
	}
	// Em OUTRA missão, não aparece.
	outra, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana", MissionID: "missao-2"}, "detalhe", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outra) != 0 {
		t.Fatalf("memória da missão um não pode aparecer na missão dois: %+v", outra)
	}
	// O EXPORT também respeita: exportar sem missão traz só o do projeto.
	export, err := store.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 {
		t.Fatalf("o export sem escopo de missão não pode incluir a memória da missão: %+v", export.Memories)
	}
}

func TestAgentAndTeamScopesRestrictDisclosureOnly(t *testing.T) {
	store, project := scopeStore(t)
	// PRIVADA com escopo: é o caso que combina as duas regras. Memória de projeto
	// (`organization`) é conhecimento compartilhado e continua alterável por ator
	// autorizado — comportamento já travado em E-088.
	doAgente, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "anotação do agente revisor",
		AgentID: "agente-revisor", Visibility: MemoryVisibilityPrivate,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	daEquipe, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "acordo da equipe azul", TeamID: "equipe-azul",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}

	semEscopo, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "anotação", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, memory := range semEscopo {
		if memory.ID == doAgente.ID || memory.ID == daEquipe.ID {
			t.Fatalf("escopo de agente/equipe vazou para leitura sem escopo: %+v", memory)
		}
	}
	noAgente, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana", AgentID: "agente-revisor"}, "anotação", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(noAgente) != 1 || noAgente[0].ID != doAgente.ID {
		t.Fatalf("o agente dono precisa ler a memória dele: %+v", noAgente)
	}
	naEquipe, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana", TeamID: "equipe-azul"}, "acordo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(naEquipe) != 1 || naEquipe[0].ID != daEquipe.ID {
		t.Fatalf("a equipe dona precisa ler o acordo dela: %+v", naEquipe)
	}

	// PROPRIEDADE não depende do escopo: a dona corrige e apaga a própria memória
	// presa ao agente, mesmo informando um escopo diferente.
	if _, err := store.UpdateMemoryForActor(context.Background(), project.ID, doAgente.ID, "ana", "anotação corrigida", "correção"); err != nil {
		t.Fatalf("a dona precisa poder corrigir a própria memória com escopo: %v", err)
	}
	// E outro ator NÃO pode, nem com o escopo certo.
	if _, err := store.UpdateMemoryForActor(context.Background(), project.ID, doAgente.ID, "bruno", "texto do bruno", ""); err == nil {
		t.Fatal("terceiro não pode alterar memória presa a agente")
	}
	if err := store.DeleteMemoryForActor(project.ID, daEquipe.ID, "ana"); err != nil {
		t.Fatalf("a dona precisa poder apagar a própria memória de equipe: %v", err)
	}
}

func TestEmptyScopePreservesPreviousBehavior(t *testing.T) {
	store, project := scopeStore(t)
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "memória sem escopo",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	// Escopo vazio = vale para o projeto: nada do comportamento anterior muda.
	encontrados, err := store.SearchMemoriesInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "sem escopo", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(encontrados) != 1 {
		t.Fatalf("escopo vazio precisa continuar valendo para o projeto: %+v", encontrados)
	}
	// E o RAG fundamentado segue o mesmo caminho de escopo.
	texto, citacoes, err := store.GroundedAnswerInScope(context.Background(), project.ID, MemoryScope{ActorID: "ana"}, "sem escopo", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(citacoes) == 0 || texto == "" {
		t.Fatalf("o RAG precisa citar a memória do projeto: texto=%q citações=%d", texto, len(citacoes))
	}
}
