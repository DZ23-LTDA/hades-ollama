package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	// ParentID responde a OUTRO comentario do MESMO projeto (referencia por id, nao por indice).
	// Complemento 48 / FASE 2 ("respostas vinculadas") - antes disto o comentario era uma lista plana.
	ParentID string `json:"parent_id,omitempty"`
	// ArtifactID pendura o comentario num ENTREGAVEL (FASE 7).
	//
	// ATENCAO, MEDIDO: nao existe registro de entregaveis NESTE store, logo `ArtifactID` **nao e' validado
	// aqui** - ele e' DECLARADO, nao verificado. *Quem grava um id errado grava um id errado*, e quem for
	// construir a tela tem de resolver o artefato no registro que existir (artefatos de missao), nao aqui.
	ArtifactID string `json:"artifact_id,omitempty"`

	// --- ANCORAGEM EM DIFF (adendo V3, FASE 11: "revisao de diffs e feedback ao agente") ---
	//
	// LIMITE DECLARADO: nada aqui e' verificado contra o diff real. Este store NAO le' o repositorio nem
	// calcula diff: ele GRAVA a ancoragem que quem comentou declarou. Validar `File`/`Line` contra o diff
	// exige o runtime de worktree/git, que nao vive neste arquivo. *O que o store garante e' que a
	// ancoragem e' PRESERVADA e que um comentario sem alvo legivel e' RECUSADO -- nao que a linha 42
	// ainda seja a linha 42.*
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Side      string `json:"side,omitempty"` // "old" | "new" -- lado do diff
	CommitRef string `json:"commit_ref,omitempty"`

	// Stale marca comentario cujo diff de referencia deixou de ser o corrente. Quem promove um novo diff
	// marca os antigos como obsoletos em vez de os apagar: *comentario obsoleto e' HISTORIA, nao lixo.*
	Stale bool `json:"stale,omitempty"`

	// Resolved/ResolvedBy/ResolvedAt: resolucao EXPLICITA. Um comentario nao "desaparece" por ter sido
	// resolvido -- ele fica com quem resolveu e quando, para a trilha ser auditavel.
	Resolved   bool       `json:"resolved,omitempty"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	// AuthorAgentID identifica o AGENTE autor, quando o comentario foi gerado por agente e nao por humano.
	// E' o que permite a regra da FASE 11: **o agente que escreveu nao pode resolver como se fosse
	// aprovacao independente.**
	AuthorAgentID string `json:"author_agent_id,omitempty"`
}
type Presence struct {
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updated_at"`
	// Origem e' QUEM DECLAROU este estado. Antes disto o `Status` era texto livre SEM origem, e um status
	// forjado era indistinguivel de um real (complemento 48 / FASE 2: "presenca com origem verificavel").
	//
	// LIMITE DECLARADO: este store ATRIBUI a origem, nao a AUTENTICA. `SetPresenceWithOrigin` recusa origem
	// vazia e grava quem declarou; **um ator que declare a origem de OUTRO nao e' detectado aqui** - *quem
	// autentica e' a sessao/credencial, que nao vive neste arquivo*. O que muda e' que agora a declaracao
	// **tem autor** e a ausencia de autor e' RECUSADA, em vez de virar texto anonimo.
	Origem string `json:"origem,omitempty"`
}
type CollaborationSnapshot struct {
	Comments  []Comment  `json:"comments"`
	Presence  []Presence `json:"presence"`
	UpdatedAt time.Time  `json:"updated_at"`
}
type CollaborationStore struct {
	mu           sync.Mutex
	root         string
	comments     map[string][]Comment
	presence     map[string]map[string]Presence
	participants map[string][]Participant
	rooms        map[string][]Room
}

