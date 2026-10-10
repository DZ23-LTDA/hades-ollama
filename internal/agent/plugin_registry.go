package agent

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Gerenciador de plugins (estágio 8).
//
// O que faltava depois da auditoria: conectar/MCP/skills tinham registro,
// habilitação e remoção, mas não existia um PLUGIN de primeira classe com
// manifesto versionado, dependências, integridade, atualização e rollback.
//
// Regras aplicadas aqui, alinhadas ao resto do runtime:
//   - **catálogo não é permissão**: instalar registra o plugin; escopo só é
//     concedido após `PromoteTrustedForOrganization`, que exige assinatura
//     válida de uma chave autorizada na política. Plugin não confiável aparece
//     com `GrantedScopes` vazio;
//   - **escopo desconhecido é recusado na instalação** (`ErrUnknownCapability`),
//     nunca aceito "para depois";
//   - **isolamento por organização**: o mesmo id não pode pertencer a duas
//     organizações (`ErrPluginOrganizationScope`);
//   - **atualização otimista**: `Update` exige a versão corrente esperada e
//     guarda a anterior no histórico, que é o que torna o rollback real;
//   - **persistência atômica** por organização com rollback do estado em
//     memória quando a gravação falha.

const (
	PluginKindConnector = "connector"
	PluginKindMCP       = "mcp"
	PluginKindRemoteMCP = "remote-mcp"
	PluginKindSkill     = "skill"
	PluginKindBundle    = "bundle"

	maxPluginHistory = 5
)

var (
	ErrPluginManifestInvalid = errors.New("plugin manifest is invalid")
	ErrPluginVersionConflict = errors.New("plugin version is out of date")
	// ErrPluginNotFound já existe em connectors.go e é reutilizado aqui: um
	// plugin ausente tem um único significado no pacote.
	ErrPluginRollbackDisabled = errors.New("plugin has no previous version to roll back to")
	ErrPluginSignatureReq     = errors.New("plugin signature is required")
	ErrPluginSignatureInvalid = errors.New("plugin signature is invalid")
	ErrPluginKeyUnauthorized  = errors.New("plugin signing key is not authorized")
)

// PluginManifest descreve um plugin de forma versionada e verificável.
type PluginManifest struct {
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	Description   string   `json:"description,omitempty"`
	Scopes        []string `json:"scopes,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
	MinEngine     string   `json:"min_engine,omitempty"`
	License       string   `json:"license,omitempty"`
	Homepage      string   `json:"homepage,omitempty"`
	SigningKeyID  string   `json:"signing_key_id,omitempty"`
	ContentSHA256 string   `json:"content_sha256,omitempty"`
	Signature     string   `json:"signature,omitempty"`
}

// PluginInstallation é o estado instalado de um plugin em uma organização.
// `Trusted` é decidido pelo servidor (assinatura + consentimento), nunca pelo
// manifesto enviado.
type PluginInstallation struct {
	OrganizationID  string           `json:"organization_id"`
	Manifest        PluginManifest   `json:"manifest"`
	Digest          string           `json:"digest"`
	Trusted         bool             `json:"trusted"`
	Enabled         bool             `json:"enabled"`
	RequestedScopes []string         `json:"requested_scopes"`
	GrantedScopes   []string         `json:"granted_scopes"`
	InstalledAt     time.Time        `json:"installed_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	History         []PluginManifest `json:"history,omitempty"`
	// Conteúdo verificado do pacote (segunda fatia do estágio 8).
	ContentSHA256   string `json:"content_sha256,omitempty"`
	ContentPath     string `json:"content_path,omitempty"`
	ContentVerified bool   `json:"content_verified"`
}

// PluginManifestSignatureBytes é a representação canônica assinada. Os campos
// de assinatura ficam de fora para que o conteúdo assinado não dependa da
// própria assinatura.
func PluginManifestSignatureBytes(manifest PluginManifest) ([]byte, error) {
	manifest.Signature = ""
	manifest.ContentSHA256 = ""
	manifest.Scopes = append([]string(nil), manifest.Scopes...)
	manifest.Dependencies = append([]string(nil), manifest.Dependencies...)
	sort.Strings(manifest.Scopes)
	sort.Strings(manifest.Dependencies)
	return json.Marshal(manifest)
}

