package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Company Factory (estágio 23): criação e operação inicial ASSISTIDA de uma empresa em
// sandbox, com **artefatos reais** e checklist honesto.
//
// O que já existia: `CompanyStore.CreateRequest` criava o **registro** da empresa. Faltava
// o que o estágio pede de verdade — **artefatos** e uma operação inicial verificável.
//
// DECISÕES DELIBERADAS:
//   - **NÃO SE INVENTA CLIENTE**: a fábrica cria catálogo, marca, site, plano e checklist,
//     mas **não** cria contatos no CRM. Popular o CRM com clientes fictícios seria
//     exatamente "dados falsos" que o estágio proíbe; há teste travando CRM vazio após o
//     lançamento;
//   - **PREÇO DERIVADO DE CUSTO E MARGEM, com aritmética conferida**: margem de 100% ou
//     mais é recusada (não existe preço finito), e o preço é arredondado para centavos uma
//     única vez, sem perder o total esperado;
//   - **O QUE DEPENDE DE AUTORIDADE É DECLARADO BLOQUEADO**: CNPJ, contrato social,
//     contabilidade e domínio próprio aparecem no checklist como
//     `BLOQUEADO_BY_EXTERNAL_DEPENDENCY` com o **procedimento exato** de desbloqueio. A
//     fábrica **não** emite dado fiscal nem registro legal;
//   - **ARTEFATOS SÃO ARQUIVOS REAIS**: marca, site e plano são gravados em disco sob a
//     raiz de artefatos da empresa, e o relatório aponta o caminho — um "artefato" que só
//     existe na memória não é entregável;
//   - **RELATÓRIO COM EVIDÊNCIA POR ITEM**, como na certificação E2E.

const (
	// CompanyArtifactBrand é o kit de marca; CompanyArtifactSite é a página inicial;
	// CompanyArtifactPricing é a tabela de preços; CompanyArtifactPlan é o plano inicial.
	CompanyArtifactBrand   = "marca"
	CompanyArtifactSite    = "site"
	CompanyArtifactPricing = "precificacao"
	CompanyArtifactPlan    = "plano"

	// Status possíveis de um item do checklist.
	CompanyChecklistDone    = "FEITO"
	CompanyChecklistBlocked = "BLOQUEADO_BY_EXTERNAL_DEPENDENCY"
	CompanyChecklistPending = "NAO_FEITO"
)

var (
	ErrCompanyFactoryInvalid = errors.New("company factory request invalid")
	ErrCompanyPricingInvalid = errors.New("pricing margin invalid")
)

// CompanyPricingInput descreve um produto inicial da empresa.
type CompanyPricingInput struct {
	SKU           string `json:"sku"`
	Name          string `json:"name"`
	CostCents     int64  `json:"cost_cents"`
	MarginPercent int64  `json:"margin_percent"`
	InitialStock  int64  `json:"initial_stock"`
}

// CompanyFactoryRequest é o pedido de lançamento da empresa em sandbox.
type CompanyFactoryRequest struct {
	OrganizationID string                `json:"organization_id"`
	Company        CompanyCreateRequest  `json:"company"`
	Tagline        string                `json:"tagline"`
	Palette        []string              `json:"palette"`
	Products       []CompanyPricingInput `json:"products"`
}

// CompanyChecklistItem é um item do checklist de abertura, com motivo quando bloqueado.
type CompanyChecklistItem struct {
	Item     string `json:"item"`
	Status   string `json:"status"`
	Evidence string `json:"evidence,omitempty"`
	// Unblock descreve o PROCEDIMENTO EXATO quando o item depende de terceiro.
	Unblock string `json:"unblock_procedure,omitempty"`
}

// CompanyArtifact é um artefato gravado em disco.
type CompanyArtifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	// Bytes é o tamanho real do arquivo, para o relatório não afirmar sem medir.
	Bytes int64 `json:"bytes"`
}

// CompanyFactoryReport é o resultado do lançamento.
type CompanyFactoryReport struct {
	CompanyID      string                 `json:"company_id"`
	OrganizationID string                 `json:"organization_id"`
	LaunchedAt     time.Time              `json:"launched_at"`
	Company        Company                `json:"company"`
	Products       []CommerceProduct      `json:"products"`
	Artifacts      []CompanyArtifact      `json:"artifacts"`
	Checklist      []CompanyChecklistItem `json:"checklist"`
	// CatalogValue é a soma do valor de tabela do estoque inicial, em centavos.
	CatalogValueCents int64 `json:"catalog_value_cents"`
	// CustomersCreated é sempre 0: a fábrica NÃO inventa clientes.
	CustomersCreated int `json:"customers_created"`
}

