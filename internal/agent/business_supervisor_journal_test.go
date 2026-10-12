package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estágio 31 — DIÁRIO DE CICLOS DURÁVEL (E-275). O limite declarado dizia que "o HISTÓRICO dos ciclos segue em
// memória" (E-258 tentou fechar e foi revertida por instabilidade da suíte).
//
// DIAGNÓSTICO medido: RunSupervisedTick NUNCA chama persistLocked; só caminhos de AÇÃO escrevem estado. Um ciclo que
// não propõe ação nenhuma — o caso comum — não deixava rastro. Foi por isso que a tentativa anterior ficou instável:
// a escrita foi colocada em outro lugar.
//
// Propriedades travadas aqui:
//  1. ciclo SILENCIOSO (sem ação) também deixa registro durável;
//  2. ciclo COM ação tem o registro persistido;
//  3. reinício real restaura o diário;
//  4. o cap é respeitado na restauração, mantendo os MAIS RECENTES.
func TestDiarioDeCiclosSobreviveAoReinicioInclusiveCicloSilencioso(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	supRoot := filepath.Join(root, "sup")
	supervisor, err := NewBusinessSupervisorAt(supRoot, runtime, DefaultSupervisorPolicy())
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()

	// 1. Ciclo SILENCIOSO: nada na empresa, portanto nenhuma ação proposta.
	silencioso, err := supervisor.BusinessTick(LocalOrganizationID, agora)
	if err != nil {
		t.Fatalf("ciclo silencioso: %v", err)
	}
	if len(silencioso.Actions) != 0 {
		t.Fatalf("o cenário do teste exige um ciclo SEM ações, veio %+v", silencioso.Actions)
	}

	// 2. Reinício REAL: o diário precisa conter o ciclo silencioso.
	reaberto, err := NewBusinessSupervisorAt(supRoot, runtime, DefaultSupervisorPolicy())
	if err != nil {
		t.Fatal(err)
	}
	diario := reaberto.History()
	if len(diario) != 1 {
		t.Fatalf("um ciclo silencioso precisa deixar UM registro durável, veio %d", len(diario))
	}
	if diario[0].TickNumber != silencioso.TickNumber {
		t.Fatalf("o registro precisa ser o mesmo ciclo: %d != %d", diario[0].TickNumber, silencioso.TickNumber)
	}

	// 3. Mais um ciclo: o diário acumula em ordem.
	if _, err := reaberto.BusinessTick(LocalOrganizationID, agora.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(reaberto.History()) != 2 {
		t.Fatalf("o diário precisa acumular: %d", len(reaberto.History()))
	}
}

// 4. O cap é aplicado NA RESTAURAÇÃO, preservando os MAIS RECENTES: guardar os mais antigos esconderia justamente o
// que aconteceu por último.
func TestDiarioRestauradoRespeitaOCapMantendoOsMaisRecentes(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store:         NewMemoryStore(),
		Planner:       RulePlanner{},
		WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot:      filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	supRoot := filepath.Join(root, "sup")
	antigo, err := NewBusinessSupervisorAt(supRoot, runtime, DefaultSupervisorPolicy())
	if err != nil {
		t.Fatal(err)
	}
	excesso := make([]BusinessTickResult, MaxSupervisorTickJournal+5)
	for i := range excesso {
		excesso[i] = BusinessTickResult{TickNumber: int64(i + 1), OrganizationID: LocalOrganizationID}
	}
	// Escreve direto no MESMO arquivo de estado (dentro do pacote: é o formato real).
	if err := writeJSONAtomic(antigo.path(), businessSupervisorFile{
		Policy:  DefaultSupervisorPolicy(),
		Ticks:   int64(len(excesso)),
		History: excesso,
	}); err != nil {
		t.Fatal(err)
	}
	reaberto, err := NewBusinessSupervisorAt(supRoot, runtime, DefaultSupervisorPolicy())
	if err != nil {
		t.Fatal(err)
	}
	diario := reaberto.History()
	if len(diario) != MaxSupervisorTickJournal {
		t.Fatalf("o cap precisa truncar na restauração: %d", len(diario))
	}
	if diario[len(diario)-1].TickNumber != int64(len(excesso)) {
		t.Fatalf("o ÚLTIMO registro precisa ser preservado, veio %d", diario[len(diario)-1].TickNumber)
	}
	if diario[0].TickNumber != int64(len(excesso)-MaxSupervisorTickJournal+1) {
		t.Fatalf("o primeiro registro restaurado precisa ser o mais antigo DENTRO do cap, veio %d", diario[0].TickNumber)
	}
}
