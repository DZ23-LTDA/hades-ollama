package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// I1b — editar um artefacto SEM reescrever a historia.
//
// A regra do ficheiro de versoes governa isto: restaurar (e agora editar) NAO substitui a versao
// corrente, produz uma versao NOVA e deixa a anterior consultavel. Este ficheiro e a parte PURA:
// valida o conteudo e constroi o manifesto N+1. Nada aqui toca no disco — a escrita e' feita pelo
// chamador, no caminho que vem do MANIFESTO (nunca do corpo do pedido), com prefixo verificado.
const (
	// LimiteConteudoArtefacto limita uma edicao a 1 MiB. Um preview e' HTML/CSS/JS pequeno; aceitar
	// mais seria transformar uma rota de edicao num canal de upload de ficheiros grandes.
	LimiteConteudoArtefacto = 1 << 20
)

var (
	ErrConteudoArtefactoVazio      = errors.New("artifact content is empty")
	ErrConteudoArtefactoGrande     = errors.New("artifact content exceeds the edit limit")
	ErrConteudoArtefactoInalterado = errors.New("artifact content is identical to the current version")
)

// ArtefactoPrepararNovaVersao valida o conteudo e devolve o manifesto N+1.
//
// O `esperado` e' o CAS: se a versao corrente ja' nao for a que o cliente viu, devolve
// ErrRestauroConflitante (409 no handler) em vez de sobrescrever trabalho de outra pessoa.
func ArtefactoPrepararNovaVersao(
	atual ArtifactManifest,
	conteudo []byte,
	esperado int,
	agora time.Time,
) (ArtifactManifest, error) {
	if len(conteudo) == 0 {
		return ArtifactManifest{}, ErrConteudoArtefactoVazio
	}
	if len(conteudo) > LimiteConteudoArtefacto {
		return ArtifactManifest{}, ErrConteudoArtefactoGrande
	}
	if esperado <= 0 {
		// Sem `expected_version` nao ha CAS: escrever assim seria o mesmo que aceitar cegamente
		// que ninguem editou entretanto.
		return ArtifactManifest{}, ErrRestauroConflitante
	}
	if atual.Version != esperado {
		return ArtifactManifest{}, ErrRestauroConflitante
	}

	soma := sha256.Sum256(conteudo)
	digest := hex.EncodeToString(soma[:])
	if strings.EqualFold(digest, strings.TrimSpace(atual.SHA256)) {
		// Conteudo identico: nao inventar versao nova para uma edicao que nao mudou nada.
		return ArtifactManifest{}, ErrConteudoArtefactoInalterado
	}

	novo := atual // copia: a versao anterior fica intacta no indice
	novo.Version = atual.Version + 1
	novo.SHA256 = digest
	novo.Size = int64(len(conteudo))
	novo.CreatedAt = agora.UTC()
	if novo.MediaType == "" {
		novo.MediaType = mediaTypeDoArtefacto(atual.Name)
	}
	return novo, nil
}

// mediaTypeDoArtefacto classifica pelo sufixo do NOME do manifesto.
func mediaTypeDoArtefacto(nome string) string {
	switch strings.ToLower(filepath.Ext(nome)) {
	case ".json":
		return "application/json"
	case ".css":
		return "text/css"
	case ".js", ".mjs":
		return "text/javascript"
	case ".svg":
		return "image/svg+xml"
	case ".txt", ".md":
		return "text/plain"
	default:
		return "text/html"
	}
}