func PluginManifestDigest(manifest PluginManifest) (string, error) {
	data, err := PluginManifestSignatureBytes(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

// SignPluginManifest assina o manifesto com uma chave privada ed25519. A
// assinatura NÃO concede confiança: a promoção continua sendo um passo do
// servidor com chave pública autorizada.
func SignPluginManifest(manifest PluginManifest, privateKey ed25519.PrivateKey, keyID string) (PluginManifest, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return PluginManifest{}, errors.New("invalid plugin signing key")
	}
	manifest.SigningKeyID = strings.TrimSpace(keyID)
	data, err := PluginManifestSignatureBytes(manifest)
	if err != nil {
		return PluginManifest{}, err
	}
	digest := sha256.Sum256(data)
	manifest.ContentSHA256 = hex.EncodeToString(digest[:])
	manifest.Signature = base64.RawStdEncoding.EncodeToString(ed25519.Sign(privateKey, data))
	return manifest, nil
}

// VerifyPluginManifestSignature confere assinatura E digest do manifesto.
func VerifyPluginManifestSignature(manifest PluginManifest, publicKey ed25519.PublicKey) error {
	if strings.TrimSpace(manifest.Signature) == "" || strings.TrimSpace(manifest.SigningKeyID) == "" {
		return ErrPluginSignatureReq
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return ErrPluginSignatureInvalid
	}
	data, err := PluginManifestSignatureBytes(manifest)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	if !strings.EqualFold(manifest.ContentSHA256, hex.EncodeToString(digest[:])) {
		return ErrPluginSignatureInvalid
	}
	signature, err := base64.RawStdEncoding.DecodeString(manifest.Signature)
	if err != nil || !ed25519.Verify(publicKey, data, signature) {
		return ErrPluginSignatureInvalid
	}
	return nil
}

// pluginTrustedKey devolve a chave pública autorizada a assinar plugins. O
// conjunto é o MESMO já usado para atestação de skills: um plugin não ganha
// raiz de confiança própria, e uma chave nova precisa ser autorizada
// explicitamente pela política.
func (p CapabilityPolicy) pluginTrustedKey(keyID string) (ed25519.PublicKey, bool) {
	key, ok := p.trustedSkillKeys[strings.TrimSpace(keyID)]
	return key, ok
}

// pluginFileComponent deixa o identificador da organização seguro para virar
// nome de arquivo: separadores viram `_`, o resto é preservado.
func pluginFileComponent(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "..", "_")
	return replacer.Replace(strings.TrimSpace(value))
}

