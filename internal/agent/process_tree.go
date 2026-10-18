package agent

// FASE 28.1 item 9, apoio: matar a ARVORE de processos, nao so' o processo directo.
//
// O PROBLEMA, MEDIDO
//
// O teste de timeout do `TestProvider` usou `cmd.exe /c ping -n 60 127.0.0.1` com timeout de 400 ms.
// O `exec.CommandContext` matou o `cmd.exe` -- e o `ping` filho ficou vivo, a segurar o pipe. O
// `CombinedOutput` so' regressou quando o `ping` terminou: **59,5 segundos para um timeout de 400 ms**.
//
//	*Um timeout que mata o pai e deixa o filho nao e' um timeout: e' uma promessa.*
//
// COMO SE MATA UMA ARVORE POR PLATAFORMA
//
//   - Windows: `taskkill /T /F /PID <pid>` -- `/T` mata a arvore, `/F` forca.
//   - Unix: mata-se o GRUPO de processos (`kill -KILL -<pgid>`), e o processo tem de ter sido
//     lancado num grupo proprio para o grupo existir.
//
// Nos dois casos o pior resultado aceitavel e' "nao foi possivel matar": devolve-se o erro e o
// `WaitDelay` do `exec.Cmd` fecha os pipes de qualquer forma.

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// matarArvoreDeProcessos termina o comando e todos os seus descendentes.
//
// Devolve o erro da tentativa para o `exec.Cmd` poder decidir o que fazer. Um erro aqui NAO e' fatal
// para o smoke: o `WaitDelay` garante que a leitura termina.
func matarArvoreDeProcessos(comando *exec.Cmd) error {
	if comando == nil || comando.Process == nil {
		return nil
	}
	pid := comando.Process.Pid
	if runtime.GOOS == "windows" {
		// `taskkill` e' o mecanismo nativo. Nao se usa biblioteca externa para uma coisa que o SO ja'
		// faz, e nao se usa `os.Kill` porque ele mata so' o processo directo.
		matador := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
		matador.Stdout = nil
		matador.Stderr = nil
		if err := matador.Run(); err != nil {
			// Se o processo ja' morreu, `taskkill` falha: nao e' um erro que importe.
			return fmt.Errorf("taskkill %d: %w", pid, err)
		}
		return nil
	}
	// Unix: o grupo de processos e' o pid do lider quando se usa `Setpgid`. Como nao o definimos,
	// matamos o processo directo com SIGKILL -- o melhor que se pode fazer sem grupo.
	if err := comando.Process.Kill(); err != nil {
		return fmt.Errorf("kill %d: %w", pid, err)
	}
	return nil
}

// garantirQueExisteProcesso permite aos testes verificarem que `os` continua a ser usado neste
// ficheiro quando a plataforma nao precisa dele.
var _ = os.Getpid
