package agent

// Testes do `TestProvider` (E-1267).
//
// O que estes testes travam:
//
//  1. um artefacto que corre e responde e' aprovado;
//  2. um artefacto que corre mas devolve o exit code errado REPROVA;
//  3. um artefacto que nao produz a saida esperada REPROVA;
//  4. um artefacto que nao termina dentro do timeout devolve `Timeout: true` -- e nao "aprovado";
//  5. um caminho inexistente e' recusado com `ErrArtefactoInexistente`;
//  6. um directorio e' recusado;
//  7. **sem argumentos e' recusado** -- executar no modo por omissao nao e' um teste;
//  8. o resultado declara `SemSandbox` e o aviso;
//  9. `AlvosNaoTestaveis` declara o que este host nao consegue, com motivo.
//
// Os artefactos sao GERADOS no teste (binarios Go minusculos compilados na hora sao caros; usamos o
// `go` do proprio ambiente para compilar um ou usamos um script que o SO executa). Para nao depender
// de compilador no teste, usamos o proprio executavel do `go` quando existe -- e `skip` honesto
// quando nao.

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// executavelDeTeste devolve um executavel real deste host que aceita argumentos e termina.
//
// No Windows e' `cmd.exe`; nos outros e' `/bin/sh`. Nao inventamos um binario: usamos um que existe,
// para o teste medir o MECANISMO e nao um mock do mecanismo.
//
//	*Um teste que substitui o cliente testa o mock, nao o produto.*
func executavelDeTeste(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		caminho, err := exec.LookPath("cmd.exe")
		if err != nil {
			t.Skipf("cmd.exe nao encontrado: %v", err)
		}
		return caminho
	}
	caminho, err := exec.LookPath("sh")
	if err != nil {
		t.Skipf("sh nao encontrado: %v", err)
	}
	return caminho
}

// argumentosDeEcho devolve os argumentos que fazem o executavel imprimir a marca.
func argumentosDeEcho(marca string) []string {
	if runtime.GOOS == "windows" {
		return []string{"/c", "echo " + marca}
	}
	return []string{"-c", "echo " + marca}
}

func TestProviderDeSmokeAprovaArtefactoQueResponde(t *testing.T) {
	executavel := executavelDeTeste(t)
	provider := ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:       executavel,
		Argumentos:    argumentosDeEcho("HADES-SMOKE-OK"),
		SaidaEsperada: "HADES-SMOKE-OK",
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !resultado.Aprovado {
		t.Fatalf("nao aprovado: %+v", resultado)
	}
	if !resultado.Executado {
		t.Fatal("executado = false num artefacto que correu")
	}
	if !strings.Contains(resultado.Saida, "HADES-SMOKE-OK") {
		t.Fatalf("saida nao contem a marca: %q", resultado.Saida)
	}
	// A fronteira de sandbox tem de estar no RESULTADO, nao so' na documentacao.
	if !resultado.SemSandbox {
		t.Fatal("sem_sandbox = false: nao ha' isolamento nesta maquina")
	}
	if resultado.AvisoDeSandbox == "" {
		t.Fatal("aviso_de_sandbox vazio")
	}
}

// O ARTEFACTO CORRE MAS NAO FAZ O QUE SE ESPERA: reprova.
//
//	*Um ficheiro com bytes nao e' um artefacto utilizavel.*
func TestProviderDeSmokeReprovaSaidaErrada(t *testing.T) {
	executavel := executavelDeTeste(t)
	provider := ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:       executavel,
		Argumentos:    argumentosDeEcho("OUTRA-COISA"),
		SaidaEsperada: "HADES-SMOKE-OK",
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if resultado.Aprovado {
		t.Fatal("aprovado com a saida errada")
	}
	if !strings.Contains(resultado.Explicacao, "nao contem") {
		t.Fatalf("explicacao nao diz o que faltou: %q", resultado.Explicacao)
	}
}

