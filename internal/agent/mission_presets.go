package agent

// FASE 23 do adendo V3: presets como configuracao versionada, revisavel e testavel --
// nunca como prompts magicos escondidos.
//
// O HADES ja' tinha 13 presets de PROVIDER em `internal/multillm/presets.go`. Aquilo responde a
// "que motor usar". Nao responde a "que trabalho e' este" -- e as duas perguntas tem respostas
// diferentes: `ollama` e' um preset de provider; `briefing-diario` e' um preset de MISSAO que pode
// correr em qualquer provider. *Um preset de motor nao e' um preset de trabalho: um escolhe onde
// corre, o outro escolhe o que corre.*
//
// Este arquivo acrescenta o segundo, sem tocar no primeiro.

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MissionPreset e' configuracao de TAREFA, declarada e inspecionavel antes de executar.
//
// LIMITE DECLARADO: um preset DESCREVE. Ele nao concede permissao nem contorna a policy: os campos
// `Permissions` e `Approvals` sao o que o preset PEDE, e o policy engine continua a ser quem
// DECIDE. *Um preset que se auto-autorizasse seria um formulario que assina por baixo.*
type MissionPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Goal orienta o planejador; nao e' a instrucao final do utilizador.
	Goal string `json:"goal"`
	// Agents sao os papeis candidatos. Vazio significa "decide o Auto Mode".
	Agents []string `json:"agents,omitempty"`
	// ModelsPref sao preferencias, nunca pins rigidos.
	ModelsPref []string `json:"models_pref,omitempty"`
	Tools      []string `json:"tools,omitempty"`
	Skills     []string `json:"skills,omitempty"`
	MCPs       []string `json:"mcps,omitempty"`
	Memory     []string `json:"memory,omitempty"`
	// Environment e' o nome do ambiente de execucao (FASE 6), nao o host.
	Environment string `json:"environment,omitempty"`
	// Permissions e' o que o preset PEDE; a policy decide.
	Permissions []string `json:"permissions,omitempty"`
	// Approvals sao os passos que EXIGEM humano neste preset.
	Approvals []string `json:"approvals,omitempty"`
	// Budget em segundos e tokens. Zero significa "usa o tecto global".
	BudgetSeconds int      `json:"budget_seconds,omitempty"`
	BudgetTokens  int      `json:"budget_tokens,omitempty"`
	Metrics       []string `json:"metrics,omitempty"`
	Tests         []string `json:"tests,omitempty"`
	// CompletionCriteria e' OBRIGATORIO e nao vazio: um preset sem criterio de conclusao e' um
	// preset que nunca acaba. *"Fazer o melhor que conseguir" nao e' um criterio: e' uma desculpa
	// com formato de campo.*
	CompletionCriteria []string `json:"completion_criteria"`
	// Schedule vazio = sob demanda. "daily" e "weekly" sao os unicos valores aceitos hoje --
	// declarar um cron arbitrario sem valida-lo seria vender uma promessa que nao se cumpre.
	Schedule string `json:"schedule,omitempty"`
}

// ErrPresetInvalido: o preset nao pode ser carregado.
var ErrPresetInvalido = errors.New("invalid mission preset")

var presetsDeMissaoValidos = map[string]bool{
	"": true, "daily": true, "weekly": true,
}

// Validate recusa presets que nao podem ser executados honestamente.
//
// Nao valida "boa ideia": valida o que impede execucao. *Um preset que nao se consegue terminar
// nao falha por ma sorte: falha por construcao.*
func (p MissionPreset) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("%w: empty id", ErrPresetInvalido)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: %s has no name", ErrPresetInvalido, p.ID)
	}
	if strings.TrimSpace(p.Goal) == "" {
		return fmt.Errorf("%w: %s has no goal", ErrPresetInvalido, p.ID)
	}
	if len(p.CompletionCriteria) == 0 {
		return fmt.Errorf("%w: %s has no completion criteria", ErrPresetInvalido, p.ID)
	}
	for _, criterio := range p.CompletionCriteria {
		if strings.TrimSpace(criterio) == "" {
			return fmt.Errorf("%w: %s has an empty completion criterion", ErrPresetInvalido, p.ID)
		}
	}
	if !presetsDeMissaoValidos[strings.TrimSpace(p.Schedule)] {
		return fmt.Errorf("%w: %s schedule %q is not one of (daily, weekly)", ErrPresetInvalido, p.ID, p.Schedule)
	}
	if p.BudgetSeconds < 0 || p.BudgetTokens < 0 {
		return fmt.Errorf("%w: %s has a negative budget", ErrPresetInvalido, p.ID)
	}
	return nil
}