func NewCollaborationStore(root string) (*CollaborationStore, error) {
	store := &CollaborationStore{root: root, comments: map[string][]Comment{}, presence: map[string]map[string]Presence{}, participants: map[string][]Participant{}, rooms: map[string][]Room{}}
	if strings.TrimSpace(root) == "" {
		return store, nil
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(root, "comments.json"), &store.comments); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load collaboration comments: %w", err)
	}
	if err := readJSON(filepath.Join(root, "presence.json"), &store.presence); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load collaboration presence: %w", err)
	}
	// PARTICIPANTS (complemento 48, FASE 2/F6): humanos e agentes na MESMA lista - *um agente que nao esta' na
	// lista nao esta' na sala*, e e' isso que impede um mencionado inexistente de ser tratado como membro.
	if err := readJSON(filepath.Join(root, "participantes.json"), &store.participants); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load collaboration participants: %w", err)
	}
	// ROOMS (complemento 48, FASE 2): a sala e' o lugar onde os membros se reúnem. *Uma sala que aceitasse
	// qualquer id seria uma via paralela de permissao* - por isso `JoinRoom` checa `participants` primeiro.
	if err := readJSON(filepath.Join(root, "rooms.json"), &store.rooms); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load collaboration rooms: %w", err)
	}
	return store, nil
}

// VinculoDoComentario declara ONDE o comentario se pendura: numa RESPOSTA (`ParentID`) e/ou num ENTREGAVEL
// (`ArtifactID`). Complemento 48, FASES 2 e 7. *Vazio e' vazio* - e o vinculo que aponta para o nada e'
// **recusado** (ver `AddCommentWithLink`).
type VinculoDoComentario struct {
	ParentID   string
	ArtifactID string

	// Ancoragem em diff (adendo V3, FASE 11). Opcional: vinculo vazio continua a gravar um comentario
	// simples, exatamente como antes -- a extensao e' COMPATIVEL com os chamadores existentes.
	File          string
	Line          int
	Side          string
	CommitRef     string
	AuthorAgentID string
}

// ErrComentarioSemPai: o comentario respondido nao existe NESTE projeto. *Uma resposta orfa e' um fio
// partido: melhor recusar na gravacao do que descobrir na leitura.*
var ErrComentarioSemPai = errors.New("comment parent does not exist in this project")

// ErrAncoragemInvalida: um comentario de diff sem ficheiro, ou com linha negativa, nao tem alvo.
// *Um comentario "na linha -1 do ficheiro vazio" nao e' revisao: e' ruido com aparencia de dado.*
var ErrAncoragemInvalida = errors.New("diff comment requires a file and a non-negative line")

// ErrAutoResolucaoDeAgente: o AGENTE autor nao pode resolver o proprio comentario.
//
// E' a regra textual da FASE 11: «O proprio agente nao pode marcar sua revisao como aprovacao
// independente.» Sem esta guarda, um agente escreveria uma observacao e fecharia o proprio achado --
// e a trilha de revisao passaria a mentir sem que ninguem tivesse mentido.
var ErrAutoResolucaoDeAgente = errors.New("the author agent cannot resolve its own review comment")

