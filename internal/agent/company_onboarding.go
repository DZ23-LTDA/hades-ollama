package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ONBOARDING DE EMPRESA JÁ EXISTENTE (V2.2 / adendo §3 P1-B) — E-327.
//
// O V2.2 fala em *"onboarding por integração/importação/mapeamento com preview, dedupe, rollback, conflitos e
// conectores versionados"*. **AUDITADO ANTES DE ESCREVER**: o repositório tinha `CompanyStore` (empresa CRIADA pelo
// Hades) e `project_import` (importação de **projeto**), mas **nada** para trazer uma empresa que **já opera** — com
// catálogo, SKUs e preços vindos de fora. ***Uma empresa que já existe não pode ser tratada como se o Hades a tivesse
// criado: os dados dela têm precedência e o sistema precisa saber disso.***
//
// PRINCÍPIOS (cada um com teste negativo):
//
//  1. **PREVIEW NÃO ESCREVE.** O dry-run calcula o que aconteceria e **não toca** no catálogo. *Um preview que grava
//     deixou de ser preview.*
//  2. **DEDUPE DENTRO DO PRÓPRIO PAYLOAD.** A mesma SKU repetida na planilha de origem é **contada** como duplicata e
//     não gera duas vezes — *importar duas vezes a mesma linha é duplicar o catálogo do cliente.*
//  3. **CONFLITO NÃO É SOBRESCRITO EM SILÊNCIO.** SKU existente com **preço diferente** vira **conflito declarado**; só
//     é aplicado com `Overwrite` explícito. *Sobrescrever preço do cliente sem avisar é o tipo de "integração" que
//     destrói confiança.*
//  4. **ROLLBACK RESTAURA O ESTADO ANTERIOR EXATO.** Quem importa precisa poder voltar atrás.
//  5. **REAPLICAÇÃO É IDEMPOTENTE.** A mesma carga duas vezes não duplica produto nem estoque inicial.

var (
	// ErrOnboardingInvalid recusa plano malformado.
	ErrOnboardingInvalid = errors.New("company onboarding plan is invalid")
	// ErrOnboardingConflict recusa aplicar conflito sem Overwrite explícito.
	ErrOnboardingConflict = errors.New("company onboarding has unresolved conflicts")
)

