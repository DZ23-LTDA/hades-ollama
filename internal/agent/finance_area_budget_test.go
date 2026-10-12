package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// Estagio 30 — ORCAMENTO POR AREA (E-262). O limite declarado dizia "sem rateio por centro de custo e sem
// orcamento por area".
//
// Propriedades travadas: sem orcamento o custo passa (declarado); com orcamento, o custo que ESTOURA e RECUSADO e
// NADA e lancado; o status mostra limite/gasto/saldo; repetir o mesmo custo NAO debita duas vezes; outra empresa nao
// ve nem o orcamento nem o gasto; e tudo SOBREVIVE ao reinicio.
func TestAreaBudgetLimitaCustoEProtegeContraCobrancaDupla(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "data")
	runtime, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: filepath.Join(root, "ws"), DataRoot: dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	empresa, err := runtime.CompanyStore().Create(Company{OrganizationID: LocalOrganizationID, Name: "Loja", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	financas := runtime.Finance()

	// Sem orcamento definido o custo passa: ausencia de limite e' DECLARADA, nao um bloqueio silencioso.
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "ref-1", "frete", 1000, "ana", now); err != nil {
		t.Fatalf("sem orcamento o custo precisa passar: %v", err)
	}
	// Agora com orcamento de 10000 na area.
	if _, err := financas.SetAreaBudget(empresa.ID, "logistica", 10000, now); err != nil {
		t.Fatal(err)
	}
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "ref-2", "frete 2", 6000, "ana", now); err != nil {
		t.Fatalf("custo dentro do orcamento: %v", err)
	}
	// 1000 + 6000 + 5000 = 12000 > 10000: RECUSADO e nada lancado.
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "ref-3", "frete 3", 5000, "ana", now); !errors.Is(err, ErrAreaBudgetExceeded) {
		t.Fatalf("custo que estoura o orcamento precisa ser recusado: %v", err)
	}
	status, err := financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 1 {
		t.Fatalf("esperava UMA area com orcamento: %+v", status)
	}
	if status[0].SpentCents != 7000 || status[0].LimitCents != 10000 || status[0].RemainingCents != 3000 {
		t.Fatalf("status do orcamento errado (a recusa NAO pode ter lancado): %+v", status[0])
	}
	// IDEMPOTENCIA: repetir o mesmo custo (area + referencia) nao debita duas vezes.
	if _, err := financas.PostAreaCost(empresa.ID, "logistica", "ref-2", "frete 2", 6000, "ana", now); err != nil {
		t.Fatalf("repetir o mesmo custo nao pode falhar: %v", err)
	}
	status, err = financas.AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if status[0].SpentCents != 7000 {
		t.Fatalf("cobranca dupla: gasto virou %d", status[0].SpentCents)
	}

	// OUTRA empresa nao ve o orcamento nem o gasto.
	outra, err := runtime.CompanyStore().Create(Company{OrganizationID: "org-b", Name: "Outra", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	statusOutra, err := financas.AreaBudgetStatus(outra.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(statusOutra) != 0 {
		t.Fatalf("orcamento nao pode vazar entre empresas: %+v", statusOutra)
	}

	// PERSISTENCIA: novo runtime sobre o MESMO diretorio de dados.
	reaberto, err := NewRuntime(RuntimeConfig{
		Store: NewMemoryStore(), Planner: RulePlanner{}, WorkspaceRoot: filepath.Join(root, "ws"), DataRoot: dataRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	statusReaberto, err := reaberto.Finance().AreaBudgetStatus(empresa.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(statusReaberto) != 1 || statusReaberto[0].LimitCents != 10000 || statusReaberto[0].SpentCents != 7000 {
		t.Fatalf("orcamento e gasto precisam sobreviver ao reinicio: %+v", statusReaberto)
	}
}
