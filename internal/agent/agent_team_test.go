// Testes das equipas e presets.
//
// Discriminantes: os presets embutidos validam contra o catalogo REAL; nenhum se declara funcional; um
// ciclo e' recusado; uma persona inexistente e' recusada; o que falta aparece dito.
package agent

import (
	"strings"
	"testing"
)

// *Uma equipa que so' lista personas e' um menu, nao uma equipa.*
func TestPresetsEmbutidosValidamContraOCatalogo(t *testing.T) {
	if TeamsCount() == 0 {
		t.Fatal("nenhum preset embutido")
	}
	for _, team := range Teams() {
		v := ValidateTeam(team, CatalogLookup)
		if !v.OK {
			t.Fatalf("%s: preset embutido invalido: %v", team.ID, v.Problems)
		}
		if len(v.MissingPersonas) != 0 {
			t.Fatalf("%s: personas em falta: %v", team.ID, v.MissingPersonas)
		}
	}
}

// *Uma equipe nao e' considerada funcional ate' completar pelo menos uma missao representativa.*
func TestPresetsEmbutidosNaoSeDizemFuncionais(t *testing.T) {
	for _, team := range Teams() {
		if team.Functional {
			t.Fatalf("%s: preset embutido declara-se funcional sem missao provada", team.ID)
		}
		if team.EvidenceOfFunctionality != "" {
			t.Fatalf("%s: preset embutido declara evidencia que nao existe", team.ID)
		}
	}
}

func TestPresetsEmbutidosExplicamCadaPapel(t *testing.T) {
	for _, team := range Teams() {
		for _, r := range team.Roles {
			if strings.TrimSpace(r.Why) == "" {
				t.Fatalf("%s: papel %q nao diz porque foi escolhido", team.ID, r.Role)
			}
		}
	}
}

func TestPresetsEmbutidosDeclaramCondicaoDeParagem(t *testing.T) {
	for _, team := range Teams() {
		if strings.TrimSpace(team.StopCondition) == "" {
			t.Fatalf("%s: sem condicao de paragem", team.ID)
		}
	}
}

func TestPresetsOrdemDeterministica(t *testing.T) {
	a := Teams()
	b := Teams()
	if len(a) != len(b) {
		t.Fatal("tamanhos differentes")
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("ordem mudou: %q vs %q", a[i].ID, b[i].ID)
		}
	}
	for i := 1; i < len(a); i++ {
		if a[i-1].ID >= a[i].ID {
			t.Fatalf("ordem por ID violada: %q antes de %q", a[i-1].ID, a[i].ID)
		}
	}
}

func TestTeamsDevolveCopia(t *testing.T) {
	primeira := Teams()
	if len(primeira) == 0 {
		t.Fatal("sem presets")
	}
	primeira[0].Name = "ALTERADO"
	segunda := Teams()
	for _, t2 := range segunda {
		if t2.Name == "ALTERADO" {
			t.Fatal("Teams() devolveu a fatia interna")
		}
	}
}

func TestTeamLookupEExacto(t *testing.T) {
	if _, ok := TeamLookup("dev-completo"); !ok {
		t.Fatal("dev-completo devia existir")
	}
	if _, ok := TeamLookup("dev"); ok {
		t.Fatal("lookup aceitou um prefixo")
	}
	if _, ok := TeamLookup("DEV-COMPLETO"); ok {
		t.Fatal("lookup ignorou a caixa")
	}
	if _, ok := TeamLookup(""); ok {
		t.Fatal("lookup aceitou id vazio")
	}
}

func equipaMinima() AgentTeam {
	return AgentTeam{
		ID: "t", Name: "T", Version: "1", Source: "hades-catalogo", License: "MIT",
		Division: "engenharia", MaxAgents: 2, MaxParallel: 1,
		StopCondition: "quando o teste passar",
		Roles: []TeamRole{
			{
				PersonaID: "hades-planejador", Role: "plano",
				Responsibility: "planear", Deliverable: "plano", Why: "sem plano nao ha ordem",
			},
			{
				PersonaID: "hades-executor", Role: "autor",
				Responsibility: "escrever", Deliverable: "diff",
				DependsOn: []string{"hades-planejador"}, Why: "alguem escreve",
			},
		},
	}
}

func TestValidateTeamAceitaEquipaValida(t *testing.T) {
	if v := ValidateTeam(equipaMinima(), CatalogLookup); !v.OK {
		t.Fatalf("equipa valida recusada: %v", v.Problems)
	}
}

