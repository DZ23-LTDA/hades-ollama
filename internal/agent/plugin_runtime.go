package agent

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

// Plugin Runtime — terceira fatia do estágio 8.
//
// O gerenciador de plugins já instalava, verificava, promovia, atualizava e
// revertia plugins; o que faltava era EXECUTAR. Executar não pode ser um caminho
// novo de processo: o Hades já tem um executor governado (`sandbox.exec`, com
// allowlist de interpretador, modo vinculado à aprovação e recusa explícita onde
// não há isolamento). Este runtime apenas decide SE pode executar e então delega.
//
// Portas (todas fail-closed, nesta ordem, antes de qualquer execução):
//  1. o plugin precisa estar instalado para a organização;
//  2. precisa estar habilitado;
//  3. precisa ser confiável, decidido pelo servidor (assinatura aceita);
//  4. o conteúdo precisa estar verificado E continuar intacto no disco;
//  5. o entrypoint precisa ser relativo, com extensão do runtime declarado, e
//     ficar DENTRO do bundle (leitura via os.Root, à prova de travessia);
//  6. os escopos concedidos na instalação precisam cobrir o escopo exigido pelo
//     executor (`sandbox:execute`) — catálogo não concede capacidade;
//  7. a execução precisa estar vinculada a um modo de sandbox (e, em
//     best-effort, com o aviso de aprovação explícito).
//
// Em host sem o executor Linux o resultado é erro explícito, nunca uma execução
// simulada.
const (
	pluginExecutionMaxCodeBytes = 512 << 10
	pluginExecutionAuditLimit   = 25
)

var (
	ErrPluginExecutionNotInstalled = errors.New("plugin is not installed for the organization")
	ErrPluginExecutionDisabled     = errors.New("plugin is disabled")
	ErrPluginExecutionUntrusted    = errors.New("plugin is not trusted by the server")
	ErrPluginExecutionUnverified   = errors.New("plugin content is not verified")
	ErrPluginExecutionEntrypoint   = errors.New("plugin entrypoint is invalid")
	ErrPluginExecutionScopeDenied  = errors.New("granted scopes do not authorize plugin execution")
	ErrPluginExecutionApproval     = errors.New("plugin execution requires an approved sandbox mode")
	ErrPluginExecutionSandbox      = errors.New("plugin sandbox execution failed")
)

// PluginExecutionRequest descreve uma execução pedida.
//
// O entrypoint e o runtime NÃO vêm daqui: vêm do manifesto assinado
// (`PluginManifest.Entrypoint` / `.Runtime`). O pedido só pode CONFIRMAR o que o
// manifesto declara — se divergir, a execução é recusada.
type PluginExecutionRequest struct {
	OrganizationID string
	PluginID       string
	MissionID      string
	StepID         string
	ActorID        string
	// EntrypointConfirmation e RuntimeConfirmation são opcionais e precisam ser
	// iguais ao que o manifesto assinado declara.
	EntrypointConfirmation string
	RuntimeConfirmation    string
	// SandboxMode é o modo aprovado para esta execução ("strict" ou "best-effort").
	// Precisa ser igual ao modo configurado no host; o executor valida o vínculo.
	SandboxMode string
	// ApprovalNotice é obrigatório quando SandboxMode é "best-effort".
	ApprovalNotice string
}

// PluginExecutionRecord é o registro auditável de uma execução.
type PluginExecutionRecord struct {
	PluginID      string    `json:"plugin_id"`
	Version       string    `json:"version"`
	Digest        string    `json:"digest"`
	Entrypoint    string    `json:"entrypoint"`
	Language      string    `json:"language"`
	SandboxMode   string    `json:"sandbox_mode"`
	ActorID       string    `json:"actor_id,omitempty"`
	StepID        string    `json:"step_id"`
	GrantedScopes []string  `json:"granted_scopes"`
	ExecutedAt    time.Time `json:"executed_at"`
	Isolation     string    `json:"isolation"`
	// Succeeded distingue execução comprovada de tentativa recusada: é o que
	// permite ao status do plugin exigir execução real antes de dizer "pronto".
	Succeeded bool `json:"succeeded"`
}

// PluginExecutionResult devolve o registro auditável e a saída do executor.
type PluginExecutionResult struct {
	Record    PluginExecutionRecord
	Output    any
	Artifacts []ArtifactManifest
}

// PluginRuntime executa plugins instalados dentro do executor governado.
type PluginRuntime struct {
	Registry *PluginRegistry
	Policy   CapabilityPolicy

	audit []PluginExecutionRecord
}

// NewPluginRuntime cria o runtime com o registro e a política vigentes.
func NewPluginRuntime(registry *PluginRegistry, policy CapabilityPolicy) *PluginRuntime {
	return &PluginRuntime{Registry: registry, Policy: policy}
}

// AuditTrail devolve uma cópia do histórico limitado de execuções.
func (r *PluginRuntime) AuditTrail() []PluginExecutionRecord {
	if r == nil {
		return nil
	}
	out := make([]PluginExecutionRecord, len(r.audit))
	copy(out, r.audit)
	return out
}