// priceFromCostAndMargin calcula o preço de venda a partir de custo e margem.
//
// Margem é percentual sobre o CUSTO: preço = custo × (1 + margem/100). Margem de 100% ou
// mais é recusada porque o preço deixaria de ser finito em em comparação com a margem, e custo
// negativo não existe.
func priceFromCostAndMargin(costCents, marginPercent int64) (int64, error) {
	if costCents < 0 {
		return 0, fmt.Errorf("%w: custo não pode ser negativo", ErrCompanyPricingInvalid)
	}
	if marginPercent < 0 || marginPercent >= 100 {
		return 0, fmt.Errorf("%w: margem precisa estar entre 0 e 99 por cento (recebido %d)",
			ErrCompanyPricingInvalid, marginPercent)
	}
	// Arredonda UMA vez, para não acumular erro linha a linha.
	return (costCents*(100+marginPercent) + 50) / 100, nil
}

// artifactsRoot devolve a raiz de artefatos da empresa.
func (s *CompanyStore) artifactsRoot(companyID string) string {
	if s.root == "" {
		return ""
	}
	return filepath.Join(s.root, "artifacts", sanitizeIDComponent(companyID))
}

// writeTextFile grava texto de forma ATÔMICA: escreve num temporário e renomeia, então
// um artefato nunca aparece pela metade se a escrita falhar no meio.
func writeTextFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, []byte(content), 0o600); err != nil {
		return err
	}
	if err := os.Rename(temp, path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}

