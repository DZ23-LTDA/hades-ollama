package agent

// FASE 28.1 item 9 do adendo V3: o `TestProvider`.
//
// O QUE O ADENDO PEDE, E O QUE E' POSSIVEL AQUI
//
// O adendo pede: "instala e executa smoke/E2E em maquina, VM, emulador, simulador ou dispositivo
// real". Neste host **nao ha' Docker, emulador Android, simulador iOS nem VM** -- e isso e' um
// bloqueio externo honesto, nao uma desculpa para nao implementar nada.
//
// O que **pode** ser feito, e e' o degrau que falta no ciclo actual:
//
//	build -> verificar checksum -> **EXECUTAR o artefacto e observar o efeito**
//
// A E-1261 verifica que o ficheiro nao mudou desde o build. Isso prova integridade, **nao prova que o
// binario corre**. Um `hades-windows-amd64.exe` com o checksum certo e um `main` que devolve 1 na
// primeira linha passa em todos os gates anteriores:
//
//	*Um ficheiro com bytes nao e' um artefacto utilizavel.*
//
// AS TRES FRONTEIRAS QUE ESTE PROVIDER DECLARA
//
//  1. **Nao ha' sandbox.** Executar um binario e' executa-lo com os privilegios deste processo. Sem
//     Docker nesta maquina, nao ha' isolamento -- e o resultado diz isso.
//
//  2. **Nao instala.** Instalar exigiria o instalador a funcionar, e a E-1252 mediu que nao funciona
//     (4 tentativas). O provider testa o artefacto JA' produzido.
//
//  3. **Nao aceita comando arbitrario.** So' executa o proprio artefacto com os argumentos
//     declarados. Um provider que aceitasse um comando qualquer seria *codigo com um nome diferente*.
//
//	*Um evento sem allowlist nao e' um evento: e' codigo com um nome diferente.*

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TimeoutPorOmissaoDeSmoke limita a execucao de um artefacto.
//
//	*Um binario que nao termina nao e' um binario que passou: e' uma pausa que o servidor nao pode
//	recusar depois de comecar.*
const TimeoutPorOmissaoDeSmoke = 60 * time.Second

// maxSaidaDeSmoke limita quanto da saida e' guardado.
const maxSaidaDeSmoke = 8192

// ErrArtefactoInexistente e' devolvido quando o caminho nao existe.
var ErrArtefactoInexistente = errors.New("artefacto inexistente")

// ErrArtefactoNaoExecutavel e' devolvido quando o ficheiro nao tem permissao de execucao.
var ErrArtefactoNaoExecutavel = errors.New("artefacto sem permissao de execucao")

// ErrSmokeSemArgumentos e' devolvido quando nao se declara nenhum argumento.
//
//	*Executar um binario sem argumentos e' executa-lo no modo que o programador nao escolheu.*
var ErrSmokeSemArgumentos = errors.New(
	"smoke sem argumentos: executar o artefacto sem argumentos e' executa-lo no modo por omissao, " +
		"que ninguem escolheu",
)

// PedidoDeSmoke descreve o que executar.
type PedidoDeSmoke struct {
	// Caminho e' o artefacto absoluto a executar.
	Caminho string `json:"caminho"`
	// Argumentos sao passados ao artefacto. O primeiro elemento NAO e' o nome do programa: e' sempre
	// o proprio artefacto que e' executado.
	Argumentos []string `json:"argumentos"`
	// Timeout limita a execucao. Zero usa `TimeoutPorOmissaoDeSmoke`.
	Timeout time.Duration `json:"-"`
	// SaidaEsperada, quando nao vazio, tem de aparecer na saida (stdout+stderr) para o smoke passar.
	SaidaEsperada string `json:"saida_esperada,omitempty"`
	// CodigoEsperado, quando nao nil, e' o exit code que o smoke aceita.
	CodigoEsperado *int `json:"codigo_esperado,omitempty"`
}

