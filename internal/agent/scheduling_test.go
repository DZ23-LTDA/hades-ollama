package agent

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// INCREMENTO I (V2.2) — AGENDAMENTO: CONCORRÊNCIA, BUFFER, IDEMPOTÊNCIA, CANCELAMENTO, REINÍCIO E ISOLAMENTO (E-313).
//
// O V2.2 exige E2E com "dois agentes concorrendo pelo último horário". Se ambos conseguirem, o cliente recebe uma
// confirmação e chega num horário já ocupado: o pior desfecho de um sistema de agenda.
func TestAgendamentoImpedeDoubleBookingSobConcorrencia(t *testing.T) {
	store, org, recurso, servico := schedulingFixture(t)
	agora := time.Now().UTC().Truncate(time.Minute).Add(24 * time.Hour)

	// DOIS CANAIS DIFERENTES (voz e WhatsApp) pedindo o MESMO horário ao mesmo tempo.
	pedidos := []BookRequest{
		{OrganizationID: org, ResourceID: recurso, ServiceID: servico, ContactID: "+5511911112222", Channel: "voice", StartsAt: agora, IdempotencyKey: "voz-1"},
		{OrganizationID: org, ResourceID: recurso, ServiceID: servico, ContactID: "+5511933334444", Channel: "whatsapp", StartsAt: agora, IdempotencyKey: "zap-1"},
	}
	resultados := make([]error, len(pedidos))
	var wg sync.WaitGroup
	for i, pedido := range pedidos {
		wg.Add(1)
		go func(indice int, p BookRequest) {
			defer wg.Done()
			_, err := store.Book(p)
			resultados[indice] = err
		}(i, pedido)
	}
	wg.Wait()

	sucessos, recusas := 0, 0
	for _, err := range resultados {
		switch {
		case err == nil:
			sucessos++
		case errors.Is(err, ErrSchedulingSlotTaken):
			recusas++
		default:
			t.Fatalf("recusa inesperada: %v", err)
		}
	}
	if sucessos != 1 || recusas != 1 {
		t.Fatalf("o último horário precisa ter UM ganhador e UMA recusa explícita: sucessos=%d recusas=%d", sucessos, recusas)
	}
	agendamentos, err := store.Appointments(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(agendamentos) != 1 {
		t.Fatalf("só pode existir UM agendamento no horário: %+v", agendamentos)
	}

	// BUFFER: o serviço tem 30min + 10min de preparo. Um atendimento que comece dentro do preparo é recusado.
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511955556666", Channel: "web", StartsAt: agora.Add(15 * time.Minute)}); !errors.Is(err, ErrSchedulingSlotTaken) {
		t.Fatalf("início dentro do buffer precisa ser recusado: %v", err)
	}
	// Depois do buffer, o recurso volta a aceitar.
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511955556666", Channel: "web", StartsAt: agora.Add(40 * time.Minute)}); err != nil {
		t.Fatalf("depois do buffer o horário precisa estar livre: %v", err)
	}

	// IDEMPOTÊNCIA: reentrega do mesmo canal com a mesma chave não cria segundo agendamento (recusa explícita).
	// MEDIDO: o vencedor da corrida é NÃO-DETERMINÍSTICO por desenho — então a idempotência é provada num horário
	// próprio, para a asserção medir a propriedade e não a sorte do escalonador.
	livre := agora.Add(90 * time.Minute)
	repetivel := BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511900001111", Channel: "whatsapp", StartsAt: livre, IdempotencyKey: "replay-1"}
	if _, err := store.Book(repetivel); err != nil {
		t.Fatalf("agendamento base para o teste de idempotência: %v", err)
	}
	if _, err := store.Book(repetivel); !errors.Is(err, ErrSchedulingDuplicate) {
		t.Fatalf("reentrega com a mesma chave precisa ser recusada explicitamente: %v", err)
	}

	// CANCELAMENTO liberta o horário.
	cancelado, err := store.Cancel(org, agendamentos[0].ID, "cliente desistiu")
	if err != nil {
		t.Fatal(err)
	}
	if cancelado.Status != SchedulingAppointmentCancelled {
		t.Fatalf("o cancelamento precisa ficar registrado: %+v", cancelado)
	}
	if _, err := store.Book(BookRequest{OrganizationID: org, ResourceID: recurso, ServiceID: servico,
		ContactID: "+5511977778888", Channel: "voice", StartsAt: agora}); err != nil {
		t.Fatalf("horário cancelado precisa voltar a ser agendável: %v", err)
	}

	// REINÍCIO: store NOVO sobre a MESMA raiz enxerga a agenda (fonte da verdade é o disco, não a memória).
	// MEDIDO: contar com aritmética fixa foi um erro meu (a agenda já tinha 4 itens quando o reinício aconteceu).
	// A asserção correta compara com o que o store ORIGINAL tinha no mesmo instante — é a propriedade, não a conta.
	antesDoReinicio, err := store.Appointments(org)
	if err != nil {
		t.Fatal(err)
	}
	reaberto := NewSchedulingStore(filepath.Join(filepath.Dir(store.Root()), "scheduling"))
	aposReinicio, err := reaberto.Appointments(org)
	if err != nil {
		t.Fatal(err)
	}
	if len(aposReinicio) != len(antesDoReinicio) {
		t.Fatalf("após reinício a agenda precisa ser a MESMA: antes=%d depois=%d", len(antesDoReinicio), len(aposReinicio))
	}

	// ISOLAMENTO: outra organização não vê nem ocupa o recurso da primeira.
	slots, err := store.Availability(org, recurso, agora, agora.Add(2*time.Hour), 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("a disponibilidade precisa derivar dos agendamentos persistidos")
	}
	outraOrg := "org-b"
	if _, err := store.Book(BookRequest{OrganizationID: outraOrg, ResourceID: recurso, ServiceID: servico,
		StartsAt: agora, Channel: "web"}); err == nil {
		t.Fatal("outra organização não pode agendar no recurso de quem não é dela")
	}
	daOutra, err := store.Appointments(outraOrg)
	if err != nil {
		t.Fatal(err)
	}
	if len(daOutra) != 0 {
		t.Fatalf("a organização vizinha precisa continuar vazia: %+v", daOutra)
	}
}

// schedulingFixture monta serviço (30min + 10min de buffer) e recurso numa organização isolada.
func schedulingFixture(t *testing.T) (*SchedulingStore, string, string, string) {
	t.Helper()
	raiz := filepath.Join(t.TempDir(), "scheduling")
	store := NewSchedulingStore(raiz)
	const org = "org-a"
	servico, err := store.AddService(SchedulingService{OrganizationID: org, Name: "Corte", DurationMinutes: 30, BufferMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := store.AddResource(SchedulingResource{OrganizationID: org, Name: "Profissional Ana"})
	if err != nil {
		t.Fatal(err)
	}
	return store, org, recurso.ID, servico.ID
}
