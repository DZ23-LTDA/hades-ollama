package agent

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Workspace de execução de plugin (correção do defeito B-13, E-184).
//
// O DEFEITO, medido em E-183: `plugin_runtime.go` passava `Workspace` = o ARQUIVO do bundle
// (`plugins/content/<org>/<id>-<versão>.zip`). O executor de sandbox faz `os.OpenRoot(toolContext.Workspace)` e depois
// `Mkdir` DENTRO dessa raiz — ou seja, exige um DIRETÓRIO. Resultado real observado: *"open
// …/runtime-plugin-1.0.0.zip: not a directory"*. Instalar e verificar funcionavam; executar não.
//
// A CORREÇÃO: o bundle é EXTRAÍDO para um diretório de execução temporário e o workspace passa a ser esse diretório.
// Como o sandbox cria o arquivo do plugin num subdiretório desse workspace, os arquivos irmãos do bundle ficam
// alcançáveis a partir do código em execução (`../`), que é o que permite plugin de mais de um arquivo.
//
// Extrair conteúdo de terceiro é operação de risco, então ela é FAIL-CLOSED e limitada: nome de entrada com
// travessia (`..`), caminho absoluto, barra invertida, link simbólico, arquivo grande demais, arquivos demais ou
// total descomprimido acima do teto são RECUSADOS — nada é extraído pela metade e o diretório é removido.
const (
	pluginExecutionMaxBundleFiles     = 1024
	pluginExecutionMaxBundleFileBytes = 16 << 20
	pluginExecutionMaxBundleBytes     = 64 << 20
)

var ErrPluginExecutionBundleUnsafe = errors.New("plugin bundle is not safe to extract")

// preparePluginWorkspace extrai o bundle verificado para um diretório temporário e devolve o caminho e a limpeza.
// O diretório é criado sob `parentDir` quando informado (mantendo a execução dentro da área confiável do registro de
// plugins) e sob o diretório temporário do sistema caso contrário.
func preparePluginWorkspace(contentPath, parentDir string) (string, func(), error) {
	base := strings.TrimSpace(parentDir)
	if base != "" {
		execRoot := filepath.Join(base, "plugins", "exec-run")
		if err := os.MkdirAll(execRoot, 0o700); err != nil {
			return "", nil, fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
		}
		base = execRoot
	}
	workspace, err := os.MkdirTemp(base, "run-")
	if err != nil {
		return "", nil, fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
	}
	cleanup := func() {
		if err := os.RemoveAll(workspace); err != nil {
			// Falha de limpeza não pode esconder a execução nem derrubar o processo: o diretório é temporário.
			_ = err
		}
	}
	if err := extractPluginBundle(contentPath, workspace); err != nil {
		cleanup()
		return "", nil, err
	}
	return workspace, cleanup, nil
}

// extractPluginBundle extrai o bundle para `destination` com as proteções declaradas acima.
func extractPluginBundle(contentPath, destination string) error {
	archive, err := zip.OpenReader(contentPath)
	if err != nil {
		return fmt.Errorf("%w: bundle não é um artefato zip legível: %v", ErrPluginExecutionBundleUnsafe, err)
	}
	defer func() { _ = archive.Close() }()
	if len(archive.File) > pluginExecutionMaxBundleFiles {
		return fmt.Errorf("%w: bundle com %d arquivos excede o limite de %d", ErrPluginExecutionBundleUnsafe, len(archive.File), pluginExecutionMaxBundleFiles)
	}
	var total int64
	for _, file := range archive.File {
		relative, err := safeBundleEntryName(file.Name)
		if err != nil {
			return err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: bundle contém link simbólico %q", ErrPluginExecutionBundleUnsafe, file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filepath.Join(destination, relative), 0o700); err != nil {
				return fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
			}
			continue
		}
		if file.UncompressedSize64 > pluginExecutionMaxBundleFileBytes {
			return fmt.Errorf("%w: arquivo %q acima do limite de %d bytes", ErrPluginExecutionBundleUnsafe, file.Name, pluginExecutionMaxBundleFileBytes)
		}
		total += int64(file.UncompressedSize64)
		if total > pluginExecutionMaxBundleBytes {
			return fmt.Errorf("%w: bundle descomprimido acima do limite de %d bytes", ErrPluginExecutionBundleUnsafe, pluginExecutionMaxBundleBytes)
		}
		target := filepath.Join(destination, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
		}
		if err := writeBundleEntry(file, target); err != nil {
			return err
		}
	}
	return nil
}

// writeBundleEntry grava uma entrada lendo no máximo o limite por arquivo: um zip que MENTE no cabeçalho não enche o
// disco (o `UncompressedSize64` é declaração do próprio artefato).
func writeBundleEntry(file *zip.File, target string) error {
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
	}
	defer func() { _ = source.Close() }()
	destination, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
	}
	defer func() { _ = destination.Close() }()
	written, err := io.Copy(destination, io.LimitReader(source, pluginExecutionMaxBundleFileBytes+1))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPluginExecutionBundleUnsafe, err)
	}
	if written > pluginExecutionMaxBundleFileBytes {
		return fmt.Errorf("%w: arquivo %q excedeu o limite ao ser lido", ErrPluginExecutionBundleUnsafe, file.Name)
	}
	return nil
}

// safeBundleEntryName valida o nome de uma entrada do zip e devolve o caminho relativo utilizável.
func safeBundleEntryName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("%w: entrada sem nome", ErrPluginExecutionBundleUnsafe)
	}
	if strings.Contains(trimmed, "\\") {
		return "", fmt.Errorf("%w: entrada com barra invertida %q", ErrPluginExecutionBundleUnsafe, name)
	}
	if bundleEntryIsAbs(trimmed) {
		return "", fmt.Errorf("%w: entrada com caminho absoluto %q", ErrPluginExecutionBundleUnsafe, name)
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("%w: entrada com travessia de caminho %q", ErrPluginExecutionBundleUnsafe, name)
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".." || segment == "" {
			return "", fmt.Errorf("%w: entrada com segmento inválido %q", ErrPluginExecutionBundleUnsafe, name)
		}
	}
	return filepath.FromSlash(cleaned), nil
}

// bundleEntryIsAbs existe para manter explícita a checagem de caminho absoluto no formato do ZIP (sempre com "/"), sem
// depender do separador da plataforma que está rodando a verificação.
func bundleEntryIsAbs(name string) bool {
	return strings.HasPrefix(name, "/") || (len(name) > 1 && name[1] == ':')
}
