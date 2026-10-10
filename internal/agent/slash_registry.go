package agent

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Registro canônico de comandos slash (§6 / §11.1).
//
// Regra de honestidade desta tabela: um comando só aparece como DISPONÍVEL se o
// backend dele existe de fato. Comando sem rota fica marcado como indisponível,
// com o motivo, para que a interface nunca mostre um botão que não executa nada —
// e há teste no pacote `server` que compara cada caminho declarado com o roteador
// realmente registrado.

var (
	// ErrSlashCommandUnavailable indica comando reconhecido e ainda não implementado.
	ErrSlashCommandUnavailable = errors.New("slash command is not available")
	// ErrSlashCommandArgsInvalid indica argumento obrigatório ausente ou inválido.
	ErrSlashCommandArgsInvalid = errors.New("slash command arguments are invalid")
)

// SlashArgKind descreve como um argumento é validado.
type SlashArgKind string

const (
	SlashArgText SlashArgKind = "texto"
	SlashArgID   SlashArgKind = "id"
	SlashArgURL  SlashArgKind = "url"
)

// SlashArg é um parâmetro informado por posição no comando.
type SlashArg struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Required    bool         `json:"required"`
	Kind        SlashArgKind `json:"kind"`
}

// SlashCommandSpec é uma linha do registro canônico.
type SlashCommandSpec struct {
	Name              string     `json:"name"`
	Summary           string     `json:"summary"`
	Stage             string     `json:"stage,omitempty"`
	Backend           string     `json:"backend,omitempty"`
	Available         bool       `json:"available"`
	UnavailableReason string     `json:"unavailable_reason,omitempty"`
	Args              []SlashArg `json:"args,omitempty"`
}

// SlashInvocation é um comando interpretado e validado.
type SlashInvocation struct {
	Spec SlashCommandSpec  `json:"spec"`
	Args map[string]string `json:"args,omitempty"`
	Raw  string            `json:"raw"`
}

var slashIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]*$`)

func slashSpec(name, summary, stage, backend, unavailableReason string, args ...SlashArg) SlashCommandSpec {
	return SlashCommandSpec{
		Name:              name,
		Summary:           summary,
		Stage:             stage,
		Backend:           backend,
		Available:         backend != "",
		UnavailableReason: unavailableReason,
		Args:              args,
	}
}

func required(name, description string, kind SlashArgKind) SlashArg {
	return SlashArg{Name: name, Description: description, Required: true, Kind: kind}
}

func optional(name, description string, kind SlashArgKind) SlashArg {
	return SlashArg{Name: name, Description: description, Kind: kind}
}

// DefaultSlashRegistry devolve o registro, ordenado por nome. Retorna cópia: o
// chamador não consegue alterar o registro global por acidente.
func DefaultSlashRegistry() []SlashCommandSpec {
	specs := []SlashCommandSpec{
		slashSpec("/goal", "Define ou consulta o objetivo de uma missão durável", "10",
			"POST /api/agent/v1/missions", "",
			required("objetivo", "objetivo da missão", SlashArgText),
			optional("workspace", "workspace isolado da missão", SlashArgText)),
		slashSpec("/plan", "Gera o plano da missão (o plano nasce com a missão)", "10",
			"POST /api/agent/v1/missions", "",
			required("missao", "id da missão", SlashArgID)),
		slashSpec("/status", "Mostra o estado REAL da missão, incluindo falhas", "22",
			"GET /api/agent/v1/missions/:id", "",
			required("missao", "id da missão", SlashArgID)),
		slashSpec("/resume", "Retoma a missão a partir do checkpoint, sem repetir efeitos", "22",
			"POST /api/agent/v1/missions/:id/run", "",
			required("missao", "id da missão", SlashArgID)),
		slashSpec("/cancel", "Cancela apenas esta missão e registra o encerramento", "10",
			"POST /api/agent/v1/missions/:id/cancel", "",
			required("missao", "id da missão", SlashArgID)),
		slashSpec("/approve", "Aprova uma ação de risco, com trilha de auditoria", "10",
			"POST /api/agent/v1/missions/:id/approvals/:approval_id", "",
			required("missao", "id da missão", SlashArgID),
			required("aprovacao", "id da aprovação", SlashArgID)),
		slashSpec("/deny", "Rejeita uma ação de risco, com trilha de auditoria", "10",
			"POST /api/agent/v1/missions/:id/approvals/:approval_id", "",
			required("missao", "id da missão", SlashArgID),
			required("aprovacao", "id da aprovação", SlashArgID)),
		slashSpec("/build", "Constrói uma aplicação no Studio", "12",
			"POST /api/agent/v1/builders", "",
			required("pedido", "o que construir", SlashArgText)),
		slashSpec("/code", "Importa o repositório e prepara a engenharia (segue em missão)", "11",
			"POST /api/agent/v1/projects/import/github", "",
			required("repositorio", "URL do repositório", SlashArgURL)),
		slashSpec("/review", "Abre um artefato da missão para revisão", "11,18",
			"GET /api/agent/v1/missions/:id/artifacts/:artifact_id", "",
			required("missao", "id da missão", SlashArgID),
			required("artefato", "id do artefato", SlashArgID)),
		slashSpec("/research", "Pesquisa autorizada na web", "10", "",
			"não existe rota de pesquisa dedicada nesta versão: a busca na web acontece dentro do fluxo de chat, não como comando"),
		slashSpec("/company", "Cria e opera uma empresa assistida", "23",
			"POST /api/agent/v1/companies", "",
			required("nome", "nome da empresa", SlashArgText)),
		slashSpec("/store", "Gerencia o catálogo da loja", "24",
			"POST /api/agent/v1/companies/:id/products", "",
			required("empresa", "id da empresa", SlashArgID),
			required("produto", "nome ou descrição do produto", SlashArgText)),
		slashSpec("/sell", "Registra operação comercial (pedido) da empresa", "27",
			"POST /api/agent/v1/companies/:id/orders", "",
			required("empresa", "id da empresa", SlashArgID),
			required("pedido", "descrição do pedido", SlashArgText)),
		slashSpec("/marketing", "Cria e acompanha campanha aprovada", "26,36",
			"POST /api/agent/v1/companies/:id/campaigns", "",
			required("empresa", "id da empresa", SlashArgID),
			required("campanha", "nome ou objetivo da campanha", SlashArgText)),
		slashSpec("/social", "Cria rascunho de publicação para aprovação", "26,36",
			"POST /api/agent/v1/companies/:id/social/drafts", "",
			required("empresa", "id da empresa", SlashArgID),
			required("conteudo", "texto do rascunho", SlashArgText)),
		slashSpec("/phone", "Aciona o atendimento telefônico da empresa", "28,37",
			"POST /api/agent/v1/companies/:id/tel-agent", "",
			required("empresa", "id da empresa", SlashArgID)),
		slashSpec("/voice", "Configura o agente de voz da empresa", "28,37",
			"POST /api/agent/v1/companies/:id/tel-agent", "",
			required("empresa", "id da empresa", SlashArgID)),
		slashSpec("/calls", "Consulta o histórico de chamadas autorizadas", "28,37",
			"GET /api/agent/v1/companies/:id/tel-agent/history", "",
			required("empresa", "id da empresa", SlashArgID)),
		slashSpec("/support", "Atendimento e suporte ao cliente", "29,35", "",
			"a caixa de entrada unificada ainda não existe: sem rota de atendimento nesta versão"),
		slashSpec("/agents", "Coordena agentes e subagentes especializados", "9",
			"POST /api/agent/v1/orchestration/jobs", "",
			required("objetivo", "objetivo da coordenação", SlashArgText),
			optional("papeis", "papéis separados por vírgula", SlashArgText)),
		slashSpec("/mcp", "Lista e gerencia servidores MCP", "7",
			"GET /api/agent/v1/mcp", ""),
		slashSpec("/skills", "Lista e gerencia skills", "8",
			"GET /api/agent/v1/skills", ""),
		slashSpec("/automate", "Cria automação agendada com retry e DLQ", "13",
			"POST /api/agent/v1/schedules", "",
			required("objetivo", "o que automatizar", SlashArgText),
			optional("quando", "expressão de agendamento", SlashArgText)),
		slashSpec("/memory", "Consulta a memória governada do projeto", "15",
			"GET /api/agent/v1/projects/:id/memories", "",
			required("projeto", "id do projeto", SlashArgID),
			optional("consulta", "texto a buscar", SlashArgText)),
		slashSpec("/deploy", "Faz deploy autorizado com rollback", "20",
			"POST /api/agent/v1/builders/:id/deploy/:provider", "",
			required("builder", "id do builder", SlashArgID),
			required("provedor", "provedor de deploy", SlashArgText)),
		slashSpec("/whatsapp", "Mostra o estado do canal WhatsApp", "33",
			"GET /api/agent/v1/whatsapp/status", ""),
		slashSpec("/telegram", "Gerencia o canal Telegram", "33", "",
			"o adaptador de Telegram não está integrado a esta base: sem rota registrada"),
		slashSpec("/inbox", "Abre a central de atendimento unificada", "29", "",
			"a caixa de entrada unificada ainda não existe nesta versão"),
		slashSpec("/channels", "Lista e configura os canais disponíveis", "34",
			"GET /api/agent/v1/whatsapp/status", ""),
		slashSpec("/crm", "Administra contatos, leads e oportunidades", "27,35", "",
			"não existe CRM nesta versão: sem rota de contatos/leads"),
		slashSpec("/orders", "Consulta e gerencia pedidos da empresa", "24,36",
			"POST /api/agent/v1/companies/:id/orders", "",
			required("empresa", "id da empresa", SlashArgID),
			required("pedido", "descrição do pedido", SlashArgText)),
		slashSpec("/handoff", "Encaminha o atendimento para uma pessoa", "29,35", "",
			"sem caixa de entrada não há transferência humano/IA para registrar"),
		slashSpec("/campaign", "Cria e acompanha campanhas aprovadas", "26,36",
			"POST /api/agent/v1/companies/:id/campaigns", "",
			required("empresa", "id da empresa", SlashArgID),
			required("campanha", "nome ou objetivo da campanha", SlashArgText)),
		slashSpec("/business-status", "Acompanha as operações da empresa", "31",
			"GET /api/agent/v1/companies/:id/report", "",
			required("empresa", "id da empresa", SlashArgID)),
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	return specs
}

// SlashCommandSpecByName procura um comando pelo nome (com ou sem barra).
func SlashCommandSpecByName(name string) (SlashCommandSpec, bool) {
	name = strings.TrimSpace(strings.ToLower(name))
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	for _, spec := range DefaultSlashRegistry() {
		if spec.Name == name {
			return spec, true
		}
	}
	return SlashCommandSpec{}, false
}

// ParseSlashInvocation interpreta o texto contra o registro e valida os
// parâmetros. Comando indisponível é recusado com o motivo — nunca "executado".
func ParseSlashInvocation(input string) (SlashInvocation, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return SlashInvocation{}, ErrUnknownSlashCommand
	}
	fields := strings.Fields(trimmed)
	name := strings.ToLower(fields[0])
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	spec, ok := SlashCommandSpecByName(name)
	if !ok {
		return SlashInvocation{}, fmt.Errorf("%w: %s", ErrUnknownSlashCommand, fields[0])
	}
	if !spec.Available {
		return SlashInvocation{}, fmt.Errorf("%w: %s — %s", ErrSlashCommandUnavailable, spec.Name, spec.UnavailableReason)
	}
	provided := fields[1:]
	var requiredCount int
	for _, arg := range spec.Args {
		if arg.Required {
			requiredCount++
		}
	}
	if len(provided) < requiredCount {
		return SlashInvocation{}, fmt.Errorf("%w: %s exige %d argumento(s) (%s) e recebeu %d",
			ErrSlashCommandArgsInvalid, spec.Name, requiredCount, slashArgNames(spec.Args), len(provided))
	}
	if len(provided) > len(spec.Args) {
		return SlashInvocation{}, fmt.Errorf("%w: %s aceita no máximo %d argumento(s) e recebeu %d",
			ErrSlashCommandArgsInvalid, spec.Name, len(spec.Args), len(provided))
	}
	args := make(map[string]string, len(provided))
	for index, value := range provided {
		arg := spec.Args[index]
		if err := validateSlashArg(arg, value); err != nil {
			return SlashInvocation{}, err
		}
		args[arg.Name] = value
	}
	return SlashInvocation{Spec: spec, Args: args, Raw: trimmed}, nil
}

func validateSlashArg(arg SlashArg, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%w: %s não pode ser vazio", ErrSlashCommandArgsInvalid, arg.Name)
	}
	switch arg.Kind {
	case SlashArgID:
		if !slashIdentifier.MatchString(value) {
			return fmt.Errorf("%w: %s deve ser um identificador (letras, números, . _ : -)", ErrSlashCommandArgsInvalid, arg.Name)
		}
	case SlashArgURL:
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("%w: %s deve ser uma URL http(s) válida", ErrSlashCommandArgsInvalid, arg.Name)
		}
	}
	return nil
}

func slashArgNames(args []SlashArg) string {
	names := make([]string, 0, len(args))
	for _, arg := range args {
		if arg.Required {
			names = append(names, arg.Name)
		}
	}
	return strings.Join(names, ", ")
}

// SlashHelp devolve a ajuda em pt-BR, dizendo explicitamente o que não existe.
func SlashHelp() string {
	var builder strings.Builder
	builder.WriteString("Comandos disponíveis:\n")
	for _, spec := range DefaultSlashRegistry() {
		if !spec.Available {
			continue
		}
		fmt.Fprintf(&builder, "%s — %s\n", spec.Name, spec.Summary)
	}
	builder.WriteString("\nComandos indisponíveis nesta versão (não executam nada):\n")
	for _, spec := range DefaultSlashRegistry() {
		if spec.Available {
			continue
		}
		fmt.Fprintf(&builder, "%s — %s (%s)\n", spec.Name, spec.Summary, spec.UnavailableReason)
	}
	return strings.TrimRight(builder.String(), "\n")
}

// CompleteSlashCommand devolve os comandos que começam com o prefixo (autocomplete).
func CompleteSlashCommand(prefix string) []SlashCommandSpec {
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	matches := make([]SlashCommandSpec, 0, len(DefaultSlashRegistry()))
	for _, spec := range DefaultSlashRegistry() {
		if strings.HasPrefix(spec.Name, prefix) {
			matches = append(matches, spec)
		}
	}
	return matches
}
