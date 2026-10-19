// Equipas e presets do Agent Center (FASE 30.5).
//
// UMA EQUIPA NAO E' UMA LISTA DE NOMES
//
// Um preset tem de declarar o papel de cada persona, o que ela entrega, de quem depende e PORQUE foi
// escolhida. E tem de declarar tambem o que FALTA:
//
//	*Cada preset deve explicar por que cada agente foi escolhido, quais capacidades reais possui, quais
//	ferramentas faltam e quais riscos permanecem.*
//
// Uma equipa que so' lista personas e' um menu, nao uma equipa.
//
// TODA A REFERENCIA A UMA PERSONA E' VALIDADA CONTRA O CATALOGO
//
// Um `PersonaID` que nao existe no catalogo e' uma vaga com nome de pessoa. `ValidateTeam` recusa.
//
//	*Uma vaga que ninguem pode preencher e' uma promessa, nao um papel.*
//
// # CICLOS SAO RECUSADOS
//
// `DependsOn` e' uma ordem de execucao. Um ciclo nao e' "complexidade": e' uma equipa que nunca arranca.
//
// NENHUMA EQUIPA E' FUNCIONAL POR ESTAR DECLARADA
//
// O campo `Functional` comeca `false` e SO' uma missao representativa com ferramentas reais o promove.
//
//	*Uma equipe nao e' considerada funcional ate' completar pelo menos uma missao representativa com
//	ferramentas reais controladas.*
package agent

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrTeamInvalida cobre equipas mal formadas.
var ErrTeamInvalida = errors.New("agent team is invalid")

// TeamRole e' um lugar na equipa: quem, para que, entregando o que, dependendo de quem.
type TeamRole struct {
	// PersonaID tem de existir no catalogo.
	PersonaID string `json:"persona_id"`
	// Role e' o nome do papel na equipa (nao o da persona): "autor", "revisor", "seguranca".
	Role string `json:"role"`
	// Responsibility e' o que este papel responde.
	Responsibility string `json:"responsibility"`
	// Deliverable e' o artefacto concreto.
	Deliverable string `json:"deliverable"`
	// DependsOn sao os IDs das PERSONAS de que este papel depende para comecar.
	DependsOn []string `json:"depends_on,omitempty"`
	// Why explica porque e' que este papel foi escolhido.
	Why string `json:"why"`
	// MissingTools sao as ferramentas que o papel pede e que o HADES ainda nao tem ligadas. Vazio = nada
	// em falta conhecido.
	MissingTools []string `json:"missing_tools,omitempty"`
	// Risk e' o risco que continua neste papel.
	Risk string `json:"risk,omitempty"`
	// RequiresApproval: acoes deste papel que exigem confirmacao humana no momento da acao.
	RequiresApproval []string `json:"requires_approval,omitempty"`
}

// AgentTeam e' um preset versionado.
type AgentTeam struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	// Division agrupa o preset (desenvolvimento, produto, seguranca...).
	Division string `json:"division"`
	// Source e License: um preset sem proveniencia nao se audita.
	Source  string `json:"source"`
	License string `json:"license"`

	Roles []TeamRole `json:"roles"`

	// Budget declarado, nao inferido.
	MaxAgents   int `json:"max_agents"`
	MaxParallel int `json:"max_parallel"`
	// StopCondition diz quando a equipa para. Vazio e' recusado.
	StopCondition string `json:"stop_condition"`

	// Functional comeca `false`. Nenhuma declaracao o promove.
	Functional bool `json:"functional"`
	// EvidenceOfFunctionality e' o que prova que a equipa correu uma missao representativa.
	EvidenceOfFunctionality string `json:"evidence_of_functionality,omitempty"`
}

// TeamValidation e' o resultado da validacao, com tudo o que falta.
type TeamValidation struct {
	OK       bool     `json:"ok"`
	Problems []string `json:"problems,omitempty"`
	// MissingPersonas: PersonaIDs que o catalogo nao tem.
	MissingPersonas []string `json:"missing_personas,omitempty"`
	// MissingTools agregadas de todos os papeis.
	MissingTools []string `json:"missing_tools,omitempty"`
	// Cycles: caminhos de dependencia que fecham em circulo.
	Cycles [][]string `json:"cycles,omitempty"`
}

