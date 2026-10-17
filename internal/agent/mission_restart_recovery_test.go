package agent

import (
	"context"
	"path/filepath"
	"testing"
)

// T1 — ESTADO RECUPERADO APÓS REINÍCIO (E-368).
//
// **MEDIDO antes**: o estágio de paridade (T1) exigia *"estado recuperado após reinício"* e eu havia registrado esse
// item como **⚠️ PARCIAL**: os checkpoints persistentes existiam, mas **não havia teste dedicado** provando que o
// estado de uma missão **sobrevive à morte do processo**.
//
// **COMO SE TESTA "REINÍCIO" SEM MATAR O PROCESSO**: o que define a durabilidade não é o PID, é o **disco**. Um
// `Runtime` novo — com **store** e **contexto** novos, construídos sobre o **mesmo diretório** — é o equivalente
// fiel de um processo que morreu e outro que subiu. *Se o segundo enxerga o que o primeiro gravou, o estado é
// durável; se não enxerga, ele vivia só na memória.*
//
// ***Estado que só existe em memória não sobrevive a um restart — e um teste que não reinicia nada não prova que
// sobrevive.***

// runtimeSobre abre store e contexto NOVOS sobre um mesmo diretório (o "processo" que acabou de subir).
func runtimeSobre(t *testing.T, raiz string) (*Runtime, *ContextStore) {
	t.Helper()
	store, err := NewJSONStore(filepath.Join(raiz, "store"))
	if err != nil {
		t.Fatal(err)
	}
	contextStore, err := NewContextStore(filepath.Join(raiz, "ctx"))
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(raiz, "workspace")
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Context:       contextStore,
		Planner:       RulePlanner{},
		WorkspaceRoot: workspace, DataRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return runtime.WithOrganization("org-a"), contextStore
}

func TestMissaoSobreviveAoReinicioComEstadoEEventos(t *testing.T) {
	raiz := t.TempDir()
	// PROCESSO 1: cria o projeto e a missão, e registra eventos reais.
	primeiro, contextoPrimeiro := runtimeSobre(t, raiz)
	workspace := filepath.Join(raiz, "workspace")
	projeto, err := contextoPrimeiro.CreateProject("projeto durável", workspace, "org-a")
	if err != nil {
		t.Fatalf("criar projeto no primeiro processo: %v", err)
	}
	missao, err := primeiro.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "objetivo que precisa sobreviver ao restart",
		ProjectID:      projeto.ID,
		OrganizationID: "org-a",
	})
	if err != nil {
		t.Fatalf("criar missão no primeiro processo: %v", err)
	}
	// Eventos são o rastro de execução: sem eles, "recuperado" significaria apenas "o id existe".
	if err := primeiro.observeEvent(missao, "missao.evento_de_teste", "passo-1",
		map[string]any{"marcador": "antes-do-restart"}); err != nil {
		t.Fatalf("registrar evento no primeiro processo: %v", err)
	}

	// PROCESSO 2: store e contexto NOVOS sobre o MESMO diretório — o reinício.
	segundo, _ := runtimeSobre(t, raiz)
	recuperadas, err := segundo.ListMissions()
	if err != nil {
		t.Fatalf("listar missões após o reinício: %v", err)
	}
	if len(recuperadas) != 1 {
		t.Fatalf("ARMADILHA: após o reinício a missão não foi recuperada (encontradas %d) — o estado vivia só na memória",
			len(recuperadas))
	}
	recuperada := recuperadas[0]
	if recuperada.ID != missao.ID {
		t.Fatalf("a missão recuperada é outra: %s (esperada %s)", recuperada.ID, missao.ID)
	}
	if recuperada.Objective != missao.Objective {
		t.Fatalf("o OBJETIVO não sobreviveu: %q (esperado %q)", recuperada.Objective, missao.Objective)
	}
	if recuperada.OrganizationID != "org-a" {
		t.Fatalf("o escopo de organização não sobreviveu: %q", recuperada.OrganizationID)
	}

	// E OS EVENTOS: sem eles a recuperação seria só um identificador órfão.
	eventos, err := segundo.store.ListEvents(missao.ID)
	if err != nil {
		t.Fatalf("listar eventos da missão recuperada: %v", err)
	}
	achouEvento := false
	for _, evento := range eventos {
		if evento.Type == "missao.evento_de_teste" {
			achouEvento = true
		}
	}
	if !achouEvento {
		t.Fatalf("ARMADILHA: os EVENTOS da missão não sobreviveram ao reinício (%d eventos recuperados)", len(eventos))
	}
}

func TestReinicioNaoVeEstadoDeOutroDiretorio(t *testing.T) {
	// ARMADILHA NEGATIVA: recuperação não pode significar "lembrar de tudo". Um diretório DIFERENTE é outro mundo.
	raizA := t.TempDir()
	primeiro, contextoA := runtimeSobre(t, raizA)
	projeto, err := contextoA.CreateProject("projeto A", filepath.Join(raizA, "workspace"), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := primeiro.CreateMission(context.Background(), CreateMissionRequest{
		Objective: "só existe em A", ProjectID: projeto.ID, OrganizationID: "org-a",
	}); err != nil {
		t.Fatal(err)
	}

	outro, _ := runtimeSobre(t, t.TempDir())
	missoes, err := outro.ListMissions()
	if err != nil {
		t.Fatalf("listar missões do diretório novo: %v", err)
	}
	if len(missoes) != 0 {
		t.Fatalf("ARMADILHA: o diretório novo enxergou %d missão(ões) de outro diretório", len(missoes))
	}
}