func TestProviderDeSmokeReprovaExitCodeErrado(t *testing.T) {
	executavel := executavelDeTeste(t)
	esperado := 0
	provider := ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:        executavel,
		Argumentos:     argumentosDeEcho("x"),
		CodigoEsperado: &esperado,
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !resultado.Aprovado {
		t.Fatalf("devia aprovar com exit 0: %+v", resultado)
	}

	// Agora com um codigo que nao acontece.
	tres := 3
	resultadoErrado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:        executavel,
		Argumentos:     argumentosDeEcho("x"),
		CodigoEsperado: &tres,
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if resultadoErrado.Aprovado {
		t.Fatal("aprovado com exit code 0 quando se esperava 3")
	}
}

// Um artefacto que nao termina NAO e' um artefacto que passou.
func TestProviderDeSmokeDetetaTimeout(t *testing.T) {
	executavel := executavelDeTeste(t)
	var argumentos []string
	if runtime.GOOS == "windows" {
		// `ping` sem destino conta ate' ao infinito; `timeout` precisa de consola. Usamos o proprio
		// cmd a esperar.
		argumentos = []string{"/c", "ping -n 60 127.0.0.1 > nul"}
	} else {
		argumentos = []string{"-c", "sleep 30"}
	}
	provider := ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:    executavel,
		Argumentos: argumentos,
		Timeout:    400 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if resultado.Aprovado {
		t.Fatal("aprovado um processo que nao terminou")
	}
	if !resultado.Timeout {
		t.Fatalf("timeout = false: %+v", resultado)
	}
	if !strings.Contains(resultado.Explicacao, "nao terminou") {
		t.Fatalf("explicacao nao declara o timeout: %q", resultado.Explicacao)
	}
}

func TestProviderDeSmokeRecusaCaminhoInexistente(t *testing.T) {
	provider := ProviderDeSmoke{}
	_, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:    filepath.Join(t.TempDir(), "nao-existe.exe"),
		Argumentos: []string{"--version"},
	})
	if !errors.Is(err, ErrArtefactoInexistente) {
		t.Fatalf("err = %v; esperado ErrArtefactoInexistente", err)
	}
}

func TestProviderDeSmokeRecusaDirectorio(t *testing.T) {
	provider := ProviderDeSmoke{}
	_, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho:    t.TempDir(),
		Argumentos: []string{"--version"},
	})
	if !errors.Is(err, ErrArtefactoInexistente) {
		t.Fatalf("err = %v; esperado ErrArtefactoInexistente para um directorio", err)
	}
}

// Sem argumentos e' recusado: executar no modo por omissao nao e' um teste.
func TestProviderDeSmokeRecusaSemArgumentos(t *testing.T) {
	executavel := executavelDeTeste(t)
	provider := ProviderDeSmoke{}
	resultado, err := provider.ExecutarSmoke(context.Background(), PedidoDeSmoke{
		Caminho: executavel,
	})
	if !errors.Is(err, ErrSmokeSemArgumentos) {
		t.Fatalf("err = %v; esperado ErrSmokeSemArgumentos", err)
	}
	if resultado.Executado {
		t.Fatal("executado = true sem argumentos: o artefacto nao podia ter corrido")
	}
}

func TestProviderDeSmokeDeclaraAlvos(t *testing.T) {
	alvos := AlvosTestaveis()
	if len(alvos) == 0 {
		t.Fatal("nenhum alvo testavel declarado")
	}
	// *Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
	// A lista tem de ser curta e verdadeira.
	for _, alvo := range alvos {
		if alvo == "SIMULADOR_IOS" || alvo == "EMULADOR_ANDROID" {
			t.Fatalf("alvo %q declarado testavel num host que nao tem emulador/simulador", alvo)
		}
	}
	naoTestaveis := AlvosNaoTestaveis()
	for _, obrigatorio := range []string{"EMULADOR_ANDROID", "SIMULADOR_IOS", "CONTENTOR_DOCKER"} {
		motivo, ok := naoTestaveis[obrigatorio]
		if !ok {
			t.Fatalf("%q nao esta' em AlvosNaoTestaveis", obrigatorio)
		}
		if !strings.Contains(motivo, "BLOCKED_EXTERNAL") {
			t.Fatalf("%q sem classificacao BLOCKED_EXTERNAL: %q", obrigatorio, motivo)
		}
	}
}