func validatePluginManifest(manifest PluginManifest, policy CapabilityPolicy) (PluginManifest, error) {
	manifest.ID = strings.TrimSpace(manifest.ID)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Kind = strings.TrimSpace(manifest.Kind)
	if manifest.ID == "" || strings.ContainsAny(manifest.ID, "/: \\") {
		return PluginManifest{}, fmt.Errorf("%w: id must be non-empty and may not contain separators", ErrPluginManifestInvalid)
	}
	if manifest.Version == "" || len(manifest.Version) > 64 || strings.ContainsAny(manifest.Version, " \t\r\n") {
		return PluginManifest{}, fmt.Errorf("%w: version must be a single token", ErrPluginManifestInvalid)
	}
	if manifest.Name == "" {
		return PluginManifest{}, fmt.Errorf("%w: name is required", ErrPluginManifestInvalid)
	}
	switch manifest.Kind {
	case PluginKindConnector, PluginKindMCP, PluginKindRemoteMCP, PluginKindSkill, PluginKindBundle:
	default:
		return PluginManifest{}, fmt.Errorf("%w: unsupported kind %q", ErrPluginManifestInvalid, manifest.Kind)
	}
	for _, dependency := range manifest.Dependencies {
		normalized := strings.TrimSpace(dependency)
		if normalized == "" {
			return PluginManifest{}, fmt.Errorf("%w: empty dependency", ErrPluginManifestInvalid)
		}
		if normalized == manifest.ID {
			return PluginManifest{}, fmt.Errorf("%w: plugin cannot depend on itself", ErrPluginManifestInvalid)
		}
	}
	known := map[string]struct{}{}
	for _, scope := range policy.KnownScopes() {
		known[scope] = struct{}{}
	}
	if len(known) == 0 {
		known = map[string]struct{}{}
		for _, scope := range DefaultCapabilityPolicy().KnownScopes() {
			known[scope] = struct{}{}
		}
	}
	scopes := make([]string, 0, len(manifest.Scopes))
	seen := map[string]struct{}{}
	for _, scope := range manifest.Scopes {
		normalized := strings.TrimSpace(scope)
		if normalized == "" {
			continue
		}
		if _, ok := known[normalized]; !ok {
			return PluginManifest{}, fmt.Errorf("%w: %s", ErrUnknownCapability, normalized)
		}
		if _, duplicated := seen[normalized]; duplicated {
			continue
		}
		seen[normalized] = struct{}{}
		scopes = append(scopes, normalized)
	}
	sort.Strings(scopes)
	manifest.Scopes = scopes
	manifest.Dependencies = append([]string(nil), manifest.Dependencies...)
	sort.Strings(manifest.Dependencies)
	return manifest, nil
}

// PluginRegistry guarda as instalações por organização.
type PluginRegistry struct {
	root  string
	mu    sync.RWMutex
	byOrg map[string]map[string]*PluginInstallation
}

