package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// MOTOR DE WORKFLOW — TESTE T6 DO ADENDO (E-324).
//
// Cobre: webhook → condição → AÇÃO REAL (efeito observável em sandbox) · condição falsa com ramo PULADO e registrado ·
// retry por nó com tentativas contadas · DLQ em falha definitiva · idempotência por chave (com armadilha ARMADA) ·
// nó de tipo desconhecido RECUSADO (o V2.2 proíbe declarar sucesso com nó ignorado) · persistência após reinício ·
// isolamento por organização.
func TestWorkflowWebhookCondicaoAcaoRetryEDLQ(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "workflows")

	// A AÇÃO REAL é um efeito observável: um contador em memória compartilhado pelo executor.
	var mu sync.Mutex
	contador := map[string]int{}
	falhasRestantes := map[string]int{} // quantas vezes uma ação ainda deve falhar antes de ter sucesso

	executor := func(_ context.Context, _ WorkflowDefinition, node WorkflowNode, event map[string]string) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		alvo := node.Config["action"]
		// Mapa com chave ausente devolve 0 em Go: o helper que eu havia escrito era desnecessário.
		if falhasRestantes[alvo] > 0 {
			falhasRestantes[alvo]--
			return "", fmt.Errorf("falha simulada em %s", alvo)
		}
		contador[alvo]++
		return fmt.Sprintf("%s executado para %s", alvo, event["contato"]), nil
	}

	motor := NewWorkflowEngine(raiz, executor)
	fluxo, err := motor.Register(WorkflowDefinition{
		OrganizationID: "org-a",
		Name:           "atendimento-por-webhook",
		StartNode:      "gatilho",
		Nodes: []WorkflowNode{
			{ID: "gatilho", Kind: WorkflowNodeTrigger, Next: "decisao"},
			{ID: "decisao", Kind: WorkflowNodeCondition, Next: "registrar",
				OnFalse: "descartar", Config: map[string]string{"field": "urgencia", "equals": "alta"}},
			{ID: "registrar", Kind: WorkflowNodeAction, Next: "", Retry: NodeRetryPolicy{MaxAttempts: 3},
				Config: map[string]string{"action": "abrir_ticket"}},
			{ID: "descartar", Kind: WorkflowNodeAction, Config: map[string]string{"action": "arquivar"}},
		},
	})
	if err != nil {
		t.Fatalf("registrar workflow: %v", err)
	}

	// 1. WEBHOOK → CONDIÇÃO VERDADEIRA → AÇÃO REAL.
	execucao, err := motor.Run(context.Background(), "org-a", fluxo.ID,
		map[string]string{"urgencia": "alta", "contato": "+5511900000001"}, "evento-1")
	if err != nil {
		t.Fatalf("executar: %v", err)
	}
	if execucao.State != WorkflowRunSucceeded {
		t.Fatalf("fluxo com condição verdadeira precisa ter sucesso: %+v", execucao)
	}
	mu.Lock()
	if contador["abrir_ticket"] != 1 || contador["arquivar"] != 0 {
		t.Fatalf("o ramo verdadeiro precisa agir e o falso NÃO: %+v", contador)
	}
	mu.Unlock()

	// EVIDÊNCIA POR NÓ: cada nó com estado, tentativas e horários.
	if len(execucao.NodeEvidence) != 3 {
		t.Fatalf("precisa haver evidência dos três nós executados: %+v", execucao.NodeEvidence)
	}
	for _, ev := range execucao.NodeEvidence {
		if ev.State == "" || ev.StartedAt.IsZero() || ev.FinishedAt.IsZero() || ev.Attempts == 0 {
			t.Fatalf("evidência incompleta do nó %s: %+v", ev.NodeID, ev)
		}
	}

	// 2. CONDIÇÃO FALSA: o ramo é PULADO COM REGISTRO (não "sucesso por acidente").
	pulado, err := motor.Run(context.Background(), "org-a", fluxo.ID,
		map[string]string{"urgencia": "baixa", "contato": "+5511900000002"}, "evento-2")
	if err != nil {
		t.Fatalf("executar ramo falso: %v", err)
	}
	var evidenciaDecisao *NodeEvidence
	for i := range pulado.NodeEvidence {
		if pulado.NodeEvidence[i].NodeID == "decisao" {
			evidenciaDecisao = &pulado.NodeEvidence[i]
		}
	}
	if evidenciaDecisao == nil || evidenciaDecisao.State != "skipped" {
		t.Fatalf("condição falsa precisa ficar registrada como skipped: %+v", pulado.NodeEvidence)
	}
	mu.Lock()
	if contador["arquivar"] != 1 {
		t.Fatalf("o ramo falso precisa executar a própria ação: %+v", contador)
	}
	mu.Unlock()

	// 3. RETRY POR NÓ: a ação falha duas vezes e vence na terceira.
	mu.Lock()
	falhasRestantes["abrir_ticket"] = 2
	mu.Unlock()
	comRetry, err := motor.Run(context.Background(), "org-a", fluxo.ID,
		map[string]string{"urgencia": "alta", "contato": "+5511900000003"}, "evento-3")
	if err != nil {
		t.Fatalf("executar com retry: %v", err)
	}
	if comRetry.State != WorkflowRunSucceeded {
		t.Fatalf("com retry o nó precisa vencer: %+v", comRetry)
	}
	for _, ev := range comRetry.NodeEvidence {
		if ev.NodeID == "registrar" && ev.Attempts != 3 {
			t.Fatalf("a evidência precisa contar as TRÊS tentativas: %+v", ev)
		}
	}

	// 4. IDEMPOTÊNCIA COM ARMADILHA ARMADA: a mesma chave NÃO pode produzir segundo efeito.
	mu.Lock()
	antes := contador["abrir_ticket"]
	mu.Unlock()
	repetida, err := motor.Run(context.Background(), "org-a", fluxo.ID,
		map[string]string{"urgencia": "alta", "contato": "+5511900000001"}, "evento-1")
	if err != nil {
		t.Fatalf("repetir: %v", err)
	}
	if !repetida.Replayed || repetida.ID != execucao.ID {
		t.Fatalf("a mesma chave precisa devolver a MESMA execução: %+v", repetida)
	}
	mu.Lock()
	depois := contador["abrir_ticket"]
	mu.Unlock()
	if depois != antes {
		t.Fatalf("ARMADILHA: repetição com a mesma chave executou a ação de novo (%d -> %d)", antes, depois)
	}

	// 5. FALHA DEFINITIVA → DLQ, com nó e erro registrados.
	mu.Lock()
	falhasRestantes["abrir_ticket"] = 99
	mu.Unlock()
	morta, err := motor.Run(context.Background(), "org-a", fluxo.ID,
		map[string]string{"urgencia": "alta", "contato": "+5511900000004"}, "evento-4")
	if err != nil {
		t.Fatalf("executar com falha definitiva: %v", err)
	}
	if morta.State != WorkflowRunDeadLetter {
		t.Fatalf("falha definitiva precisa ir para a DLQ: %+v", morta)
	}
	dlq, err := motor.DeadLetters("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(dlq) != 1 || dlq[0].ID != morta.ID {
		t.Fatalf("a DLQ precisa conter a execução morta: %+v", dlq)
	}
	ultimo := dlq[0].NodeEvidence[len(dlq[0].NodeEvidence)-1]
	if ultimo.NodeID != "registrar" || ultimo.Error == "" || ultimo.Attempts != 3 {
		t.Fatalf("a DLQ precisa registrar nó, erro e tentativas: %+v", ultimo)
	}

	// 6. NÓ DESCONHECIDO: recusado no REGISTRO (não se declara sucesso ignorando nó).
	if _, err := motor.Register(WorkflowDefinition{OrganizationID: "org-a", Name: "ruim", StartNode: "x",
		Nodes: []WorkflowNode{{ID: "x", Kind: "loop_infinito"}}}); !errors.Is(err, ErrWorkflowUnknownNodeKind) {
		t.Fatalf("tipo de nó desconhecido precisa ser recusado no registro: %v", err)
	}
	// E também recusado na EXECUÇÃO quando a definição chega por outro caminho (importada/editada no disco).
	// Este é o caminho REAL: a validação de registro não é a única defesa.
	raizImportada := filepath.Join(t.TempDir(), "importado")
	importado := NewWorkflowEngine(raizImportada, executor)
	bruto := `{"workflows":[{"id":"wf_importado","organization_id":"org-a","name":"importado","version":1,` +
		`"start_node":"x","nodes":[{"id":"x","kind":"loop_infinito"}]}],"runs":[]}`
	if err := os.MkdirAll(raizImportada, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raizImportada, "org-a.json"), []byte(bruto), 0o600); err != nil {
		t.Fatal(err)
	}
	execucaoImportada, err := importado.Run(context.Background(), "org-a", "wf_importado",
		map[string]string{"urgencia": "alta"}, "importado-1")
	if err != nil {
		t.Fatalf("executar definição importada: %v", err)
	}
	if execucaoImportada.State != WorkflowRunDeadLetter {
		t.Fatalf("nó desconhecido NÃO pode ser ignorado: %+v", execucaoImportada)
	}
	achouErro := false
	for _, ev := range execucaoImportada.NodeEvidence {
		if strings.Contains(ev.Error, ErrWorkflowUnknownNodeKind.Error()) {
			achouErro = true
		}
	}
	if !achouErro {
		t.Fatalf("a evidência precisa dizer que o tipo do nó não é suportado: %+v", execucaoImportada.NodeEvidence)
	}

	// 7. PERSISTÊNCIA após reinício e ISOLAMENTO entre organizações.
	reaberto := NewWorkflowEngine(raiz, executor)
	execucoes, err := reaberto.Runs("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(execucoes) != 4 {
		t.Fatalf("após reinício a trilha precisa estar completa: %d", len(execucoes))
	}
	vizinha, err := reaberto.Runs("org-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(vizinha) != 0 {
		t.Fatalf("a organização vizinha precisa continuar vazia: %+v", vizinha)
	}
	_ = time.Now()
}
