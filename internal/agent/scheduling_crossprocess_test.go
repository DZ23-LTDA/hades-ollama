package agent

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// AGENDAMENTO — FATIA 4: CORREÇÃO ENTRE PROCESSOS (E-323).
//
// As fatias anteriores protegiam contra concorrência **dentro do mesmo processo**. Aqui a pergunta é outra: dois
// processos (duas instâncias do servidor) sobre o MESMO diretório de dados. Cada um tem o seu cache em memória e a sua
// trava — ***um cliente confirmado em horário já ocupado por OUTRO processo é o mesmo defeito de double booking***.
//
// APROXIMAÇÃO DECLARADA: dois `SchedulingStore` independentes sobre a mesma raiz exercitam o mesmo caminho crítico
// (trava de arquivo + releitura do disco) que dois processos. Não é um teste com `fork`; é o teste do mecanismo.
//
// A PROVA NUNCA VEM DO CACHE DE QUEM ESCREVEU: as asserções de persistência são feitas em um store **novo**.

func TestAgendamentoNaoDuplicaEntreInstanciasIndependentes(t *testing.T) {
	raiz := t.TempDir() + "/scheduling"
	a := NewSchedulingStore(raiz)
	servico, err := a.AddService(SchedulingService{OrganizationID: "org-a", Name: "Corte", DurationMinutes: 30, BufferMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := a.AddResource(SchedulingResource{OrganizationID: "org-a", Name: "Profissional Ana"})
	if err != nil {
		t.Fatal(err)
	}
	for dia := 0; dia <= 6; dia++ {
		if _, err := a.AddShift(SchedulingShift{OrganizationID: "org-a", Weekday: dia, OpensAt: 0, ClosesAt: 24 * 60}); err != nil {
			t.Fatal(err)
		}
	}
	// Segunda INSTÂNCIA, sem qualquer estado em comum além do diretório.
	b := NewSchedulingStore(raiz)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	alvo := time.Date(2026, 12, 7, 9, 0, 0, 0, loc)

	// 1. DUAS INSTÂNCIAS pedindo o MESMO horário ao mesmo tempo.
	pedido := BookRequest{OrganizationID: "org-a", ResourceID: recurso.ID, ServiceID: servico.ID,
		ContactID: "+5511911112222", Channel: "voice", StartsAt: alvo, IdempotencyKey: "inst-1"}
	erros := make([]error, 2)
	var wg sync.WaitGroup
	for i, loja := range []*SchedulingStore{a, b} {
		wg.Add(1)
		go func(indice int, s *SchedulingStore) {
			defer wg.Done()
			_, err := s.Book(pedido)
			erros[indice] = err
		}(i, loja)
	}
	wg.Wait()
	sucessos, recusas := 0, 0
	for _, err := range erros {
		switch {
		case err == nil:
			sucessos++
		case errors.Is(err, ErrSchedulingSlotTaken), errors.Is(err, ErrSchedulingDuplicate):
			recusas++
		default:
			t.Fatalf("recusa inesperada entre instâncias: %v", err)
		}
	}
	if sucessos != 1 || recusas != 1 {
		t.Fatalf("entre instâncias o último horário precisa ter UM ganhador e UMA recusa: sucessos=%d recusas=%d", sucessos, recusas)
	}

	// A prova do que ficou gravado vem de um store NOVO (nem A nem B): cache não é evidência.
	c := NewSchedulingStore(raiz)
	gravados, err := c.Appointments("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(gravados) != 1 {
		t.Fatalf("o disco precisa ter UM agendamento, não %d", len(gravados))
	}

	// 2. SEQUENCIAL ENTRE INSTÂNCIAS: B precisa RECUSAR o horário que A ocupou (releitura sob a trava).
	if _, err := b.Book(BookRequest{OrganizationID: "org-a", ResourceID: recurso.ID, ServiceID: servico.ID,
		ContactID: "+5511933334444", Channel: "whatsapp", StartsAt: alvo, IdempotencyKey: "inst-2"}); err == nil {
		t.Fatal("a segunda instância NÃO pode confirmar horário já ocupado pela primeira")
	}

	// 3. CANCELAMENTO entre instâncias: B cancela o que A criou, e o horário volta a ser agendável.
	if _, err := b.Cancel("org-a", gravados[0].ID, "cliente desmarcou"); err != nil {
		t.Fatalf("cancelar entre instâncias: %v", err)
	}
	depois, err := NewSchedulingStore(raiz).Appointments("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if depois[0].Status != SchedulingAppointmentCancelled {
		t.Fatalf("o cancelamento entre instâncias precisa persistir: %+v", depois[0])
	}
	if _, err := b.Book(BookRequest{OrganizationID: "org-a", ResourceID: recurso.ID, ServiceID: servico.ID,
		ContactID: "+5511955556666", Channel: "web", StartsAt: alvo, IdempotencyKey: "inst-3"}); err != nil {
		t.Fatalf("horário cancelado precisa voltar a ser agendável: %v", err)
	}

	// 4. A TRAVA É POR ORGANIZAÇÃO, não global: outra organização não fica bloqueada nem vê o horário alheio.
	outra := NewSchedulingStore(raiz)
	servicoB, err := outra.AddService(SchedulingService{OrganizationID: "org-b", Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	recursoB, err := outra.AddResource(SchedulingResource{OrganizationID: "org-b", Name: "Profissional Bia"})
	if err != nil {
		t.Fatal(err)
	}
	for dia := 0; dia <= 6; dia++ {
		if _, err := outra.AddShift(SchedulingShift{OrganizationID: "org-b", Weekday: dia, OpensAt: 0, ClosesAt: 24 * 60}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := outra.Book(BookRequest{OrganizationID: "org-b", ResourceID: recursoB.ID, ServiceID: servicoB.ID,
		StartsAt: alvo, Channel: "web", IdempotencyKey: "orgb-1"}); err != nil {
		t.Fatalf("a organização vizinha não pode ser bloqueada pela trava da outra: %v", err)
	}
	daVizinha, err := NewSchedulingStore(raiz).Appointments("org-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(daVizinha) != 1 {
		t.Fatalf("cada organização tem a sua agenda: %d", len(daVizinha))
	}
}