func NewPluginRegistry(root string) (*PluginRegistry, error) {
	registry := &PluginRegistry{root: strings.TrimSpace(root), byOrg: map[string]map[string]*PluginInstallation{}}
	if registry.root != "" {
		if err := os.MkdirAll(filepath.Join(registry.root, "plugins"), 0o755); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (r *PluginRegistry) pluginPath(organizationID string) string {
	return filepath.Join(r.root, "plugins", pluginFileComponent(organizationID)+".json")
}

func (r *PluginRegistry) persistLocked(organizationID string) error {
	if r.root == "" {
		return nil
	}
	scoped := r.byOrg[organizationID]
	list := make([]PluginInstallation, 0, len(scoped))
	ids := make([]string, 0, len(scoped))
	for id := range scoped {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		list = append(list, *scoped[id])
	}
	return writeJSONAtomic(r.pluginPath(organizationID), list)
}

func clonePluginInstallation(installation PluginInstallation) PluginInstallation {
	cloned := installation
	cloned.Manifest.Scopes = append([]string(nil), installation.Manifest.Scopes...)
	cloned.Manifest.Dependencies = append([]string(nil), installation.Manifest.Dependencies...)
	cloned.RequestedScopes = append([]string(nil), installation.RequestedScopes...)
	cloned.GrantedScopes = append([]string(nil), installation.GrantedScopes...)
	cloned.History = append([]PluginManifest(nil), installation.History...)
	return cloned
}

// LoadOrganization lê o estado persistido de uma organização. O manifesto
// gravado é conferido contra o digest: estado adulterado no disco é recusado em
// vez de carregar um plugin alterado.
func (r *PluginRegistry) LoadOrganization(organizationID string) error {
	organizationID = strings.TrimSpace(organizationID)
	if r.root == "" || organizationID == "" {
		return nil
	}
	raw, err := os.ReadFile(r.pluginPath(organizationID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var list []PluginInstallation
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("decode plugins: %w", err)
	}
	scoped := make(map[string]*PluginInstallation, len(list))
	for index := range list {
		installation := list[index]
		if strings.TrimSpace(installation.Manifest.ID) == "" {
			continue
		}
		digest, err := PluginManifestDigest(installation.Manifest)
		if err != nil {
			return err
		}
		if installation.Digest != "" && !strings.EqualFold(installation.Digest, digest) {
			return fmt.Errorf("%w: persisted manifest of %s does not match its digest", ErrPluginSignatureInvalid, installation.Manifest.ID)
		}
		installation.Digest = digest
		installation.OrganizationID = organizationID
		copied := installation
		scoped[installation.Manifest.ID] = &copied
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byOrg[organizationID] = scoped
	return nil
}

// InstallForOrganization registra um plugin novo. Recusa id já pertencente a
// outra organização, versão já instalada e escopo desconhecido.
func (r *PluginRegistry) InstallForOrganization(organizationID string, manifest PluginManifest, policy CapabilityPolicy) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return PluginInstallation{}, fmt.Errorf("%w: organization is required", ErrPluginManifestInvalid)
	}
	validated, err := validatePluginManifest(manifest, policy)
	if err != nil {
		return PluginInstallation{}, err
	}
	digest, err := PluginManifestDigest(validated)
	if err != nil {
		return PluginInstallation{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for otherOrg, scoped := range r.byOrg {
		if otherOrg == organizationID {
			continue
		}
		if _, taken := scoped[validated.ID]; taken {
			return PluginInstallation{}, fmt.Errorf("%w: %s", ErrPluginOrganizationScope, validated.ID)
		}
	}
	if r.byOrg[organizationID] == nil {
		r.byOrg[organizationID] = map[string]*PluginInstallation{}
	}
	if _, exists := r.byOrg[organizationID][validated.ID]; exists {
		return PluginInstallation{}, fmt.Errorf("%w: %s@%s already installed", ErrPluginVersionConflict, validated.ID, validated.Version)
	}
	now := time.Now().UTC()
	installation := &PluginInstallation{
		OrganizationID:  organizationID,
		Manifest:        validated,
		Digest:          digest,
		Trusted:         false,
		Enabled:         true,
		RequestedScopes: append([]string(nil), validated.Scopes...),
		GrantedScopes:   []string{},
		InstalledAt:     now,
		UpdatedAt:       now,
	}
	r.byOrg[organizationID][validated.ID] = installation
	if err := r.persistLocked(organizationID); err != nil {
		delete(r.byOrg[organizationID], validated.ID)
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*installation), nil
}

func (r *PluginRegistry) getLocked(organizationID, pluginID string) (*PluginInstallation, error) {
	scoped := r.byOrg[organizationID]
	if scoped == nil {
		return nil, ErrPluginNotFound
	}
	installation, ok := scoped[pluginID]
	if !ok {
		return nil, ErrPluginNotFound
	}
	return installation, nil
}

func (r *PluginRegistry) GetForOrganization(organizationID, pluginID string) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*installation), nil
}

func (r *PluginRegistry) ListForOrganization(organizationID string) []PluginInstallation {
	organizationID = strings.TrimSpace(organizationID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	scoped := r.byOrg[organizationID]
	ids := make([]string, 0, len(scoped))
	for id := range scoped {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]PluginInstallation, 0, len(ids))
	for _, id := range ids {
		out = append(out, clonePluginInstallation(*scoped[id]))
	}
	return out
}

// UpdateForOrganization troca a versão instalada com concorrência otimista e
// guarda a versão anterior no histórico (base do rollback).
func (r *PluginRegistry) UpdateForOrganization(organizationID, pluginID, expectedVersion string, manifest PluginManifest, policy CapabilityPolicy) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	expectedVersion = strings.TrimSpace(expectedVersion)
	validated, err := validatePluginManifest(manifest, policy)
	if err != nil {
		return PluginInstallation{}, err
	}
	if validated.ID != pluginID {
		return PluginInstallation{}, fmt.Errorf("%w: manifest id does not match the target plugin", ErrPluginManifestInvalid)
	}
	digest, err := PluginManifestDigest(validated)
	if err != nil {
		return PluginInstallation{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return PluginInstallation{}, err
	}
	if installation.Manifest.Version != expectedVersion {
		return PluginInstallation{}, fmt.Errorf("%w: installed %s, expected %s", ErrPluginVersionConflict, installation.Manifest.Version, expectedVersion)
	}
	if validated.Version == installation.Manifest.Version {
		return PluginInstallation{}, fmt.Errorf("%w: version %s is already installed", ErrPluginVersionConflict, validated.Version)
	}
	previous := clonePluginInstallation(*installation)
	installation.History = append(installation.History, installation.Manifest)
	if len(installation.History) > maxPluginHistory {
		installation.History = installation.History[len(installation.History)-maxPluginHistory:]
	}
	installation.Manifest = validated
	installation.Digest = digest
	installation.RequestedScopes = append([]string(nil), validated.Scopes...)
	// Versão nova perde a confiança anterior: precisa ser promovida de novo.
	installation.Trusted = false
	installation.GrantedScopes = []string{}
	installation.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(organizationID); err != nil {
		*installation = previous
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*installation), nil
}

// RollbackForOrganization volta para a última versão do histórico. O passo é
// consumido: cada rollback desfaz uma atualização.
func (r *PluginRegistry) RollbackForOrganization(organizationID, pluginID string) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.Lock()
	defer r.mu.Unlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return PluginInstallation{}, err
	}
	if len(installation.History) == 0 {
		return PluginInstallation{}, ErrPluginRollbackDisabled
	}
	previous := clonePluginInstallation(*installation)
	target := installation.History[len(installation.History)-1]
	installation.History = installation.History[:len(installation.History)-1]
	digest, err := PluginManifestDigest(target)
	if err != nil {
		return PluginInstallation{}, err
	}
	installation.Manifest = target
	installation.Digest = digest
	installation.RequestedScopes = append([]string(nil), target.Scopes...)
	installation.Trusted = false
	installation.GrantedScopes = []string{}
	installation.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(organizationID); err != nil {
		*installation = previous
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*installation), nil
}

