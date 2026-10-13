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
	// ErrOnboardingTooLarge recusa plano acima dos limites DECLARADOS de volume.
	ErrOnboardingTooLarge = errors.New("onboarding plan exceeds declared limits")
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
	// E-353: LIMITE DE VOLUME conferido ANTES de qualquer trabalho (este e' o ponto de entrada comum a plan e apply).
	if err := ValidarVolumeDoPlano(plan, DefaultOnboardingLimits()); err != nil {
		return CompanyOnboardingPreview{}, err
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
// ApplyCompanyOnboarding valida o volume DO PLANO **e** o acervo ACUMULADO antes de aplicar (E-359).
//
// ***O limite por plano não limita nada sozinho: três planos de 4000 linhas passam um a um e o resultado são
// 12000 linhas carregadas.*** Aqui o teto é conferido contra o estado real do comércio, que é a única fonte de
// verdade que não pode divergir de si mesma.
func ApplyCompanyOnboarding(commerce *CommerceStore, plan CompanyOnboardingPlan, now time.Time) (CompanyOnboardingPreview, CompanyOnboardingRollback, error) {
	return aplicarComOrcamento(commerce, plan, now)
}

func aplicarSemOrcamento(commerce *CommerceStore, plan CompanyOnboardingPlan, now time.Time) (CompanyOnboardingPreview, CompanyOnboardingRollback, error) {
	preview, err := PlanCompanyOnboarding(commerce, plan)
	if err != nil {
		return CompanyOnboardingPreview{}, CompanyOnboardingRollback{}, err
	}
	if len(preview.Conflicts) > 0 && !plan.Overwrite {
		return preview, CompanyOnboardingRollback{}, fmt.Errorf("%w: %s (use Overwrite para decidir sobrescrever)",
			ErrOnboardingConflict, strings.Join(preview.Conflicts, ", "))
	}

	antes := commerce.Products(plan.OrganizationID)
	token := CompanyOnboardingRollback{
		OrganizationID: plan.OrganizationID, Origin: plan.Origin,
		Before: append([]CommerceProduct(nil), antes...), AppliedAt: now.UTC(),
	}

	vistos := map[string]bool{}
	aplicados := map[string]bool{}
	lote := make([]CommerceProduct, 0, len(plan.Rows))
	estoques := make([]estoqueInicial, 0, len(plan.Rows))
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
		// E-364: LOTE. Antes, cada linha gravava o arquivo inteiro da organização (O(n²) em varredura e I/O: 2000
		// linhas em 54,13s/50,01s). E o erro era DESCARTADO — uma falha de gravação virava "sucesso" no relatório.
		lote = append(lote, produto)
		aplicados[sku] = true

		// ESTOQUE INICIAL: registrado UMA vez por SKU, DEPOIS de o produto existir (a lição de E-305). O guarda fica
		// aqui (a decisão) e a gravação acontece na segunda passada — *duas passadas preservam a ordem que o teste de
		// E-305 verifica: produto primeiro, movimento depois*.
		if linha.InitialStock > 0 && len(commerce.Movements(plan.OrganizationID, sku)) == 0 {
			estoques = append(estoques, estoqueInicial{sku: sku, quantidade: linha.InitialStock})
		}
	}
	// PASSADA 1: UMA gravação para o lote inteiro. O erro NÃO é descartado: falha de gravação precisa chegar a quem chamou.
	if _, err := commerce.UpsertProducts(lote, now); err != nil {
		return preview, token, fmt.Errorf("%w: falha ao gravar o lote do onboarding: %v", ErrOnboardingInvalid, err)
	}
	// PASSADA 2: estoque inicial, agora com TODOS os produtos já gravados — a ordem que o teste de E-305 exige — e
	// gravado em LOTE (E-365): uma gravação em vez de uma por linha (ADV-44: 19,11s medidos antes).
	if len(estoques) > 0 {
		movimentos := make([]MovimentoDeEstoque, 0, len(estoques))
		for _, estoque := range estoques {
			movimentos = append(movimentos, MovimentoDeEstoque{
				SKU: estoque.sku, Delta: estoque.quantidade,
				Kind: "carga inicial de empresa existente", Reference: plan.Origin,
			})
		}
		if _, err := commerce.RecordStocks(plan.OrganizationID, movimentos, now); err != nil {
			return preview, token, fmt.Errorf("%w: falha ao gravar o estoque inicial: %v", ErrOnboardingInvalid, err)
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
	// 🚨 ADV-57/5 (4ª auditoria independente, MEDIDO): aqui era `commerce.UpsertProduct(produto, now)` **por linha**,
	// com o erro **ignorado**: desfazer 500 linhas custava **1000 gravações em 2,72s** (o mesmo O(n²) do aplicar, que
	// o E-364 corrigiu). ***A minha própria tese — "uma API mais rápida que ninguém usa não acelera nada" — se
	// aplicava ao desfazer, e nenhuma prova publicada cobria isso.*** Agora é **um lote** (1 gravação), com o erro
	// **checado**: *rollback que falha em silêncio deixa o operador achando que desfez.*
	if len(token.Before) > 0 {
		if _, err := commerce.UpsertProducts(token.Before, now); err != nil {
			return fmt.Errorf("%w: falha ao restaurar os produtos anteriores: %v", ErrOnboardingInvalid, err)
		}
	}
	// 2. Remove os que passaram a existir por causa desta carga.
	for _, sku := range token.CreatedSKUs {
		atual, existe := commerce.ProductBySKU(token.OrganizationID, sku)
		if !existe || !criados[sku] {
			continue
		}
		// O `UpsertProduct` intermediário que marcava `Active=false` foi REMOVIDO: ele persistia o arquivo inteiro
		// (uma gravação por SKU) para um estado que o `RemoveProduct` logo em seguida apaga. *Gravar um estado
		// intermediário que ninguém pode observar é custo sem efeito.* E o erro da remoção deixou de ser ignorado.
		atual.Active = false
		if err := commerce.RemoveProduct(token.OrganizationID, sku); err != nil {
			return fmt.Errorf("%w: falha ao remover o produto criado %q: %v", ErrOnboardingInvalid, sku, err)
		}
	}
	return nil
}

// OnboardingLimits são os limites DECLARADOS de volume do onboarding.
//
// POR QUE ISTO EXISTE (E-353): o caminho aceitava **qualquer** quantidade de linhas — só recusava o vazio. Um payload
// de volume (milhões de linhas, ou um nome de gigabytes) consumiria memória e disco sem que nada no código dissesse
// qual era o teto. ***Limite que não está escrito não é limite: é surpresa.***
type OnboardingLimits struct {
	MaxRows        int
	MaxNameBytes   int
	MaxOriginBytes int
	MaxTotalBytes  int64
}

// DefaultOnboardingLimits devolve os limites explícitos (nada de número mágico espalhado).
func DefaultOnboardingLimits() OnboardingLimits {
	return OnboardingLimits{MaxRows: 5000, MaxNameBytes: 512, MaxOriginBytes: 256, MaxTotalBytes: 16 << 20}
}

func (l OnboardingLimits) normalizar() OnboardingLimits {
	padrao := DefaultOnboardingLimits()
	if l.MaxRows <= 0 {
		l.MaxRows = padrao.MaxRows
	}
	if l.MaxNameBytes <= 0 {
		l.MaxNameBytes = padrao.MaxNameBytes
	}
	if l.MaxOriginBytes <= 0 {
		l.MaxOriginBytes = padrao.MaxOriginBytes
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = padrao.MaxTotalBytes
	}
	return l
}

// ValidarVolumeDoPlano recusa plano acima dos limites declarados, ANTES de qualquer trabalho.
func ValidarVolumeDoPlano(plan CompanyOnboardingPlan, limites OnboardingLimits) error {
	limites = limites.normalizar()
	if len(plan.Rows) > limites.MaxRows {
		return fmt.Errorf("%w: %d linhas excedem o limite de %d", ErrOnboardingTooLarge, len(plan.Rows), limites.MaxRows)
	}
	if len(plan.Origin) > limites.MaxOriginBytes {
		return fmt.Errorf("%w: origem com %d bytes excede o limite de %d", ErrOnboardingTooLarge, len(plan.Origin), limites.MaxOriginBytes)
	}
	total := int64(len(plan.Origin))
	for _, linha := range plan.Rows {
		if len(linha.Name) > limites.MaxNameBytes {
			return fmt.Errorf("%w: nome do SKU %q com %d bytes excede o limite de %d", ErrOnboardingTooLarge,
				linha.SKU, len(linha.Name), limites.MaxNameBytes)
		}
		total += int64(len(linha.SKU) + len(linha.Name) + len(linha.Currency))
		if total > limites.MaxTotalBytes {
			return fmt.Errorf("%w: payload com mais de %d bytes", ErrOnboardingTooLarge, limites.MaxTotalBytes)
		}
	}
	return nil
}

// estoqueInicial é o saldo pendente de gravação (E-364): decidido na varredura, gravado depois que o produto existe.
type estoqueInicial struct {
	sku        string
	quantidade int64
}