// *Um ciclo nao e' "complexidade": e' uma equipa que nunca arranca.*
func TestValidateTeamRecusaCiclo(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[0].DependsOn = []string{"hades-executor"}
	v := ValidateTeam(eq, CatalogLookup)
	if v.OK {
		t.Fatal("ciclo nao foi recusado")
	}
	if len(v.Cycles) == 0 {
		t.Fatal("o ciclo nao foi reportado")
	}
}

// *Uma vaga que ninguem pode preencher e' uma promessa, nao um papel.*
func TestValidateTeamRecusaPersonaInexistente(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[1].PersonaID = "nao-existe"
	v := ValidateTeam(eq, CatalogLookup)
	if v.OK {
		t.Fatal("persona inexistente nao foi recusada")
	}
	if len(v.MissingPersonas) != 1 || v.MissingPersonas[0] != "nao-existe" {
		t.Fatalf("missing personas: %v", v.MissingPersonas)
	}
}

func TestValidateTeamRecusaDependenciaForaDaEquipa(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[1].DependsOn = []string{"hades-seguranca"}
	v := ValidateTeam(eq, CatalogLookup)
	if v.OK {
		t.Fatal("dependencia fora da equipa nao foi recusada")
	}
}

func TestValidateTeamRecusaSemParagem(t *testing.T) {
	eq := equipaMinima()
	eq.StopCondition = ""
	if v := ValidateTeam(eq, CatalogLookup); v.OK {
		t.Fatal("equipa sem condicao de paragem foi aceita")
	}
}

func TestValidateTeamRecusaMaxAgentsMenorQuePapeis(t *testing.T) {
	eq := equipaMinima()
	eq.MaxAgents = 1
	if v := ValidateTeam(eq, CatalogLookup); v.OK {
		t.Fatal("max_agents menor que os papeis foi aceito")
	}
}

func TestValidateTeamRecusaPapelDuplicado(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[1].Role = "plano"
	if v := ValidateTeam(eq, CatalogLookup); v.OK {
		t.Fatal("papel duplicado foi aceito")
	}
}

func TestValidateTeamRecusaPapelSemWhy(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[0].Why = ""
	if v := ValidateTeam(eq, CatalogLookup); v.OK {
		t.Fatal("papel sem justificacao foi aceito")
	}
}

func TestValidateTeamRecusaPapelSemEntregavel(t *testing.T) {
	eq := equipaMinima()
	eq.Roles[0].Deliverable = ""
	if v := ValidateTeam(eq, CatalogLookup); v.OK {
		t.Fatal("papel sem entregavel foi aceito")
	}
}

func TestCoverageApontaDivisaoEmFalta(t *testing.T) {
	eq := equipaMinima() // so' engenharia (planejador + executor)
	cobertas, faltam := TeamCoverage(eq, CatalogLookup)
	junta := strings.Join(cobertas, ",")
	if !strings.Contains(junta, "engenharia") {
		t.Fatalf("engenharia devia estar coberta: %v", cobertas)
	}
	if len(faltam) == 0 || faltam[0] != "qualidade" {
		t.Fatalf("qualidade devia faltar: %v", faltam)
	}
}

func TestDescribeTeamDizNaoFuncional(t *testing.T) {
	eq := equipaMinima()
	v := ValidateTeam(eq, CatalogLookup)
	cob, falt := TeamCoverage(eq, CatalogLookup)
	d := DescribeTeam(eq, v, cob, falt)
	if !strings.Contains(d, "NÃO FUNCIONAL") {
		t.Fatalf("a descricao nao diz que nao e' funcional: %q", d)
	}
	if !strings.Contains(d, "qualidade") {
		t.Fatalf("a descricao nao diz o que falta: %q", d)
	}
}

func TestPresetsDeclaramFerramentasEmFaltaQuandoAplicavel(t *testing.T) {
	// O preset de seguranca tem de declarar scanner em falta -- o HADES nao tem Semgrep neste host.
	eq, ok := TeamLookup("seguranca-release")
	if !ok {
		t.Fatal("seguranca-release devia existir")
	}
	v := ValidateTeam(eq, CatalogLookup)
	if len(v.MissingTools) == 0 {
		t.Fatal("seguranca-release devia declarar ferramentas em falta (semgrep)")
	}
	junta := strings.Join(v.MissingTools, ",")
	if !strings.Contains(junta, "semgrep") {
		t.Fatalf("semgrep devia estar declarado em falta: %v", v.MissingTools)
	}
}
