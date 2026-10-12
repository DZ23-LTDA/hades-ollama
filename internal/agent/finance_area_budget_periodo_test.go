package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 30 — RECORTE MENSAL do orcamento (E-263), limite declarado em E-262.
//
// Propriedades travadas: o gasto de um mes NAO consome o limite do mes seguinte; a MESMA referencia em outro mes e
// OUTRO custo (nao pode ser engolida como "ja lancado"); e o mes fechado continua consultavel.
func TestAreaBudgetRespeitaRecorteMensal(t *testing.T) {
	root := t.TempDir()
	runtime, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: filepath.Join(root, "ws"),
		DataRoot: filepath.Join(root, "data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	empresa, err := runtime.CompanyStore().Create(Company{OrganizationID: LocalOrganizationID, Name: "Loja", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	financas := runtime.Finance()
	janeiro := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	fevereiro := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	if _, err := financas.SetAreaBudget(empresa.ID, "logistica", 10000, janeiro); err != nil {
		t.Fatal(err)
	}
	// Janeiro consome 9000 de 10000.
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "NF-1", "frete jan", 9000, "ana", janeiro); err != nil {
		t.Fatal(err)
	}
	// Fevereiro COMECA ZERADO: o gasto de janeiro nao consome o limite de fevereiro.
	status, err := financas.AreaBudgetStatus(empresa.ID, fevereiro)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 || status[0].SpentCents != 0 || status[0].RemainingCents != 10000 {
		t.Fatalf("fevereiro precisa comecar zerado: %+v", status)
	}
	if status[0].Periodo != "2026-02" {
		t.Fatalf("periodo do status: %q", status[0].Periodo)
	}
	// A MESMA referencia em outro mes e OUTRO custo: se o periodo nao entrasse na chave, seria engolida.
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "NF-1", "frete fev", 9000, "ana", fevereiro); err != nil {
		t.Fatalf("a mesma referencia em outro mes precisa ser lancada: %v", err)
	}
	statusFev, err := financas.AreaBudgetStatus(empresa.ID, fevereiro)
	if err != nil {
		t.Fatal(err)
	}
	if statusFev[0].SpentCents != 9000 {
		t.Fatalf("fevereiro precisa ter o proprio gasto: %+v", statusFev[0])
	}
	// Janeiro continua consultavel e intacto.
	statusJan, err := financas.AreaBudgetStatus(empresa.ID, janeiro)
	if err != nil {
		t.Fatal(err)
	}
	if statusJan[0].SpentCents != 9000 {
		t.Fatalf("janeiro precisa continuar com o proprio gasto: %+v", statusJan[0])
	}
	// Estouro NO MES corrente e recusado (9000 + 2000 > 10000).
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "NF-2", "frete fev 2", 2000, "ana", fevereiro); !errors.Is(err, ErrAreaBudgetExceeded) {
		t.Fatalf("estouro no mes corrente precisa ser recusado: %v", err)
	}
	// E o mes SEGUINTE segue livre.
	marco := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "NF-3", "frete mar", 9500, "ana", marco); err != nil {
		t.Fatalf("o mes seguinte precisa estar livre: %v", err)
	}
}
