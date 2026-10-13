package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ADV-24 — CAUSA-RAIZ ENCONTRADA (E-350).
//
// Em E-349 a execução com `Runtime` real terminou sem erro e **nenhum evento** apareceu. Duas hipóteses foram
// levantadas e **nenhuma foi escolhida por palpite**: foram medidas.
//
// **RESULTADO**: `JSONStore.AppendEvent` **exige ids válidos** (`validSnapshotID(event.MissionID)`) — e os ids usados
// na sonda anterior (`"missao-sonda"`, `"evt_teste"`) **não eram válidos** ✗. O laço **emitiu** o evento; o **store o
// rejeitou**. ***Não era defeito do produto: eram os MEUS dados de teste.*** A declaração de E-349 (não declarar
// verificado o que não se observou) estava **certa** — e a investigação mostrou que a causa era minha.
//
// ***Hipótese não é diagnóstico; sonda é.***

func storeComRaiz(t *testing.T) *JSONStore {
	t.Helper()
	store, err := NewJSONStore(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatalf("criar store com raiz: %v", err)
	}
	return store
}

func TestAppendEventExigeIdsValidos(t *testing.T) {
	store := storeComRaiz(t)
	// ARMADILHA: id de missão inválido precisa ser RECUSADO — foi exatamente isto que aconteceu em E-349.
	eventoInvalido := Event{ID: uuid.NewString(), MissionID: "missao-sonda", Type: "tipo",
		Payload: map[string]any{"a": 1}, CreatedAt: time.Now().UTC()}
	if err := store.AppendEvent(eventoInvalido); err == nil {
		t.Fatal("ARMADILHA: id de missão inválido foi aceito; a rejeição observada em E-349 não se explica")
	}
	// E id de EVENTO inválido também.
	eventoComEventIDInvalido := Event{ID: "evt_teste", MissionID: uuid.NewString(), Type: "tipo",
		Payload: map[string]any{"a": 1}, CreatedAt: time.Now().UTC()}
	if err := store.AppendEvent(eventoComEventIDInvalido); err == nil {
		t.Fatal("id de evento inválido precisa ser recusado")
	}
}

func TestComIdsValidosOStorePersisteELista(t *testing.T) {
	store := storeComRaiz(t)
	missao := uuid.NewString()
	// MEDIDO (E-350): `AppendEvent` exige a missão EXISTENTE e a MESMA organização (`os.ErrNotExist` / `os.ErrPermission`).
	// *Isolamento de tenant no próprio store de eventos.*
	if err := store.CreateMission(Mission{ID: missao, OrganizationID: "org-a"}); err != nil {
		t.Fatalf("criar missão: %v", err)
	}
	if err := store.AppendEvent(Event{ID: uuid.NewString(), MissionID: missao, OrganizationID: "org-a",
		Type: "mission.repair_attempt", StepID: "step-1", Payload: map[string]any{"attempt": 1},
		CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("com ids válidos o evento precisa ser aceito: %v", err)
	}
	eventos, err := store.ListEvents(missao)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(eventos) != 1 || eventos[0].Type != "mission.repair_attempt" {
		t.Fatalf("o evento anexado precisa ser listado: %+v", eventos)
	}
	// Isolamento por missão — e o contrato MEDIDO: missão INEXISTENTE devolve erro (`os.ErrNotExist`), não lista vazia.
	// Eu havia suposto "lista vazia"; a medição mostrou erro. *Contrato do store, não preferência minha.*
	if _, err := store.ListEvents(uuid.NewString()); err == nil {
		t.Fatal("missão inexistente precisa devolver erro no store (contrato medido), não lista vazia")
	}
	// E a missão criada com OUTRA organização não aceita evento desta: isolamento de tenant no store de eventos.
	outraMissao := uuid.NewString()
	if err := store.CreateMission(Mission{ID: outraMissao, OrganizationID: "org-b"}); err != nil {
		t.Fatalf("criar missão da outra organização: %v", err)
	}
	if err := store.AppendEvent(Event{ID: uuid.NewString(), MissionID: outraMissao, OrganizationID: "org-a",
		Type: "tipo", Payload: map[string]any{"a": 1}, CreatedAt: time.Now().UTC()}); err == nil {
		t.Fatal("ARMADILHA: evento de uma organização não pode ser aceito em missão de outra")
	}
	if eventos, err := store.ListEvents(outraMissao); err != nil || len(eventos) != 0 {
		t.Fatalf("a missão da outra organização precisa continuar sem eventos: %v %d", err, len(eventos))
	}
}

// A PROVA DO CAMINHO COMPLETO: o laço REAL, com Runtime REAL e missão de id VÁLIDO, persiste os seus eventos.
func TestLacoRealPersisteEventosComMissaoValida(t *testing.T) {
	repo := repoComFixtureT2(t)
	tarefa := FixtureT2()
	ctx, cancelar := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancelar()

	store := storeComRaiz(t)
	runtime, err := NewRuntime(RuntimeConfig{Store: store, WorkspaceRoot: repo, DataRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("construir Runtime: %v", err)
	}
	missaoID := uuid.NewString()
	// A missão precisa existir ANTES dos eventos: sem isso o store responde `os.ErrNotExist` (medido em E-350).
	if err := store.CreateMission(Mission{ID: missaoID, OrganizationID: "org-a"}); err != nil {
		t.Fatalf("criar missão do laço: %v", err)
	}
	opcoes := DefaultSpecialistLoopOptions()
	opcoes.Runtime = runtime
	opcoes.MissionID = missaoID
	opcoes.OrganizationID = "org-a"

	fixer := &fixerDeArquivos{arquivos: map[string]string{"calc.go": tarefa.ReferenceFix["calc.go"]}}
	relatorio, err := RunSpecialistRepairLoops(ctx, repo, t.TempDir(),
		[]SpecialistTask{{Name: "calc", Segment: []string{"calc.go"}}}, fixer, opcoes)
	if err != nil {
		t.Fatalf("execução com missão válida: %v", err)
	}
	if relatorio.Verdict != "MESCLADO" {
		t.Fatalf("veredito: %+v", relatorio)
	}
	eventos, err := store.ListEvents(missaoID)
	if err != nil {
		t.Fatalf("ListEvents da missão do laço: %v", err)
	}
	if len(eventos) == 0 {
		t.Fatal("ADV-24 NÃO resolvido: com ids válidos e Runtime real, os eventos do laço precisam ser persistidos")
	}
	tipos := map[string]bool{}
	for _, evento := range eventos {
		tipos[evento.Type] = true
	}
	if !tipos["mission.repair_attempt"] {
		t.Fatalf("o laço precisa registrar `mission.repair_attempt`; tipos observados: %v", tipos)
	}
	t.Logf("ADV-24 RESOLVIDO: %d evento(s) do laço persistidos (tipos: %v)", len(eventos), tipos)
}
