package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FATIA 1 (publicar): o artefacto de uma missao passa a ter uma copia PUBLICADA, servida por URL.
//
// Porque nao se reusa o `BuilderService.PublishLocal`: aquele publica PROJECTOS de builder; o painel
// "Construcao ao vivo" mostra ARTEFACTOS de missao. Sao eixos diferentes -- ligar um ao outro daria
// a ilusao de publicar. Aqui o artefacto e' copiado para um directorio publicado e devolvido o URL.
//
// Regra de seguranca: os nomes usados no caminho publicado sao SANITIZADOS (nunca vem caminho do
// cliente) e o url devolvido e' sempre construido a partir dos identificadores, nao do disco.

// ErrArtefactoPublicacaoInvalida recusa identificadores que nao passam na sanitizacao.
var ErrArtefactoPublicacaoInvalida = errors.New("artifact publish: identificador invalido")

var idPublicacaoSeguro = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,128}$`)

// nomePublicado devolve um nome de ficheiro seguro para o directorio publicado.
// Sem separadores, sem `..`, sem nomes reservados do Windows (CON, PRN, ...).
func nomePublicado(nome string) (string, error) {
	limpo := strings.TrimSpace(nome)
	if strings.ContainsAny(limpo, "/\\") {
		// Recusar e melhor do que remapear em silencio: um nome com separador pode ser
		// tentativa de travessia, e transformar "a/b" em "b" esconderia a tentativa.
		return "", ErrArtefactoPublicacaoInvalida
	}
	base := filepath.Base(limpo)
	if base == "." || base == ".." || base == string(filepath.Separator) || base == "" {
		return "", ErrArtefactoPublicacaoInvalida
	}
	if !idPublicacaoSeguro.MatchString(base) {
		return "", ErrArtefactoPublicacaoInvalida
	}
	semExt := strings.TrimSuffix(base, filepath.Ext(base))
	switch strings.ToUpper(semExt) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "LPT1", "LPT2", "LPT3":
		return "", ErrArtefactoPublicacaoInvalida
	}
	return base, nil
}

// chavePublicacao e' o segmento estavel do URL: artefacto + versao.
func chavePublicacao(artifactID string, versao int) (string, error) {
	if !idPublicacaoSeguro.MatchString(strings.TrimSpace(artifactID)) {
		return "", ErrArtefactoPublicacaoInvalida
	}
	if versao < 0 {
		return "", ErrArtefactoPublicacaoInvalida
	}
	return fmt.Sprintf("%s-v%d", artifactID, versao), nil
}

// ArtifactPublish copia o conteudo do artefacto para o directorio publicado e devolve
// (caminhoPublicado, urlRelativo). Nao sobrescreve uma publicacao anterior de outra versao: a chave
// inclui a versao, por isso cada versao publicada mantem o seu URL.
//
// A escrita e' ATOMICA (temporario + rename), como a edicao.
func (r *Runtime) ArtifactPublish(missionID, artifactID string) (string, string, error) {
	manifest, origem, err := r.Artifact(missionID, artifactID)
	if err != nil {
		return "", "", err
	}
	chave, err := chavePublicacao(manifest.ID, manifest.Version)
	if err != nil {
		return "", "", err
	}
	nome, err := nomePublicado(manifest.Name)
	if err != nil {
		return "", "", err
	}
	raiz, err := r.directorioPublicado()
	if err != nil {
		return "", "", err
	}
	destinoDir := filepath.Join(raiz, chave)
	if err := os.MkdirAll(destinoDir, 0o755); err != nil {
		return "", "", fmt.Errorf("artifact publish: criar directorio: %w", err)
	}
	bytes, err := os.ReadFile(origem)
	if err != nil {
		return "", "", fmt.Errorf("artifact publish: ler artefacto: %w", err)
	}
	destino := filepath.Join(destinoDir, nome)
	tmp, err := os.CreateTemp(destinoDir, ".publish-*")
	if err != nil {
		return "", "", fmt.Errorf("artifact publish: temporario: %w", err)
	}
	nomeTmp := tmp.Name()
	defer os.Remove(nomeTmp)
	if _, err := tmp.Write(bytes); err != nil {
		tmp.Close()
		return "", "", fmt.Errorf("artifact publish: escrever: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", "", fmt.Errorf("artifact publish: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", "", fmt.Errorf("artifact publish: fechar: %w", err)
	}
	if err := os.Rename(nomeTmp, destino); err != nil {
		return "", "", fmt.Errorf("artifact publish: substituir: %w", err)
	}
	return destino, "/api/agent/v1/published/" + chave + "/" + nome, nil
}

// directorioPublicado e' `<workspaceRoot>/.hades/published`, criado a pedido.
func (r *Runtime) directorioPublicado() (string, error) {
	base := filepath.Clean(strings.TrimSpace(r.workspaceRoot))
	if base == "" || base == "." {
		return "", ErrCaminhoForaDoWorkspace
	}
	return filepath.Join(base, ".hades", "published"), nil
}

// caminhoPublicado resolve um URL publicado no disco, recusando qualquer escape.
// caminhoPublicado resolve um URL publicado no disco, recusando qualquer escape.
func (r *Runtime) caminhoPublicado(chave, nome string) (string, error) {
	segChave := strings.TrimSpace(chave)
	if !idPublicacaoSeguro.MatchString(segChave) {
		return "", ErrArtefactoPublicacaoInvalida
	}
	nomeSeguro, err := nomePublicado(nome)
	if err != nil {
		return "", err
	}
	raiz, err := r.directorioPublicado()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(raiz, segChave)
	alvo := filepath.Join(dir, nomeSeguro)
	if !strings.HasPrefix(alvo, dir+string(os.PathSeparator)) {
		return "", ErrCaminhoForaDoWorkspace
	}
	return alvo, nil
}

// PublishedPath expoe o resolvedor para o handler HTTP.
func (r *Runtime) PublishedPath(chave, nome string) (string, error) {
	return r.caminhoPublicado(chave, nome)
}
