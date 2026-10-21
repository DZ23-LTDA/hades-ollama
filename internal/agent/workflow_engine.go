package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MOTOR DE WORKFLOW DETERMINÍSTICO (V2.2 / adendo §3 P1-A, teste T6) — E-324.
//
// **AUDITORIA ANTES DE ESCREVER (e ela corrigiu o meu próprio inventário):** o inventário dizia P1-A "PARCIAL
// (workflow com retry/DLQ/compensação provados)". MEDIDO: **não existia motor de workflow** — `Workflow` aparecia em
// apenas 3 arquivos, e os arquivos `schedule_*` são o **scheduler do supervisor de negócios** (agendar tarefas), que é
// outra coisa. Chamar aquilo de "workflow estilo n8n" seria confundir **agendador** com **motor de automação**.
//
// O adendo exige: *"workflow determinístico estilo n8n/Dify com nodes reais, retry, idempotência, eventos, aprovação e
// evidência por nó"*. Este motor entrega o núcleo com as propriedades que a missão cobra:
//
//   - **NÓ DESCONHECIDO FALHA ALTO** — o V2.2 proíbe declarar sucesso com nós ignorados: um `Kind` que o motor não
//     reconhece **não** é pulado em silêncio, é erro. ***Um workflow que ignora o que não entende é uma mentira
//     organizada.***
//   - **IDEMPOTÊNCIA POR CHAVE** — a mesma chave de evento não produz segunda execução nem segundo efeito.
//   - **RETRY POR NÓ** com política explícita, e **evidência por nó** (tentativas, início/fim, saída, erro).
//   - **DLQ** — falha definitiva vai para a lista de mortos com o nó, o erro e a tentativa, em vez de sumir.
//   - **CONDIÇÃO AVALIADA DE VERDADE** — condição falsa faz o nó de ação ser **pulado com registro**, não "bem
//     sucedido por acidente".
//
// LIMITE DECLARADO: o motor executa **cadeia linear com `Next`** (sem fan-out/join/loops) e a ação é injetada por um
// `NodeExecutor` — quem decide o que é "ação real" é o runtime, não o motor. Não há interface visual nem aprovação
// humana nesta fatia; **aprovação** entra em fatia própria, e declará-la aqui seria a mesma promessa vazia que o
// V2.2 proíbe.

var (
	// ErrWorkflowUnknownNodeKind recusa um nó cujo tipo o motor não reconhece (nada de pular em silêncio).
	ErrWorkflowUnknownNodeKind = errors.New("workflow node kind is not supported")
	// ErrWorkflowUnknownNode recusa uma aresta que aponta para nó inexistente.
	ErrWorkflowUnknownNode = errors.New("workflow edge points to an unknown node")
	// ErrWorkflowInvalid recusa definição malformada.
	ErrWorkflowInvalid = errors.New("workflow definition is invalid")
	// ErrWorkflowLoopLimit recusa execução que não termina: ciclo entre nós ou passos acima do limite.
	//
	// ACHADO DA AUDITORIA (E-330): um `Next` cíclico fazia o motor girar PARA SEMPRE. O V2.2 fala em "loops
	// limitados" e proíbe loop infinito — ***um motor que não termina não é automação, é travamento.***
	ErrWorkflowLoopLimit = errors.New("workflow exceeded the step limit")
)

// Tipos de nó suportados. Qualquer outro valor FALHA.
// WorkflowMaxSteps limita a execução. É um BACKSTOP: o ciclo entre nós já é detectado de imediato.
const WorkflowMaxSteps = 64

const (
	WorkflowNodeTrigger   = "trigger"
	WorkflowNodeCondition = "condition"
	WorkflowNodeAction    = "action"
)

// Estados de execução.
const (
	WorkflowRunSucceeded  = "succeeded"
	WorkflowRunFailed     = "failed"
	WorkflowRunDeadLetter = "dead_letter"
)

// NodeRetryPolicy define quantas vezes um nó pode ser tentado e o intervalo entre tentativas.
type NodeRetryPolicy struct {
	MaxAttempts  int           `json:"max_attempts"`
	BackoffDelay time.Duration `json:"backoff_delay,omitempty"`
}

// WorkflowNode é um passo da automação.
type WorkflowNode struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Next string `json:"next,omitempty"`
	// Condicao é usada por nós `condition`: compara `Config["field"]` do evento com `Config["equals"]`.
	Config  map[string]string `json:"config,omitempty"`
	Retry   NodeRetryPolicy   `json:"retry,omitempty"`
	OnFalse string            `json:"on_false,omitempty"`
}

// WorkflowDefinition é a automação versionada.
type WorkflowDefinition struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Version        int            `json:"version"`
	StartNode      string         `json:"start_node"`
	Nodes          []WorkflowNode `json:"nodes"`
}

