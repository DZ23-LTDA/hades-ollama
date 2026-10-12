package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestRuntimeExecutaMissaoMultiEtapaComCadeiaDeAprovacoes cobre um caminho que os testes existentes
// NÃO cobriam: todos afirmavam uma única aprovação. Aqui o plano tem DUAS etapas de escrita, cada uma
// exigindo aprovação própria, mais uma etapa de leitura no fim.
//
// O que isso prova: o runtime não trava na primeira aprovação, não executa etapa sem aprovação, e a
// missão só conclui depois de a cadeia inteira ser aprovada — com DOIS artefatos verificáveis.
func TestRuntimeExecutaMissaoMultiEtapaComCadeiaDeAprovacoes(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{
		{ID: "step_1", Kind: "workspace.write", Title: "escrever a.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "a.txt", "content": "primeiro\n"}},
		{ID: "step_2", Kind: "workspace.write", Title: "escrever b.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "b.txt", "content": "segundo\n"}},
		{ID: "step_3", Kind: "workspace.list", Title: "listar o workspace", Risk: RiskRead, State: StepPending,
			Input: map[string]any{"path": ".", "max_entries": 10}},
	}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:    "escrever dois arquivos e listar",
		Capabilities: []string{"workspace:read", "workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if missao.State != MissionAwaitingApproval || len(missao.Approvals) != 2 {
		t.Fatalf("plano com duas escritas deve pedir DUAS aprovacoes: state=%s approvals=%d", missao.State, len(missao.Approvals))
	}

	// Cadeia de aprovacoes: aprovar todas as pendentes e retomar, com limite para não girar em falso.
	aprovacoesFeitas := 0
	for tentativa := 0; tentativa < 5; tentativa++ {
		atual, err := runtime.GetMission(missao.ID)
		if err != nil {
			t.Fatal(err)
		}
		if atual.State == MissionCompleted {
			break
		}
		if atual.State != MissionAwaitingApproval {
			if err := runtime.Run(ctx, missao.ID); err != nil {
				t.Fatalf("execucao falhou em %s: %v", atual.State, err)
			}
			continue
		}
		// Nada pode ter sido escrito antes da aprovacao correspondente.
		for _, aprovacao := range atual.Approvals {
			if aprovacao.Status != ApprovalPending {
				continue
			}
			if _, err := runtime.DecideApproval(missao.ID, aprovacao.ID, true, "aprovado pelo teste de cadeia"); err != nil {
				t.Fatalf("aprovacao falhou: %v", err)
			}
			aprovacoesFeitas++
		}
		if err := runtime.Run(ctx, missao.ID); err != nil {
			t.Fatalf("execucao apos aprovacao falhou: %v", err)
		}
	}

	final, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if aprovacoesFeitas != 2 {
		t.Fatalf("a cadeia exigia 2 aprovacoes, foram feitas %d", aprovacoesFeitas)
	}
	if final.State != MissionCompleted {
		t.Fatalf("missao multi-etapa nao concluiu: %s", final.State)
	}
	if len(final.Artifacts) != 2 {
		t.Fatalf("esperava 2 artefatos, veio %d: %+v", len(final.Artifacts), final.Artifacts)
	}
	for _, artefato := range final.Artifacts {
		if artefato.SHA256 == "" || artefato.StepID == "" {
			t.Fatalf("artefato sem SHA-256 ou sem vinculo com a etapa: %+v", artefato)
		}
	}
	for nome, conteudo := range map[string]string{"a.txt": "primeiro\n", "b.txt": "segundo\n"} {
		dados, err := os.ReadFile(filepath.Join(root, nome))
		if err != nil {
			t.Fatalf("arquivo %s nao existe: %v", nome, err)
		}
		if string(dados) != conteudo {
			t.Fatalf("conteudo de %s = %q, quer %q", nome, dados, conteudo)
		}
	}
	for _, etapa := range final.Plan {
		if etapa.State != StepSucceeded {
			t.Fatalf("etapa %q terminou em %s, queria SUCCEEDED", etapa.ID, etapa.State)
		}
	}
}

// TestRuntimeNaoExecutaEtapaSemAprovacaoNaCadeia garante o lado negativo: com uma aprovação NEGADA,
// a etapa correspondente não escreve e nenhum artefato daquela etapa aparece.
func TestRuntimeNaoExecutaEtapaSemAprovacaoNaCadeia(t *testing.T) {
	root := t.TempDir()
	planner := fixedPlanner{steps: []Step{
		{ID: "step_1", Kind: "workspace.write", Title: "escrever a.txt", Risk: RiskWrite, RequiresApproval: true, State: StepPending,
			Input: map[string]any{"path": "a.txt", "content": "nao deve existir\n"}},
	}}
	runtime, err := NewRuntime(RuntimeConfig{Store: NewMemoryStore(), Planner: planner, WorkspaceRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	missao, err := runtime.CreateMission(ctx, CreateMissionRequest{
		Objective:    "escrever arquivo",
		Capabilities: []string{"workspace:read", "workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(missao.Approvals) != 1 {
		t.Fatalf("esperava 1 aprovacao, veio %d", len(missao.Approvals))
	}
	if _, err := runtime.DecideApproval(missao.ID, missao.Approvals[0].ID, false, "negado pelo teste"); err != nil {
		t.Fatal(err)
	}
	_ = runtime.Run(ctx, missao.ID)
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("arquivo foi escrito apesar da aprovacao NEGADA: %v", err)
	}
	final, err := runtime.GetMission(missao.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, artefato := range final.Artifacts {
		t.Fatalf("artefato apareceu apesar da aprovacao negada: %+v", artefato)
	}
	if final.State == MissionCompleted {
		t.Fatal("missao NAO pode concluir quando a etapa obrigatoria foi negada")
	}
}
