package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Evidência obrigatória do estágio 31: "logs de supervisão".
//
// Os contadores viviam só em memória, então ao reiniciar a operação perdia a
// resposta para "o que o supervisor fez nas últimas horas?". O diário guarda os
// últimos ticks de forma limitada e persistida — INCLUINDO os erros, porque um
// diário que só guarda sucesso não serve para investigar incidente.
func TestTickJournalRecordsEveryTickWithErrors(t *testing.T) {
	supervisor, _ := incidentSupervisor(t)
	now := time.Now().UTC()
	for index := range 3 {
		if _, err := supervisor.Tick(t.Context(), now.Add(time.Duration(index)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	journal := supervisor.TickJournal()
	if len(journal) != 3 {
		t.Fatalf("cada tick precisa virar uma linha: %d", len(journal))
	}
	// Ordem: do mais recente para o mais antigo, e numeração crescente.
	if journal[0].TickNumber <= journal[len(journal)-1].TickNumber {
		t.Fatalf("o diário precisa vir do mais recente para o mais antigo: %+v", journal)
	}
	for _, record := range journal {
		if record.ExecutedAt.IsZero() {
			t.Fatalf("a linha precisa registrar quando o tick rodou: %+v", record)
		}
		if record.ErrorCount != len(record.Errors) {
			t.Fatalf("a contagem de erro precisa bater com a lista: %+v", record)
		}
	}
}

func TestTickJournalSurvivesRestartAndIsBounded(t *testing.T) {
	root := t.TempDir()
	store := failingStore{JSONStore: NewMemoryStore()}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Planner:       RulePlanner{},
		WorkspaceRoot: root + "-ws",
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	supervisor := NewSupervisor(runtime, SupervisorConfig{Enabled: true})
	now := time.Now().UTC()
	for index := range MaxSupervisorTickJournal + 5 {
		if _, err := supervisor.Tick(t.Context(), now.Add(time.Duration(index)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	journal := supervisor.TickJournal()
	if len(journal) > MaxSupervisorTickJournal {
		t.Fatalf("o diário precisa respeitar o teto de %d: %d", MaxSupervisorTickJournal, len(journal))
	}
	// A falha de retomada precisa estar registrada no diário, com o erro visível.
	foundError := false
	for _, record := range journal {
		if record.ErrorCount > 0 && len(record.Errors) > 0 {
			foundError = true
		}
	}
	if !foundError {
		t.Fatalf("o diário precisa registrar os erros do tick: %+v", journal)
	}

	// REINÍCIO: o diário vem do disco.
	restartedRuntime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: root + "-ws2",
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewSupervisor(restartedRuntime, SupervisorConfig{Enabled: true})
	loaded := restarted.TickJournal()
	if len(loaded) != len(journal) {
		t.Fatalf("o diário precisa sobreviver ao reinício: antes=%d depois=%d", len(journal), len(loaded))
	}
	if loaded[0].TickNumber == 0 {
		t.Fatalf("as linhas precisam vir íntegras: %+v", loaded[0])
	}
}
