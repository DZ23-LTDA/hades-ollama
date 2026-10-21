// Presets de equipa embutidos (FASE 30.5).
//
// NENHUM E' FUNCIONAL, E ISSO E' A VERDADE
//
// Todos comecam `Functional: false`. Uma equipa so' se torna funcional quando completa uma missao
// representativa com ferramentas reais controladas -- e isso ainda nao aconteceu.
//
//	*Uma equipe nao e' considerada funcional ate' completar pelo menos uma missao representativa com
//	ferramentas reais controladas.*
//
// Cada papel declara `MissingTools` quando o HADES ainda nao tem a ferramenta ligada. Uma lista vazia
// significa "nada em falta conhecido", nao "tudo funciona".
//
//	*Um preset que nao declara o que falta e' um prospecto; um que declara e' um plano.*
package agent

import "sort"

// teamPresetVersion sobe quando o CONTEUDO dos presets muda.
const teamPresetVersion = "1"

// TeamPresetVersion devolve a versao dos presets embutidos.
func TeamPresetVersion() string { return teamPresetVersion }

var teamPresetsBuiltin = []AgentTeam{
	{
		ID:            "dev-completo",
		Name:          "Desenvolvimento completo",
		Version:       teamPresetVersion,
		Description:   "Do plano ao diff revisado: planeja, implementa, revisa e verifica seguranca.",
		Division:      "engenharia",
		Source:        "hades-catalogo",
		License:       "MIT",
		MaxAgents:     4,
		MaxParallel:   1,
		StopCondition: "para quando os gates do repositorio passarem ou quando um papel reprovar duas vezes",
		Functional:    false,
		Roles: []TeamRole{
			{
				PersonaID: "hades-planejador", Role: "plano",
				Responsibility: "decompor o objetivo em passos com teste discriminante",
				Deliverable:    "plano com dependencias e criterios de aceite",
				Why:            "sem plano, o executor escolhe o proximo passo por impulso",
				Risk:           "um plano que parece bom e nao tem teste associado passa despercebido",
			},
			{
				PersonaID: "hades-executor", Role: "autor",
				Responsibility:   "fazer a menor mudanca que resolve o passo",
				Deliverable:      "diff com testes e gates corridos",
				DependsOn:        []string{"hades-planejador"},
				Why:              "alguem tem de escrever o codigo",
				MissingTools:     []string{"run_tests_sandboxed"},
				Risk:             "o executor edita o teste para passar em vez de corrigir a causa",
				RequiresApproval: []string{"commit", "push"},
			},
			{
				PersonaID: "hades-revisor", Role: "revisao",
				Responsibility: "apontar o defeito no diff, com arquivo e linha",
				Deliverable:    "lista de achados com severidade",
				DependsOn:      []string{"hades-executor"},
				Why:            "o autor nao aprova a propria entrega",
				Risk:           "revisao que so' elogia e' decoracao",
			},
			{
				PersonaID: "hades-seguranca", Role: "seguranca",
				Responsibility: "procurar fronteira de tenant, segredo e permissao excessiva",
				Deliverable:    "parecer de seguranca com negacoes testadas",
				DependsOn:      []string{"hades-executor"},
				Why:            "um diff funcional pode abrir uma fronteira sem ninguem notar",
				MissingTools:   []string{"scan_secrets_full"},
				Risk:           "sem scanner completo, a procura e' manual e incompleta",
			},
		},
	},
	{
		ID:            "app-movel",
		Name:          "Aplicativo movel",
		Version:       teamPresetVersion,
		Description:   "Planeja, desenha, implementa e revisa um app movel com estados reais.",
		Division:      "engenharia",
		Source:        "hades-catalogo",
		License:       "MIT",
		MaxAgents:     4,
		MaxParallel:   2,
		StopCondition: "para quando o build de preview abrir no emulador ou quando o design reprovar",
		Functional:    false,
		Roles: []TeamRole{
			{
				PersonaID: "hades-planejador", Role: "plano",
				Responsibility: "definir telas, fluxos e criterios de aceite",
				Deliverable:    "plano de telas com testes",
				Why:            "app sem plano vira tela solta",
			},
			{
				PersonaID: "hades-design", Role: "design",
				Responsibility: "declarar os estados de cada tela e medir contraste",
				Deliverable:    "especificacao com estados e contraste medido",
				DependsOn:      []string{"hades-planejador"},
				Why:            "o caminho feliz nao e' a tela",
				Risk:           "aprovacao estetica sem medicao passa como acessibilidade",
			},
			{
				PersonaID: "hades-executor", Role: "autor",
				Responsibility: "implementar as telas sem quebrar o WIP nativo",
				Deliverable:    "diff com build de preview",
				DependsOn:      []string{"hades-design"},
				Why:            "alguem tem de escrever o app",
				MissingTools:   []string{"expo_mcp_oauth"},
				Risk:           "prebuild automatico destroi trabalho nativo em andamento",
			},
			{
				PersonaID: "hades-revisor", Role: "revisao",
				Responsibility: "revisar o diff e os estados de UI",
				Deliverable:    "achados com arquivo e linha",
				DependsOn:      []string{"hades-executor"},
				Why:            "o autor nao aprova a propria entrega",
			},
		},
	},
	{
		ID:            "pesquisa-profunda",
		Name:          "Pesquisa com fontes",
		Version:       teamPresetVersion,
		Description:   "Planeja a pergunta, pesquisa com citacao rastreavel e revisa o que ficou sem fonte.",
		Division:      "pesquisa",
		Source:        "hades-catalogo",
		License:       "MIT",
		MaxAgents:     3,
		MaxParallel:   1,
		StopCondition: "para quando cada afirmacao tiver fonte ou estiver marcada como nao verificada",
		Functional:    false,
		Roles: []TeamRole{
			{
				PersonaID: "hades-planejador", Role: "pergunta",
				Responsibility: "transformar o pedido em perguntas verificaveis",
				Deliverable:    "lista de perguntas com criterio de resposta",
				Why:            "pergunta vaga devolve resposta vaga com aparencia de rigor",
			},
			{
				PersonaID: "hades-pesquisa", Role: "pesquisador",
				Responsibility:   "responder com fonte, trecho e data",
				Deliverable:      "resposta com citacoes",
				DependsOn:        []string{"hades-planejador"},
				Why:              "sem fonte, uma afirmacao e' uma opiniao formatada",
				Risk:             "modelo inventa citacao que parece real",
				RequiresApproval: []string{"network_access"},
			},
			{
				PersonaID: "hades-revisor", Role: "revisao",
				Responsibility: "marcar o que ficou sem fonte",
				Deliverable:    "lista do que nao foi verificado",
				DependsOn:      []string{"hades-pesquisa"},
				Why:            "o pesquisador nao audita a propria pesquisa",
			},
		},
	},
	{
		ID:            "seguranca-release",
		Name:          "Seguranca e release",
		Version:       teamPresetVersion,
		Description:   "Revisa fronteiras, verifica negacoes e prepara a evidencia de release.",
		Division:      "segurança",
		Source:        "hades-catalogo",
		License:       "MIT",
		MaxAgents:     3,
		MaxParallel:   1,
		StopCondition: "para quando as negacoes passarem ou quando houver um achado P0/P1",
		Functional:    false,
		Roles: []TeamRole{
			{
				PersonaID: "hades-seguranca", Role: "seguranca",
				Responsibility: "procurar fronteira de tenant, segredo exposto e permissao excessiva",
				Deliverable:    "parecer com achados e negacoes testadas",
				Why:            "seguranca nao se revisa a si propria no fim; revisa-se antes de publicar",
				MissingTools:   []string{"scan_secrets_full", "semgrep"},
				Risk:           "sem Semgrep nem Docker, a analise estatica e' parcial",
			},
			{
				PersonaID: "hades-revisor", Role: "revisao",
				Responsibility: "revisar o parecer e o diff de correcao",
				Deliverable:    "achados com arquivo e linha",
				DependsOn:      []string{"hades-seguranca"},
				Why:            "um parecer sem revisao e' a palavra de quem o escreveu",
			},
			{
				PersonaID: "hades-executor", Role: "correcao",
				Responsibility:   "corrigir a causa raiz de cada achado",
				Deliverable:      "diff de correcao com teste que falharia sem ela",
				DependsOn:        []string{"hades-revisor"},
				Why:              "achado sem correcao e' relatorio",
				RequiresApproval: []string{"commit", "push"},
			},
		},
	},
}

// Teams devolve os presets embutidos, ordenados por ID.
//
// Copia defensiva, como o catalogo: um chamador que altere a fatia nao altera o processo seguinte.
func Teams() []AgentTeam {
	out := make([]AgentTeam, len(teamPresetsBuiltin))
	copy(out, teamPresetsBuiltin)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// TeamLookup encontra um preset pelo ID exacto.
func TeamLookup(id string) (AgentTeam, bool) {
	for _, t := range teamPresetsBuiltin {
		if t.ID == id {
			return t, true
		}
	}
	return AgentTeam{}, false
}

// TeamsCount devolve o numero de presets embutidos.
func TeamsCount() int { return len(teamPresetsBuiltin) }
