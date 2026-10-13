package agent

import (
	"fmt"
	"time"
)

// ORÇAMENTO ACUMULADO DE ONBOARDING (E-359).
//
// **MEDIDO antes** (declarado como limite em E-353): o teto era **por plano**. Três planos de 4000 linhas cada passavam
// um a um e o resultado eram **12000 linhas** carregadas — *o limite existia no papel e não no sistema*.
//
// AQUI O ORÇAMENTO É MEDIDO CONTRA O **ESTADO REAL**, não contra um contador paralelo: o `CommerceStore` já guarda o
// que foi importado, e `Products(org)` é a soma verdadeira. ***Contador paralelo é a segunda fonte de verdade que
// alguém vai esquecer de atualizar; o estado real não tem como divergir de si mesmo.***
//
// O limite acumulado é o MESMO `MaxRows`/`MaxTotalBytes` declarados, aplicados ao **total depois** da importação:
// o sistema recusa antes de gravar, e diz por que.

// MedirAcumuladoDoOnboarding devolve quantas linhas e quantos bytes a organização JÁ tem.
func MedirAcumuladoDoOnboarding(commerce *CommerceStore, organizationID string) (linhas int, bytes int) {
	if commerce == nil {
		return 0, 0
	}
	for _, produto := range commerce.Products(organizationID) {
		linhas++
		bytes += len(produto.SKU) + len(produto.Name) + len(produto.Currency)
	}
	return linhas, bytes
}

// ValidarOrcamentoAcumulado recusa quando o plano, SOMADO ao que já existe, passa dos limites declarados.
func ValidarOrcamentoAcumulado(commerce *CommerceStore, plan CompanyOnboardingPlan, limites OnboardingLimits) error {
	limites = limites.normalizar()
	jaLinhas, jaBytes := MedirAcumuladoDoOnboarding(commerce, consentScope(plan.OrganizationID))

	quantasNovas := 0
	bytesNovos := int64(len(plan.Origin))
	for _, linha := range plan.Rows {
		// Linhas que já existem não crescem o acervo: o onboarding as ATUALIZA.
		if _, existente := commerce.ProductBySKU(plan.OrganizationID, linha.SKU); existente {
			continue
		}
		quantasNovas++
		bytesNovos += int64(len(linha.SKU) + len(linha.Name) + len(linha.Currency))
	}

	if jaLinhas+quantasNovas > limites.MaxRows {
		return fmt.Errorf("%w: acervo acumulado chegaria a %d linhas (já tem %d, o plano acrescenta %d; limite %d)",
			ErrOnboardingTooLarge, jaLinhas+quantasNovas, jaLinhas, quantasNovas, limites.MaxRows)
	}
	if int64(jaBytes)+bytesNovos > limites.MaxTotalBytes {
		return fmt.Errorf("%w: acervo acumulado chegaria a %d bytes (já tem %d, o plano acrescenta %d; limite %d)",
			ErrOnboardingTooLarge, int64(jaBytes)+bytesNovos, jaBytes, bytesNovos, limites.MaxTotalBytes)
	}
	return nil
}

// aplicarComOrcamento é o ponto por onde o onboarding passa: valida o volume do plano E o acumulado.
func aplicarComOrcamento(commerce *CommerceStore, plan CompanyOnboardingPlan, agora time.Time) (CompanyOnboardingPreview, CompanyOnboardingRollback, error) {
	limites := DefaultOnboardingLimits()
	if err := ValidarVolumeDoPlano(plan, limites); err != nil {
		return CompanyOnboardingPreview{}, CompanyOnboardingRollback{}, err
	}
	if err := ValidarOrcamentoAcumulado(commerce, plan, limites); err != nil {
		return CompanyOnboardingPreview{}, CompanyOnboardingRollback{}, err
	}
	return aplicarSemOrcamento(commerce, plan, agora)
}
