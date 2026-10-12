package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estágio 31 — AGENDADOR PRÓPRIO DO SUPERVISOR (E-274). O limite declarado dizia que "o LOOP com agendador próprio
// continua ausente".
//
// As quatro propriedades que um agendador errado viola:
//  1. rodar ANTES da hora;
//  2. pular um intervalo em silêncio quando o lease está com outra instância (o relógio avançar sem o ciclo rodar);
//  3. rodar N ciclos atrasados de uma vez depois de uma queda (tempestade de efeitos);
//  4. esquecer o último disparo e reexecutar o ciclo ao reiniciar.
//
// NOTA DE ORDEM: o próprio ciclo MANTÉM o lease enquanto roda, então o bloqueio por OUTRA instância precisa ser
// montado ANTES do primeiro ciclo. Testar depois seria testar uma premissa falsa.
func TestAgendadorDoSupervisorRespeitaIntervaloLeaseECaptura(t *testing.T) {
	root := t.TempDir()
	// Runtime COMPLETO: um &Runtime{} vazio panica dentro do ciclo (mapas nulos). O fixture abaixo e' o mesmo que os
	// testes existentes do supervisor usam.
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	supervisor, err := NewBusinessSupervisorAt(filepath.Join(root, "sup"), runtime, DefaultSupervisorPolicy())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	agendador, err := NewSupervisorScheduler(filepath.Join(root, "sched"), supervisor, LocalOrganizationID, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	// 1. LEASE com outra instância ANTES do primeiro ciclo: não roda E o relógio NÃO avança.
	ocupado, err := supervisor.AcquireSupervisorLease("instancia-2", agora, 10*time.Minute)
	if err != nil || !ocupado {
		t.Fatalf("preparar lease da outra instância: ocupado=%v err=%v", ocupado, err)
	}
	if _, rodou, err := agendador.RunDue("instancia-1", agora); err != nil || rodou {
		t.Fatalf("com o lease de outra instância o ciclo não pode rodar: rodou=%v err=%v", rodou, err)
	}
	if agendador.Runs() != 0 {
		t.Fatalf("tentativa bloqueada não pode contar ciclo: %d", agendador.Runs())
	}
	if !agendador.Due(agora) {
		t.Fatal("o relógio NÃO pode avançar quando o ciclo não rodou: o intervalo seria pulado em silêncio")
	}

	// 2. Liberado o lease, o MESMO instante ainda está vencido: o ciclo acontece (não foi perdido).
	if err := supervisor.ReleaseSupervisorLease("instancia-2"); err != nil {
		t.Fatal(err)
	}
	resultado, rodou, err := agendador.RunDue("instancia-1", agora)
	if err != nil || !rodou {
		t.Fatalf("após liberar o lease o ciclo precisa rodar: rodou=%v err=%v", rodou, err)
	}
	if resultado.TickNumber == 0 {
		t.Fatalf("o primeiro ciclo precisa numerar o tick: %+v", resultado)
	}
	primeiroTick := resultado.TickNumber
	if agendador.Runs() != 1 {
		t.Fatalf("contagem de ciclos: %d", agendador.Runs())
	}

	// 3. ANTES da hora não roda de novo.
	if _, rodou, _ = agendador.RunDue("instancia-1", agora.Add(time.Minute)); rodou {
		t.Fatal("não pode rodar antes do intervalo")
	}

	// 4. CATCH-UP COLAPSADO: 10 intervalos perdidos => UM ciclo, e o próximo não vence na hora.
	atrasado := agora.Add(10 * 30 * time.Minute)
	resultado, rodou, err = agendador.RunDue("instancia-1", atrasado)
	if err != nil || !rodou {
		t.Fatalf("depois da queda o ciclo precisa rodar UMA vez: rodou=%v err=%v", rodou, err)
	}
	if resultado.TickNumber <= primeiroTick {
		t.Fatalf("o tick precisa avançar: %d -> %d", primeiroTick, resultado.TickNumber)
	}
	if _, rodou, _ = agendador.RunDue("instancia-1", atrasado.Add(time.Second)); rodou {
		t.Fatal("catch-up colapsado: não pode rodar de novo no mesmo instante")
	}
	if agendador.Runs() != 2 {
		t.Fatalf("dois disparos esperados, veio %d", agendador.Runs())
	}

	// 5. REINÍCIO: o relógio é persistido e o intervalo não é reexecutado.
	reaberto, err := NewSupervisorScheduler(filepath.Join(root, "sched"), supervisor, LocalOrganizationID, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if reaberto.Due(atrasado.Add(time.Minute)) {
		t.Fatal("após reiniciar, o ciclo não pode vencer antes do intervalo")
	}
	if !reaberto.Due(atrasado.Add(31 * time.Minute)) {
		t.Fatal("após reiniciar, o intervalo seguinte precisa vencer normalmente")
	}

	// 6. Intervalo inválido é recusado na construção.
	if _, err := NewSupervisorScheduler(filepath.Join(root, "sched"), supervisor, LocalOrganizationID, -time.Second); err == nil {
		t.Fatal("intervalo negativo precisa ser recusado")
	}
	if _, err := NewSupervisorScheduler(filepath.Join(root, "sched"), supervisor, "", time.Minute); err == nil {
		t.Fatal("organização vazia precisa ser recusada")
	}
	if _, err := NewSupervisorScheduler(filepath.Join(root, "sched"), nil, LocalOrganizationID, time.Minute); err == nil {
		t.Fatal("sem supervisor precisa ser recusado")
	}
}