// LaunchCompany cria a empresa e MATERIALIZA os artefatos da operação inicial.
func (r *Runtime) LaunchCompany(request CompanyFactoryRequest, now time.Time) (CompanyFactoryReport, error) {
	report := CompanyFactoryReport{LaunchedAt: now, OrganizationID: consentScope(request.OrganizationID)}
	if r == nil {
		return report, fmt.Errorf("runtime indisponível")
	}
	company, err := r.CompanyStore().CreateRequest(request.Company, request.OrganizationID)
	if err != nil {
		return report, err
	}
	report.CompanyID = company.ID
	report.Company = company

	// 1. PLANO INICIAL: artefato em disco com o que foi pedido, para revisão humana.
	if root := r.CompanyStore().artifactsRoot(company.ID); root != "" {
		if err := os.MkdirAll(root, 0o700); err != nil {
			return report, err
		}
		plan := strings.Builder{}
		fmt.Fprintf(&plan, "# Plano inicial — %s\n\n", company.Name)
		fmt.Fprintf(&plan, "- Missão: %s\n", company.Mission)
		fmt.Fprintf(&plan, "- Posicionamento: %s\n", company.Positioning)
		fmt.Fprintf(&plan, "- Modelo de negócio: %s\n", company.BusinessModel)
		fmt.Fprintf(&plan, "- Público: %s\n", company.TargetAudience)
		fmt.Fprintf(&plan, "- Oferta: %s\n", company.Offer)
		fmt.Fprintf(&plan, "- Moeda: %s\n", company.Currency)
		plan.WriteString("\n## Itens que dependem de terceiros (não são emitidos por esta fábrica)\n")
		plan.WriteString("- CNPJ e registro legal\n- Contrato social\n- Contabilidade\n- Domínio próprio e certificado\n")
		planPath := filepath.Join(root, "plano-inicial.md")
		if err := writeTextFile(planPath, plan.String()); err != nil {
			return report, err
		}
		if info, err := os.Stat(planPath); err == nil {
			report.Artifacts = append(report.Artifacts, CompanyArtifact{Kind: CompanyArtifactPlan, Path: planPath, Bytes: info.Size()})
		}
	}

	// 2. MARCA: kit com tagline e paleta declaradas pelo operador.
	if root := r.CompanyStore().artifactsRoot(company.ID); root != "" {
		brand := strings.Builder{}
		fmt.Fprintf(&brand, "# Marca — %s\n\n%s\n\n", company.Name, strings.TrimSpace(request.Tagline))
		brand.WriteString("## Paleta\n")
		for _, color := range request.Palette {
			fmt.Fprintf(&brand, "- %s\n", strings.TrimSpace(color))
		}
		brand.WriteString("\n## Personalidade\n")
		fmt.Fprintf(&brand, "Posicionamento declarado: %s\n", company.Positioning)
		brandPath := filepath.Join(root, "marca.md")
		if err := writeTextFile(brandPath, brand.String()); err != nil {
			return report, err
		}
		if info, err := os.Stat(brandPath); err == nil {
			report.Artifacts = append(report.Artifacts, CompanyArtifact{Kind: CompanyArtifactBrand, Path: brandPath, Bytes: info.Size()})
		}
	}

	// 3. CATÁLOGO COM PREÇO DERIVADO (comércio real) — sem cliente fictício nenhum.
	pricingRows := make([]string, 0, len(request.Products))
	for _, input := range request.Products {
		price, err := priceFromCostAndMargin(input.CostCents, input.MarginPercent)
		if err != nil {
			return report, err
		}
		product, err := r.Commerce().UpsertProduct(CommerceProduct{
			OrganizationID: request.OrganizationID, SKU: input.SKU, Name: input.Name,
			PriceCents: price, Currency: company.Currency, Active: true,
		}, now)
		if err != nil {
			return report, err
		}
		if input.InitialStock > 0 {
			if _, err := r.Commerce().RecordStock(request.OrganizationID, input.SKU, input.InitialStock,
				CommerceMovementEntry, "carga inicial da fabrica", now); err != nil {
				return report, err
			}
		}
		report.Products = append(report.Products, product)
		report.CatalogValueCents += price * input.InitialStock
		pricingRows = append(pricingRows, fmt.Sprintf("| %s | %s | %d | %d%% | %d | %d |",
			product.SKU, product.Name, input.CostCents, input.MarginPercent, product.PriceCents, input.InitialStock))
	}

	// 4. PRECIFICAÇÃO: artefato com a CONTA à vista, para o operador conferir.
	if root := r.CompanyStore().artifactsRoot(company.ID); root != "" {
		table := strings.Builder{}
		fmt.Fprintf(&table, "# Precificação — %s (%s)\n\n", company.Name, company.Currency)
		table.WriteString("| SKU | Produto | Custo (centavos) | Margem | Preço (centavos) | Estoque inicial |\n")
		table.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, row := range pricingRows {
			table.WriteString(row + "\n")
		}
		fmt.Fprintf(&table, "\nValor de tabela do estoque inicial: %d centavos.\n", report.CatalogValueCents)
		table.WriteString("\nFórmula: preço = custo × (1 + margem/100), arredondado uma única vez.\n")
		table.WriteString("\n## Não incluído\n\n- Impostos e obrigações fiscais (dependem de regime e contador).\n- Frete e devoluções.\n")
		pricingPath := filepath.Join(root, "precificacao.md")
		if err := writeTextFile(pricingPath, table.String()); err != nil {
			return report, err
		}
		if info, err := os.Stat(pricingPath); err == nil {
			report.Artifacts = append(report.Artifacts, CompanyArtifact{Kind: CompanyArtifactPricing, Path: pricingPath, Bytes: info.Size()})
		}
	}

	// 5. SITE: página inicial estática com os dados REAIS da empresa.
	if root := r.CompanyStore().artifactsRoot(company.ID); root != "" {
		page := strings.Builder{}
		page.WriteString("<!doctype html>\n<html lang=\"pt-BR\">\n<head>\n<meta charset=\"utf-8\">\n")
		fmt.Fprintf(&page, "<title>%s</title>\n", company.Name)
		page.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n</head>\n<body>\n")
		fmt.Fprintf(&page, "<h1>%s</h1>\n", company.Name)
		if strings.TrimSpace(request.Tagline) != "" {
			fmt.Fprintf(&page, "<p>%s</p>\n", strings.TrimSpace(request.Tagline))
		}
		fmt.Fprintf(&page, "<p>%s</p>\n", company.Offer)
		fmt.Fprintf(&page, "<p>Público: %s</p>\n", company.TargetAudience)
		if len(report.Products) > 0 {
			page.WriteString("<h2>Produtos</h2>\n<ul>\n")
			for _, product := range report.Products {
				fmt.Fprintf(&page, "<li>%s — %d centavos</li>\n", product.Name, product.PriceCents)
			}
			page.WriteString("</ul>\n")
		}
		page.WriteString("</body>\n</html>\n")
		sitePath := filepath.Join(root, "index.html")
		if err := writeTextFile(sitePath, page.String()); err != nil {
			return report, err
		}
		if info, err := os.Stat(sitePath); err == nil {
			report.Artifacts = append(report.Artifacts, CompanyArtifact{Kind: CompanyArtifactSite, Path: sitePath, Bytes: info.Size()})
		}
	}

	// 6. CHECKLIST com o que foi FEITO e o que depende de terceiros, com procedimento.
	report.Checklist = append(report.Checklist,
		CompanyChecklistItem{Item: "empresa registrada no Hades", Status: CompanyChecklistDone, Evidence: "company=" + company.ID},
		CompanyChecklistItem{
			Item: "catálogo com preço derivado de custo e margem", Status: CompanyChecklistDone,
			Evidence: fmt.Sprintf("%d produtos, %d centavos de valor de tabela", len(report.Products), report.CatalogValueCents),
		},
		CompanyChecklistItem{
			Item: "marca, site e plano inicial gravados como arquivos", Status: CompanyChecklistDone,
			Evidence: fmt.Sprintf("%d artefatos", len(report.Artifacts)),
		},
		CompanyChecklistItem{
			Item: "CNPJ e registro legal", Status: CompanyChecklistBlocked,
			Unblock: "exige dados reais da empresa e autoridade para registrar; fornecer os dados e executar o registro no órgão competente",
		},
		CompanyChecklistItem{
			Item: "contrato social e documentos societários", Status: CompanyChecklistBlocked,
			Unblock: "exige definição societária e assinatura das partes; a fábrica não emite documento legal",
		},
		CompanyChecklistItem{
			Item: "contabilidade e obrigações fiscais", Status: CompanyChecklistBlocked,
			Unblock: "depende de regime tributário e contador responsável; nenhum dado fiscal é gerado aqui",
		},
		CompanyChecklistItem{
			Item: "domínio próprio com certificado", Status: CompanyChecklistBlocked,
			Unblock: "exige registro de domínio e emissão de certificado por terceiro",
		},
	)
	// Os clientes NÃO são criados: registrar isso no próprio relatório evita que alguém
	// interprete a empresa como "com clientes".
	report.Checklist = append(report.Checklist, CompanyChecklistItem{
		Item: "clientes cadastrados", Status: CompanyChecklistPending,
		Evidence: "0 (a fábrica não inventa clientes; o CRM fica vazio até existir contato real)",
	})

	sort.SliceStable(report.Products, func(i, j int) bool { return report.Products[i].SKU < report.Products[j].SKU })
	return report, nil
}

