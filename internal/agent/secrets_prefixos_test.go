package agent

import (
	"strings"
	"testing"
)

// CATRACA DO PRÉ-FILTRO POR LITERAIS (E-559 / E-563).
//
// POR QUE ESTE TESTE EXISTE: o E-559 deu ao `ScanDLP` um pré-filtro que salta um
// padrão quando nenhum dos seus literais aparece no texto. Isso é seguro ENQUANTO o
// literal declarado for uma condição NECESSÁRIA do padrão — e deixou de ser seguro
// no dia em que eu escrevi `ContainsAny(text, ":")` para um padrão que exige `[:=]`:
// o `api_key=abcdefghijklmnop` deixou de ser detetado e três testes de segurança
// falharam. ***Uma condição necessária errada é pior que nenhuma: silencia a deteção,
// e a ausência PARECE um texto limpo.***
//
// A auditoria de segurança independente verificou as 12 entradas à mão e não achou
// literal declarado que não fosse obrigatório — mas notou que NADA o travava. Este
// teste trava. Ele não lê a struct interna: ele mede o COMPORTAMENTO, que é o que
// importa (um pré-filtro errado manifesta-se por um verdadeiro positivo que
// desaparece, não por um campo mal preenchido).

// amostrasDLP é um verdadeiro positivo por padrão. Se o pré-filtro silenciar
// qualquer um deles, o teste falha e nomeia a forma.
var amostrasDLP = []struct {
	nome    string
	amostra string
}{
	{"private_key", "-----BEGIN RSA PRIVATE KEY-----\nMIIabc\n-----END RSA PRIVATE KEY-----"},
	{"github_token (ghp_)", "ghp_" + strings.Repeat("a1B2c3D4e5", 2)},
	{"github_token (github_pat_)", "github_pat_" + strings.Repeat("a1B2c3D4e5", 2)},
	{"openrouter_token", "sk-or-v1-" + strings.Repeat("a1B2c3D4e5", 2)},
	{"openai_token", "sk-" + strings.Repeat("a1B2c3D4e5", 2)},
	{"xai_token", "xai-" + strings.Repeat("a1B2c3D4e5", 2)},
	{"aws_access_key", "AKIA" + strings.Repeat("A1B2C3D4", 2)},
	{"slack_token", "xoxb-" + strings.Repeat("a1b2c3d4", 2)},
	{"bearer_token", "Bearer " + strings.Repeat("a1B2c3D4", 2)},
	{"basic_authorization", "Authorization: Basic " + strings.Repeat("aB1cD2", 2)},
	{"cookie_credential (session=)", "session=a1b2c3d4e5f6"},
	{"cookie_credential (cookie:)", "Cookie: sid=a1b2c3d4e5f6"},
	{"jwt_token", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
	{"credential_assignment (api_key=)", "api_key=abcdefghijklmnop"},
	{"credential_assignment (token:)", "token: abcdefghijklmnop"},
	{"credential_assignment (apikey=)", "apikey=abcdefghijklmnop"},
	{"credential_assignment (api-key=)", "api-key=abcdefghijklmnop"},
	{"credential_assignment (password=)", "password=abcdefghijklmnop"},
	{"credential_assignment (refresh_token=)", "refresh_token=abcdefghijklmnop"},
}

func TestPreFiltroDLPNaoSilenciaVerdadeiroPositivo(t *testing.T) {
	for _, caso := range amostrasDLP {
		achados := ScanDLP(caso.amostra)
		if len(achados) == 0 {
			t.Errorf("SILENCIADO: %s não foi detetado — o pré-filtro está a saltar um padrão que casa.\n  amostra: %q",
				caso.nome, caso.amostra)
		}
	}
}

// A CAIXA ALTA É O CASO QUE MAIS PROVAVELMENTE QUEBRA: o pré-filtro compara o texto
// em MINÚSCULAS contra literais em minúsculas, e é isso que permite detetar `GHP_` e
// `API_KEY` num padrão `(?i)`. Se alguém "optimizar" a comparação para o texto
// original, estes desaparecem.
func TestPreFiltroDLPDetetaLiteraisEmCaixaAlta(t *testing.T) {
	caixaAlta := []string{
		"GHP_" + strings.Repeat("a1B2c3D4e5", 2),
		"BEARER " + strings.Repeat("a1B2c3D4", 2),
		"API_KEY=abcdefghijklmnop",
		"PASSWORD=abcdefghijklmnop",
		"COOKIE: SESSION=a1b2c3d4e5f6",
	}
	for _, amostra := range caixaAlta {
		if len(ScanDLP(amostra)) == 0 {
			t.Errorf("SILENCIADO em caixa alta: %q", amostra)
		}
	}
}

// O CONTROLO NEGATIVO: um pré-filtro que devolvesse SEMPRE deteção também estaria
// errado (e caro). Texto sem nenhum dos literais não pode gerar achado.
func TestPreFiltroDLPNaoInventaDeteccao(t *testing.T) {
	limpos := []string{
		"",
		"o relatório de vendas cresceu 12 por cento no trimestre",
		"função soma(a, b) { return a + b }",
		"a chave do problema é a ordem das guardas, não o literal",
		"sk- curto demais para ser token",
	}
	for _, texto := range limpos {
		if achados := ScanDLP(texto); len(achados) > 0 {
			t.Errorf("falso positivo em %q: %v", texto, achados)
		}
	}
}
