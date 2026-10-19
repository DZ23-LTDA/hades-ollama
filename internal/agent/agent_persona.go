// Agent Center -- modelo canonico de agente, hash de conteudo, renderizacao determinista e reconciliacao.
//
// FRONTEIRA (FASE 30 do adendo, e o que o `agency-agents-app` NAO e')
//
// O catalogo externo declara explicitamente que NAO e' runtime, NAO executa as personas e NAO e' ponte para
// shell arbitrario. Este ficheiro respeita essa fronteira: uma persona e' CONTEUDO. Nao autoriza nada, nao
// executa nada, nao eleva privilegios. Quem executa continua a ser o runtime, o Auto Mode e as politicas.
//
//	*Persona nao e' capacidade. Um prompt bonito nao prova que a ferramenta existe, que a permissao foi
//	concedida ou que o resultado e' bom.*
//
// DETERMINISMO E' O PRODUTO, NAO UM DETALHE
//
// "A mesma entrada canonica, na mesma versao, produz bytes identicos" -- senao a reconciliacao por hash e'
// ruido: dois renders do mesmo agente pareceriam drift um do outro. Por isso TUDO o que e' mapa ou conjunto
// e' ordenado antes de serializar, e o hash e' do conteudo NORMALIZADO.
//
//	*Uma reconciliacao que compara hashes de bytes nao-deterministicos nao detecta drift: detecta a hora.*
package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrPersonaInvalid cobre o schema do agente.
	ErrPersonaInvalid = errors.New("agent persona is invalid")
	// ErrPersonaUnsupportedTarget cobre formatos que nao sabemos renderizar.
	ErrPersonaUnsupportedTarget = errors.New("agent persona target is not supported")
)

// AgentPersona e' o contrato canonico de um agente especializado.
//
// O prompt e' CONTEUDO: nunca e' executado, nunca e' autoridade sobre o runtime, e nunca concede permissao.
type AgentPersona struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Division    string `json:"division"`
	Category    string `json:"category"`

	// Proveniencia. Sem isto o agente e' anonimo, e um agente anonimo nao se audita.
	Source    string `json:"source"`
	SourceRef string `json:"source_ref"`
	License   string `json:"license"`

	Prompt string `json:"prompt"`

	// Capacidades PEDIDAS. As concedidas vivem na instalacao, nao aqui.
	Capabilities []string `json:"capabilities,omitempty"`
	// Scopes permitidos: user, organization, project, workspace, goal.
	Scopes []string `json:"scopes,omitempty"`
	// Ferramentas que o agente diz precisar.
	Tools []string `json:"tools,omitempty"`
	// Acoes que exigem aprovacao humana no momento da acao.
	RequiresApproval []string `json:"requires_approval,omitempty"`
	// Acoes absolutamente proibidas, mesmo com aprovacao.
	Forbidden []string `json:"forbidden,omitempty"`

	// Evaluation e' o estado medido. Vazio significa NAO AVALIADO -- nunca "aprovado".
	Evaluation string `json:"evaluation,omitempty"`
}

// Validate recusa o que nao da' para instalar. Nao avalia qualidade: avalia forma.
func (p AgentPersona) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("%w: id is required", ErrPersonaInvalid)
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrPersonaInvalid)
	}
	if strings.TrimSpace(p.Version) == "" {
		return fmt.Errorf("%w: version is required", ErrPersonaInvalid)
	}
	// Proveniencia: um agente sem fonte e sem licenca nao se pode auditar nem redistribuir.
	if strings.TrimSpace(p.Source) == "" {
		return fmt.Errorf("%w: source is required", ErrPersonaInvalid)
	}
	if strings.TrimSpace(p.License) == "" {
		return fmt.Errorf("%w: license is required", ErrPersonaInvalid)
	}
	if strings.ContainsAny(p.ID, "/\\") || strings.Contains(p.ID, "..") {
		// O id vira nome de ficheiro no destino. Um id que reescreve o caminho e' um caminho, nao um id.
		return fmt.Errorf("%w: id must not contain path separators or ..", ErrPersonaInvalid)
	}
	return nil
}