// CompanyOnboardingRow é uma linha vinda do sistema de origem (ERP/planilha/CSV convertido).
type CompanyOnboardingRow struct {
	SKU        string `json:"sku"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Currency   string `json:"currency"`
	// InitialStock é o saldo inicial informado pela origem (0 = não informado).
	InitialStock int64 `json:"initial_stock,omitempty"`
}

// CompanyOnboardingPlan é a carga a importar.
type CompanyOnboardingPlan struct {
	OrganizationID string                 `json:"organization_id"`
	Origin         string                 `json:"origin"` // de onde veio (ex.: "erp-x", "planilha-2026-10")
	Rows           []CompanyOnboardingRow `json:"rows"`
	// Overwrite autoriza, EXPLICITAMENTE, sobrescrever preço divergente de SKU existente.
	Overwrite bool `json:"overwrite,omitempty"`
}

// CompanyOnboardingPreview é o resultado do dry-run: o que ACONTECERIA, sem ter acontecido.
type CompanyOnboardingPreview struct {
	ToCreate      []string `json:"to_create"`
	ToUpdate      []string `json:"to_update"`
	Unchanged     []string `json:"unchanged"`
	Conflicts     []string `json:"conflicts"`
	Duplicates    []string `json:"duplicates"`
	InvalidRows   []string `json:"invalid_rows"`
	ConflictKnown bool     `json:"conflict_known"`
}

// CompanyOnboardingRollback guarda o necessário para desfazer uma aplicação.
type CompanyOnboardingRollback struct {
	OrganizationID string            `json:"organization_id"`
	Origin         string            `json:"origin"`
	Before         []CommerceProduct `json:"before"`
	CreatedSKUs    []string          `json:"created_skus"`
	AppliedAt      time.Time         `json:"applied_at"`
}

// PlanCompanyOnboarding calcula o PREVIEW sem escrever nada.
func PlanCompanyOnboarding(commerce *CommerceStore, plan CompanyOnboardingPlan) (CompanyOnboardingPreview, error) {
	if commerce == nil {
		return CompanyOnboardingPreview{}, fmt.Errorf("%w: comércio ausente", ErrOnboardingInvalid)
	}
	// E-351: MESMA CLASSE DO ADV-11 — `consentScope("")` devolve a organização LOCAL, então validar DEPOIS dele nunca
	// dispara. Aqui isso era pior do que num caminho qualquer: `plan.OrganizationID` também alimenta o IDENTIFICADOR do
	// produto (`"prod_" + sanitizeIDComponent(...)`), e um identificador com escopo vazio faz o produto de uma empresa
	// aparecer no onboarding de outra. *Identificador compartilhado é a mesma classe de defeito que arquivo compartilhado
	// — só muda o objeto.*
	if err := requireOrganization(plan.OrganizationID); err != nil {
		return CompanyOnboardingPreview{}, fmt.Errorf("%w: %v", ErrOnboardingInvalid, err)
	}
	plan.OrganizationID = consentScope(plan.OrganizationID)
	if plan.OrganizationID == "" || len(plan.Rows) == 0 {
		return CompanyOnboardingPreview{}, fmt.Errorf("%w: organização e linhas são obrigatórias", ErrOnboardingInvalid)
	}
	existentes := map[string]CommerceProduct{}
	for _, produto := range commerce.Products(plan.OrganizationID) {
		existentes[produto.SKU] = produto
	}

	preview := CompanyOnboardingPreview{ConflictKnown: plan.Overwrite}
	vistos := map[string]bool{}
	for _, linha := range plan.Rows {
		sku := strings.TrimSpace(linha.SKU)
		nome := strings.TrimSpace(linha.Name)
		if sku == "" || nome == "" || linha.PriceCents < 0 {
			preview.InvalidRows = append(preview.InvalidRows, fmt.Sprintf("%s (nome, sku e preço >= 0 são obrigatórios)", sku))
			continue
		}
		// DEDUPE dentro do payload: a mesma SKU repetida na origem conta uma vez.
		if vistos[sku] {
			preview.Duplicates = append(preview.Duplicates, sku)
			continue
		}
		vistos[sku] = true

		atual, existe := existentes[sku]
		switch {
		case !existe:
			preview.ToCreate = append(preview.ToCreate, sku)
		case atual.PriceCents != linha.PriceCents || strings.TrimSpace(atual.Name) != nome:
			// Conflito: dado do cliente diverge. Sem Overwrite explícito, NÃO se sobrescreve.
			if plan.Overwrite {
				preview.ToUpdate = append(preview.ToUpdate, sku)
			} else {
				preview.Conflicts = append(preview.Conflicts, sku)
			}
		default:
			preview.Unchanged = append(preview.Unchanged, sku)
		}
	}
	sort.Strings(preview.ToCreate)
	sort.Strings(preview.ToUpdate)
	sort.Strings(preview.Unchanged)
	sort.Strings(preview.Conflicts)
	sort.Strings(preview.Duplicates)
	return preview, nil
}

// ApplyCompanyOnboarding aplica a carga e devolve o token de rollback.
//
// Recusa aplicar quando há conflito e `Overwrite` não foi pedido: *o conflito tem de ser uma decisão, não um efeito
// colateral.*
func ApplyCompanyOnboarding(commerce *CommerceStore, plan CompanyOnboardingPlan, now time.Time) (CompanyOnboardingPreview, CompanyOnboardingRollback, error) {
	preview, err := PlanCompanyOnboarding(commerce, plan)
	if err != nil {
		return CompanyOnboardingPreview{}, CompanyOnboardingRollback{}, err
	}
	if len(preview.Conflicts) > 0 && !plan.Overwrite {
		return preview, CompanyOnboardingRollback{}, fmt.Errorf("%w: %s (use Overwrite para decidir sobrescrever)",
			ErrOnboardingConflict, strings.Join(preview.Conflicts, ", "))
	}

	antes := commerce.Products(plan.OrganizationID)
	token := CompanyOnboardingRollback{OrganizationID: plan.OrganizationID, Origin: plan.Origin,
		Before: append([]CommerceProduct(nil), antes...), AppliedAt: now.UTC()}

	vistos := map[string]bool{}
	aplicados := map[string]bool{}
	for _, linha := range plan.Rows {
		sku := strings.TrimSpace(linha.SKU)
		nome := strings.TrimSpace(linha.Name)
		if sku == "" || nome == "" || linha.PriceCents < 0 || vistos[sku] {
			continue
		}
		vistos[sku] = true

		existente, existe := commerce.ProductBySKU(plan.OrganizationID, sku)
		if existe {
			if !plan.Overwrite && (existente.PriceCents != linha.PriceCents || strings.TrimSpace(existente.Name) != nome) {
				continue // conflito já recusado acima; aqui é defesa em profundidade
			}
			if existente.PriceCents == linha.PriceCents && strings.TrimSpace(existente.Name) == nome {
				aplicados[sku] = true
				continue // inalterado: reaplicação idempotente não reescreve
			}
		}
		moeda := strings.TrimSpace(linha.Currency)
		if moeda == "" {
			moeda = "BRL"
		}
		produto := CommerceProduct{
			ID:             "prod_" + sanitizeIDComponent(plan.OrganizationID) + "_" + sanitizeIDComponent(sku),
			OrganizationID: plan.OrganizationID,
			SKU:            sku,
			Name:           nome,
			PriceCents:     linha.PriceCents,
			Currency:       moeda,
			Active:         true,
			CreatedAt:      now.UTC(),
		}
		if existe {
			produto.CreatedAt = existente.CreatedAt
		} else {
			token.CreatedSKUs = append(token.CreatedSKUs, sku)
		}
		commerce.UpsertProduct(produto, now)
		aplicados[sku] = true

		// ESTOQUE INICIAL: registrado UMA vez. Reaplicar não infla (a lição de E-305, agora no onboarding).
		if linha.InitialStock > 0 && len(commerce.Movements(plan.OrganizationID, sku)) == 0 {
			commerce.RecordStock(plan.OrganizationID, sku, linha.InitialStock, "carga inicial de empresa existente", plan.Origin, now)
		}
	}
	sort.Strings(token.CreatedSKUs)
	return preview, token, nil
}

// RollbackCompanyOnboarding desfaz uma aplicação: remove o que foi criado e restaura o que existia.
func RollbackCompanyOnboarding(commerce *CommerceStore, token CompanyOnboardingRollback, now time.Time) error {
	if commerce == nil {
		return fmt.Errorf("%w: comércio ausente", ErrOnboardingInvalid)
	}
	criados := map[string]bool{}
	for _, sku := range token.CreatedSKUs {
		criados[sku] = true
	}
	// 1. Volta cada produto ao estado anterior EXATO.
	for _, produto := range token.Before {
		commerce.UpsertProduct(produto, now)
	}
	// 2. Remove os que passaram a existir por causa desta carga.
	for _, sku := range token.CreatedSKUs {
		atual, existe := commerce.ProductBySKU(token.OrganizationID, sku)
		if !existe || !criados[sku] {
			continue
		}
		atual.Active = false
		commerce.UpsertProduct(atual, now)
		commerce.RemoveProduct(token.OrganizationID, sku)
	}
	return nil
}
