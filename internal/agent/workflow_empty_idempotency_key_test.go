package agent

import (
	"context"
	"testing"
)

// E-485 — **CHAVE DE IDEMPOTÊNCIA VAZIA: NÃO DEDUPLICA. É DEFEITO?**
//
// **O LIMITE QUE EU DECLAREI NO E-461**: *«chaves vazias continuam a não deduplicar (comportamento do motor, **não medido
// aqui**)»* ✗ — *eu escrevi o limite e segui em frente, sem medir o que ele significava.*
//
// **E A PERGUNTA É BINÁRIA, com consequência**: *duas execuções do mesmo workflow **sem chave** criam **duas execuções**?*
//
//   - Se **criam**: é o **comportamento certo** ⇒ *«sem chave» significa «não pedi idempotência», e inventar uma chave
//     (o hash do evento, por exemplo) tornaria **todo** evento deduplicado — o que quebraria a reexecução legítima.*
//   - Se **não criam**: o motor estaria a deduplicar por acidente, e *a segunda execução desapareceria em silêncio* —
//     ***que é exactamente a família do ADV-77.***
//
// ***Meço para saber QUAL das duas coisas o motor faz — e a diferença entre elas é a diferença entre um contrato e um
// acidente.***
func TestChaveDeIdempotenciaVaziaNoMotorDeWorkflow(t *testing.T) {
	motor := NewWorkflowEngine(t.TempDir(), func(_ context.Context, _ WorkflowDefinition, _ WorkflowNode, _ map[string]string) (string, error) {
		return "ok", nil
	})
	definicao, err := motor.Register(WorkflowDefinition{
		OrganizationID: "org-a", Name: "sem-chave", StartNode: "gatilho",
		Nodes: []WorkflowNode{
			{ID: "gatilho", Kind: WorkflowNodeTrigger, Next: "acao"},
			{ID: "acao", Kind: WorkflowNodeAction},
		},
	})
	if err != nil {
		t.Fatalf("registar: %v", err)
	}
	evento := map[string]string{"evento": "evt-sem-chave"}

	// (1) Duas execuções SEM chave.
	primeira, err := motor.Run(context.Background(), "org-a", definicao.ID, evento, "")
	if err != nil {
		t.Fatalf("primeira execução: %v", err)
	}
	segunda, err := motor.Run(context.Background(), "org-a", definicao.ID, evento, "")
	if err != nil {
		t.Fatalf("segunda execução: %v", err)
	}

	// A MEDIDA.
	if segunda.ID == primeira.ID {
		// Deduplicou por acidente - *a familia do ADV-77: a segunda execução desaparece em silêncio.*
		t.Fatalf("MEDIDO — O MOTOR DEDUPLICA CHAVE VAZIA: a segunda execução devolveu o MESMO run (%q), e `Replayed=%v` "+
			"⇒ ***duas execuções sem chave viraram uma, em silêncio. É a família do ADV-77*** (o efeito que desaparece "+
			"sem erro). *Se isto fosse intencional, teria de estar escrito - e não está.*", segunda.ID, segunda.Replayed)
	}
	// Dois runs distintos: o comportamento é «sem chave ⇒ sem deduplicação».
	if segunda.Replayed {
		t.Fatalf("a segunda execução, com chave VAZIA, veio marcada como `Replayed=%v` — *replay sem chave não faz "+
			"sentido: não havia intenção a repetir*", segunda.Replayed)
	}
	execucoes, err := motor.Runs("org-a")
	if err != nil {
		t.Fatalf("listar execuções: %v", err)
	}
	if len(execucoes) != 2 {
		t.Fatalf("esperava 2 execuções para 2 chamadas sem chave, veio %d — *«sem chave» tem de significar «não pedi "+
			"idempotência», senão a reexecução legítima seria engolida*", len(execucoes))
	}

	// (2) E O CONTROLE, que separa os dois comportamentos: *com chave, a segunda NÃO cria.*
	comChave, err := motor.Run(context.Background(), "org-a", definicao.ID, evento, "k-medida")
	if err != nil {
		t.Fatalf("primeira com chave: %v", err)
	}
	repetida, err := motor.Run(context.Background(), "org-a", definicao.ID, evento, "k-medida")
	if err != nil {
		t.Fatalf("segunda com chave: %v", err)
	}
	if repetida.ID != comChave.ID || !repetida.Replayed {
		t.Fatalf("com chave, a repetição tem de devolver o MESMO run com `Replayed`: id=%q vs %q, replayed=%v — *sem "+
			"este controle, o teste acima poderia estar a medir um motor que NUNCA deduplica*",
			repetida.ID, comChave.ID, repetida.Replayed)
	}
	if final, err := motor.Runs("org-a"); err != nil || len(final) != 3 {
		t.Fatalf("esperava 3 execuções no total (2 sem chave + 1 com chave), veio %d (err=%v) — *a chave deduplica, a "+
			"ausência de chave não: a distinção e' o contrato*", len(final), err)
	}
}