// AddCommentWithLink grava um comentario COM vinculo, validando o que da' para validar.
//
// Mantem `AddComment` (assinatura antiga) funcionando: aquele passa a delegar aqui com vinculo vazio, de modo
// que existe **um unico caminho de escrita** (uma unica persistencia, um unico rollback) - nao dois.
func (s *CollaborationStore) AddCommentWithLink(projectID, userID, body string, vinculo VinculoDoComentario) (Comment, error) {
	projectID = strings.TrimSpace(projectID)
	userID = strings.TrimSpace(userID)
	body = strings.TrimSpace(body)
	if projectID == "" || userID == "" || body == "" {
		return Comment{}, errors.New("project, user and comment are required")
	}
	if len(body) > 10000 {
		return Comment{}, errors.New("comment is too long")
	}
	pai := strings.TrimSpace(vinculo.ParentID)
	artefato := strings.TrimSpace(vinculo.ArtifactID)
	ficheiro := strings.TrimSpace(vinculo.File)
	lado := strings.TrimSpace(vinculo.Side)
	if lado == "" {
		lado = "new"
	}
	// ANCORAGEM: ou nao ha' alvo nenhum (comentario simples, como antes), ou ha' ficheiro E linha >= 0.
	// Recusar o meio-termo e' o que impede um comentario de diff sem alvo de entrar no ledger.
	if ficheiro == "" && vinculo.Line != 0 {
		return Comment{}, fmt.Errorf("%w: line without file", ErrAncoragemInvalida)
	}
	if ficheiro != "" && vinculo.Line < 0 {
		return Comment{}, fmt.Errorf("%w: negative line", ErrAncoragemInvalida)
	}
	if lado != "old" && lado != "new" {
		return Comment{}, fmt.Errorf("%w: side must be old or new", ErrAncoragemInvalida)
	}
	comment := Comment{
		ID: "com_" + uuid.NewString(), ProjectID: projectID, UserID: userID, Body: body,
		CreatedAt: time.Now().UTC(), ParentID: pai, ArtifactID: artefato,
		File: ficheiro, Line: vinculo.Line, Side: lado, CommitRef: strings.TrimSpace(vinculo.CommitRef),
		AuthorAgentID: strings.TrimSpace(vinculo.AuthorAgentID),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// O PAI TEM DE EXISTIR NO MESMO PROJETO - a checagem e' feita sob o lock, contra o estado corrente.
	if pai != "" && !s.comentarioExisteLocked(projectID, pai) {
		return Comment{}, fmt.Errorf("%w: %s", ErrComentarioSemPai, pai)
	}
	previous := append([]Comment(nil), s.comments[projectID]...)
	s.comments[projectID] = append(s.comments[projectID], comment)
	if err := s.persistLocked(); err != nil {
		if len(previous) == 0 {
			delete(s.comments, projectID)
		} else {
			s.comments[projectID] = previous
		}
		return Comment{}, err
	}
	return comment, nil
}

func (s *CollaborationStore) comentarioExisteLocked(projectID, commentID string) bool {
	for _, item := range s.comments[projectID] {
		if item.ID == commentID {
			return true
		}
	}
	return false
}

// Respostas devolve as respostas DIRETAS de um comentario, em ordem cronologica. E' o que a FASE 2 pede
// ("respostas vinculadas as mensagens") e o que uma tela de time consome sem varrer a lista inteira.
func (s *CollaborationStore) Respostas(projectID, parentID string) []Comment {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		return nil
	}
	todos := s.Comments(projectID)
	respostas := make([]Comment, 0)
	for _, item := range todos {
		if item.ParentID == parentID {
			respostas = append(respostas, item)
		}
	}
	return respostas
}

// AddComment grava um comentario simples (sem vinculo). Delega para o caminho unico de escrita.
func (s *CollaborationStore) AddComment(projectID, userID, body string) (Comment, error) {
	return s.AddCommentWithLink(projectID, userID, body, VinculoDoComentario{})
}

func (s *CollaborationStore) Comments(projectID string) []Comment {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]Comment(nil), s.comments[projectID]...)
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result
}

// ErrPresencaSemOrigem: presenca sem quem a declarou nao entra no store. *Um estado que nao se sabe de quem
// veio nao e' informacao: e' ruido com aparencia de informacao.*
var ErrPresencaSemOrigem = errors.New("presence requires an origin")

// SetPresenceWithOrigin registra a presenca ATRIBUINDO quem a declarou (complemento 48, FASE 2).
func (s *CollaborationStore) SetPresenceWithOrigin(projectID, userID, status, origem string) (Presence, error) {
	projectID = strings.TrimSpace(projectID)
	userID = strings.TrimSpace(userID)
	status = strings.TrimSpace(status)
	origem = strings.TrimSpace(origem)
	if projectID == "" || userID == "" {
		return Presence{}, errors.New("project and user are required")
	}
	if origem == "" {
		return Presence{}, ErrPresencaSemOrigem
	}
	if status == "" {
		status = "online"
	}
	presence := Presence{ProjectID: projectID, UserID: userID, Status: status, UpdatedAt: time.Now().UTC(), Origem: origem}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.presence[projectID] == nil {
		s.presence[projectID] = map[string]Presence{}
	}
	previous, existed := s.presence[projectID][userID]
	s.presence[projectID][userID] = presence
	if err := s.persistLocked(); err != nil {
		if existed {
			s.presence[projectID][userID] = previous
		} else {
			delete(s.presence[projectID], userID)
			if len(s.presence[projectID]) == 0 {
				delete(s.presence, projectID)
			}
		}
		return Presence{}, err
	}
	return presence, nil
}

// SetPresence mantem a assinatura antiga e significa **AUTO-DECLARACAO**: a origem e' o proprio `userID`.
// Nao ha caminho sem origem - o antigo passou a delegar, de modo que existe **um unico caminho de escrita**.
func (s *CollaborationStore) SetPresence(projectID, userID, status string) (Presence, error) {
	return s.SetPresenceWithOrigin(projectID, userID, status, strings.TrimSpace(userID))
}

