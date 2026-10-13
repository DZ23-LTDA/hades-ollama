package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ESTÁGIO 42 — DOIS PROCESSOS REAIS (E-358).
//
// E-356 e E-357 provaram a correção com **duas instâncias no mesmo processo** — e eu declarei esse limite. Aqui a
// medição é com **processos de sistema operacional separados**, que é o único caso em que a semântica do lock de
// arquivo é inequívoca: cada processo abre o seu próprio descritor e o travamento do sistema operacional (Windows
// `LockFileEx`, Unix `flock`) entra em ação de verdade.
//
// O padrão de subprocesso segue o que o repositório já usa em `mcp_test.go` (`os.Args[0]` + `-test.run=...` + variável
// de ambiente). ***A variável de ambiente aqui vive num arquivo de TESTE — não no código de produção.*** A diferença
// em relação ao ADV-32 não é de estilo: um interruptor em produção desliga a trava para quem quiser; um helper de
// teste só existe quando o binário de teste é executado com aquele nome de teste.

const (
	variavelAjudante = "HADES_LIVE_HELPER"
	variavelRaiz     = "HADES_LIVE_HELPER_ROOT"
	variavelPedido   = "HADES_LIVE_HELPER_ORDER"
	variavelValor    = "HADES_LIVE_HELPER_AMOUNT"
)

// TestProcessoAjudanteDoEstagio42 é o FILHO: só faz trabalho quando chamado como subprocesso.
func TestProcessoAjudanteDoEstagio42(t *testing.T) {
	if os.Getenv(variavelAjudante) != "1" {
		t.Skip("processo ajudante: executa apenas quando chamado pelo teste pai")
	}
	raiz := os.Getenv(variavelRaiz)
	if raiz == "" {
		fmt.Fprintln(os.Stderr, "raiz ausente")
		os.Exit(2)
	}
	// BARREIRA DE LARGADA: todos os processos esperam o mesmo arquivo para largar juntos, maximizando a corrida.
	largada := filepath.Join(raiz, "LARGADA")
	limite := time.Now().Add(2 * time.Minute)
	for {
		if _, err := os.Stat(largada); err == nil {
			break
		}
		if time.Now().After(limite) {
			fmt.Fprintln(os.Stderr, "tempo esgotado esperando a largada")
			os.Exit(3)
		}
		time.Sleep(5 * time.Millisecond)
	}
	store := NewLiveCommerceStore(raiz)
	valor := int64(1000)
	fmt.Sscanf(os.Getenv(variavelValor), "%d", &valor)
	evento := CommerceEvent{
		OrganizationID: "org-a", ExternalID: "proc-" + os.Getenv(variavelPedido),
		Kind: "purchase_confirmed", SKU: "SKU-" + os.Getenv(variavelPedido), OrderID: os.Getenv(variavelPedido),
		AmountCents: valor, SignatureVerified: true,
	}
	if _, _, err := store.IngestLiveCommerceEvent(evento); err != nil {
		fmt.Fprintln(os.Stderr, "ingestao falhou:", err)
		os.Exit(4)
	}
	os.Exit(0)
}

// TestDoisProcessosReaisNaoPerdemEscrita mede a invariante com processos SEPARADOS do sistema operacional.
func TestDoisProcessosReaisNaoPerdemEscrita(t *testing.T) {
	if testing.Short() {
		t.Skip("teste com subprocessos: pulado em modo curto")
	}
	raiz := filepath.Join(t.TempDir(), "live")
	if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatal(err)
	}
	pedidos := []string{"REAL-A", "REAL-B", "REAL-C"}

	var wg sync.WaitGroup
	saidas := make([]string, len(pedidos))
	erros := make([]error, len(pedidos))
	for i, pedido := range pedidos {
		i, pedido := i, pedido
		wg.Add(1)
		go func() {
			defer wg.Done()
			comando := exec.Command(os.Args[0], "-test.run=TestProcessoAjudanteDoEstagio42", "-test.v")
			comando.Env = append(os.Environ(),
				variavelAjudante+"=1", variavelRaiz+"="+raiz,
				variavelPedido+"="+pedido, variavelValor+"=1000")
			saida, err := comando.CombinedOutput()
			saidas[i], erros[i] = string(saida), err
		}()
	}
	// Dá tempo de os três processos abrirem os seus descritores, e só então libera a largada.
	time.Sleep(1500 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(raiz, "LARGADA"), []byte("vai"), 0o600); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	for i, err := range erros {
		if err != nil {
			t.Fatalf("processo %d falhou: %v\n%s", i, err, saidas[i])
		}
	}

	// A prova é lida de um TERCEIRO processo lógico (o pai), lendo o mesmo diretório.
	store := NewLiveCommerceStore(raiz)
	resumo, err := store.ConfirmedSales("org-a")
	if err != nil {
		t.Fatal(err)
	}
	if resumo.Orders != len(pedidos) {
		t.Fatalf("ARMADILHA: %d processos gravaram e só %d pedido(s) sobreviveram (receita=%d) — perda de escrita entre PROCESSOS",
			len(pedidos), resumo.Orders, resumo.Revenue)
	}
	if resumo.Revenue != int64(1000*len(pedidos)) {
		t.Fatalf("receita somada errada com processos reais: %d", resumo.Revenue)
	}
	t.Logf("processos reais medidos: %d processos, %d pedidos, receita=%d", len(pedidos), resumo.Orders, resumo.Revenue)
}
