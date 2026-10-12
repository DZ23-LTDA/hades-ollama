package agent

import (
	"context"
	"os"
	"time"
)

type MissionState string

const (
	MissionCreated          MissionState = "CREATED"
	MissionPlanning         MissionState = "PLANNING"
	MissionAwaitingApproval MissionState = "AWAITING_APPROVAL"
	MissionReady            MissionState = "READY"
	MissionRunning          MissionState = "RUNNING"
	MissionObserving        MissionState = "OBSERVING"
	MissionRecovering       MissionState = "RECOVERING"
	MissionCompleted        MissionState = "COMPLETED"
	MissionFailed           MissionState = "FAILED"
	MissionCancelled        MissionState = "CANCELLED"
)

type StepState string

const (
	StepPending   StepState = "PENDING"
	StepRunning   StepState = "RUNNING"
	StepSucceeded StepState = "SUCCEEDED"
	StepFailed    StepState = "FAILED"
	StepBlocked   StepState = "BLOCKED"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalRejected ApprovalStatus = "REJECTED"
)

type RiskClass string

const (
	RiskRead               RiskClass = "read"
	RiskWrite              RiskClass = "write"
	RiskExternalSideEffect RiskClass = "external_side_effect"
	RiskDestructive        RiskClass = "destructive"
)

type CreateMissionRequest struct {
	MissionID string `json:"-"`
	ActorID   string `json:"-"`
	// IdempotencyKey torna a criação repetível sem duplicar missão. Quando
	// presente, o ID é DERIVADO da chave (organização + chave), então repetir a
	// mesma criação devolve a missão já existente em vez de criar outra. É o
	// pré-requisito para retry de automação: sem isso, repetir um passo que falhou
	// depois de persistir geraria missões fantasma.
	IdempotencyKey   string   `json:"-"`
	Objective        string   `json:"objective"`
	Provider         string   `json:"provider,omitempty"`
	Model            string   `json:"model,omitempty"`
	Workspace        string   `json:"workspace,omitempty"`
	IsolateWorkspace bool     `json:"isolate_workspace,omitempty"`
	ProjectID        string   `json:"project_id,omitempty"`
	OrganizationID   string   `json:"organization_id,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	AutoRun          bool     `json:"auto_run,omitempty"`
	IsolateWorktree  bool     `json:"isolate_worktree,omitempty"`
	WorktreeBranch   string   `json:"worktree_branch,omitempty"`
	AutoRepair       bool     `json:"auto_repair,omitempty"`
	MaxRepairTries   int      `json:"max_repair_tries,omitempty"`
}

type Mission struct {
	ID                      string             `json:"id"`
	Version                 int64              `json:"version"`
	Objective               string             `json:"objective"`
	Provider                string             `json:"provider,omitempty"`
	Model                   string             `json:"model,omitempty"`
	Workspace               string             `json:"workspace,omitempty"`
	WorkspaceIdentity       string             `json:"workspace_identity,omitempty"`
	WorkspaceIsolated       bool               `json:"workspace_isolated,omitempty"`
	WorkspaceSnapshotID     string             `json:"workspace_snapshot_id,omitempty"`
	WorkspaceSnapshotSHA256 string             `json:"workspace_snapshot_sha256,omitempty"`
	ProjectID               string             `json:"project_id,omitempty"`
	OrganizationID          string             `json:"organization_id,omitempty"`
	Capabilities            []string           `json:"capabilities,omitempty"`
	AutoRun                 bool               `json:"auto_run,omitempty"`
	State                   MissionState       `json:"state"`
	Plan                    []Step             `json:"plan"`
	Approvals               []Approval         `json:"approvals,omitempty"`
	Artifacts               []ArtifactManifest `json:"artifacts,omitempty"`
	GitRepoRoot             string             `json:"git_repo_root,omitempty"`
	GitBranch               string             `json:"git_branch,omitempty"`
	GitWorktreePath         string             `json:"git_worktree_path,omitempty"`
	GitBaseCommit           string             `json:"git_base_commit,omitempty"`
	GitMergeStatus          string             `json:"git_merge_status,omitempty"`
	GitWorktreeActive       bool               `json:"git_worktree_active,omitempty"`
	AutoRepair              bool               `json:"auto_repair,omitempty"`
	MaxRepairTries          int                `json:"max_repair_tries,omitempty"`
	LastError               string             `json:"last_error,omitempty"`
	// Usage leva o consumo real de modelo (tokens e custo) para dentro do registro da
	// missão. Sem isto, o custo ficava só no catálogo de provedores e a linha do tempo
	// não conseguia responder "quanto custou esta missão".
	Usage *ModelUsage `json:"usage,omitempty"`
	// UsageOrigin diz DE ONDE veio o consumo registrado em Usage. Hoje só existe a origem
	// "planning": a chamada de planejamento produz o plano INTEIRO e não há chamada de modelo por
	// etapa. Sem esta marca, um custo único poderia ser lido como se estivesse distribuído por
	// etapa — rateio que ninguém mediu.
	UsageOrigin string     `json:"usage_origin,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type Step struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Title                string         `json:"title"`
	Input                map[string]any `json:"input,omitempty"`
	Risk                 RiskClass      `json:"risk"`
	RequiresApproval     bool           `json:"requires_approval,omitempty"`
	ToolDescriptorSHA256 string         `json:"tool_descriptor_sha256,omitempty"`
	ToolConfigSHA256     string         `json:"tool_config_sha256,omitempty"`
	State                StepState      `json:"state"`
	Attempts             int            `json:"attempts"`
	Result               any            `json:"result,omitempty"`
	Error                string         `json:"error,omitempty"`
}

