package agent

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Atividade do estágio 31: REGISTRO DE INCIDENTE E RECUPERAÇÃO.
//
// As propriedades travadas aqui:
//  1. a mesma falha repetida INCREMENTA tentativas em vez de criar incidente novo;
//  2. a recuperação é OBSERVADA: sucesso posterior resolve com data, sem ninguém
//     precisar marcar como resolvido;
//  3. incidente resolvido que volta a falhar REABRE;
//  4. ciência humana não é resolução, e exige ator identificado;
//  5. o registro sobrevive ao reinício e é limitado.
func incidentSupervisor(t *testing.T) (*Supervisor, string) {
	t.Helper()
	root := t.TempDir()
	// O runtime EXIGE data root separado do workspace de execução: dado e código
	// não podem morar no mesmo lugar. O teste segue a mesma regra do produto.
	store := NewMemoryStore()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "workspace"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewSupervisor(runtime, SupervisorConfig{Enabled: true}), filepath.Join(root, "data")
}

func TestIncidentIsIdempotentPerSubjectAndCountsAttempts(t *testing.T) {
	supervisor, _ := incidentSupervisor(t)
	now := time.Now().UTC()
	if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure, "sched-1", "primeira falha", now); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure, "sched-1", "segunda falha", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// Sujeito diferente é incidente diferente.
	if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure, "sched-2", "falha do outro", now); err != nil {
		t.Fatal(err)
	}
	incidents := supervisor.Incidents()
	if len(incidents) != 2 {
		t.Fatalf("a repetição não pode criar incidente novo: %+v", incidents)
	}
	var primeiro SupervisorIncident
	for _, incident := range incidents {
		if incident.Subject == "sched-1" {
			primeiro = incident
		}
	}
	if primeiro.RecoveryAttempts != 2 {
		t.Fatalf("a repetição precisa somar tentativa: %+v", primeiro)
	}
	if primeiro.Detail != "segunda falha" {
		t.Fatalf("o detalhe precisa refletir a falha mais recente: %+v", primeiro)
	}
	if primeiro.State != SupervisorIncidentOpen {
		t.Fatalf("falha repetida mantém aberto: %+v", primeiro)
	}
}

