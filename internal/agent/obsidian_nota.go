package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Camada duravel da memoria: uma nota Obsidian com frontmatter versionado.
//
// Esta fatia NAO escreve no vault e NAO e' um segundo store: ela so' converte uma `Memoria` (a autoridade
// canonica, ver memoria_viva.go) no formato de nota e de volta. O adaptador completo (acha o vault, lock,
// watcher, DLP) encaixa em cima disto.
//
// Duas regras que o formato tem de sustentar, e que os testes discriminam:
//   1. NEGACAO POR DEFEITO -- nota sem escopo ou sem memory_id nao e' montavel. Nao existe escopo "implicito".
//   2. IDEMPOTENCIA -- reanexar a mesma memory_id nao duplica a nota nem reescreve os bytes.

// EscopoDeMemoria: a que esfera a memoria pertence. Lista fechada, deliberadamente.
type EscopoDeMemoria string

const (
	EscopoSession          EscopoDeMemoria = "SESSION"
	EscopoPerson           EscopoDeMemoria = "PERSON"
	EscopoTenant           EscopoDeMemoria = "TENANT"
	EscopoProject          EscopoDeMemoria = "PROJECT"
	EscopoWorkspace        EscopoDeMemoria = "WORKSPACE"
	EscopoWorktree         EscopoDeMemoria = "WORKTREE"
	EscopoGoal             EscopoDeMemoria = "GOAL"
	EscopoAgent            EscopoDeMemoria = "AGENT"
	EscopoSharedAuthorized EscopoDeMemoria = "SHARED_AUTHORIZED"
)

// VersaoDoSchemaDaNota sobe quando o frontmatter muda de forma incompativel.
const VersaoDoSchemaDaNota = 1

var (
	// ErrEscopoAusente: a nota nao declara escopo. Recusa-se antes de escrever.
	ErrEscopoAusente = errors.New("memory note without scope")
	// ErrEscopoDesconhecido: o escopo nao esta' na lista fechada.
	ErrEscopoDesconhecido = errors.New("unknown memory scope")
	// ErrCampoDesconhecido: campo fora do schema. Nao se ignora em silencio (um typo viraria campo ausente).
	ErrCampoDesconhecido = errors.New("memory note with unknown field")
	// ErrNotaSemFrontmatter: o ficheiro nao comeca com delimitador `---`.
	ErrNotaSemFrontmatter = errors.New("memory note without frontmatter")
	// ErrNotaSemID: nota sem `memory_id`.
	ErrNotaSemID = errors.New("memory note without memory_id")
	// ErrConteudoAlterado: o corpo nao corresponde ao `content_hash` gravado. Alguem editou a nota (ou ela foi
	// truncada) -- nao se reconcilia em silencio, avisa-se.
	ErrConteudoAlterado = errors.New("memory note content does not match its content_hash")
)

// MetaDaNota: o frontmatter. So' os campos que a fatia usa; `supersedes` fica declarado para nao ser inventado
// depois com outro nome.
type MetaDaNota struct {
	MemoryID      string
	SchemaVersion int
	Escopo        EscopoDeMemoria
	Tenant        string
	Project       string
	Goal          string
	Author        string
	Aprovador     string
	CreatedAt     string
	UpdatedAt     string
	Confidence    string
	Sensitivity   string
	Retention     string
	ContentHash   string
	Supersedes    string
}

func escopoConhecido(e EscopoDeMemoria) bool {
	switch e {
	case EscopoSession, EscopoPerson, EscopoTenant, EscopoProject, EscopoWorkspace,
		EscopoWorktree, EscopoGoal, EscopoAgent, EscopoSharedAuthorized:
		return true
	}
	return false
}

// HashDoConteudo: identificador do corpo. Serve para detectar edicao humana e para recusar reescrita silenciosa.
//
// Normaliza as quebras de linha FINAIS: `MontarNota` garante que o corpo termina em "\n", e sem normalizar o hash
// gravado nunca bateria com o corpo lido -- e uma edicao humana seria indistinguivel de uma ida ao disco.
func HashDoConteudo(conteudo string) string {
	s := sha256.Sum256([]byte(strings.TrimRight(conteudo, "\n")))
	return hex.EncodeToString(s[:])
}

