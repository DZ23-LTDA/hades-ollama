package agent

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

// PROVA DE NÃO INFERÊNCIA ENTRE TENANTS (estágio 18, E-189).
//
// "Rejeitar cross-tenant" NÃO é a mesma coisa que "não deixar inferir". Se o Tenant B recebe uma resposta DIFERENTE
// para um recurso que existe em A e para um recurso que não existe, ele aprende — por diferença — que aquele ID
// existe em algum lugar. Isso é um ORÁCULO DE ENUMERAÇÃO: dá para varrer IDs e descobrir a existência de recursos de
// outros tenants sem acessar nenhum dado. É o vetor que a seção 18 chama de "prova de não inferência".
//
// Estes testes medem o OBSERVÁVEL (erro, forma, tamanho e ordem de grandeza de tempo), não a intenção do código.

type resultadoObservavel struct {
	erro       error
	mensagem   string
	itens      int
	durou      time.Duration
	erroEhNulo bool
}

func observarLeitura(store *CompanyStore, companyID, organização string, repeticoes int) resultadoObservavel {
	var ultimo resultadoObservavel
	var maior time.Duration
	for range repeticoes {
		inicio := time.Now()
		histórico, err := store.TelAgentHistoryForOrganization(companyID, organização, 10)
		decorrido := time.Since(inicio)
		if decorrido > maior {
			maior = decorrido
		}
		ultimo = resultadoObservavel{
			erro:       err,
			itens:      len(histórico),
			erroEhNulo: err == nil,
			durou:      decorrido,
		}
		if err != nil {
			ultimo.mensagem = err.Error()
		}
	}
	ultimo.durou = maior
	return ultimo
}

// formatoDaMensagem troca os NÚMEROS e os IDs da mensagem por marcadores: o que importa é a FORMA do erro, não o
// literal. Duas negações com a mesma forma não permitem inferência; duas com formas diferentes permitem.
func formatoDaMensagem(mensagem string) string {
	trocado := mensagem
	for _, palavra := range strings.Fields(mensagem) {
		if strings.ContainsAny(palavra, "0123456789_") && len(palavra) > 3 {
			trocado = strings.ReplaceAll(trocado, palavra, "<id>")
		}
	}
	return trocado
}

func TestCrossTenantRefusalDoesNotRevealExistence(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	daOrgA, err := store.Create(Company{OrganizationID: "org-a", Name: "Empresa da Org A"})
	if err != nil {
		t.Fatal(err)
	}
	// Histórico real no tenant dono: se o dono não tem nada, o caso não mede nada.
	if _, _, err := store.ExecuteTelAgent(daOrgA.ID, "org-a", "user-a", TelAgentRequest{Message: "estado", Operation: "report.read"}); err != nil {
		t.Fatal(err)
	}

	// (2) recurso EXISTE, mas é de outro tenant.
	existenteDeOutro := observarLeitura(store, daOrgA.ID, "org-b", 5)
	// (3) recurso NÃO existe.
	inexistente := observarLeitura(store, "cmp_que_nao_existe_em_lugar_nenhum", "org-b", 5)
	// (1) controle positivo: o dono lê.
	doDono := observarLeitura(store, daOrgA.ID, "org-a", 5)

	if existenteDeOutro.erroEhNulo {
		t.Fatal("leitura cross-tenant não pode ter sucesso")
	}
	if !doDono.erroEhNulo {
		t.Fatalf("o dono precisa conseguir ler o próprio histórico: %v", doDono.erro)
	}

	formaExistente := formatoDaMensagem(existenteDeOutro.mensagem)
	formaInexistente := formatoDaMensagem(inexistente.mensagem)
	t.Logf("negação para recurso de OUTRO tenant: %q", formaExistente)
	t.Logf("negação para recurso INEXISTENTE:    %q", formaInexistente)
	t.Logf("tempo (máximo de 5 tentativas): outro=%v inexistente=%v", existenteDeOutro.durou, inexistente.durou)

	// A PROPRIEDADE: as duas negações precisam ser indistinguíveis para o tenant que não é dono.
	if formaExistente != formaInexistente {
		t.Fatalf(
			"ORÁCULO DE ENUMERAÇÃO: a negação cross-tenant (%q) difere da negação para recurso inexistente (%q) — "+
				"o tenant B consegue inferir que o ID existe", formaExistente, formaInexistente)
	}
	if existenteDeOutro.itens != inexistente.itens {
		t.Fatalf("itens vazados na negação: outro=%d inexistente=%d", existenteDeOutro.itens, inexistente.itens)
	}
}

