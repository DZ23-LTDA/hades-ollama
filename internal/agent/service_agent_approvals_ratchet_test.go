package agent

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// E-907 / ALTO-4 - CATRACA DA TRAVA DO STORE DE APROVACOES.
//
// O achado: `Request`, `expireStale`, `SetApprovers` e `Save` faziam leitura-modificacao-escrita SEM trava,
// enquanto `Consume` (E-424) e `Decide` (E-426) ja' a tinham. A janela: uma dessas quatro LE o ficheiro,
// depois o `Consume` grava "consumida", e a gravacao seguinte reescreve o ficheiro INTEIRO com o instantaneo
// antigo - a aprovacao CONSUMIDA **ressuscita**.
//
// Estes testes discriminam o defeito: com a versao anterior (sem trava e sem releitura sob a trava) o
// primeiro falha de forma deterministica, porque `expireStale`/`Save`/`SetApprovers` gravavam a lista que
// tinham lido ANTES do consumo.

func storeDeAprovacaoParaCatraca(t *testing.T) (*ServiceAgentApprovalStore, string) {
	t.Helper()
	root := t.TempDir()
	store := NewServiceAgentApprovalStore(filepath.Join(root, "approvals"))
	if store == nil {
		t.Fatal("NewServiceAgentApprovalStore devolveu nil")
	}
	return store, "org-catraca"
}

// pedidoAprovadoEConsumido cria um pedido, aprova-o com ator humano e consome-o. Devolve o ID.
func pedidoAprovadoEConsumido(t *testing.T, store *ServiceAgentApprovalStore, org, fingerprint string) string {
	t.Helper()
	agora := time.Now().UTC()
	pedido, err := store.Request(org, ServiceAgentApproval{
		Fingerprint: fingerprint,
		ValueCents:  100,
	})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if _, err := store.Decide(org, pedido.ID, true, "humano-1", "aprovado no teste", agora); err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if _, err := store.Consume(org, pedido.ID, agora); err != nil {
		t.Fatalf("Consume: %v", err)
	}
	return pedido.ID
}

func statusDaAprovacao(t *testing.T, store *ServiceAgentApprovalStore, org, id string) string {
	t.Helper()
	aprovacoes, err := store.Load(org)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, a := range aprovacoes {
		if a.ID == id {
			return a.Status
		}
	}
	t.Fatalf("aprovacao %q nao encontrada", id)
	return ""
}

// TestExpireStaleNaoRessuscitaAprovacaoConsumida e' o discriminante DIRETO do achado.
//
// Magnetude: `expireStale` reescreve o FICHEIRO INTEIRO. Se ele gravar a lista que leu antes do `Consume`,
// a aprovacao que o `Consume` marcou como CONSUMIDA volta a ficar APPROVED - e uma ferramenta ja' executada
// poderia executar de novo. Com a releitura sob a trava, o status permanece CONSUMIDA.
func TestExpireStaleNaoRessuscitaAprovacaoConsumida(t *testing.T) {
	store, org := storeDeAprovacaoParaCatraca(t)
	id := pedidoAprovadoEConsumido(t, store, org, "fp-ressuscita")

	if s := statusDaAprovacao(t, store, org, id); s != ServiceAgentApprovalConsumed {
		t.Fatalf("pre-condicao: esperava %q, obtive %q", ServiceAgentApprovalConsumed, s)
	}

	// Uma expiracao "varre o ficheiro" depois do consumo. Com TTL zero nada expira - mas a funcao ainda pode
	// reescrever se achar que mudou; o que importa e' que o estado CONSUMIDO sobreviva.
	if err := store.expireStale(org, time.Now().UTC().Add(48*time.Hour)); err != nil {
		t.Fatalf("expireStale: %v", err)
	}

	if s := statusDaAprovacao(t, store, org, id); s == ServiceAgentApprovalApproved {
		t.Fatalf("REGRESSAO: `expireStale` ressuscitou a aprovacao consumida (status %q)", s)
	} else if s != ServiceAgentApprovalConsumed {
		t.Fatalf("status inesperado apos expireStale: %q", s)
	}
}