// MontarNota produz os bytes da nota. Determinista: a mesma entrada, na mesma versao, da' os mesmos bytes.
func MontarNota(meta MetaDaNota, m Memoria) ([]byte, error) {
	// A autoridade do rotulo continua a ser `Memoria.Validar` -- nao se reimplementa aqui.
	if err := m.Validar(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(meta.Escopo)) == "" {
		return nil, ErrEscopoAusente
	}
	if !escopoConhecido(meta.Escopo) {
		return nil, fmt.Errorf("%w: %q", ErrEscopoDesconhecido, string(meta.Escopo))
	}
	if strings.TrimSpace(meta.MemoryID) == "" {
		return nil, ErrNotaSemID
	}

	versao := meta.SchemaVersion
	if versao == 0 {
		versao = VersaoDoSchemaDaNota
	}

	campos := [][2]string{
		{"memory_id", meta.MemoryID},
		{"schema_version", strconv.Itoa(versao)},
		{"scope", string(meta.Escopo)},
		{"rotulo", string(m.Rotulo)},
		{"tenant_id", meta.Tenant},
		{"project_id", meta.Project},
		{"goal_id", meta.Goal},
		{"source", m.Fonte},
		{"aprovador", m.Aprovador},
		{"author", meta.Author},
		{"created_at", meta.CreatedAt},
		{"updated_at", meta.UpdatedAt},
		{"confidence", meta.Confidence},
		{"sensitivity", meta.Sensitivity},
		{"retention", meta.Retention},
		{"content_hash", HashDoConteudo(m.Conteudo)},
		{"supersedes", meta.Supersedes},
	}

	var b strings.Builder
	b.WriteString("---\n")
	for _, c := range campos {
		b.WriteString(c[0])
		b.WriteString(": ")
		b.WriteString(c[1])
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	b.WriteString(m.Conteudo)
	if !strings.HasSuffix(m.Conteudo, "\n") {
		b.WriteString("\n")
	}
	return []byte(b.String()), nil
}

// LerNota faz o caminho inverso. Recusa campo desconhecido em vez de o descartar: um campo ignorado e' um campo
// que o leitor acredita ter lido.
func LerNota(dados []byte) (MetaDaNota, Memoria, error) {
	var meta MetaDaNota
	var m Memoria

	texto := strings.ReplaceAll(string(dados), "\r\n", "\n")
	if !strings.HasPrefix(texto, "---\n") {
		return meta, m, ErrNotaSemFrontmatter
	}
	resto := texto[len("---\n"):]
	fim := strings.Index(resto, "\n---\n")
	if fim < 0 {
		return meta, m, ErrNotaSemFrontmatter
	}
	cabecalho := resto[:fim]
	corpo := resto[fim+len("\n---\n"):]
	m.Conteudo = corpo

	for _, linha := range strings.Split(cabecalho, "\n") {
		if linha == "" {
			continue
		}
		i := strings.Index(linha, ": ")
		if i < 0 {
			return meta, m, fmt.Errorf("%w: %q", ErrCampoDesconhecido, linha)
		}
		chave, valor := linha[:i], linha[i+2:]
		switch chave {
		case "memory_id":
			meta.MemoryID = valor
			m.ID = valor
		case "schema_version":
			n, err := strconv.Atoi(valor)
			if err != nil {
				return meta, m, fmt.Errorf("memory note with invalid schema_version: %q", valor)
			}
			meta.SchemaVersion = n
		case "scope":
			meta.Escopo = EscopoDeMemoria(valor)
		case "rotulo":
			m.Rotulo = RotuloDeMemoria(valor)
		case "tenant_id":
			meta.Tenant = valor
		case "project_id":
			meta.Project = valor
		case "goal_id":
			meta.Goal = valor
		case "source":
			m.Fonte = valor
		case "aprovador":
			m.Aprovador = valor
		case "author":
			meta.Author = valor
		case "created_at":
			meta.CreatedAt = valor
		case "updated_at":
			meta.UpdatedAt = valor
		case "confidence":
			meta.Confidence = valor
		case "sensitivity":
			meta.Sensitivity = valor
		case "retention":
			meta.Retention = valor
		case "content_hash":
			meta.ContentHash = valor
		case "supersedes":
			meta.Supersedes = valor
		default:
			return meta, m, fmt.Errorf("%w: %q", ErrCampoDesconhecido, chave)
		}
	}

	if strings.TrimSpace(meta.MemoryID) == "" {
		return meta, m, ErrNotaSemID
	}
	if !escopoConhecido(meta.Escopo) {
		return meta, m, fmt.Errorf("%w: %q", ErrEscopoDesconhecido, string(meta.Escopo))
	}
	return meta, m, nil
}

// AnexarNotaIdempotente acrescenta a nota a um ficheiro de notas existente.
//
// Devolve `anexou=false` e os bytes ORIGINAIS quando a `memory_id` ja' la' esta'. Nao compara o corpo: reanexar
// a mesma memoria nao pode ser uma reescrita, senao o "append idempotente" seria um "append que as vezes
// sobrescreve".
func AnexarNotaIdempotente(existente []byte, meta MetaDaNota, m Memoria) ([]byte, bool, error) {
	nota, err := MontarNota(meta, m)
	if err != nil {
		return nil, false, err
	}
	if idJaPresente(existente, meta.MemoryID) {
		return existente, false, nil
	}

	saida := make([]byte, 0, len(existente)+len(nota)+2)
	saida = append(saida, existente...)
	if len(existente) > 0 {
		if !strings.HasSuffix(string(existente), "\n") {
			saida = append(saida, '\n')
		}
		saida = append(saida, '\n')
	}
	saida = append(saida, nota...)
	return saida, true, nil
}

// IntegridadeDaNota diz se o corpo lido ainda corresponde ao `content_hash` gravado no frontmatter.
//
// Devolve erro de proposito em vez de um bool: quem le precisa de saber QUE a nota divergiu, e nao apenas que
// "algo esta' diferente". Nao reescreve nada -- detetar uma edicao humana nao autoriza apaga-la.
func IntegridadeDaNota(meta MetaDaNota, m Memoria) error {
	if strings.TrimSpace(meta.ContentHash) == "" {
		return fmt.Errorf("%w: nota sem content_hash gravado", ErrConteudoAlterado)
	}
	if HashDoConteudo(m.Conteudo) != meta.ContentHash {
		return fmt.Errorf("%w: memory_id %q", ErrConteudoAlterado, meta.MemoryID)
	}
	return nil
}

func idJaPresente(existente []byte, id string) bool {
	if len(existente) == 0 || strings.TrimSpace(id) == "" {
		return false
	}
	alvo := "memory_id: " + id
	for _, linha := range strings.Split(strings.ReplaceAll(string(existente), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(linha) == alvo {
			return true
		}
	}
	return false
}
