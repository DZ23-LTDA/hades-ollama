package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrCaminhoForaDoWorkspace recusa escrever fora da raiz do workspace.
//
// I1b parte 2: editar um artefacto ESCREVE no disco. O caminho vem SEMPRE do manifesto
// (`ArtifactManifest.Path`), nunca do corpo do pedido, e mesmo assim e' verificado: um manifesto
// forjado ou corrompido nao pode transformar a edicao numa escrita arbitraria.
var ErrCaminhoForaDoWorkspace = errors.New("artifact path escapes the workspace root")

// caminhoDoArtefactoNoWorkspace resolve o caminho RELATIVO do manifesto dentro da raiz do
// workspace e recusa qualquer resultado que saia dela.
//
// Cobre: `..` , caminho absoluto colado ao join, e prefixo que apenas PARECE estar dentro
// (`/ws-x` com raiz `/ws`). O separador e' incluido na comparacao para nao aceitar irmaos.
func caminhoDoArtefactoNoWorkspace(raiz, caminhoRelativo string) (string, error) {
	base := filepath.Clean(strings.TrimSpace(raiz))
	if base == "" || base == "." {
		return "", ErrCaminhoForaDoWorkspace
	}
	rel := strings.TrimSpace(caminhoRelativo)
	if rel == "" {
		return "", ErrCaminhoForaDoWorkspace
	}
	if filepath.IsAbs(rel) {
		return "", ErrCaminhoForaDoWorkspace
	}
	alvo := filepath.Clean(filepath.Join(base, rel))
	if alvo == base {
		// O proprio directorio nao e' um artefacto.
		return "", ErrCaminhoForaDoWorkspace
	}
	if !strings.HasPrefix(alvo, base+string(os.PathSeparator)) {
		return "", ErrCaminhoForaDoWorkspace
	}
	return alvo, nil
}

// ArtifactEdit grava conteudo NOVO e devolve o manifesto da versao N+1.
//
// Duas garantias que o chamador nao pode contornar:
//   - a versao e' calculada por ArtefactoPrepararNovaVersao (vazio, 1 MiB, CAS, conteudo igual);
//   - o destino e' resolvido e verificado por caminhoDoArtefactoNoWorkspace.
//
// A escrita e' ATOMICA (ficheiro temporario no mesmo directorio + rename): um erro a meio nao
// deixa o artefacto truncado.
func (r *Runtime) ArtifactEdit(missionID, artifactID string, conteudo []byte, esperado int, agora time.Time) (ArtifactManifest, error) {
	atual, _, err := r.Artifact(missionID, artifactID)
	if err != nil {
		return ArtifactManifest{}, err
	}
	novo, err := ArtefactoPrepararNovaVersao(atual, conteudo, esperado, agora)
	if err != nil {
		return ArtifactManifest{}, err
	}
	alvo, err := caminhoDoArtefactoNoWorkspace(r.workspaceRoot, atual.Path)
	if err != nil {
		return ArtifactManifest{}, err
	}
	dir := filepath.Dir(alvo)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ArtifactManifest{}, fmt.Errorf("artifact edit: criar directorio: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".hades-edit-*")
	if err != nil {
		return ArtifactManifest{}, fmt.Errorf("artifact edit: temporario: %w", err)
	}
	nomeTmp := tmp.Name()
	defer os.Remove(nomeTmp) // no-op depois do rename bem-sucedido
	if _, err := tmp.Write(conteudo); err != nil {
		tmp.Close()
		return ArtifactManifest{}, fmt.Errorf("artifact edit: escrever: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return ArtifactManifest{}, fmt.Errorf("artifact edit: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return ArtifactManifest{}, fmt.Errorf("artifact edit: fechar: %w", err)
	}
	if err := os.Rename(nomeTmp, alvo); err != nil {
		return ArtifactManifest{}, fmt.Errorf("artifact edit: substituir: %w", err)
	}
	return novo, nil
}