func (r *PluginRuntime) record(record PluginExecutionRecord) {
	if r == nil {
		return
	}
	r.audit = append(r.audit, record)
	if len(r.audit) > pluginExecutionAuditLimit {
		r.audit = r.audit[len(r.audit)-pluginExecutionAuditLimit:]
	}
}

// Execute aplica as portas e delega a execução ao sandbox governado.
func (r *PluginRuntime) Execute(ctx context.Context, request PluginExecutionRequest) (PluginExecutionResult, error) {
	if r == nil || r.Registry == nil {
		return PluginExecutionResult{}, errors.New("plugin runtime is not configured")
	}
	if err := validateStepID(request.StepID); err != nil {
		return PluginExecutionResult{}, err
	}
	organizationID := strings.TrimSpace(request.OrganizationID)
	pluginID := strings.TrimSpace(request.PluginID)
	if organizationID == "" || pluginID == "" {
		return PluginExecutionResult{}, fmt.Errorf("%w: organization and plugin are required", ErrPluginExecutionNotInstalled)
	}
	if err := r.Registry.LoadOrganization(organizationID); err != nil {
		return PluginExecutionResult{}, err
	}
	installation, err := r.Registry.GetForOrganization(organizationID, pluginID)
	if err != nil {
		return PluginExecutionResult{}, fmt.Errorf("%w: %v", ErrPluginExecutionNotInstalled, err)
	}
	if !installation.Enabled {
		return PluginExecutionResult{}, fmt.Errorf("%w: %s", ErrPluginExecutionDisabled, pluginID)
	}
	if !installation.Trusted {
		return PluginExecutionResult{}, fmt.Errorf("%w: %s", ErrPluginExecutionUntrusted, pluginID)
	}
	if !installation.ContentVerified || strings.TrimSpace(installation.ContentSHA256) == "" {
		return PluginExecutionResult{}, fmt.Errorf("%w: %s", ErrPluginExecutionUnverified, pluginID)
	}
	// Reverificação a cada execução: o bundle pode ter sido alterado depois da
	// instalação, e executar conteúdo divergente do digest seria confiar em disco
	// em vez de confiar no que foi verificado.
	if err := r.Registry.VerifyInstalledContent(organizationID, pluginID); err != nil {
		return PluginExecutionResult{}, fmt.Errorf("%w: %v", ErrPluginExecutionUnverified, err)
	}
	language, extension, err := pluginExecutionLanguage(installation.Manifest.Runtime)
	if err != nil {
		return PluginExecutionResult{}, err
	}
	entrypoint := strings.TrimSpace(installation.Manifest.Entrypoint)
	if err := validatePluginEntrypoint(entrypoint, extension); err != nil {
		return PluginExecutionResult{}, err
	}
	// Só o que o manifesto assinado declara pode executar: confirmações
	// divergentes são recusadas em vez de aceitas como override.
	if requested := strings.TrimSpace(request.EntrypointConfirmation); requested != "" && requested != entrypoint {
		return PluginExecutionResult{}, fmt.Errorf("%w: entrypoint confirmado %q difere do manifesto %q", ErrPluginExecutionEntrypoint, requested, entrypoint)
	}
	if requested := strings.TrimSpace(request.RuntimeConfirmation); requested != "" && !strings.EqualFold(requested, installation.Manifest.Runtime) {
		return PluginExecutionResult{}, fmt.Errorf("%w: runtime confirmado %q difere do manifesto %q", ErrPluginExecutionEntrypoint, requested, installation.Manifest.Runtime)
	}
	// Escopos: o executor exige `sandbox:execute`; a instalação só executa se os
	// escopos concedidos cobrirem exatamente o que o executor pede.
	if !r.Policy.Allows(sandboxExecTool{}.Descriptor(), installation.GrantedScopes) {
		return PluginExecutionResult{}, fmt.Errorf("%w: %s precisa de %v", ErrPluginExecutionScopeDenied, pluginID, sandboxExecTool{}.Descriptor().Scopes)
	}
	mode := strings.ToLower(strings.TrimSpace(request.SandboxMode))
	switch mode {
	case "strict":
	case "best-effort":
		if strings.TrimSpace(request.ApprovalNotice) == "" {
			return PluginExecutionResult{}, fmt.Errorf("%w: best-effort exige o aviso de aprovação explícito", ErrPluginExecutionApproval)
		}
	default:
		return PluginExecutionResult{}, fmt.Errorf("%w: modo %q", ErrPluginExecutionApproval, request.SandboxMode)
	}
	code, err := readPluginEntrypoint(installation.ContentPath, entrypoint)
	if err != nil {
		return PluginExecutionResult{}, err
	}

	input := map[string]any{
		"code":         code,
		"language":     language,
		"sandbox_mode": mode,
	}
	if mode == "best-effort" {
		input["sandbox_gate_status"] = string(GateStatusNotConfigured)
		input["sandbox_approval_notice"] = strings.TrimSpace(request.ApprovalNotice)
	}
	toolContext := ToolContext{
		MissionID:      strings.TrimSpace(request.MissionID),
		StepID:         strings.TrimSpace(request.StepID),
		Workspace:      installation.ContentPath,
		OrganizationID: organizationID,
	}
	result, execErr := (sandboxExecTool{}).Execute(ctx, toolContext, input)
	record := PluginExecutionRecord{
		PluginID:      installation.Manifest.ID,
		Version:       installation.Manifest.Version,
		Digest:        installation.ContentSHA256,
		Entrypoint:    entrypoint,
		Language:      language,
		SandboxMode:   mode,
		ActorID:       strings.TrimSpace(request.ActorID),
		StepID:        strings.TrimSpace(request.StepID),
		GrantedScopes: append([]string(nil), installation.GrantedScopes...),
		ExecutedAt:    time.Now().UTC(),
		Isolation:     "sandbox.exec",
		Succeeded:     execErr == nil,
	}
	r.record(record)
	if execErr != nil {
		// A falha do executor é propagada como falha de execução de plugin: nunca
		// como sucesso parcial. Onde não há executor (host não Linux) ou o sandbox
		// estrito não está configurado, o plugin simplesmente não roda.
		return PluginExecutionResult{Record: record}, fmt.Errorf("%w: %v", ErrPluginExecutionSandbox, execErr)
	}
	return PluginExecutionResult{Record: record, Output: result.Value, Artifacts: result.Artifacts}, nil
}