type Approval struct {
	ID             string         `json:"id"`
	MissionID      string         `json:"mission_id"`
	StepID         string         `json:"step_id"`
	OrganizationID string         `json:"organization_id,omitempty"`
	ActorID        string         `json:"actor_id,omitempty"`
	RequestedBy    string         `json:"requested_by,omitempty"`
	Policy         string         `json:"policy,omitempty"`
	PayloadSHA256  string         `json:"payload_sha256,omitempty"`
	Nonce          string         `json:"nonce,omitempty"`
	Status         ApprovalStatus `json:"status"`
	Reason         string         `json:"reason,omitempty"`
	ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Event struct {
	ID             string    `json:"id"`
	MissionID      string    `json:"mission_id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	Type           string    `json:"type"`
	StepID         string    `json:"step_id,omitempty"`
	Payload        any       `json:"payload,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ArtifactManifest struct {
	ID        string    `json:"id"`
	MissionID string    `json:"mission_id"`
	StepID    string    `json:"step_id,omitempty"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	MediaType string    `json:"media_type,omitempty"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

type ToolDescriptor struct {
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	Description      string    `json:"description"`
	Risk             RiskClass `json:"risk"`
	Scopes           []string  `json:"scopes,omitempty"`
	RequiresApproval bool      `json:"requires_approval"`
	// RequiredInputs declara os campos de input SEM os quais a etapa não executa. O runtime
	// recusa no PLANEJAMENTO, nomeando etapa e campo, em vez de deixar o operador descobrir na
	// execução (ex.: "anchored workspace path is required" não diz qual etapa nem qual campo
	// faltou). Descritor sem RequiredInputs continua válido: zero valor = nenhuma exigência.
	RequiredInputs []string `json:"required_inputs,omitempty"`
}

type ToolContext struct {
	MissionID      string
	StepID         string
	Workspace      string
	WorkspaceRoot  *os.Root
	OrganizationID string
	// Consent é o registro de consentimento/opt-out aplicado pelas ferramentas que
	// falam com terceiros (estágio 27). Ausente significa NEGAR o contato.
	Consent                   *ConsentRegistry
	ToolConfigSHA256          string
	WorkspaceSnapshotManifest *WorkspaceSnapshotManifest
	WorkspaceSnapshotPrefix   string
	MissionArtifacts          []ArtifactManifest
}

type ToolResult struct {
	Value     any
	Artifacts []ArtifactManifest
}

type Tool interface {
	Descriptor() ToolDescriptor
	Execute(ctx context.Context, context ToolContext, input map[string]any) (ToolResult, error)
}

type Project struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Root           string    `json:"root"`
	OrganizationID string    `json:"organization_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Memory struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id,omitempty"`
	Kind       string    `json:"kind"`
	Content    string    `json:"content"`
	Source     string    `json:"source,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
	Embedding  []float32 `json:"embedding,omitempty"`
	// EmbeddingModel diz QUAL modelo gerou o vetor acima.
	//
	// Sem este carimbo, trocar o modelo de embedding deixaria vetores de espaços diferentes sendo comparados
	// por cosseno — e a similaridade entre espaços distintos é ruído que PARECE resultado. Com o carimbo, a
	// busca sabe que aquele vetor não é comparável com a consulta atual, cai para correspondência literal e
	// DECLARA o fato, em vez de devolver lixo plausível.
	EmbeddingModel string    `json:"embedding_model,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	// Isolamento por usuário (estágio 15): vazio significa conhecimento do
	// projeto (`organization`), preservando dados anteriores.
	Visibility string `json:"visibility,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	// Versionamento (estágio 15): `Revision` começa em 0 e sobe a cada alteração
	// real; `Versions` guarda o conteúdo anterior com quem alterou, data e motivo.
	Revision int             `json:"revision,omitempty"`
	Versions []MemoryVersion `json:"versions,omitempty"`
	// `EncryptedContent` existe SÓ no disco: com chave configurada, `Content` sai
	// vazio do arquivo e o cifrado fica aqui. Em memória ele nunca é usado.
	EncryptedContent string `json:"encrypted_content,omitempty"`
	// Escopos (estágio 15): vazio significa "vale para o projeto". Preenchido
	// restringe a DIVULGAÇÃO àquela missão/agente/equipe; a PROPRIEDADE continua
	// sendo do ator, que segue podendo corrigir e apagar.
	MissionID string `json:"mission_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	TeamID    string `json:"team_id,omitempty"`
}

type SkillManifest struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Scopes         []string `json:"scopes,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	Trusted        bool     `json:"trusted"`
	Enabled        bool     `json:"enabled"`
	SigningKeyID   string   `json:"signing_key_id,omitempty"`
	ContentSHA256  string   `json:"content_sha256,omitempty"`
	Signature      string   `json:"signature,omitempty"`
	// Campos do estágio 8 (catálogo estruturado). Todos com `omitempty` para não
	// alterar os bytes canônicos de manifestos já assinados.
	Category     string   `json:"category,omitempty"`
	License      string   `json:"license,omitempty"`
	Homepage     string   `json:"homepage,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	// TokenBudget é o custo declarado da skill no orçamento de contexto. Zero
	// significa "não declarado" e não consome orçamento — nunca é tratado como
	// custo zero real por conveniência.
	TokenBudget int `json:"token_budget,omitempty"`
}

