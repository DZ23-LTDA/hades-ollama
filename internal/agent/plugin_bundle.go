package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Verificação de CONTEÚDO de plugin (estágio 8, segunda fatia).
//
// A fatia anterior governava manifesto, confiança, atualização e rollback. Esta
// verifica o PACOTE: assinatura destacada (ed25519) sobre o SHA-256 do arquivo,
// cópia para um diretório do próprio registro e recheque do digest depois de
// instalado — de modo que alterar o arquivo em disco seja detectado.
//
// Regras de segurança aplicadas:
//   - a chave que assina precisa estar autorizada na política; sem ela, 403;
//   - arquivo alterado depois de assinado invalida a verificação;
//   - o conteúdo só é aceito de um diretório permitido pelo operador (a rota
//     nunca lê caminho arbitrário do host);
//   - se a cópia do conteúdo falhar, a instalação é REVERTIDA: nunca fica um
//     plugin "instalado" sem pacote.

const (
	// PluginBundleDirEnv limita de onde o servidor aceita pacotes de plugin.
	PluginBundleDirEnv = "OLLAMA_AGENT_PLUGIN_BUNDLE_DIR"
	// maxPluginBundleBytes evita que um pacote gigante consuma o disco.
	maxPluginBundleBytes = 64 << 20
)

var ErrPluginBundleUnavailable = errors.New("plugin bundle is not available")

// VerifyPluginBundle confere a assinatura destacada do arquivo apontado por
// bundlePath e devolve o digest verificado. A chave precisa estar autorizada.
func VerifyPluginBundle(bundlePath string, signed SignedArtifact, policy CapabilityPolicy) (string, error) {
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return "", ErrPluginBundleUnavailable
	}
	info, err := os.Stat(bundlePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPluginBundleUnavailable, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: path is a directory", ErrPluginBundleUnavailable)
	}
	if info.Size() > maxPluginBundleBytes {
		return "", fmt.Errorf("%w: bundle exceeds %d bytes", ErrPluginBundleUnavailable, maxPluginBundleBytes)
	}
	keyID := strings.TrimSpace(signed.SigningKeyID)
	if keyID == "" {
		return "", ErrPluginSignatureReq
	}
	publicKey, ok := policy.pluginTrustedKey(keyID)
	if !ok {
		return "", ErrPluginKeyUnauthorized
	}
	if err := VerifyArtifactSignature(bundlePath, signed, publicKey); err != nil {
		return "", fmt.Errorf("%w: %v", ErrPluginSignatureInvalid, err)
	}
	return signed.SHA256, nil
}

// ResolvePluginBundlePath aceita apenas caminhos dentro do diretório permitido
// pelo operador. Sem diretório configurado, nada é aceito: falha fechada.
func ResolvePluginBundlePath(allowedDir, requestedPath string) (string, error) {
	allowedDir = strings.TrimSpace(allowedDir)
	requestedPath = strings.TrimSpace(requestedPath)
	if allowedDir == "" || requestedPath == "" {
		return "", ErrPluginBundleUnavailable
	}
	root, err := filepath.Abs(allowedDir)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPluginBundleUnavailable, err)
	}
	candidate, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPluginBundleUnavailable, err)
	}
	if resolved, err := filepath.EvalSymlinks(candidate); err == nil {
		candidate = resolved
	}
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}
	relative, err := filepath.Rel(root, candidate)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: bundle must live inside the allowed directory", ErrPluginBundleUnavailable)
	}
	return candidate, nil
}

func pluginBundleDigestFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, io.LimitReader(file, maxPluginBundleBytes+1)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (r *PluginRegistry) contentPath(organizationID string, manifest PluginManifest) string {
	return filepath.Join(r.root, "plugins", "content", pluginFileComponent(organizationID), manifest.ID+"-"+manifest.Version+".zip")
}

// InstallBundleForOrganization verifica o pacote, instala o plugin e só então
// publica o conteúdo. Qualquer falha depois da instalação reverte o registro.
func (r *PluginRegistry) InstallBundleForOrganization(organizationID string, manifest PluginManifest, bundlePath string, signed SignedArtifact, policy CapabilityPolicy) (PluginInstallation, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return PluginInstallation{}, fmt.Errorf("%w: organization is required", ErrPluginManifestInvalid)
	}
	digest, err := VerifyPluginBundle(bundlePath, signed, policy)
	if err != nil {
		return PluginInstallation{}, err
	}
	installation, err := r.InstallForOrganization(organizationID, manifest, policy)
	if err != nil {
		return PluginInstallation{}, err
	}
	// Rollback da instalação se qualquer passo posterior falhar.
	rollback := func() { _ = r.RemoveForOrganization(organizationID, installation.Manifest.ID) }
	if r.root == "" {
		rollback()
		return PluginInstallation{}, fmt.Errorf("%w: registry has no content root", ErrPluginBundleUnavailable)
	}
	target := r.contentPath(organizationID, installation.Manifest)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		rollback()
		return PluginInstallation{}, err
	}
	if err := copyFile(bundlePath, target); err != nil {
		rollback()
		return PluginInstallation{}, err
	}
	r.mu.Lock()
	stored, err := r.getLocked(organizationID, installation.Manifest.ID)
	if err != nil {
		r.mu.Unlock()
		rollback()
		return PluginInstallation{}, err
	}
	stored.ContentSHA256 = digest
	stored.ContentPath = target
	stored.ContentVerified = true
	stored.UpdatedAt = time.Now().UTC()
	if err := r.persistLocked(organizationID); err != nil {
		r.mu.Unlock()
		rollback()
		return PluginInstallation{}, err
	}
	result := clonePluginInstallation(*stored)
	r.mu.Unlock()
	return result, nil
}

// VerifyInstalledContent recomputa o digest do pacote guardado e compara com o
// digest registrado: alteração posterior à instalação é detectada.
func (r *PluginRegistry) VerifyInstalledContent(organizationID, pluginID string) error {
	organizationID = strings.TrimSpace(organizationID)
	pluginID = strings.TrimSpace(pluginID)
	r.mu.RLock()
	installation, err := r.getLocked(organizationID, pluginID)
	if err != nil {
		r.mu.RUnlock()
		return err
	}
	expected := installation.ContentSHA256
	path := installation.ContentPath
	r.mu.RUnlock()
	if expected == "" || path == "" {
		return ErrPluginBundleUnavailable
	}
	actual, err := pluginBundleDigestFile(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPluginBundleUnavailable, err)
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%w: stored bundle does not match its recorded digest", ErrPluginSignatureInvalid)
	}
	return nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	temp := destination + ".tmp"
	output, err := os.OpenFile(temp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, io.LimitReader(input, maxPluginBundleBytes+1)); err != nil {
		output.Close()
		_ = os.Remove(temp)
		return err
	}
	if err := output.Close(); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return os.Rename(temp, destination)
}