// Normalized devolve a persona com todos os conjuntos ordenados e os campos de texto aparados.
//
// E' esta forma que se hasheia e se renderiza. Duas personas iguais escritas em ordens diferentes tem de
// produzir o MESMO hash.
func (p AgentPersona) Normalized() AgentPersona {
	out := p
	out.ID = strings.TrimSpace(p.ID)
	out.Slug = strings.TrimSpace(p.Slug)
	out.Name = strings.TrimSpace(p.Name)
	out.Version = strings.TrimSpace(p.Version)
	out.Description = strings.TrimSpace(p.Description)
	out.Division = strings.TrimSpace(p.Division)
	out.Category = strings.TrimSpace(p.Category)
	out.Source = strings.TrimSpace(p.Source)
	out.SourceRef = strings.TrimSpace(p.SourceRef)
	out.License = strings.TrimSpace(p.License)
	out.Prompt = strings.TrimSpace(p.Prompt)
	out.Evaluation = strings.TrimSpace(p.Evaluation)
	out.Capabilities = ordenarUnicos(p.Capabilities)
	out.Scopes = ordenarUnicos(p.Scopes)
	out.Tools = ordenarUnicos(p.Tools)
	out.RequiresApproval = ordenarUnicos(p.RequiresApproval)
	out.Forbidden = ordenarUnicos(p.Forbidden)
	return out
}

// ContentHash e' o sha256 da forma normalizada renderizada em HADES nativo.
//
// Hashear a FORMA NORMALIZADA, e nao os bytes do ficheiro de origem, e' o que faz o mesmo agente importado
// de um JSON indentado e de um JSON compacto ter o mesmo hash.
func (p AgentPersona) ContentHash() (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	corpo, err := RenderHades(p)
	if err != nil {
		return "", err
	}
	soma := sha256.Sum256(corpo)
	return hex.EncodeToString(soma[:]), nil
}

func ordenarUnicos(valores []string) []string {
	vistos := make(map[string]struct{}, len(valores))
	saida := make([]string, 0, len(valores))
	for _, v := range valores {
		limpo := strings.TrimSpace(v)
		if limpo == "" {
			continue
		}
		if _, repetido := vistos[limpo]; repetido {
			// Duplicar uma capacidade nao a pede duas vezes: pede-a uma e polui o diff.
			continue
		}
		vistos[limpo] = struct{}{}
		saida = append(saida, limpo)
	}
	sort.Strings(saida)
	return saida
}