func pluginExecutionLanguage(language string) (resolved string, extension string, err error) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "python3":
		return "python", ".py", nil
	case "node", "javascript", "js":
		return "node", ".js", nil
	default:
		return "", "", fmt.Errorf("%w: runtime %q não suportado (use python ou node)", ErrPluginExecutionEntrypoint, language)
	}
}

func validatePluginEntrypoint(entrypoint, extension string) error {
	if entrypoint == "" {
		return fmt.Errorf("%w: entrypoint é obrigatório", ErrPluginExecutionEntrypoint)
	}
	if len(entrypoint) > 256 {
		return fmt.Errorf("%w: entrypoint muito longo", ErrPluginExecutionEntrypoint)
	}
	if strings.Contains(entrypoint, "\\") {
		return fmt.Errorf("%w: use separador '/'", ErrPluginExecutionEntrypoint)
	}
	// Padrão de allowlist: caminho relativo simples, sem traversal e sem absoluto.
	if strings.HasPrefix(entrypoint, "/") || path.IsAbs(entrypoint) || entrypoint != path.Clean(entrypoint) {
		return fmt.Errorf("%w: %q não é um caminho relativo limpo", ErrPluginExecutionEntrypoint, entrypoint)
	}
	if strings.HasPrefix(entrypoint, "../") || strings.Contains(entrypoint, "/../") || entrypoint == ".." {
		return fmt.Errorf("%w: %q escapa do bundle", ErrPluginExecutionEntrypoint, entrypoint)
	}
	if path.Ext(entrypoint) != extension {
		return fmt.Errorf("%w: %q precisa terminar em %s", ErrPluginExecutionEntrypoint, entrypoint, extension)
	}
	return nil
}

// readPluginEntrypoint lê o código do plugin de DENTRO do bundle (um artefato
// zip guardado como arquivo). A busca é por nome exato no arquivo compactado:
// nomes com traversal (`../`) são recusados e nada é extraído para o disco.
func readPluginEntrypoint(contentPath, entrypoint string) (string, error) {
	contentPath = strings.TrimSpace(contentPath)
	if contentPath == "" {
		return "", fmt.Errorf("%w: bundle sem caminho de conteúdo", ErrPluginExecutionUnverified)
	}
	archive, err := zip.OpenReader(contentPath)
	if err != nil {
		return "", fmt.Errorf("%w: bundle não é um artefato zip legível: %v", ErrPluginExecutionUnverified, err)
	}
	defer archive.Close()
	for _, file := range archive.File {
		name := file.Name
		if strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
			// Bundle com nome suspeito não é executado; o conteúdo verificado não
			// pode esconder um caminho de escape.
			return "", fmt.Errorf("%w: bundle contém caminho suspeito %q", ErrPluginExecutionEntrypoint, name)
		}
		if name != entrypoint {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrPluginExecutionEntrypoint, err)
		}
		defer reader.Close()
		content, err := io.ReadAll(io.LimitReader(reader, pluginExecutionMaxCodeBytes+1))
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrPluginExecutionEntrypoint, err)
		}
		if len(content) > pluginExecutionMaxCodeBytes {
			return "", fmt.Errorf("%w: código do plugin excede o limite do executor", ErrPluginExecutionEntrypoint)
		}
		return string(content), nil
	}
	return "", fmt.Errorf("%w: %q não existe no bundle", ErrPluginExecutionEntrypoint, entrypoint)
}
