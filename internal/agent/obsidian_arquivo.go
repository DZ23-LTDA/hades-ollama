package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// I/O do adaptador, ainda SEM tocar no vault operacional.
//
// Duas guardas, e nenhuma delas e' opcional:
//   1. RAIZ -- todo o caminho e' resolvido e comparado com a raiz autorizada. `..`, caminho absoluto e symlink para
//      fora sao recusados antes de qualquer escrita.
//   2. ATomicidade -- escreve-se num temporario no MESMO directorio e so' depois se renomeia. Um crash a meio deixa
//      o ficheiro antigo intacto, nunca um ficheiro truncado que se parece com o novo.

// ErrForaDaRaiz: o caminho pedido sai da raiz autorizada.
var ErrForaDaRaiz = errors.New("path escapes the authorized memory root")

// CaminhoAutorizado resolve `relativo` dentro de `raiz` e recusa tudo o que saia.
//
// Resolve os symlinks que JA' existem (`EvalSymlinks` na raiz) e compara com `filepath.Rel`: uma travessia que so'
// aparece depois de resolver o caminho verdadeiro nao e' apanhada por uma comparacao de strings.
func CaminhoAutorizado(raiz, relativo string) (string, error) {
	if strings.TrimSpace(raiz) == "" {
		return "", fmt.Errorf("%w: raiz vazia", ErrForaDaRaiz)
	}
	if strings.TrimSpace(relativo) == "" {
		return "", fmt.Errorf("%w: caminho vazio", ErrForaDaRaiz)
	}
	// `filepath.IsAbs` NAO basta: no Windows, "/etc/passwd" nao e' absoluto (nao tem volume) e seria
	// silenciosamente reinterpretado como relativo -- entrando na raiz com outro significado. Caminho enraizado
	// recusa-se: reinterpretar em silencio e' a forma mais discreta de escrever onde nao se devia.
	if filepath.IsAbs(relativo) || filepath.VolumeName(relativo) != "" ||
		strings.HasPrefix(relativo, "/") || strings.HasPrefix(relativo, `\`) {
		return "", fmt.Errorf("%w: %q e' enraizado/absoluto", ErrForaDaRaiz, relativo)
	}

	// Duas raizes, e a distincao e' o que estava a falhar em CI:
	//   raizPedida -- o que o chamador passou; e' SOB ELA que o caminho devolvido tem de estar.
	//   raizReal   -- o que `EvalSymlinks` resolve; serve apenas para VALIDAR a travessia.
	// Devolver o caminho montado sobre `raizReal` fazia o resultado deixar de estar sob a raiz pedida sempre que a
	// raiz e' uma juncao/symlink -- no Windows de CI, `RUNNER~1` expande para `runneradmin` --, e o chamador via um
	// caminho "fora da raiz" que era, afinal, o caminho autorizado. PATHFIX E-1444.
	raizPedida, err := filepath.Abs(raiz)
	if err != nil {
		return "", err
	}
	raizReal := raizPedida
	if resolvida, err := filepath.EvalSymlinks(raizPedida); err == nil {
		raizReal = resolvida
	}

	limpo := filepath.Clean(relativo)

	// Validacao sobre a raiz REAL: e' aqui que uma travessia escondida por symlink e' apanhada.
	if rel, err := filepath.Rel(raizReal, filepath.Join(raizReal, limpo)); err != nil {
		return "", fmt.Errorf("%w: %q", ErrForaDaRaiz, relativo)
	} else if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q resolve para %q", ErrForaDaRaiz, relativo, rel)
	}
	if limpo == "." || limpo == string(filepath.Separator) {
		return "", fmt.Errorf("%w: %q aponta para a propria raiz", ErrForaDaRaiz, relativo)
	}

	// Devolucao sobre a raiz PEDIDA: o contrato com o chamador e' "dentro da raiz que eu dei".
	return filepath.Join(raizPedida, limpo), nil
}

// AnexarEmArquivo e' o append governado num ficheiro de notas dentro da raiz autorizada.
//
// Devolve (anexou, erro). Conflito de versao devolve `ErrVersaoDivergente` e deixa o ficheiro como estava.
func AnexarEmArquivo(raiz, relativo string, meta MetaDaNota, m Memoria, versaoEsperada int) (bool, error) {
	alvo, err := CaminhoAutorizado(raiz, relativo)
	if err != nil {
		return false, err
	}

	existente, err := os.ReadFile(alvo)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	novo, err := AnexarComVersao(existente, meta, m, versaoEsperada)
	if err != nil {
		return false, err
	}
	if len(novo) == len(existente) || string(novo) == string(existente) {
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(alvo), 0o700); err != nil {
		return false, err
	}
	if err := escreverAtomico(alvo, novo); err != nil {
		return false, err
	}
	return true, nil
}

// escreverAtomico: temporario no mesmo directorio -> sync -> rename. O rename no mesmo volume e' atomico, logo
// nunca existe um instante em que o ficheiro tenha metade do conteudo novo.
func escreverAtomico(alvo string, dados []byte) error {
	dir := filepath.Dir(alvo)
	tmp, err := os.CreateTemp(dir, ".nota-*.tmp")
	if err != nil {
		return err
	}
	nome := tmp.Name()
	defer os.Remove(nome) // se o rename falhar, o temporario nao fica para tras

	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(nome, alvo)
}