// RenderHades produz o formato canonico do HADES. E' a referencia: os outros formatos sao projeccoes disto.
func RenderHades(p AgentPersona) ([]byte, error) {
	n := p.Normalized()
	if err := n.Validate(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(n.Name)
	b.WriteString("\n\n")
	b.WriteString("<!-- hades-agent: ")
	b.WriteString(n.ID)
	b.WriteString("@")
	b.WriteString(n.Version)
	b.WriteString(" source=")
	b.WriteString(n.SourceRef)
	if n.SourceRef == "" {
		b.WriteString("desconhecido")
	}
	b.WriteString(" license=")
	b.WriteString(n.License)
	b.WriteString(" -->\n\n")
	if n.Description != "" {
		b.WriteString(n.Description)
		b.WriteString("\n\n")
	}
	escreverSecao(&b, "Divisao", n.Division)
	escreverSecao(&b, "Categoria", n.Category)
	escreverLista(&b, "Capacidades pedidas", n.Capabilities)
	escreverLista(&b, "Escopos permitidos", n.Scopes)
	escreverLista(&b, "Ferramentas necessarias", n.Tools)
	escreverLista(&b, "Exige aprovacao", n.RequiresApproval)
	escreverLista(&b, "Absolutamente proibido", n.Forbidden)
	if n.Prompt != "" {
		b.WriteString("## Instrucoes\n\n")
		b.WriteString(n.Prompt)
		b.WriteString("\n\n")
	}
	if n.Evaluation == "" {
		// NAO AVALIADO e' um estado, e e' diferente de "aprovado".
		b.WriteString("## Avaliacao\n\nNAO_AVALIADO\n")
	} else {
		b.WriteString("## Avaliacao\n\n")
		b.WriteString(n.Evaluation)
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}

func escreverSecao(b *strings.Builder, titulo, valor string) {
	if valor == "" {
		return
	}
	b.WriteString(titulo)
	b.WriteString(": ")
	b.WriteString(valor)
	b.WriteString("\n\n")
}

func escreverLista(b *strings.Builder, titulo string, valores []string) {
	if len(valores) == 0 {
		return
	}
	b.WriteString("## ")
	b.WriteString(titulo)
	b.WriteString("\n\n")
	for _, v := range valores {
		b.WriteString("- ")
		b.WriteString(v)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

// TargetFormat e' o destino de renderizacao. Cada formato tem um suporte DECLARADO.
type TargetFormat string

const (
	TargetHades      TargetFormat = "hades"
	TargetCodex      TargetFormat = "codex"
	TargetClaudeCode TargetFormat = "claude-code"
)

// SupportLevel diz o que o formato realmente suporta. Nao e' marketing: e' o que o renderizador faz.
type SupportLevel string

const (
	// SupportInstallable: o renderizador produz bytes que a ferramenta aceita como agente.
	SupportInstallable SupportLevel = "SUPPORTED_INSTALLABLE"
	// SupportRecognized: conhecemos o formato, mas nao afirmamos que instala.
	SupportRecognized SupportLevel = "RECOGNIZED_ONLY"
	// SupportUnsupported: nao renderizamos.
	SupportUnsupported SupportLevel = "UNSUPPORTED"
)

// FormatSupport devolve o nivel de suporte, sem inventar.
func FormatSupport(t TargetFormat) SupportLevel {
	switch t {
	case TargetHades, TargetCodex, TargetClaudeCode:
		return SupportInstallable
	default:
		return SupportUnsupported
	}
}

// Render produz os bytes para o formato pedido.
//
// Capacidades que o formato NAO representa falham fechado -- nunca desaparecem em silencio.
//
//	*Uma capacidade que some no render e' uma permissao que o utilizador pensa ter pedido e nao pediu.*
func Render(p AgentPersona, target TargetFormat) ([]byte, error) {
	switch target {
	case TargetHades:
		return RenderHades(p)
	case TargetCodex:
		return renderCodex(p)
	case TargetClaudeCode:
		return renderClaudeCode(p)
	default:
		return nil, fmt.Errorf("%w: %s", ErrPersonaUnsupportedTarget, target)
	}
}

// renderCodex: as seccoes que o Codex le sao as instrucoes; o resto vai como nota auditavel.
func renderCodex(p AgentPersona) ([]byte, error) {
	n := p.Normalized()
	if err := n.Validate(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(n.Name)
	b.WriteString("\n\n")
	if n.Description != "" {
		b.WriteString(n.Description)
		b.WriteString("\n\n")
	}
	if n.Prompt != "" {
		b.WriteString(n.Prompt)
		b.WriteString("\n\n")
	}
	escreverLista(&b, "Ferramentas necessarias", n.Tools)
	escreverLista(&b, "Exige aprovacao humana", n.RequiresApproval)
	escreverLista(&b, "Proibido", n.Forbidden)
	return []byte(b.String()), nil
}

// renderClaudeCode: mesmo principio, com o cabecalho de proveniencia que o formato preserva.
func renderClaudeCode(p AgentPersona) ([]byte, error) {
	n := p.Normalized()
	if err := n.Validate(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("---\nname: ")
	b.WriteString(n.Slug)
	if n.Slug == "" {
		b.WriteString(n.ID)
	}
	b.WriteString("\ndescription: ")
	b.WriteString(n.Description)
	b.WriteString("\n---\n\n")
	b.WriteString("# ")
	b.WriteString(n.Name)
	b.WriteString("\n\n")
	if n.Prompt != "" {
		b.WriteString(n.Prompt)
		b.WriteString("\n\n")
	}
	escreverLista(&b, "Exige aprovacao humana", n.RequiresApproval)
	escreverLista(&b, "Proibido", n.Forbidden)
	return []byte(b.String()), nil
}

// PersonaLedgerEntry e' o registo de UMA instalacao gerida pelo HADES.
type PersonaLedgerEntry struct {
	PersonaID    string       `json:"persona_id"`
	Version      string       `json:"version"`
	Target       TargetFormat `json:"target"`
	Destination  string       `json:"destination"`
	Scope        string       `json:"scope"`
	SourceHash   string       `json:"source_hash"`   // hash da persona normalizada
	RenderedHash string       `json:"rendered_hash"` // hash dos bytes escritos
}

// DriftState classifica a distancia entre o ledger e o que esta' no disco.
type DriftState string

const (
	DriftCurrent         DriftState = "CURRENT"
	DriftOutdated        DriftState = "OUTDATED"
	DriftLocallyModified DriftState = "LOCALLY_MODIFIED"
	DriftRemoved         DriftState = "REMOVED"
	DriftForeign         DriftState = "FOREIGN_UNMANAGED"
	DriftConflicted      DriftState = "CONFLICTED"
	DriftBroken          DriftState = "BROKEN_DEPENDENCY"
)

func hashBytes(b []byte) string {
	soma := sha256.Sum256(b)
	return hex.EncodeToString(soma[:])
}

// ClassifyDrift compara o ledger com o estado observado.
//
// `atual` e' nil quando o ficheiro NAO EXISTE. `esperadoAgora` e' o render da persona que temos HOJE --
// se for diferente do que o ledger gravou, a persona de origem mudou.
func ClassifyDrift(entry PersonaLedgerEntry, atual []byte, esperadoAgora []byte) DriftState {
	if entry.SourceHash == "" || entry.RenderedHash == "" {
		// Sem hashes nao ha' reconciliacao possivel. Dizer "CURRENT" seria inventar.
		return DriftForeign
	}
	if atual == nil {
		return DriftRemoved
	}
	hashAtual := hashBytes(atual)
	if hashAtual != entry.RenderedHash {
		// O ficheiro mudou depois de o HADES escrever: alguem editou a mao.
		return DriftLocallyModified
	}
	if esperadoAgora != nil && hashBytes(esperadoAgora) != entry.RenderedHash {
		// O disco esta' como o HADES deixou, mas a persona de origem ja' rende outra coisa.
		return DriftOutdated
	}
	return DriftCurrent
}

// DescribeDrift devolve a accao segura para cada estado -- nunca "sobrescrever" por omissao.
func DescribeDrift(state DriftState) string {
	switch state {
	case DriftCurrent:
		return "Actual: nada a fazer."
	case DriftOutdated:
		return "A origem mudou. Actualize com diff e backup; nunca em silencio."
	case DriftLocallyModified:
		return "Editado fora do HADES. Preserve a edicao: actualizar por cima apagaria trabalho de uma pessoa."
	case DriftRemoved:
		return "O ficheiro desapareceu. Reinstale ou arquive o registo; nao invente que continua instalado."
	case DriftForeign:
		return "Registo sem hashes, ou ficheiro que o HADES nao escreveu. NAO remova: nao e' nosso."
	case DriftConflicted:
		return "Duas escritas concorrentes. Resolva com uma pessoa; o ultimo a escrever nao ganha por ser o ultimo."
	case DriftBroken:
		return "Dependencia em falta. Repare a dependencia antes de reclamar o agente."
	default:
		return "Estado desconhecido: nao aja."
	}
}