func (r *PluginRegistry) SetEnabledForOrganization(organizationID, pluginID string, enabled bool) error {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.Lock()
	defer r.mu.Unlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return err
	}
	previous := installation.Enabled
	installation.Enabled = enabled
	installation.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(organizationID); err != nil {
		installation.Enabled = previous
		return err
	}
	return nil
}

func (r *PluginRegistry) RemoveForOrganization(organizationID, pluginID string) error {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.Lock()
	defer r.mu.Unlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return err
	}
	removed := *installation
	delete(r.byOrg[organizationID], pluginID)
	if err := r.persistLocked(organizationID); err != nil {
		r.byOrg[organizationID][pluginID] = &removed
		return err
	}
	return nil
}

// PromoteTrustedForOrganization verifica a assinatura contra as chaves
// autorizadas da política e SÓ ENTÃO concede os escopos pedidos.
func (r *PluginRegistry) PromoteTrustedForOrganization(organizationID, pluginID string, policy CapabilityPolicy) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.Lock()
	defer r.mu.Unlock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		return PluginInstallation{}, err
	}
	manifest := installation.Manifest
	if strings.TrimSpace(manifest.Signature) == "" || strings.TrimSpace(manifest.SigningKeyID) == "" {
		return PluginInstallation{}, ErrPluginSignatureReq
	}
	publicKey, ok := policy.pluginTrustedKey(manifest.SigningKeyID)
	if !ok {
		return PluginInstallation{}, ErrPluginKeyUnauthorized
	}
	if err := VerifyPluginManifestSignature(manifest, publicKey); err != nil {
		return PluginInstallation{}, err
	}
	previousTrusted, previousGranted := installation.Trusted, installation.GrantedScopes
	installation.Trusted = true
	installation.GrantedScopes = append([]string(nil), installation.RequestedScopes...)
	installation.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(organizationID); err != nil {
		installation.Trusted = previousTrusted
		installation.GrantedScopes = previousGranted
		return PluginInstallation{}, err
	}
	return clonePluginInstallation(*installation), nil
}