// ResultadoDeSmoke e' o que aconteceu, sem interpretacao.
type ResultadoDeSmoke struct {
	Caminho        string   `json:"caminho"`
	Argumentos     []string `json:"argumentos"`
	Executado      bool     `json:"executado"`
	CodigoDeSaida  int      `json:"codigo_de_saida"`
	DuracaoMS      int64    `json:"duracao_ms"`
	Saida          string   `json:"saida"`
	SaidaTruncada  bool     `json:"saida_truncada"`
	Timeout        bool     `json:"timeout"`
	Aprovado       bool     `json:"aprovado"`
	Explicacao     string   `json:"explicacao"`
	SemSandbox     bool     `json:"sem_sandbox"`
	AvisoDeSandbox string   `json:"aviso_de_sandbox"`
}

// TestProvider e' o contrato do item 9 do adendo 28.1.
type TestProvider interface {
	// ExecutarSmoke corre o artefacto com os argumentos dados e devolve o observado.
	ExecutarSmoke(ctx context.Context, pedido PedidoDeSmoke) (ResultadoDeSmoke, error)
}

// ProviderDeSmoke executa o proprio artefacto, sem shell.
type ProviderDeSmoke struct {
	// NotFound permite substituir a procura de executaveis nos testes.
	NotFound func(string) (string, error)
}

// avisoDeSandbox diz, no resultado, que nao houve isolamento.
func avisoDeSandbox() string {
	return "este smoke executou o artefacto com os privilegios deste processo; NAO ha' sandbox, " +
		"contentor, VM, emulador ou simulador nesta maquina. Executar codigo nao confiavel exige " +
		"isolamento que este provider nao oferece."
}

// ExecutarSmoke corre o artefacto e observa.
//
// A ordem das recusas importa: existencia, depois executabilidade, depois argumentos. Recusar os
// argumentos antes de saber se o ficheiro existe mandaria corrigir a coisa errada.
func (p ProviderDeSmoke) ExecutarSmoke(
	ctx context.Context,
	pedido PedidoDeSmoke,
) (ResultadoDeSmoke, error) {
	resultado := ResultadoDeSmoke{
		Caminho:        pedido.Caminho,
		Argumentos:     append([]string{}, pedido.Argumentos...),
		SemSandbox:     true,
		AvisoDeSandbox: avisoDeSandbox(),
	}
	if err := ctx.Err(); err != nil {
		return resultado, err
	}

	if strings.TrimSpace(pedido.Caminho) == "" {
		return resultado, fmt.Errorf("%w: caminho vazio", ErrArtefactoInexistente)
	}
	absoluto, err := filepath.Abs(pedido.Caminho)
	if err != nil {
		return resultado, fmt.Errorf("%w: resolver %q: %v", ErrArtefactoInexistente, pedido.Caminho, err)
	}
	estado, err := os.Stat(absoluto)
	if err != nil {
		return resultado, fmt.Errorf("%w: %q: %v", ErrArtefactoInexistente, absoluto, err)
	}
	if estado.IsDir() {
		return resultado, fmt.Errorf("%w: %q e' um directorio", ErrArtefactoInexistente, absoluto)
	}
	resultado.Caminho = absoluto

	// Executabilidade: no Windows qualquer `.exe` corre; nos outros, o bit de execucao tem de estar
	// posto. Onde a plataforma nao tem bit de execucao, nao se inventa uma verificacao.
	if runtime.GOOS != "windows" && estado.Mode().Perm()&0o111 == 0 {
		resultado.Explicacao = fmt.Sprintf(
			"%q nao tem bit de execucao (modo %s)", absoluto, estado.Mode().Perm(),
		)
		return resultado, fmt.Errorf("%w: %s", ErrArtefactoNaoExecutavel, resultado.Explicacao)
	}

	if len(pedido.Argumentos) == 0 {
		resultado.Explicacao = ErrSmokeSemArgumentos.Error()
		return resultado, ErrSmokeSemArgumentos
	}

	limite := pedido.Timeout
	if limite <= 0 {
		limite = TimeoutPorOmissaoDeSmoke
	}
	contexto, cancelar := context.WithTimeout(ctx, limite)
	defer cancelar()

	comando := exec.CommandContext(contexto, absoluto, pedido.Argumentos...)
	// Ambiente minimo: sem herdar tudo, para o smoke nao depender de variaveis do programador.
	//
	// `ambienteDeBuild("")` e' o ambiente do HOST: o smoke executa um binario JA' COMPILADO, e o
	// `GOOS` no ambiente nao muda nada no que um executavel faz -- mas passar o alvo aqui sugeriria
	// que o smoke compila, e nao compila.
	comando.Env = ambienteDeBuild("")
	// O `CommandContext` mata o processo DIRECTO quando o contexto expira. Num Windows, matar o
	// `cmd.exe` NAO mata o `ping` que ele lancou: o `CombinedOutput` continua a esperar pelo pipe
	// aberto e o smoke fica pendurado muito depois do timeout. Medido num teste: 400 ms de timeout
	// viraram 59,5 s de espera.
	//
	//	*Um timeout que mata o pai e deixa o filho nao e' um timeout: e' uma promessa.*
	//
	// `Cancel` mata a ARVORE; o `WaitDelay` fecha os pipes e desbloqueia a leitura.
	comando.Cancel = func() error { return matarArvoreDeProcessos(comando) }
	comando.WaitDelay = 2 * time.Second

	inicio := time.Now()
	saidaBruta, err := comando.CombinedOutput()
	resultado.DuracaoMS = time.Since(inicio).Milliseconds()
	resultado.Executado = true

	texto := string(saidaBruta)
	if len(texto) > maxSaidaDeSmoke {
		texto = texto[:maxSaidaDeSmoke]
		resultado.SaidaTruncada = true
	}
	resultado.Saida = texto

	if contexto.Err() == context.DeadlineExceeded {
		resultado.Timeout = true
		resultado.Explicacao = fmt.Sprintf(
			"o artefacto nao terminou em %s; um processo que nao termina nao e' um processo que "+
				"passou", limite,
		)
		return resultado, nil
	}

	resultado.CodigoDeSaida = 0
	if err != nil {
		var erroDeSaida *exec.ExitError
		if errors.As(err, &erroDeSaida) {
			resultado.CodigoDeSaida = erroDeSaida.ExitCode()
		} else {
			// Falha de arranque: o binario nao chegou a correr.
			resultado.Executado = false
			resultado.Explicacao = fmt.Sprintf("nao foi possivel arrancar o artefacto: %v", err)
			return resultado, nil
		}
	}

	// O veredicto.
	if pedido.CodigoEsperado != nil && resultado.CodigoDeSaida != *pedido.CodigoEsperado {
		resultado.Explicacao = fmt.Sprintf(
			"exit code %d; esperado %d", resultado.CodigoDeSaida, *pedido.CodigoEsperado,
		)
		return resultado, nil
	}
	if pedido.SaidaEsperada != "" && !strings.Contains(resultado.Saida, pedido.SaidaEsperada) {
		resultado.Explicacao = fmt.Sprintf(
			"a saida nao contem %q", pedido.SaidaEsperada,
		)
		return resultado, nil
	}
	resultado.Aprovado = true
	resultado.Explicacao = fmt.Sprintf(
		"o artefacto arrancou, terminou com exit code %d em %d ms", resultado.CodigoDeSaida,
		resultado.DuracaoMS,
	)
	return resultado, nil
}