// TestSaveESetApproversNaoRessuscitamAprovacaoConsumida cobre os OUTROS dois escritores de ficheiro inteiro.
//
// `Save` e `SetApprovers` tambem gravam o ficheiro todo. O teste passa uma lista construida a partir de uma
// leitura ANTERIOR ao consumo: se a gravacao nao reler sob a trava, o estado consumido perde-se.
func TestSaveESetApproversNaoRessuscitamAprovacaoConsumida(t *testing.T) {
	store, org := storeDeAprovacaoDaCatracaComLista(t)
	agora := time.Now().UTC()

	pedido, err := store.Request(org, ServiceAgentApproval{Fingerprint: "fp-save", ValueCents: 100})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	// Instantaneo ANTIGO: lido ANTES de decidir/consumir.
	antigo, err := store.Load(org)
	if err != nil {
		t.Fatalf("Load (instantaneo antigo): %v", err)
	}

	if _, err := store.Decide(org, pedido.ID, true, "humano-2", "ok", agora); err != nil {
		t.Fatalf("Decide: %v", err)
	}
	if _, err := store.Consume(org, pedido.ID, agora); err != nil {
		t.Fatalf("Consume: %v", err)
	}

	// O escritor cego: grava o instantaneo antigo (com a aprovacao ainda PENDENTE).
	if err := store.Save(org, antigo); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if s := statusDaAprovacao(t, store, org, pedido.ID); s == ServiceAgentApprovalPending {
		t.Fatalf("REGRESSAO: `Save` gravou o instantaneo antigo e reabriu um pedido ja' consumido (status %q)", s)
	}

	// `SetApprovers` tambem reescreve o ficheiro: os aprovadores mudam, as aprovacoes NAO podem mudar.
	if err := store.SetApprovers(org, []string{"humano-3"}); err != nil {
		t.Fatalf("SetApprovers: %v", err)
	}
	if s := statusDaAprovacao(t, store, org, pedido.ID); s != ServiceAgentApprovalConsumed {
		t.Fatalf("SetApprovers alterou o estado da aprovacao: %q", s)
	}
	aprovadores, err := store.Approvers(org)
	if err != nil {
		t.Fatalf("Approvers: %v", err)
	}
	if len(aprovadores) != 1 || aprovadores[0] != "humano-3" {
		t.Fatalf("SetApprovers nao gravou a lista: %#v", aprovadores)
	}
}

// storeDeAprovacaoDaCatracaComLista cria o store com uma lista de aprovadores configurada, para o
// `SetApprovers` ter o que alterar e o `Decide` exercitar a checagem de papel.
func storeDeAprovacaoDaCatracaComLista(t *testing.T) (*ServiceAgentApprovalStore, string) {
	t.Helper()
	store, org := storeDeAprovacaoParaCatraca(t)
	if err := store.SetApprovers(org, []string{"humano-2"}); err != nil {
		t.Fatalf("SetApprovers inicial: %v", err)
	}
	return store, org
}

// TestConsumeConcorrenteComEscritoresNaoDuplicaExecucao e' a prova de concorrencia do prompt:
// *uma ferramenta aprovada uma vez NAO pode executar duas vezes*, mesmo com escritores a competir.
//
// N produtores de pedidos + N consumidores + N varredores de expiracao, todos no MESMO store (logo a mesma
// trava entre processos, porque `withFileLock` e' um lock de ficheiro). No fim, cada aprovacao tem de estar
// CONSUMIDA no maximo UMA vez - e o numero de consumos bem-sucedidos tem de igualar o numero de aprovacoes
// efetivamente consumidas no ficheiro.
func TestConsumeConcorrenteComEscritoresNaoDuplicaExecucao(t *testing.T) {
	store, org := storeDeAprovacaoParaCatraca(t)
	agora := time.Now().UTC()

	const quantos = 12
	ids := make([]string, 0, quantos)
	for i := 0; i < quantos; i++ {
		pedido, err := store.Request(org, ServiceAgentApproval{
			Fingerprint: "fp-conc-" + string(rune('a'+i)),
			ValueCents:  100,
		})
		if err != nil {
			t.Fatalf("Request %d: %v", i, err)
		}
		if _, err := store.Decide(org, pedido.ID, true, "humano-conc", "ok", agora); err != nil {
			t.Fatalf("Decide %d: %v", i, err)
		}
		ids = append(ids, pedido.ID)
	}

	var mu sync.Mutex
	consumosOK := map[string]int{}

	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		// Consumidor.
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.Consume(org, id, agora); err == nil {
				mu.Lock()
				consumosOK[id]++
				mu.Unlock()
			}
		}()
		// Escritor de ficheiro inteiro a competir com o consumo.
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.expireStale(org, agora)
		}()
	}
	wg.Wait()

	// Nenhuma aprovacao pode ter sido consumida com sucesso mais de uma vez.
	for id, n := range consumosOK {
		if n > 1 {
			t.Fatalf("aprovacao %q consumida %d vezes: a ferramenta executaria mais de uma vez", id, n)
		}
	}
	// E o ficheiro tem de concordar: consumidas = consumos bem-sucedidos.
	aprovacoes, err := store.Load(org)
	if err != nil {
		t.Fatalf("Load final: %v", err)
	}
	consumidasNoFicheiro := 0
	for _, a := range aprovacoes {
		if a.Status == ServiceAgentApprovalConsumed {
			consumidasNoFicheiro++
		}
	}
	if consumidasNoFicheiro != len(consumosOK) {
		t.Fatalf("ficheiro diz %d consumidas e houve %d consumos bem-sucedidos: estado divergente",
			consumidasNoFicheiro, len(consumosOK))
	}
}

// NOTA: o caso "repetir o turno nao reenvia o aviso" NAO e' duplicado aqui. Ele ja' existe em
// `service_agent_approval_notify_test.go` (`TestServiceAgentApprovalNotificationSendsOnceWithMinimalPayload`),
// e foi esse teste que discriminou a regressao que E-907 introduziu ao mover o `notifyPending` para o wrapper.
// *Um teste que ja' existe e ja' falhou pelo defeito certo nao precisa de gemeo.*