func (s *CollaborationStore) Presence(projectID string) []Presence {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Presence, 0, len(s.presence[projectID]))
	for _, item := range s.presence[projectID] {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UserID < result[j].UserID })
	return result
}

func (s *CollaborationStore) Snapshot(projectID string) CollaborationSnapshot {
	return CollaborationSnapshot{Comments: s.Comments(projectID), Presence: s.Presence(projectID), UpdatedAt: time.Now().UTC()}
}

func (s *CollaborationStore) persistLocked() error {
	if s.root == "" {
		return nil
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "comments.json"), s.comments); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "presence.json"), s.presence); err != nil {
		return err
	}
	if err := writeJSONAtomic(filepath.Join(s.root, "participantes.json"), s.participants); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(s.root, "rooms.json"), s.rooms)
}

// ResolveComment fecha um comentario de revisao com autor e instante explicitos.
//
// A guarda que faz esta funcao valer a pena e' a da FASE 11: **um agente nao fecha o proprio achado.**
// Recusa-se quando `resolverID` e' igual ao `AuthorAgentID` do comentario -- ou seja, quando o proprio
// agente autor tenta resolve-lo. Um humano a resolver um comentario de agente e' o caso normal; um
// agente a resolver o comentario de OUTRO agente tambem passa (a regra e' contra a AUTO-resolucao, nao
// contra agentes em geral).
//
// A resolucao NAO apaga o comentario: grava quem e quando, para a trilha poder ser auditada depois.
// *Um achado que desaparece ao ser resolvido nao e' um achado resolvido: e' um achado escondido.*
func (s *CollaborationStore) ResolveComment(projectID, commentID, resolverID string) (Comment, error) {
	projectID = strings.TrimSpace(projectID)
	commentID = strings.TrimSpace(commentID)
	resolverID = strings.TrimSpace(resolverID)
	if projectID == "" || commentID == "" || resolverID == "" {
		return Comment{}, errors.New("project, comment and resolver are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lista := s.comments[projectID]
	indice := -1
	for i := range lista {
		if lista[i].ID == commentID {
			indice = i
			break
		}
	}
	if indice < 0 {
		return Comment{}, fmt.Errorf("%w: %s", ErrComentarioSemPai, commentID)
	}
	anterior := lista[indice]
	if anterior.AuthorAgentID != "" && anterior.AuthorAgentID == resolverID {
		return Comment{}, fmt.Errorf("%w: %s", ErrAutoResolucaoDeAgente, resolverID)
	}
	if anterior.Resolved {
		// Idempotente por natureza: resolver duas vezes devolve o mesmo estado, sem reescrever o autor
		// original da resolucao. *Reexecutar uma acao ja' feita nao pode mudar quem a fez.*
		return anterior, nil
	}
	agora := time.Now().UTC()
	lista[indice].Resolved = true
	lista[indice].ResolvedBy = resolverID
	lista[indice].ResolvedAt = &agora
	if err := s.persistLocked(); err != nil {
		lista[indice] = anterior
		return Comment{}, err
	}
	return lista[indice], nil
}

// MarkCommentsStale marca como obsoletos os comentarios de um commit de referencia.
//
// Chamado quando um diff novo passa a ser o corrente. Os comentarios NAO sao apagados nem resolvidos:
// ficam `Stale` para a UI poder mostrar que aquela ancoragem ja' nao aponta para o diff actual.
// *Comentario obsoleto e' historia; apaga-lo seria reescrever a revisao.*
func (s *CollaborationStore) MarkCommentsStale(projectID, commitRef string) (int, error) {
	projectID = strings.TrimSpace(projectID)
	commitRef = strings.TrimSpace(commitRef)
	if projectID == "" || commitRef == "" {
		return 0, errors.New("project and commit ref are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	lista := s.comments[projectID]
	anteriores := append([]Comment(nil), lista...)
	marcados := 0
	for i := range lista {
		if lista[i].CommitRef == commitRef && !lista[i].Stale {
			lista[i].Stale = true
			marcados++
		}
	}
	if marcados == 0 {
		return 0, nil
	}
	if err := s.persistLocked(); err != nil {
		copy(s.comments[projectID], anteriores)
		return 0, err
	}
	return marcados, nil
}