func TestIncidentResolvesWhenRecoveryIsObserved(t *testing.T) {
	supervisor, _ := incidentSupervisor(t)
	now := time.Now().UTC()
	if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure, "sched-1", "falhou", now); err != nil {
		t.Fatal(err)
	}
	resolved, err := supervisor.resolveIncident(SupervisorIncidentScheduleFailure, "sched-1", "sucesso posterior", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !resolved {
		t.Fatal("havia incidente aberto e ele precisava ser resolvido")
	}
	incidents := supervisor.Incidents()
	if len(incidents) != 1 {
		t.Fatalf("resolver não cria registro novo: %+v", incidents)
	}
	if incidents[0].State != SupervisorIncidentResolved || incidents[0].ResolvedAt == nil {
		t.Fatalf("a recuperação observada precisa fechar com data: %+v", incidents[0])
	}
	// Resolver de novo é no-op: não havia incidente aberto.
	again, err := supervisor.resolveIncident(SupervisorIncidentScheduleFailure, "sched-1", "de novo", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if again {
		t.Fatal("não havia incidente aberto para resolver")
	}
}

func TestResolvedIncidentReopensOnNewFailure(t *testing.T) {
	supervisor, _ := incidentSupervisor(t)
	now := time.Now().UTC()
	if err := supervisor.recordIncident(SupervisorIncidentRiskPause, "empresa-1", "pausada", now); err != nil {
		t.Fatal(err)
	}
	if _, err := supervisor.resolveIncident(SupervisorIncidentRiskPause, "empresa-1", "voltou a operar", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// A empresa volta a ser pausada: o histórico que voltou a ser problema NÃO pode
	// ficar escondido como resolvido.
	if err := supervisor.recordIncident(SupervisorIncidentRiskPause, "empresa-1", "pausada de novo", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	incidents := supervisor.Incidents()
	if len(incidents) != 1 {
		t.Fatalf("reabrir não cria segundo incidente: %+v", incidents)
	}
	incident := incidents[0]
	if incident.State != SupervisorIncidentOpen || incident.ResolvedAt != nil {
		t.Fatalf("o incidente precisa reabrir limpo: %+v", incident)
	}
	if incident.RecoveryAttempts != 2 {
		t.Fatalf("reabrir soma tentativa: %+v", incident)
	}
}

func TestIncidentAcknowledgementNeedsActorAndIsNotResolution(t *testing.T) {
	supervisor, _ := incidentSupervisor(t)
	now := time.Now().UTC()
	if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure, "sched-9", "falhou", now); err != nil {
		t.Fatal(err)
	}
	id := incidentID(SupervisorIncidentScheduleFailure, "sched-9")
	if _, err := supervisor.AcknowledgeIncident(id, "   ", now); err == nil {
		t.Fatal("ciência do incidente precisa de ator identificado")
	}
	acknowledged, err := supervisor.AcknowledgeIncident(id, "ana", now)
	if err != nil {
		t.Fatal(err)
	}
	if acknowledged.State != SupervisorIncidentAcknowledged || acknowledged.ResolvedAt != nil {
		t.Fatalf("ciência não é resolução: %+v", acknowledged)
	}
	if acknowledged.AcknowledgedBy != "ana" {
		t.Fatalf("precisa registrar quem assumiu: %+v", acknowledged)
	}
	// Resolução humana explícita.
	resolved, err := supervisor.ResolveIncident(id, "ana", "corrigido pelo operador", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if resolved.State != SupervisorIncidentResolved || resolved.Resolution != "corrigido pelo operador" {
		t.Fatalf("resolução explícita precisa registrar motivo: %+v", resolved)
	}
	if _, err := supervisor.ResolveIncident("inc_inexistente", "ana", "", now); err == nil {
		t.Fatal("incidente inexistente precisa falhar")
	}
}

func TestIncidentsSurviveRestartAndAreBounded(t *testing.T) {
	supervisor, root := incidentSupervisor(t)
	now := time.Now().UTC()
	for index := range MaxSupervisorIncidents + 5 {
		if err := supervisor.recordIncident(SupervisorIncidentScheduleFailure,
			"sched-"+string(rune('a'+index%26))+string(rune('0'+index/26)), "falha", now.Add(time.Duration(index)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	incidents := supervisor.Incidents()
	if len(incidents) > MaxSupervisorIncidents {
		t.Fatalf("o registro precisa respeitar o teto de %d: %d", MaxSupervisorIncidents, len(incidents))
	}

	// REINÍCIO: um supervisor novo sobre o mesmo diretório de DADOS enxerga o registro.
	store := NewMemoryStore()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Planner:       RulePlanner{},
		WorkspaceRoot: root + "-ws-restart",
		DataRoot:      root,
	})
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewSupervisor(runtime, SupervisorConfig{Enabled: true})
	loaded := restarted.Incidents()
	if len(loaded) != len(incidents) {
		t.Fatalf("o registro precisa sobreviver ao reinício: antes=%d depois=%d", len(incidents), len(loaded))
	}
	if loaded[0].Subject == "" || loaded[0].OpenedAt.IsZero() {
		t.Fatalf("os registros precisam vir íntegros: %+v", loaded[0])
	}
}

// O TICK precisa registrar o incidente de falha de retomada, que antes era
// silenciosamente engolida quando o segundo `resumePending` também falhava.
func TestTickRecordsRecoveryFailureInsteadOfSwallowingIt(t *testing.T) {
	root := t.TempDir()
	store := failingStore{JSONStore: NewMemoryStore()}
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         store,
		Planner:       RulePlanner{},
		WorkspaceRoot: root + "-ws",
		DataRoot:      root + "-data",
	})
	if err != nil {
		t.Fatal(err)
	}
	supervisor := NewSupervisor(runtime, SupervisorConfig{Enabled: true})
	result, err := supervisor.Tick(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range result.ScheduleRunErrors {
		if strings.Contains(entry, "resume") {
			found = true
		}
	}
	if !found {
		t.Fatalf("a falha persistente de retomada precisa aparecer no tick: %+v", result.ScheduleRunErrors)
	}
	incidents := supervisor.Incidents()
	if len(incidents) != 1 || incidents[0].Kind != SupervisorIncidentRecoveryFailure {
		t.Fatalf("a falha precisa virar incidente de recuperação: %+v", incidents)
	}
}

// failingStore faz a listagem de missões falhar sempre, para provar que a falha
// persistente de retomada vira incidente em vez de desaparecer.
type failingStore struct {
	*JSONStore
}

func (f failingStore) ListMissions() ([]Mission, error) {
	return nil, context.DeadlineExceeded
}