// AlvosTestaveis declara o que este provider consegue testar NESTE host.
//
// A lista e' curta de proposito. *Um provider que existe no catalogo e nunca correu e' uma promessa
// com nome de modulo* -- e prometer emulador Android ou simulador iOS num host Windows seria
// exactamente isso.
func AlvosTestaveis() []string {
	alvos := []string{"BINARIO_DO_HOST"}
	if runtime.GOOS == "windows" {
		alvos = append(alvos, "EXE_WINDOWS")
	}
	return alvos
}

// AlvosNaoTestaveis declara, com o motivo, o que o adendo pede e este host nao permite.
func AlvosNaoTestaveis() map[string]string {
	return map[string]string{
		"EMULADOR_ANDROID":   "exige Android SDK/emulador; ausente neste host (BLOCKED_EXTERNAL)",
		"SIMULADOR_IOS":      "exige macOS com Xcode; impossivel num host Windows (BLOCKED_EXTERNAL)",
		"VM_EFEMERA":         "exige hipervisor e imagem; ausente neste host (BLOCKED_EXTERNAL)",
		"CONTENTOR_DOCKER":   "Docker ausente neste host (BLOCKED_EXTERNAL)",
		"DISPOSITIVO_FISICO": "exige hardware ligado; nao ha' dispositivo autorizado nesta sessao",
	}
}