// NodeEvidence é a evidência POR NÓ exigida pelo V2.2.
type NodeEvidence struct {
	NodeID     string    `json:"node_id"`
	Kind       string    `json:"kind"`
	State      string    `json:"state"` // succeeded | failed | skipped
	Attempts   int       `json:"attempts"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Output     string    `json:"output,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// WorkflowRun é uma execução (ou repetição idempotente) de um workflow.
type WorkflowRun struct {
	ID             string         `json:"id"`
	WorkflowID     string         `json:"workflow_id"`
	OrganizationID string         `json:"organization_id"`
	State          string         `json:"state"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	NodeEvidence   []NodeEvidence `json:"node_evidence"`
	StartedAt      time.Time      `json:"started_at"`
	FinishedAt     time.Time      `json:"finished_at"`
	// Replayed é verdadeiro quando a execução devolvida é a MESMA de uma chave já vista (idempotência).
	Replayed bool `json:"replayed,omitempty"`
}

// NodeExecutor executa a AÇÃO de um nó. O motor não sabe o que é "ação real": quem sabe é o runtime.
type NodeExecutor func(ctx context.Context, workflow WorkflowDefinition, node WorkflowNode, event map[string]string) (string, error)

// WorkflowEngine executa workflows por organização, com persistência.
type WorkflowEngine struct {
	mu        sync.Mutex
	root      string
	executar  NodeExecutor
	porOrg    map[string]*workflowOrg
	carregado map[string]bool
}

type workflowOrg struct {
	Workflows  []WorkflowDefinition `json:"workflows"`
	Runs       []WorkflowRun        `json:"runs"`
	DeadLetter []WorkflowRun        `json:"dead_letter,omitempty"`
}

// NewWorkflowEngine cria o motor no diretório informado, com o executor de ações.
func NewWorkflowEngine(root string, executor NodeExecutor) *WorkflowEngine {
	return &WorkflowEngine{
		root: strings.TrimSpace(root), executar: executor,
		porOrg: map[string]*workflowOrg{}, carregado: map[string]bool{},
	}
}

func (e *WorkflowEngine) caminho(org string) string {
	return filepath.Join(e.root, sanitizeIDComponent(org)+".json")
}

func (e *WorkflowEngine) carregarLocked(org string) error {
	if e.carregado[org] {
		return nil
	}
	e.carregado[org] = true
	if e.root == "" {
		e.porOrg[org] = &workflowOrg{}
		return nil
	}
	dados, err := os.ReadFile(e.caminho(org))
	if err != nil {
		if os.IsNotExist(err) {
			e.porOrg[org] = &workflowOrg{}
			return nil
		}
		return err
	}
	alvo := &workflowOrg{}
	if len(dados) > 0 {
		if err := json.Unmarshal(dados, alvo); err != nil {
			return err
		}
	}
	e.porOrg[org] = alvo
	return nil
}

func (e *WorkflowEngine) salvarLocked(org string) error {
	if e.root == "" {
		return nil
	}
	if err := os.MkdirAll(e.root, 0o700); err != nil {
		return err
	}
	alvo := e.porOrg[org]
	if alvo == nil {
		alvo = &workflowOrg{}
	}
	return writeJSONAtomic(e.caminho(org), alvo)
}

// Register valida e registra um workflow. Definição malformada é RECUSADA aqui, não na execução.
func (e *WorkflowEngine) Register(def WorkflowDefinition) (WorkflowDefinition, error) {
	if e == nil {
		return WorkflowDefinition{}, ErrWorkflowInvalid
	}
	if err := requireOrganization(def.OrganizationID); err != nil {
		return WorkflowDefinition{}, fmt.Errorf("%w: %v", ErrWorkflowInvalid, err)
	}
	def.OrganizationID = consentScope(def.OrganizationID)
	def.Name = strings.TrimSpace(def.Name)
	if def.OrganizationID == "" || def.Name == "" || len(def.Nodes) == 0 || strings.TrimSpace(def.StartNode) == "" {
		return WorkflowDefinition{}, fmt.Errorf("%w: organização, nome, nós e nó inicial são obrigatórios", ErrWorkflowInvalid)
	}
	conhecidos := map[string]bool{}
	for _, no := range def.Nodes {
		if strings.TrimSpace(no.ID) == "" {
			return WorkflowDefinition{}, fmt.Errorf("%w: nó sem id", ErrWorkflowInvalid)
		}
		if conhecidos[no.ID] {
			return WorkflowDefinition{}, fmt.Errorf("%w: nó duplicado %q", ErrWorkflowInvalid, no.ID)
		}
		conhecidos[no.ID] = true
		switch no.Kind {
		case WorkflowNodeTrigger, WorkflowNodeCondition, WorkflowNodeAction:
		default:
			// NADA de nó ignorado: tipo desconhecido é recusado no registro E na execução.
			return WorkflowDefinition{}, fmt.Errorf("%w: %q (%s)", ErrWorkflowUnknownNodeKind, no.Kind, no.ID)
		}
	}
	if !conhecidos[def.StartNode] {
		return WorkflowDefinition{}, fmt.Errorf("%w: nó inicial %q não existe", ErrWorkflowInvalid, def.StartNode)
	}
	for _, no := range def.Nodes {
		for _, destino := range []string{no.Next, no.OnFalse} {
			if destino != "" && !conhecidos[destino] {
				return WorkflowDefinition{}, fmt.Errorf("%w: %q aponta para %q", ErrWorkflowUnknownNode, no.ID, destino)
			}
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.carregarLocked(def.OrganizationID); err != nil {
		return WorkflowDefinition{}, err
	}
	org := e.porOrg[def.OrganizationID]
	if def.Version <= 0 {
		def.Version = 1
	}
	if def.ID == "" {
		def.ID = "wf_" + uuid.NewString()
	}
	org.Workflows = append(org.Workflows, def)
	if err := e.salvarLocked(def.OrganizationID); err != nil {
		org.Workflows = org.Workflows[:len(org.Workflows)-1]
		return WorkflowDefinition{}, err
	}
	return def, nil
}

// Run executa o workflow a partir do evento recebido (por exemplo, um webhook já validado).
func (e *WorkflowEngine) Run(ctx context.Context, organizationID, workflowID string, event map[string]string, idempotencyKey string) (WorkflowRun, error) {
	if e == nil {
		return WorkflowRun{}, ErrWorkflowInvalid
	}
	organizationID = consentScope(organizationID)
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.carregarLocked(organizationID); err != nil {
		return WorkflowRun{}, err
	}
	org := e.porOrg[organizationID]

	// IDEMPOTÊNCIA: a mesma chave **NO MESMO WORKFLOW** devolve a MESMA execução — sem segundo efeito.
	//
	// 🚨 E-461 (MEDIDO): a comparação exigia **só a chave** ✗, e ela é procurada **antes** de o workflow ser resolvido
	// (`:284-293`, onde `workflowID` aparece pela primeira vez) ⇒ *`Run(org, wf-A, …, K)` e `Run(org, wf-B, …, K)`
	// passavam pelo mesmo laço, e o segundo recebia um run do PRIMEIRO workflow* ✓. **E não era só pular**: era
	// **devolver o resultado do outro workflow como se fosse deste**.
	//
	// ***A chave costuma vir do EVENTO (id de mensagem, id de transação), não do workflow*** ⇒ *dois workflows reagindo
	// ao **mesmo evento** — o caso mais natural de todos — fariam o segundo **não executar*** ✓. É a **mesma classe** do
	// ADV-75 (*E-460*): *comparar um identificador **sem o escopo a que ele pertence***. Lá o escopo era o **contacto**;
	// aqui é o **workflow**.
	//
	// **O escopo de TENANT já estava garantido**: `org.Runs` é por organização ✓ — *e é por isso que o defeito não
	// atravessava organizações.*
	if chave := strings.TrimSpace(idempotencyKey); chave != "" {
		for _, anterior := range org.Runs {
			if anterior.IdempotencyKey == chave && anterior.WorkflowID == workflowID {
				anterior.Replayed = true
				return anterior, nil
			}
		}
	}

	var definicao *WorkflowDefinition
	for i := range org.Workflows {
		if org.Workflows[i].ID == workflowID {
			definicao = &org.Workflows[i]
			break
		}
	}
	if definicao == nil {
		return WorkflowRun{}, fmt.Errorf("%w: workflow %s não existe nesta organização", ErrWorkflowInvalid, workflowID)
	}
	porID := map[string]WorkflowNode{}
	for _, no := range definicao.Nodes {
		porID[no.ID] = no
	}

	execucao := WorkflowRun{
		ID: "run_" + uuid.NewString(), WorkflowID: definicao.ID, OrganizationID: organizationID,
		State: WorkflowRunSucceeded, IdempotencyKey: strings.TrimSpace(idempotencyKey), StartedAt: time.Now().UTC(),
	}

	atual := definicao.StartNode
	visitados := map[string]bool{}
	passos := 0
	for atual != "" {
		// AUDITORIA (E-330): ciclo entre nós é detectado DE IMEDIATO, e o limite de passos é o backstop.
		passos++
		if visitados[atual] {
			execucao.NodeEvidence = append(execucao.NodeEvidence, NodeEvidence{
				NodeID: atual, State: "failed",
				Attempts: 1, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
				Error: fmt.Sprintf("%s: ciclo entre nós no nó %q", ErrWorkflowLoopLimit.Error(), atual),
			})
			execucao.State = WorkflowRunFailed
			break
		}
		if passos > WorkflowMaxSteps {
			execucao.NodeEvidence = append(execucao.NodeEvidence, NodeEvidence{
				NodeID: atual, State: "failed",
				Attempts: 1, StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
				Error: fmt.Sprintf("%s: %d passos", ErrWorkflowLoopLimit.Error(), WorkflowMaxSteps),
			})
			execucao.State = WorkflowRunFailed
			break
		}
		visitados[atual] = true
		no, existe := porID[atual]
		if !existe {
			execucao.NodeEvidence = append(execucao.NodeEvidence, NodeEvidence{
				NodeID: atual, State: "failed",
				Error: ErrWorkflowUnknownNode.Error(), StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC(),
			})
			execucao.State = WorkflowRunFailed
			break
		}
		evidencia := NodeEvidence{NodeID: no.ID, Kind: no.Kind, StartedAt: time.Now().UTC()}

		switch no.Kind {
		case WorkflowNodeTrigger:
			evidencia.State = "succeeded"
			evidencia.Output = fmt.Sprintf("evento recebido com %d campo(s)", len(event))
			evidencia.Attempts = 1
		case WorkflowNodeCondition:
			evidencia.Attempts = 1
			campo := ""
			if no.Config != nil {
				campo = no.Config["field"]
			}
			esperado := ""
			if no.Config != nil {
				esperado = no.Config["equals"]
			}
			obtido := event[campo]
			if obtido == esperado {
				evidencia.State = "succeeded"
				evidencia.Output = fmt.Sprintf("%s == %q", campo, esperado)
			} else {
				// Condição falsa: o ramo é PULADO COM REGISTRO, não "sucesso por acidente".
				evidencia.State = "skipped"
				evidencia.Output = fmt.Sprintf("%s=%q != %q", campo, obtido, esperado)
			}
		case WorkflowNodeAction:
			if e.executar == nil {
				evidencia.State = "failed"
				evidencia.Error = "nenhum executor de ação configurado"
				evidencia.Attempts = 1
			} else {
				tentativas := no.Retry.MaxAttempts
				if tentativas <= 0 {
					tentativas = 1
				}
				for tentativa := 1; tentativa <= tentativas; tentativa++ {
					evidencia.Attempts = tentativa
					saida, err := e.executar(ctx, *definicao, no, event)
					if err == nil {
						evidencia.State = "succeeded"
						evidencia.Output = saida
						evidencia.Error = ""
						break
					}
					evidencia.State = "failed"
					evidencia.Error = err.Error()
					if tentativa < tentativas && no.Retry.BackoffDelay > 0 {
						time.Sleep(no.Retry.BackoffDelay)
					}
				}
			}
		default:
			evidencia.State = "failed"
			evidencia.Attempts = 1
			evidencia.Error = fmt.Sprintf("%s: %s", ErrWorkflowUnknownNodeKind.Error(), no.Kind)
		}

		evidencia.FinishedAt = time.Now().UTC()
		execucao.NodeEvidence = append(execucao.NodeEvidence, evidencia)

		if evidencia.State == "failed" {
			execucao.State = WorkflowRunFailed
			break
		}
		if evidencia.State == "skipped" && no.Kind == WorkflowNodeCondition {
			atual = no.OnFalse
			continue
		}
		atual = no.Next
	}

	execucao.FinishedAt = time.Now().UTC()
	if execucao.State == WorkflowRunFailed {
		// DLQ: falha definitiva vai para a lista de mortos com nó, erro e tentativas — falha não some.
		execucao.State = WorkflowRunDeadLetter
		org.DeadLetter = append(org.DeadLetter, execucao)
	}
	org.Runs = append(org.Runs, execucao)
	if err := e.salvarLocked(organizationID); err != nil {
		org.Runs = org.Runs[:len(org.Runs)-1]
		return WorkflowRun{}, err
	}
	return execucao, nil
}

// Runs devolve as execuções da organização; DeadLetters, as que falharam em definitivo.
func (e *WorkflowEngine) Runs(organizationID string) ([]WorkflowRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := e.carregarLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]WorkflowRun(nil), e.porOrg[organizationID].Runs...), nil
}

// DeadLetters devolve a DLQ da organização.
func (e *WorkflowEngine) DeadLetters(organizationID string) ([]WorkflowRun, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := e.carregarLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]WorkflowRun(nil), e.porOrg[organizationID].DeadLetter...), nil
}