// Um tenant não pode inferir NADA sobre o volume de dados de outro: criar recursos em A não pode mudar a leitura de
// B (nem em quantidade, nem em sucesso).
func TestListingDoesNotRevealAnotherTenantsVolume(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	daOrgB, err := store.Create(Company{OrganizationID: "org-b", Name: "Empresa da Org B"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.ExecuteTelAgent(daOrgB.ID, "org-b", "user-b", TelAgentRequest{Message: "estado", Operation: "report.read"}); err != nil {
		t.Fatal(err)
	}
	antesB, err := store.TelAgentHistoryForOrganization(daOrgB.ID, "org-b", 100)
	if err != nil {
		t.Fatal(err)
	}

	// A enche o próprio histórico.
	daOrgA, err := store.Create(Company{OrganizationID: "org-a", Name: "Empresa da Org A"})
	if err != nil {
		t.Fatal(err)
	}
	for range 25 {
		if _, _, err := store.ExecuteTelAgent(daOrgA.ID, "org-a", "user-a", TelAgentRequest{Message: "estado", Operation: "report.read"}); err != nil {
			t.Fatal(err)
		}
	}

	depoisB, err := store.TelAgentHistoryForOrganization(daOrgB.ID, "org-b", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(antesB) != len(depoisB) {
		t.Fatalf("o volume do tenant A mudou a leitura do tenant B: antes=%d depois=%d", len(antesB), len(depoisB))
	}
	for _, item := range depoisB {
		if item.OrganizationID != "" && item.OrganizationID != "org-b" {
			t.Fatalf("leitura de B trouxe item de %q", item.OrganizationID)
		}
	}
	// E a tentativa de A ler B continua negada depois de A crescer.
	if _, err := store.TelAgentHistoryForOrganization(daOrgB.ID, "org-a", 10); err == nil {
		t.Fatal("depois de crescer, A não pode passar a ler B")
	}
}

// O TESTE DO TESTE: a comparação de forma precisa ACUSAR uma negação que difere. Sem isto, uma asserção de
// indistinguibilidade pode passar por acidente (por exemplo, comparando vazio com vazio).
func TestFormatoDeNegacaoDetectaDiferenca(t *testing.T) {
	casos := []struct {
		nome            string
		esquerda        string
		direita         string
		esperaDiferenca bool
	}{
		{"mesma negação", "company cmp_1 not found for organization", "company cmp_2 not found for organization", false},
		{"mismatch versus inexistente", "organization mismatch for company cmp_1", "company cmp_1 not found", true},
		{"vazio versus preenchido", "", "organization mismatch", true},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			diferem := formatoDaMensagem(caso.esquerda) != formatoDaMensagem(caso.direita)
			if diferem != caso.esperaDiferenca {
				t.Fatalf("comparação de forma falhou: esquerda=%q direita=%q diferem=%v esperado=%v",
					caso.esquerda, caso.direita, diferem, caso.esperaDiferenca)
			}
		})
	}
}

// A ordem de grandeza do tempo foi MEDIDA e é registrada; ela NÃO é usada como asserção porque o relógio de um
// runner é ruidoso demais para provar não inferência por timing. Declarar isso é parte da prova: um teste que
// afirmasse "não há diferença de tempo" com este instrumento estaria mentindo.
func TestTempoDeNegacaoEhRegistradoNaoAfirmado(t *testing.T) {
	store, err := NewCompanyStore("")
	if err != nil {
		t.Fatal(err)
	}
	empresa, err := store.Create(Company{OrganizationID: "org-a", Name: "Tempo"})
	if err != nil {
		t.Fatal(err)
	}
	existente := observarLeitura(store, empresa.ID, "org-b", 50)
	inexistente := observarLeitura(store, "cmp_inexistente_para_tempo", "org-b", 50)
	razao := float64(existente.durou) / math.Max(float64(inexistente.durou), 1)
	t.Logf("razão de tempo (outro/inexistente) = %.2f — registrado, NÃO afirmado: o relógio do runner não prova "+
		"não inferência por timing", razao)
	if existente.erro == nil || !errors.Is(existente.erro, ErrTelAgentOrganizationMismatch) {
		t.Fatalf("a negação cross-tenant precisa ser o erro declarado de mismatch: %v", existente.erro)
	}
}
