package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 31 — LEASE do supervisor (E-261). O limite declarado dizia "sem loop proprio com lease".
//
// Por que e' SEGURANCA: o supervisor executa acoes com EFEITO REAL. Duas instancias rodando o ciclo ao mesmo tempo
// podem agir duas vezes sobre o mesmo estado. O lease garante UM titular por vez e a ASSUNCAO apos a expiracao
// mantem a recuperacao.
func TestSupervisorLeaseImpedeCicloDuploEPermiteAssuncao(t *testing.T) {
	runtime, dataRoot := persistentSupervisorRuntime(t)
	dir := filepath.Join(dataRoot, "business-supervisor")
	agora := time.Now().UTC()
	organizacao := LocalOrganizationID

	primeiro, err := NewBusinessSupervisorAt(dir, runtime, SupervisorPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := NewBusinessSupervisorAt(dir, runtime, SupervisorPolicy{})
	if err != nil {
		t.Fatal(err)
	}

	// 1. O primeiro toma o lease e roda.
	if _, err := primeiro.RunSupervisedTick("instancia-1", organizacao, agora, time.Minute); err != nil {
		t.Fatalf("o titular do lease precisa rodar o ciclo: %v", err)
	}
	// 2. O segundo NAO roda enquanto o lease do primeiro esta vivo.
	if _, err := segundo.RunSupervisedTick("instancia-2", organizacao, agora.Add(time.Second), time.Minute); !errors.Is(err, ErrSupervisorLeaseHeld) {
		t.Fatalf("a segunda instancia NAO pode rodar o ciclo com lease vivo: %v", err)
	}
	// 3. O titular RENOVA o proprio lease.
	if ok, err := primeiro.AcquireSupervisorLease("instancia-1", agora.Add(2*time.Second), time.Minute); err != nil || !ok {
		t.Fatalf("o titular precisa renovar o proprio lease: ok=%v err=%v", ok, err)
	}
	// 4. Depois do TTL, a segunda instancia ASSUME (recuperacao: instancia morta nao trava a operacao).
	if _, err := segundo.RunSupervisedTick("instancia-2", organizacao, agora.Add(2*time.Minute), time.Minute); err != nil {
		t.Fatalf("apos a expiracao a segunda instancia precisa assumir: %v", err)
	}
	// 5. Quem NAO e' titular nao libera o lease.
	if err := primeiro.ReleaseSupervisorLease("instancia-1"); err != nil {
		t.Fatalf("liberar lease alheio nao pode falhar: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "supervisor.lease")); err != nil {
		t.Fatalf("quem nao e' titular NAO pode remover o lease: %v", err)
	}
	// 6. O titular libera.
	if err := segundo.ReleaseSupervisorLease("instancia-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "supervisor.lease")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("o titular precisa conseguir liberar o lease: %v", err)
	}
}

// Modo em memoria: um unico processo, um unico supervisor - o lease concede e isso fica DECLARADO, nao fingido.
func TestSupervisorEmMemoriaConcedeLease(t *testing.T) {
	runtime, _ := persistentSupervisorRuntime(t)
	supervisor := NewBusinessSupervisor(runtime, SupervisorPolicy{})
	if ok, err := supervisor.AcquireSupervisorLease("qualquer", time.Now().UTC(), time.Minute); err != nil || !ok {
		t.Fatalf("modo em memoria precisa conceder o lease: ok=%v err=%v", ok, err)
	}
	if _, err := supervisor.RunSupervisedTick("qualquer", LocalOrganizationID, time.Now().UTC(), time.Minute); err != nil {
		t.Fatalf("modo em memoria precisa rodar o ciclo: %v", err)
	}
	// Holder vazio e' erro: sem titular nao ha de quem ser o lease.
	if _, err := supervisor.AcquireSupervisorLease("   ", time.Now().UTC(), time.Minute); err == nil {
		t.Fatal("lease sem titular precisa ser recusado")
	}
}
