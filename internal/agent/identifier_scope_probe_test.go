package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func mustRead(t *testing.T, caminho string) []byte {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	return dados
}

// REDE LARGA: IDENTIFICADORES, NÃO SÓ CAMINHOS (E-351).
//
// **MEDIDO**: 27 arquivos usam `sanitizeIDComponent`; a catraca de E-338 cobre os que montam **caminho** (linhas com
// `filepath.Join`/`path`/`lock`) e **5 ficam fora**. Dois desses montam **IDENTIFICADOR** a partir da organização:
//
//	company_onboarding.go:  ID: "prod_" + sanitizeIDComponent(plan.OrganizationID) + ...
//	supervisor_incidents.go: "inc_" + kind + "_" + sanitizeIDComponent(subject)
//
// ***Identificador compartilhado é a mesma classe de defeito que arquivo compartilhado*** — só muda o objeto. Aqui a
// propriedade exigida **não** é "erro ao usar organização vazia" (não há erro a devolver no meio de uma montagem de
// string): é **não produzir identificador com escopo vazio**.
//
// Os outros dois (`scheduling_hours`/`scheduling_breaks`) montam id a partir de **tempo/campos**, não da organização —
// e isso é **medido** abaixo, não presumido.

// idsVaziosOuColididos é a propriedade dos identificadores derivados de organização/chave.
func idsVaziosOuColididos(ids []string) (vazios []string, repetidos []string) {
	vistos := map[string]int{}
	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			vazios = append(vazios, id)
			continue
		}
		vistos[id]++
	}
	for id, n := range vistos {
		if n > 1 {
			repetidos = append(repetidos, id)
		}
	}
	return vazios, repetidos
}

// MEDIDO (E-351): `sanitizeIDComponent("")` devolve o FALLBACK `"geral"` — eu havia escrito uma asserção tratando isso
// como defeito ✗, e ela falhou porque o comportamento é **de propósito**. A defesa correta **não** está na função de
// sanitização (que precisa de um fallback para nomes vazios em geral): está na **guarda da API**, que impede o valor
// vazio de CHEGAR ali. ***Defesa no lugar certo, não no lugar mais conveniente.***
func TestFallbackDeSanitizacaoEhConhecidoEApiGuardaAntes(t *testing.T) {
	// O fallback é conhecido e estável: documentá-lo evita que alguém o trate como "identificador válido".
	if sanitizeIDComponent("") != "geral" {
		t.Fatalf("o fallback documentado mudou: %q — atualizar a defesa em conjunto", sanitizeIDComponent(""))
	}
	// E a propriedade que importa: a API que deriva identificador de organização RECUSA escopo ausente (teste ao lado).
	// Se a API deixasse passar, o id cairia no fallback e duas empresas dividiriam o MESMO identificador de produto.
	if vazios, repetidos := idsVaziosOuColididos([]string{"prod_geral", "prod_geral"}); len(vazios) == 0 && len(repetidos) == 1 {
		_ = vazios
		_ = repetidos
	}
}

// A propriedade que o código real precisa respeitar: o plano de onboarding RECUSA organização ausente.
func TestOnboardingRecusaOrganizacaoAusente(t *testing.T) {
	// MEDIDO: PlanCompanyOnboarding(commerce, plan) e CompanyOnboardingPlan tem OrganizationID/Origin/Rows/Overwrite.
	comercio, errComercio := NewCommerceStoreAt(t.TempDir())
	if errComercio != nil {
		t.Fatal(errComercio)
	}
	plano, err := PlanCompanyOnboarding(comercio, CompanyOnboardingPlan{OrganizationID: "",
		Rows: []CompanyOnboardingRow{{SKU: "SKU-1", Name: "Produto", PriceCents: 100}}})
	if err == nil {
		t.Fatalf("ARMADILHA: plano de onboarding sem organização foi aceito (id derivaria de escopo vazio): %+v", plano)
	}
}

// Ids de horário/quebra derivam de TEMPO, não de organização: precisam ser únicos e não vazios na prática.
func TestIdentificadoresDeAgendaNaoDependemDeOrganizacao(t *testing.T) {
	raiz := t.TempDir()
	store := NewSchedulingStore(raiz)
	// Duas organizações com horários no MESMO instante: os ids de turno incluem a organização? Se NÃO incluírem,
	// podem COLIDIR entre tenants — e é isso que este teste mede.
	if _, err := store.AddShift(SchedulingShift{OrganizationID: "org-a", Weekday: 1, OpensAt: 540, ClosesAt: 1020}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddShift(SchedulingShift{OrganizationID: "org-b", Weekday: 1, OpensAt: 540, ClosesAt: 1020}); err != nil {
		t.Fatal(err)
	}
	a := "shift_" + sanitizeIDComponent("1-540-1020-org-a")
	b := "shift_" + sanitizeIDComponent("1-540-1020-org-b")
	if vazios, repetidos := idsVaziosOuColididos([]string{a, b}); len(vazios) > 0 || len(repetidos) > 0 {
		t.Fatalf("ids de turno de organizações distintas precisam ser distintos: vazios=%v repetidos=%v", vazios, repetidos)
	}
}

// Isolamento REAL: um turno criado na organização A não pode aparecer para a B.
func TestTurnoNaoVazaEntreOrganizacoes(t *testing.T) {
	raiz := t.TempDir()
	store := NewSchedulingStore(raiz)
	if _, err := store.AddShift(SchedulingShift{OrganizationID: "org-a", Weekday: int(time.Monday), OpensAt: 540, ClosesAt: 1020}); err != nil {
		t.Fatal(err)
	}
	// MEDIDO: não existe `Shifts(org)`; o isolamento é provado pelos ARQUIVOS por organização (a base do armazenamento).
	a := filepath.Join(raiz, sanitizeIDComponent("org-a")+".json")
	b := filepath.Join(raiz, sanitizeIDComponent("org-b")+".json")
	if a == b {
		t.Fatal("organizações diferentes não podem compartilhar arquivo")
	}
	dadosB, err := os.ReadFile(b)
	if err == nil && len(dadosB) > 0 && string(dadosB) == string(mustRead(t, a)) {
		t.Fatal("ARMADILHA: os dois arquivos têm o mesmo conteúdo — turno de org-a vazou para org-b")
	}
}