type Schedule struct {
	ID               string `json:"id"`
	Objective        string `json:"objective"`
	Model            string `json:"model,omitempty"`
	Workspace        string `json:"workspace,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	OrganizationID   string `json:"organization_id,omitempty"`
	IntervalSeconds  int64  `json:"interval_seconds"`
	Enabled          bool   `json:"enabled"`
	WebhookSecretEnv string `json:"webhook_secret_env,omitempty"`
	// Steps is the persisted flow graph. When empty the schedule keeps its
	// historical behaviour of a single mission carrying Objective; when
	// populated the supervisor executes the steps in dependency order.
	Steps           []ScheduleStep `json:"steps,omitempty"`
	NextRunAt       time.Time      `json:"next_run_at"`
	LastRunAt       *time.Time     `json:"last_run_at,omitempty"`
	FailureCount    int            `json:"failure_count,omitempty"`
	LastFailureCode string         `json:"last_failure_code,omitempty"`
	// LastRun é o resultado POR NÓ da última execução. Existe para que uma
	// execução parcial (nó condicional falso, passo bloqueado) nunca apareça
	// como sucesso limpo: o operador vê quais passos rodaram, quais foram
	// ignorados e qual falhou.
	LastRun *ScheduleRunReport `json:"last_run,omitempty"`
	// DeadLetters é a DLQ por nó deste schedule: passos que esgotaram as
	// tentativas. Entra na falha e SAI quando o passo volta a ter sucesso
	// (inclusive via reprocessamento). Limitada por maxScheduleDeadLetters.
	DeadLetters []ScheduleStepDeadLetter `json:"dead_letters,omitempty"`
	// FlowVersion é a versão do GRAFO. Sobe quando os passos MUDAM (não em
	// qualquer edição), e cada execução guarda a versão que rodou: sem isso,
	// comparar execuções de grafos diferentes seria comparação inválida.
	FlowVersion int `json:"flow_version,omitempty"`
	// Approvals é o histórico de decisões humanas sobre passos approval.request e
	// PendingApprovals é o que está PARADO esperando decisão. Ambos vivem no
	// schedule (e não só no processo) porque a espera atravessa execuções.
	Approvals        []ScheduleApproval        `json:"approvals,omitempty"`
	PendingApprovals []SchedulePendingApproval `json:"pending_approvals,omitempty"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}