// ValidateTeam valida uma equipa contra o catalogo.
//
// `lookup` e' a funcao de resolucao (normalmente `CatalogLookup`). Passa-la como argumento em vez de usar
// o catalogo directamente permite testar a validacao sem depender do conteudo do catalogo.
func ValidateTeam(t AgentTeam, lookup func(string) (AgentPersona, bool)) TeamValidation {
	v := TeamValidation{OK: true}
	add := func(s string) {
		v.OK = false
		v.Problems = append(v.Problems, s)
	}

	if strings.TrimSpace(t.ID) == "" {
		add("id is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		add("name is required")
	}
	if strings.TrimSpace(t.Version) == "" {
		add("version is required")
	}
	if strings.TrimSpace(t.Source) == "" {
		add("source is required")
	}
	if strings.TrimSpace(t.License) == "" {
		add("license is required")
	}
	if strings.TrimSpace(t.StopCondition) == "" {
		// *Uma equipa sem condicao de paragem nao para: esgota o orcamento e chama-lhe fim.*
		add("stop_condition is required")
	}
	if len(t.Roles) == 0 {
		add("team has no roles")
	}
	if t.MaxAgents < 0 {
		add("max_agents cannot be negative")
	}
	if t.MaxAgents > 0 && len(t.Roles) > t.MaxAgents {
		add(fmt.Sprintf("%d roles exceed max_agents %d", len(t.Roles), t.MaxAgents))
	}
	if t.MaxParallel < 0 {
		add("max_parallel cannot be negative")
	}
	if t.MaxParallel > len(t.Roles) && len(t.Roles) > 0 {
		add(fmt.Sprintf("max_parallel %d exceeds roles %d", t.MaxParallel, len(t.Roles)))
	}

	vistosPersona := map[string]bool{}
	vistosPapel := map[string]bool{}
	ferramentas := map[string]bool{}
	for _, r := range t.Roles {
		if strings.TrimSpace(r.PersonaID) == "" {
			add("role without persona_id")
			continue
		}
		if vistosPapel[r.Role] {
			// *Dois papeis com o mesmo nome sao um papel que ninguem sabe de quem e'.*
			add(fmt.Sprintf("duplicate role %q", r.Role))
		}
		vistosPapel[r.Role] = true
		if strings.TrimSpace(r.Role) == "" {
			add(fmt.Sprintf("persona %q without a role name", r.PersonaID))
		}
		if strings.TrimSpace(r.Responsibility) == "" {
			add(fmt.Sprintf("role %q has no responsibility", r.Role))
		}
		if strings.TrimSpace(r.Deliverable) == "" {
			add(fmt.Sprintf("role %q has no deliverable", r.Role))
		}
		if strings.TrimSpace(r.Why) == "" {
			// *Uma equipa que nao explica a escolha e' uma lista de nomes.*
			add(fmt.Sprintf("role %q does not say why it was chosen", r.Role))
		}
		vistosPersona[r.PersonaID] = true
		if lookup != nil {
			if _, ok := lookup(r.PersonaID); !ok {
				v.MissingPersonas = append(v.MissingPersonas, r.PersonaID)
				v.OK = false
				v.Problems = append(v.Problems, fmt.Sprintf("persona %q is not in the catalog", r.PersonaID))
			}
		}
		for _, f := range r.MissingTools {
			ferramentas[f] = true
		}
	}

	// Dependencias: teem de apontar para personas DESTA equipa.
	for _, r := range t.Roles {
		for _, dep := range r.DependsOn {
			if !vistosPersona[dep] {
				add(fmt.Sprintf("role %q depends on %q, which is not in this team", r.Role, dep))
			}
		}
	}

	// Ciclos. *Um ciclo nao e' "complexidade": e' uma equipa que nunca arranca.*
	v.Cycles = findTeamCycles(t.Roles)
	if len(v.Cycles) > 0 {
		v.OK = false
		for _, c := range v.Cycles {
			v.Problems = append(v.Problems, "dependency cycle: "+strings.Join(c, " -> "))
		}
	}

	for f := range ferramentas {
		v.MissingTools = append(v.MissingTools, f)
	}
	sort.Strings(v.MissingTools)
	sort.Strings(v.MissingPersonas)
	return v
}

// findTeamCycles procura ciclos em `DependsOn`. Devolve caminhos fechados, ordenados.
func findTeamCycles(roles []TeamRole) [][]string {
	grafo := map[string][]string{}
	for _, r := range roles {
		grafo[r.PersonaID] = append(grafo[r.PersonaID], r.DependsOn...)
	}
	estado := map[string]int{} // 0=nao visitado, 1=em pilha, 2=feito
	var ciclos [][]string
	var pilha []string

	var visitar func(string)
	visitar = func(no string) {
		estado[no] = 1
		pilha = append(pilha, no)
		for _, dep := range grafo[no] {
			if estado[dep] == 1 {
				// encontrou o ciclo: copia o troco entre a primeira ocorrencia e agora
				inicio := 0
				for i, p := range pilha {
					if p == dep {
						inicio = i
						break
					}
				}
				ciclo := append([]string{}, pilha[inicio:]...)
				ciclo = append(ciclo, dep)
				ciclos = append(ciclos, ciclo)
				continue
			}
			if estado[dep] == 0 {
				visitar(dep)
			}
		}
		pilha = pilha[:len(pilha)-1]
		estado[no] = 2
	}

	nomes := make([]string, 0, len(grafo))
	for n := range grafo {
		nomes = append(nomes, n)
	}
	sort.Strings(nomes)
	for _, n := range nomes {
		if estado[n] == 0 {
			visitar(n)
		}
	}
	return ciclos
}

// TeamCoverage diz que divisoes a equipa cobre e o que falta.
//
// Um preset de "desenvolvimento" que nao tem revisor tem uma lacuna que o utilizador tem de ver ANTES de
// correr, nao depois.
func TeamCoverage(t AgentTeam, lookup func(string) (AgentPersona, bool)) (cobertas []string, faltam []string) {
	set := map[string]bool{}
	for _, r := range t.Roles {
		if lookup == nil {
			continue
		}
		if p, ok := lookup(r.PersonaID); ok && p.Division != "" {
			set[p.Division] = true
		}
	}
	for d := range set {
		cobertas = append(cobertas, d)
	}
	sort.Strings(cobertas)
	// O que um preset de construcao deveria cobrir. Nao e' uma regra universal: e' o que este produto
	// considera minimo para nao entregar trabalho sem revisao.
	exigidas := []string{"engenharia", "qualidade"}
	if t.Division == "segurança" {
		exigidas = append(exigidas, "segurança")
	}
	for _, d := range exigidas {
		if !set[d] {
			faltam = append(faltam, d)
		}
	}
	return cobertas, faltam
}

// DescribeTeam devolve um resumo legivel, com o que falta dito por extenso.
func DescribeTeam(t AgentTeam, v TeamValidation, cobertas, faltam []string) string {
	var b strings.Builder
	b.WriteString(t.Name + " v" + t.Version + " (" + t.Division + "): ")
	b.WriteString(fmt.Sprintf("%d papel(is)", len(t.Roles)))
	if v.OK {
		b.WriteString(" · válida")
	} else {
		b.WriteString(fmt.Sprintf(" · INVÁLIDA (%d problema(s))", len(v.Problems)))
	}
	if len(v.MissingPersonas) > 0 {
		b.WriteString(" · personas em falta: " + strings.Join(v.MissingPersonas, ", "))
	}
	if len(v.MissingTools) > 0 {
		b.WriteString(" · ferramentas em falta: " + strings.Join(v.MissingTools, ", "))
	}
	if len(faltam) > 0 {
		b.WriteString(" · divisões sem cobertura: " + strings.Join(faltam, ", "))
	}
	if !t.Functional {
		// *Um preset que nunca correu nao e' uma equipa: e' uma proposta de equipa.*
		b.WriteString(" · NÃO FUNCIONAL (nenhuma missão representativa provada)")
	}
	return b.String()
}