// BuiltInMissionPresets devolve os presets canonicos, ordenados por ID.
//
// Cada um corresponde a um caso do adendo. Nenhum e' "generico o suficiente para tudo": um preset
// que serve para tudo nao informa nada a quem o escolhe.
func BuiltInMissionPresets() []MissionPreset {
	presets := []MissionPreset{
		{
			ID: "conversa-simples", Name: "Conversa simples",
			Description:        "Pergunta ou accao curta, sem missao persistente.",
			Goal:               "Responder ou agir no menor numero de passos.",
			Agents:             []string{}, // deliberadamente vazio: um agente
			Approvals:          []string{"acao_destrutiva"},
			CompletionCriteria: []string{"resposta entregue ao utilizador"},
			Metrics:            []string{"latencia", "tokens_saida"},
		},
		{
			ID: "pesquisa-profunda", Name: "Pesquisa profunda",
			Description:   "Pesquisa multi-salto com citacoes rastreaveis.",
			Goal:          "Produzir uma sintese com fontes verificaveis para cada afirmacao.",
			Agents:        []string{"pesquisador", "revisor"},
			Tools:         []string{"web_fetch", "search"},
			Memory:        []string{"SHARED_AUTHORIZED"},
			Permissions:   []string{"network:read"},
			BudgetSeconds: 1800,
			CompletionCriteria: []string{
				"toda afirmacao tem fonte citada",
				"nenhuma fonte e' citada duas vezes como se fossem duas",
				"o relatorio distingue facto de inferencia",
			},
			Metrics: []string{"latencia", "fontes_citadas", "afirmacoes_sem_fonte"},
			Tests:   []string{"verificar que cada afirmacao tem URL", "verificar que URLs respondem"},
		},
		{
			ID: "construcao-aplicativo", Name: "Construcao de aplicativo",
			Description:   "Do pedido ao artefacto instalavel, com testes.",
			Goal:          "Entregar um aplicativo que corre, com testes que falham sem o defeito.",
			Agents:        []string{"planner", "executor", "testador", "revisor"},
			Tools:         []string{"terminal", "worktree", "artifact"},
			Permissions:   []string{"filesystem:write:workspace"},
			Approvals:     []string{"instalar_dependencia", "publicar"},
			BudgetSeconds: 7200,
			CompletionCriteria: []string{
				"o build corre a partir de um clone limpo",
				"os testes falham quando o defeito e' reintroduzido",
				"a instalacao foi testada num ambiente limpo",
			},
			Metrics: []string{"duracao", "testes_passados", "tamanho_diff"},
			Tests:   []string{"go build", "go test", "instalacao limpa"},
		},
		{
			ID: "assistente-codigo", Name: "Assistente de codigo",
			Description: "Alteracoes de codigo com diff revisto antes de integrar.",
			Goal:        "Fazer a menor alteracao que resolve, com o diff explicado.",
			Agents:      []string{"executor", "revisor"},
			Tools:       []string{"terminal", "worktree", "diff_review"},
			Permissions: []string{"filesystem:write:workspace", "git:commit:branch"},
			Approvals:   []string{"merge", "force_push"},
			CompletionCriteria: []string{
				"os testes do pacote afectado passam",
				"o diff foi revisto por alguem que nao o autor",
				"nenhum teste foi removido ou enfraquecido",
			},
			Metrics: []string{"testes_passados", "linhas_adicionadas", "linhas_removidas"},
		},
		{
			ID: "briefing-diario", Name: "Briefing diario",
			Description:   "Resumo diario com audio opcional.",
			Goal:          "Produzir um resumo curto do que mudou e do que exige accao.",
			Agents:        []string{"pesquisador"},
			Tools:         []string{"artifact"},
			Memory:        []string{"PERSON", "PROJECT"},
			Schedule:      "daily",
			BudgetSeconds: 600,
			CompletionCriteria: []string{
				"o briefing nomeia o que MUDOU, nao o que existe",
				"itens sem mudanca nao aparecem",
			},
			Metrics: []string{"duracao", "itens_relevantes"},
		},
		{
			ID: "monitor-agendado", Name: "Monitor agendado",
			Description:   "Vigia um servico e so' avisa quando houver mudanca relevante.",
			Goal:          "Detectar mudanca relevante e permanecer silencioso quando nao houver.",
			Agents:        []string{"monitor"},
			Tools:         []string{"terminal", "web_fetch"},
			Memory:        []string{"PROJECT"},
			Schedule:      "daily",
			Permissions:   []string{"network:read"},
			BudgetSeconds: 300,
			CompletionCriteria: []string{
				"sem mudanca relevante = nenhuma notificacao",
				"com mudanca relevante = uma notificacao com a evidencia",
				"o monitor tem condicao de parada e orcamento",
			},
			Metrics: []string{"execucoes", "notificacoes", "falsos_positivos"},
		},
		{
			ID: "analise-dados", Name: "Analise de dados",
			Description: "Analise com metodologia declarada e resultados reproduziveis.",
			Goal:        "Responder a pergunta com numeros que outra pessoa consegue reproduzir.",
			Agents:      []string{"analista", "revisor"},
			Tools:       []string{"terminal", "artifact"},
			Permissions: []string{"filesystem:read:workspace"},
			CompletionCriteria: []string{
				"o metodo esta' declarado",
				"os numeros podem ser reproduzidos a partir dos dados",
				"as limitacoes estao escritas",
			},
			Metrics: []string{"linhas_processadas", "duracao"},
		},
		{
			ID: "producao-conteudo", Name: "Producao de conteudo",
			Description: "Texto com voz consistente, sem inventar factos.",
			Goal:        "Produzir conteudo que nao afirma o que nao sabe.",
			Agents:      []string{"redator", "revisor"},
			Tools:       []string{"artifact"},
			CompletionCriteria: []string{
				"nenhuma afirmacao factual sem fonte ou marcada como opiniao",
				"o tom segue a marca",
			},
			Metrics: []string{"palavras", "afirmacoes_sem_fonte"},
		},
		{
			ID: "atendimento", Name: "Atendimento autorizado",
			Description: "Respostas a clientes dentro de um escopo autorizado.",
			Goal:        "Responder dentro do escopo, escalando o que estiver fora.",
			Agents:      []string{"atendente", "revisor"},
			Memory:      []string{"TENANT", "PROJECT"},
			Permissions: []string{"network:write:messaging"},
			Approvals:   []string{"resposta_publica", "envio_em_massa", "reembolso"},
			CompletionCriteria: []string{
				"nenhuma resposta saiu do escopo autorizado",
				"o que estava fora do escopo foi escalado, nao respondido",
			},
			Metrics: []string{"respostas", "escalacoes"},
		},
		{
			ID: "automacao-local", Name: "Automacao local",
			Description: "Tarefas no proprio computador, com aprovacao do dono.",
			Goal:        "Executar a tarefa local sem tocar em nada fora do escopo.",
			Agents:      []string{"executor"},
			Tools:       []string{"terminal", "filesystem"},
			Permissions: []string{"filesystem:write:workspace", "process:spawn:allowlist"},
			Approvals:   []string{"exclusao_permanente", "elevacao", "compra", "publicacao"},
			CompletionCriteria: []string{
				"o alvo de cada operacao destrutiva foi resolvido em caminho absoluto",
				"nenhuma accao de alto impacto ocorreu sem confirmacao",
			},
			Metrics: []string{"accoes", "accoes_negadas"},
		},
		{
			ID: "criacao-empresa", Name: "Criacao e operacao de empresa",
			Description:   "Pesquisa de mercado, plano e operacao assistida.",
			Goal:          "Produzir um plano verificavel, nao uma narrativa optimista.",
			Agents:        []string{"pesquisador", "analista", "revisor"},
			Tools:         []string{"web_fetch", "artifact"},
			Memory:        []string{"TENANT", "PROJECT"},
			Approvals:     []string{"ato_juridico", "transacao_financeira", "publicacao"},
			BudgetSeconds: 7200,
			CompletionCriteria: []string{
				"cada numero de mercado tem fonte",
				"as premissas estao separadas dos factos",
				"o plano diz o que faria o negocio falhar",
			},
			Metrics: []string{"fontes_citadas", "premissas_declaradas"},
			Tests:   []string{"verificar que cada numero tem fonte"},
		},
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].ID < presets[j].ID })
	return presets
}

// MissionPresetByID devolve o preset e diz se existe.
func MissionPresetByID(id string) (MissionPreset, bool) {
	alvo := strings.TrimSpace(id)
	for _, p := range BuiltInMissionPresets() {
		if p.ID == alvo {
			return p, true
		}
	}
	return MissionPreset{}, false
}

// ValidateAllPresets valida o catalogo inteiro. Existe para o teste poder falhar se alguem
// acrescentar um preset sem criterio de conclusao -- o erro aparece aqui, nao em producao.
func ValidateAllPresets() error {
	vistos := map[string]bool{}
	for _, p := range BuiltInMissionPresets() {
		if err := p.Validate(); err != nil {
			return err
		}
		if vistos[p.ID] {
			return fmt.Errorf("%w: duplicate id %s", ErrPresetInvalido, p.ID)
		}
		vistos[p.ID] = true
	}
	return nil
}
