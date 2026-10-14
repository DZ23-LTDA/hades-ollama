package agent

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// CAMINHO DO HISTÓRICO BRUTO — com recusa de travessia (E-581).
//
// PORQUE ESTA FUNÇÃO EXISTE, E PORQUE ELA RECUSA TANTO: `organizacao` e `projeto` chegam
// de um PEDIDO HTTP. Concatená-los num caminho é a receita clássica de travessia de
// directório — `projeto = "../../etc/passwd"` escreveria fora da raiz. O resto deste
// repositório já resolveu o mesmo problema noutros sítios (`safeWorkspacePath`,
// `rejectSymlinkComponents`, `containedPath`); aqui é a mesma disciplina para o histórico
// bruto, em vez de uma segunda regra que pudesse divergir.
//
// DUAS CAMADAS, DE PROPÓSITO: (1) cada identificador é validado antes de ser usado; e
// (2) o caminho FINAL é conferido como estando dentro da raiz. A segunda camada não é
// redundante — é o que transforma um erro de validação futura num erro de construção, em
// vez de numa escrita fora do sítio.

// LimiteDeIdentificadorNoCaminho limita cada segmento. Um identificador gigante é um
// caminho gigante, e nomes acima do limite do sistema de ficheiros falham de formas
// obscuras; declarar o limite aqui dá um erro legível.
const LimiteDeIdentificadorNoCaminho = 128

// CaminhoDoHistorico devolve `<root>/historico/<organizacao>/<projeto>.jsonl`.
func CaminhoDoHistorico(root, organizacao, projeto string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("raiz vazia: sem raiz não há onde guardar o histórico")
	}
	org, err := segmentoDeCaminhoSeguro("organização", organizacao)
	if err != nil {
		return "", err
	}
	proj, err := segmentoDeCaminhoSeguro("projeto", projeto)
	if err != nil {
		return "", err
	}
	caminho := filepath.Join(root, "historico", org, proj+".jsonl")
	// CAMADA 2: o caminho final tem de estar DENTRO da raiz. `filepath.Join` já limpa
	// `..`, logo isto é a rede de segurança para uma validação que alguém venha a
	// enfraquecer — e falha na construção, não na escrita.
	relativo, err := filepath.Rel(root, caminho)
	if err != nil {
		return "", fmt.Errorf("caminho do histórico fora da raiz: %w", err)
	}
	if relativo == ".." || strings.HasPrefix(relativo, ".."+string(filepath.Separator)) || filepath.IsAbs(relativo) {
		return "", fmt.Errorf("caminho do histórico escaparia da raiz: %q", caminho)
	}
	return caminho, nil
}

// segmentoDeCaminhoSeguro aceita apenas um identificador simples: sem separadores, sem
// `..`, sem NUL, sem caminho absoluto, sem dois-pontos (unidade no Windows) e não vazio.
func segmentoDeCaminhoSeguro(qual, valor string) (string, error) {
	valor = strings.TrimSpace(valor)
	switch {
	case valor == "":
		return "", fmt.Errorf("%s vazia: não há caminho sem %s", qual, qual)
	case len(valor) > LimiteDeIdentificadorNoCaminho:
		return "", fmt.Errorf("%s com %d caracteres excede o limite de %d",
			qual, len(valor), LimiteDeIdentificadorNoCaminho)
	case valor == "." || valor == "..":
		return "", fmt.Errorf("%s inválida: %q não é um identificador", qual, valor)
	case strings.ContainsAny(valor, `/\`):
		return "", fmt.Errorf("%s inválida: %q contém separador de caminho", qual, valor)
	case strings.ContainsRune(valor, 0):
		return "", fmt.Errorf("%s inválida: %q contém byte nulo", qual, valor)
	case strings.Contains(valor, ":"):
		return "", fmt.Errorf("%s inválida: %q contém dois-pontos (unidade ou fluxo no Windows)", qual, valor)
	case filepath.IsAbs(valor):
		return "", fmt.Errorf("%s inválida: %q é caminho absoluto", qual, valor)
	}
	return valor, nil
}
