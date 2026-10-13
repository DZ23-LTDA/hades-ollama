package agent

import (
	"context"
	"testing"
	"time"
)

// AUDITORIA ADVERSARIAL (V2.2 §12) — TESTES QUE ATACAM O PRÓPRIO CÓDIGO DAS FATIAS ANTERIORES (E-330).
//
// A auditoria profunda acontece **depois** das integrações, conforme a ordem exigida. Aqui eu tento **quebrar** o que
// acabei de escrever: laço infinito, oferta de serviço incompatível, contradição entre módulos. Cada achado vira
// correção + teste — *auditar sem atacar é revisar o espelho*.

// ACHADO 1 (candidato): um workflow com `Next` cíclico pode rodar para sempre. O V2.2 fala em "loops limitados" e
// proíbe loop infinito; o motor precisa LIMITAR e falhar alto, não girar.
func TestAuditoriaWorkflowCiclicoNaoGiraParaSempre(t *testing.T) {
	motor := NewWorkflowEngine("", func(_ context.Context, _ WorkflowDefinition, _ WorkflowNode, _ map[string]string) (string, error) {
		return "ok", nil
	})
	// A -> B -> A: ciclo explícito.
	ciclo, err := motor.Register(WorkflowDefinition{OrganizationID: "org-a", Name: "ciclo", StartNode: "a",
		Nodes: []WorkflowNode{
			{ID: "a", Kind: WorkflowNodeTrigger, Next: "b"},
			{ID: "b", Kind: WorkflowNodeAction, Next: "a", Config: map[string]string{"action": "x"}},
		}})
	if err != nil {
		t.Fatalf("registrar ciclo (a validação de forma não precisa recusar, mas a execução tem de limitar): %v", err)
	}

	feito := make(chan WorkflowRun, 1)
	go func() {
		execucao, _ := motor.Run(context.Background(), "org-a", ciclo.ID, map[string]string{}, "ciclo-1")
		feito <- execucao
	}()
	select {
	case execucao := <-feito:
		// Precisa TERMINAR e dizer que falhou por limite — não girar em silêncio.
		if execucao.State != WorkflowRunDeadLetter {
			t.Fatalf("workflow cíclico precisa terminar em falha explícita (dead_letter), veio %q", execucao.State)
		}
		if len(execucao.NodeEvidence) == 0 {
			t.Fatal("a falha por ciclo precisa de evidência dos nós executados")
		}
		ultimo := execucao.NodeEvidence[len(execucao.NodeEvidence)-1]
		if ultimo.Error == "" {
			t.Fatalf("a evidência precisa dizer QUE houve ciclo: %+v", ultimo)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ACHADO CONFIRMADO: o motor gira para sempre num workflow cíclico (loops precisam ser LIMITADOS)")
	}
}

// ACHADO 2 (candidato): a fila de espera casa por RECURSO, sem olhar o SERVIÇO. Se a entrada pede um serviço de 2h e o
// horário liberado comporta 30min, a oferta seria feita para um atendimento que não cabe.
func TestAuditoriaFilaNaoOfereceHorarioQueNaoCabeNoServicoPedido(t *testing.T) {
	store, org, recurso, _ := schedulingFixture(t) // expediente 24/7
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	// Serviço LONGO (2h) — a entrada da fila pede este serviço.
	longo, err := store.AddService(SchedulingService{OrganizationID: org, Name: "Avaliação longa", DurationMinutes: 120})
	if err != nil {
		t.Fatal(err)
	}
	slot := time.Date(2026, 12, 14, 9, 0, 0, 0, loc)

	// O recurso já tem um agendamento que deixa apenas 30min livres antes do próximo compromisso.
	curto, err := store.AddService(SchedulingService{OrganizationID: org, Name: "Corte", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: curto.ID,
		StartsAt: slot.Add(30 * time.Minute), Channel: "voice", IdempotencyKey: "ocupado-1"}); err != nil {
		t.Fatal(err)
	}

	entrada, err := store.AddToWaitlist(SchedulingWaitlistEntry{OrganizationID: org, ResourceID: recurso,
		ServiceID: longo.ID, ContactID: "+5511900000001", Channel: "whatsapp",
		From: slot.Add(-time.Hour), To: slot.Add(6 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	oferta, achou, err := store.OfferNextFromWaitlist(org, recurso, slot, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	// ASSERTIVA ATUALIZADA PELA SEGUNDA ONDA (E-335): na primeira onda a fila OFERECIA e a confirmação recusava — a
	// defesa existia, mas o cliente recebia oferta de horário que não caberia. Depois da correção, a propriedade é mais
	// FORTE: **a oferta nem acontece**. *Corrigir a causa-raiz torna obsoleta a expectativa antiga — e o registro diz
	// por quê.*
	if achou {
		t.Fatalf("a fila NÃO pode oferecer horário que não cabe no serviço pedido (oferta %+v)", oferta)
	}
	// E a entrada continua aguardando, em vez de virar estado terminal.
	fila, err := store.Waitlist(org)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range fila {
		if item.ID == entrada.ID && item.Status != WaitlistWaiting {
			t.Fatalf("a entrada precisa continuar aguardando: %+v", item)
		}
	}
	_ = entrada
}
