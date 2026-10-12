package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ORCAMENTO POR AREA (E-262). O limite declarado do estagio 30 dizia "sem rateio por centro de custo e sem
// orcamento por area". Orcamento que NAO limita nada e decoracao: aqui o custo que estoura a area e RECUSADO, e o
// rateio fica rastreavel porque a area vive no proprio lancamento.

// ErrAreaBudgetExceeded diz que o custo passaria do orcamento da area. O custo NAO e' lancado.
var ErrAreaBudgetExceeded = errors.New("area budget exceeded")

// ErrAreaBudgetInvalid cobre orcamento sem empresa/area ou com limite negativo.
var ErrAreaBudgetInvalid = errors.New("area budget invalid")

// AreaBudget e' o limite definido para uma area (centro de custo).
type AreaBudget struct {
	CompanyID  string    `json:"company_id"`
	Area       string    `json:"area"`
	LimitCents int64     `json:"limit_cents"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// AreaBudgetStatus e' o orcamento com o gasto acumulado.
type AreaBudgetStatus struct {
	Area           string `json:"area"`
	LimitCents     int64  `json:"limit_cents"`
	SpentCents     int64  `json:"spent_cents"`
	RemainingCents int64  `json:"remaining_cents"`
	Exceeded       bool   `json:"exceeded"`
}

type areaBudgetFile struct {
	Budgets []AreaBudget `json:"budgets"`
}

// budgetPath devolve o arquivo de orcamentos da empresa. O arquivo do razao NAO e' tocado: orcamento e' politica,
// nao lancamento.
func (f *FinanceOps) budgetPath(companyID string) string {
	if f == nil || f.companies == nil {
		return ""
	}
	root := strings.TrimSpace(f.companies.Root())
	if root == "" {
		return ""
	}
	seguro := strings.NewReplacer("/", "_", "\\", "_", "..", "_").Replace(strings.TrimSpace(companyID))
	if seguro == "" {
		return ""
	}
	return filepath.Join(root, "area_budgets_"+seguro+".json")
}

func (f *FinanceOps) loadAreaBudgets(companyID string) ([]AreaBudget, error) {
	caminho := f.budgetPath(companyID)
	if caminho == "" {
		return nil, nil
	}
	stored := areaBudgetFile{}
	if err := readJSON(caminho, &stored); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("ler orcamentos por area: %w", err)
	}
	return stored.Budgets, nil
}

// SetAreaBudget define (ou atualiza) o limite da area.
func (f *FinanceOps) SetAreaBudget(companyID, area string, limitCents int64, now time.Time) (AreaBudget, error) {
	companyID = strings.TrimSpace(companyID)
	area = strings.TrimSpace(area)
	if companyID == "" || area == "" || limitCents < 0 {
		return AreaBudget{}, fmt.Errorf("%w: empresa=%q area=%q limite=%d", ErrAreaBudgetInvalid, companyID, area, limitCents)
	}
	orcamentos, err := f.loadAreaBudgets(companyID)
	if err != nil {
		return AreaBudget{}, err
	}
	novo := AreaBudget{CompanyID: companyID, Area: area, LimitCents: limitCents, UpdatedAt: now.UTC()}
	substituido := false
	for i := range orcamentos {
		if orcamentos[i].Area == area {
			orcamentos[i] = novo
			substituido = true
			break
		}
	}
	if !substituido {
		orcamentos = append(orcamentos, novo)
	}
	caminho := f.budgetPath(companyID)
	if caminho == "" {
		// Modo em memoria: o orcamento vale nesta execucao. Declarado, nao fingido.
		return novo, nil
	}
	if err := writeJSONAtomic(caminho, areaBudgetFile{Budgets: orcamentos}); err != nil {
		return AreaBudget{}, err
	}
	return novo, nil
}

// areaSpentCents soma os CUSTOS lancados na area (despesa; receita nao consome orcamento).
func (f *FinanceOps) areaSpentCents(companyID, area string) (int64, error) {
	lancamentos, err := f.companies.ListLedgerEntries(companyID, time.Time{}, time.Time{})
	if err != nil {
		return 0, err
	}
	var total int64
	for _, lancamento := range lancamentos {
		if lancamento.Area != area || normalizeLedgerKind(lancamento.Kind) != LedgerKindExpense {
			continue
		}
		total += lancamento.AmountCents
	}
	return total, nil
}

// AreaBudgetStatus devolve o orcamento de cada area com o gasto acumulado.
func (f *FinanceOps) AreaBudgetStatus(companyID string) ([]AreaBudgetStatus, error) {
	orcamentos, err := f.loadAreaBudgets(companyID)
	if err != nil {
		return nil, err
	}
	resultado := make([]AreaBudgetStatus, 0, len(orcamentos))
	for _, orcamento := range orcamentos {
		gasto, err := f.areaSpentCents(companyID, orcamento.Area)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, AreaBudgetStatus{
			Area:           orcamento.Area,
			LimitCents:     orcamento.LimitCents,
			SpentCents:     gasto,
			RemainingCents: orcamento.LimitCents - gasto,
			Exceeded:       gasto > orcamento.LimitCents,
		})
	}
	return resultado, nil
}

// PostAreaCost lanca um CUSTO na area, respeitando o orcamento.
//
// Recusa com ErrAreaBudgetExceeded quando o custo passaria do limite - e nesse caso NADA e' lancado. A idempotencia
// usa (area + referencia): repetir o mesmo custo devolve o lancamento existente em vez de debitar duas vezes.
func (f *FinanceOps) PostAreaCost(companyID, area, reference, description string, amountCents int64, actor string, now time.Time) (LedgerEntry, error) {
	companyID = strings.TrimSpace(companyID)
	area = strings.TrimSpace(area)
	if amountCents <= 0 {
		return LedgerEntry{}, fmt.Errorf("%w: valor %d", ErrAreaBudgetInvalid, amountCents)
	}
	// A IDEMPOTENCIA VEM ANTES DA POLITICA. Se este custo JA foi lancado, a retentativa nao gasta orcamento novo:
	// checar o orcamento primeiro recusava a retentativa de um custo ACEITO - e pior, somava o valor ja lancado,
	// produzindo uma recusa aritmeticamente correta sobre semantica errada.
	identidade := "area:" + companyID + "|" + area + "|" + strings.TrimSpace(reference)
	if strings.TrimSpace(reference) != "" {
		existentes, err := f.companies.ListLedgerEntries(companyID, time.Time{}, time.Time{})
		if err != nil {
			return LedgerEntry{}, err
		}
		for _, existente := range existentes {
			if existente.Idempotency == identidade {
				return existente, nil
			}
		}
	}
	orcamentos, err := f.loadAreaBudgets(companyID)
	if err != nil {
		return LedgerEntry{}, err
	}
	var limite int64
	temOrcamento := false
	for _, orcamento := range orcamentos {
		if orcamento.Area == area {
			limite, temOrcamento = orcamento.LimitCents, true
			break
		}
	}
	if temOrcamento {
		gasto, err := f.areaSpentCents(companyID, area)
		if err != nil {
			return LedgerEntry{}, err
		}
		if gasto+amountCents > limite {
			return LedgerEntry{}, fmt.Errorf("%w: area %q gastaria %d de %d", ErrAreaBudgetExceeded, area, gasto+amountCents, limite)
		}
	}
	lancamento, _, err := f.companies.RecordLedgerEntry(LedgerEntry{
		CompanyID:   companyID,
		Kind:        LedgerKindExpense,
		Category:    area,
		Area:        area,
		AmountCents: amountCents,
		Description: strings.TrimSpace(description),
		Reference:   strings.TrimSpace(reference),
		Idempotency: identidade,
		OccurredAt:  now.UTC(),
		CreatedBy:   strings.TrimSpace(actor),
	}, now)
	if err != nil {
		return LedgerEntry{}, err
	}
	return lancamento, nil
}
