package agent

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Exclusao mutua entre processos para a escrita de uma nota.
//
// Deliberadamente PORTATIL: um ficheiro-sentinela criado com O_EXCL. Nao e' `flock` -- e' a primitiva que existe
// igual nos tres sistemas sem `syscall` por plataforma, e o preco esta' declarado em `ErrNotaBloqueada`.
//
// O limite que isto NAO resolve: se o processo morrer depois de criar o sentinela, o sentinela fica. Nao ha
// recuperacao automatica por idade -- recuperar sozinho um bloqueio alheio e' como arrombar uma porta por estar
// fechada ha muito tempo. `VersaoDoBloqueio` e `DestravarNota` existem para o operador decidir com informacao.

// ErrNotaBloqueada: outro escritor tem o bloqueio e o chamador NAO deve insistir em silencio.
var ErrNotaBloqueada = errors.New("memory note is locked by another writer")

func caminhoDoBloqueio(alvo string) string {
	return alvo + ".lock"
}

// TravarNota cria o sentinela de forma exclusiva. Devolve `ErrNotaBloqueada` (com o dono declarado, se houver)
// quando ja' existe.
func TravarNota(alvo string) error {
	caminho := caminhoDoBloqueio(alvo)
	f, err := os.OpenFile(caminho, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			dono, _ := os.ReadFile(caminho)
			return fmt.Errorf("%w: %s (%s)", ErrNotaBloqueada, caminho, string(dono))
		}
		return err
	}
	// O dono fica escrito ANTES de fechar: um sentinela sem dono obriga o operador a adivinhar quem o criou.
	_, werr := f.WriteString(fmt.Sprintf("pid=%d", os.Getpid()))
	cerr := f.Close()
	if werr != nil {
		os.Remove(caminho)
		return werr
	}
	if cerr != nil {
		os.Remove(caminho)
		return cerr
	}
	return nil
}

// DestravarNota remove o sentinela. Idempotente: destravar o que ja' nao esta' travado nao e' erro.
func DestravarNota(alvo string) error {
	err := os.Remove(caminhoDoBloqueio(alvo))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Bloqueado diz se ha' sentinela, e devolve o dono declarado quando existe.
func Bloqueado(alvo string) (bool, string) {
	conteudo, err := os.ReadFile(caminhoDoBloqueio(alvo))
	if err != nil {
		return false, ""
	}
	return true, string(conteudo)
}

// AnexarEmArquivoComBloqueio e' `AnexarEmArquivo` com exclusao mutua: adquire, escreve, liberta.
//
// Se o bloqueio estiver tomado, devolve `ErrNotaBloqueada` e NAO escreve -- um escritor que insiste em silencio
// produz exactamente a corrida que o bloqueio devia impedir.
func AnexarEmArquivoComBloqueio(raiz, relativo string, meta MetaDaNota, m Memoria, versaoEsperada int) (bool, error) {
	alvo, err := CaminhoAutorizado(raiz, relativo)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(alvo), 0o700); err != nil {
		return false, err
	}
	if err := TravarNota(alvo); err != nil {
		return false, err
	}
	defer DestravarNota(alvo)
	return AnexarEmArquivo(raiz, relativo, meta, m, versaoEsperada)
}
