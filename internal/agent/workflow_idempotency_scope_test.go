package agent

import (
	"context"
	"testing"
)

// E-461 — **A GENERALIZACAO DO ADV-75: A MESMA CHAVE EM WORKFLOWS DIFERENTES.**
//
// **O PADRAO, medido em E-460**: *comparar um identificador **sem o escopo a que ele pertence***. No agendamento o escopo
// que faltava era o **contacto** (*duas pessoas colidiam*). Aqui o escopo **de tenant** está garantido (`org.Runs` é por
// organização ✓), mas há **outro**: **o WORKFLOW**.
//
// **POR QUE A SUSPEITA É FUNDADA, e não um palpite**: em `workflow_engine.go:268-276` a chave é procurada **antes** de o
// workflow ser resolvido (`:278-287`, onde `workflowID` é usado pela primeira vez) ⇒ *`Run(org, wf-A, …, K)` e
// `Run(org, wf-B, …, K)` passam pelo **mesmo laço**, e o segundo encontra o run do primeiro* ✗.
//
// **E A CONSEQUÊNCIA**: *a chave costuma vir do **evento** (id de mensagem, id de transação) e **não** do workflow* ⇒
// dois workflows reagindo ao **mesmo evento** — *o caso mais natural de todos* — fariam o segundo **não executar**, e
// ainda devolver o run do primeiro como se fosse dele.
//
// ***A pergunta é binária: ou o escopo é o workflow, ou o segundo workflow nunca roda.***
func TestMesmaChaveEmWorkflowsDiferentes(t *testing.T) {
	motor := NewWorkflowEngine(t.TempDir(), func(_ context.Context, _ WorkflowDefinition, _ WorkflowNode, _ map[string]string) (string, error) {
		return "ok", nil
	})

	registar := func(nome string) WorkflowDefinition {
		// ⚠️ Os `Kind` são VALIDADOS pelo motor (`ErrWorkflowUnknownNodeKind`): *nada de pular em silêncio*. Eu tentei
		// `"noop"` e o motor recusou - **com razão**. Os válidos, medidos: `WorkflowNodeTrigger`, `WorkflowNodeCondition`,
		// `WorkflowNodeAction`. *A recusa do motor é a mesma classe de tudo que esta missão vem registando: o produto
		// recusou o que eu não medi.*
		definicao, err := motor.Register(WorkflowDefinition{
			OrganizationID: "org-a", Name: nome, StartNode: "gatilho",
			Nodes: []WorkflowNode{
				{ID: "gatilho", Kind: WorkflowNodeTrigger, Next: "acao"},
				{ID: "acao", Kind: WorkflowNodeAction},
			},
		})
		if err != nil {
			t.Fatalf("registar %s: %v", nome, err)
		}
		return definicao
	}
	wfA := registar("primeiro")
	wfB := registar("segundo")

	// O `Register` devolve a definicao; o ID do workflow é o que `Run` recebe.
	primeiro, err := motor.Run(context.Background(), "org-a", wfA.ID, map[string]string{"evento": "evt-1"}, "K-compartilhada")
	if err != nil {
		t.Fatalf("wf-A: %v", err)
	}

	// wf-B roda com a MESMA chave. *Se o escopo for o workflow, isto é uma execução NOVA.*
	segundo, err := motor.Run(context.Background(), "org-a", wfB.ID, map[string]string{"evento": "evt-1"}, "K-compartilhada")
	if err != nil {
		t.Fatalf("wf-B: %v", err)
	}

	// A MEDIDA: a que workflow pertence o run devolvido a wf-B?
	if segundo.WorkflowID != wfB.ID {
		t.Fatalf("MEDIDO: A CHAVE NAO TEM O WORKFLOW COMO ESCOPO. `%s` com a chave de `%s` devolveu um run do workflow "+
			"%q (`Replayed=%v`). ***A chave e' procurada ANTES de o workflow ser resolvido (workflow_engine.go:268 vs "+
			":278) - e a chave costuma vir do EVENTO, nao do workflow: dois workflows reagindo ao MESMO evento fariam o "+
			"segundo NAO EXECUTAR.***", wfB.ID, wfA.ID, segundo.WorkflowID, segundo.Replayed)
	}
	if segundo.ID == primeiro.ID {
		t.Fatalf("MEDIDO: `%s` recebeu o MESMO run de `%s` (%q) — *nao e' so' pular: e' devolver o resultado do outro "+
			"workflow como se fosse deste*", wfB.ID, wfA.ID, segundo.ID)
	}
	if segundo.Replayed {
		t.Fatalf("MEDIDO: `%s` foi marcado como REPLAY do `%s` — *o escopo da chave nao inclui o workflow*", wfB.ID, wfA.ID)
	}

	// CONTROLE: a idempotência DENTRO do mesmo workflow continua valendo? *Sem isto, o teste poderia estar a medir um
	// motor que simplesmente nunca deduplica - que e' um defeito diferente.*
	repetido, err := motor.Run(context.Background(), "org-a", wfA.ID, map[string]string{"evento": "evt-1"}, "K-compartilhada")
	if err != nil {
		t.Fatalf("repeticao do wf-A: %v", err)
	}
	if repetido.ID != primeiro.ID || !repetido.Replayed {
		t.Fatalf("a repeticao DENTRO do mesmo workflow tinha de devolver o mesmo run marcado como Replayed "+
			"(id=%q vs %q, replayed=%v) — *a idempotencia existe e tem de continuar valendo*",
			repetido.ID, primeiro.ID, repetido.Replayed)
	}
}