// CompanyFactoryReportFor devolve o relatório de uma empresa já lançada, reconstruído a
// partir do estado real (empresa, catálogo e arquivos em disco).
func (r *Runtime) CompanyFactoryReportFor(organizationID, companyID string) (CompanyFactoryReport, error) {
	report := CompanyFactoryReport{OrganizationID: consentScope(organizationID)}
	company, err := r.CompanyStore().Get(companyID)
	if err != nil {
		return report, err
	}
	if consentScope(company.OrganizationID) != consentScope(organizationID) {
		// NÃO INFERÊNCIA (E-190): devolver `ErrCompanyNotFound` COM o ID só aqui criava assimetria observável — a
		// recusa de um ID inexistente era `"company not found"` e a de uma empresa EXISTENTE de outro tenant era
		// `"company not found: <id>"`. Ecoar o ID apenas no caso que existe entrega a existência. O chamador já sabe
		// o ID que perguntou, então a mensagem não precisa carregá-lo.
		return report, ErrCompanyNotFound
	}
	report.CompanyID = company.ID
	report.Company = company
	// Só o catálogo da PRÓPRIA organização entra no relatório.
	report.Products = append(report.Products, r.Commerce().Products(organizationID)...)
	root := r.CompanyStore().artifactsRoot(company.ID)
	if root != "" {
		for _, kind := range []struct{ kind, name string }{
			{CompanyArtifactPlan, "plano-inicial.md"},
			{CompanyArtifactBrand, "marca.md"},
			{CompanyArtifactPricing, "precificacao.md"},
			{CompanyArtifactSite, "index.html"},
		} {
			path := filepath.Join(root, kind.name)
			if info, err := os.Stat(path); err == nil {
				report.Artifacts = append(report.Artifacts, CompanyArtifact{Kind: kind.kind, Path: path, Bytes: info.Size()})
			}
		}
	}
	return report, nil
}
