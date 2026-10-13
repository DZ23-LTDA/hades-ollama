package agent

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// CONFIRMAÇÃO POR COMPORTAMENTO DAS DUAS CATEGORIAS DECLARADAS (E-347).
//
// A catraca parou em **2 pendências**, ambas declaradas com **motivo categórico** e **nenhuma medida**:
//  1. `scheduling_crossprocess` — **DELEGAÇÃO**: monta o caminho do lock e o chamador valida.
//  2. `benchmark_suite` — **NÃO É ORGANIZAÇÃO**: o componente vem de **nome de tarefa** (identificador interno).
//
// ***Motivo categórico não é medição.*** Este arquivo mede as duas — e é honesto sobre o que **não** consegue provar.

// --- 1. DELEGAÇÃO: os pontos de entrada públicos recusam escopo ausente ---

func TestDelegacaoDeEscopoRecusaOrganizacaoVazia(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "sched")
	store := NewSchedulingStore(raiz)
	agora := time.Now().UTC()

	// Book: a organização vem DENTRO do pedido.
	requisicao := BookRequest{
		OrganizationID: "", ResourceID: "recurso", ServiceID: "servico",
		ContactID: "contato", Channel: "voice", StartsAt: agora.Add(time.Hour), IdempotencyKey: "k1",
	}
	if _, err := store.Book(requisicao); err == nil {
		t.Fatal("ARMADILHA: Book com organização vazia foi aceito (o wrapper delega, mas o chamador valida)")
	}
	// Cancel e Reschedule: organização como primeiro parâmetro.
	if _, err := store.Cancel("", "agendamento-qualquer", "motivo"); err == nil {
		t.Fatal("ARMADILHA: Cancel com organização vazia foi aceito")
	}
	if _, err := store.Reschedule("", "agendamento-qualquer", agora.Add(2*time.Hour)); err == nil {
		t.Fatal("ARMADILHA: Reschedule com organização vazia foi aceito")
	}
	// A recusa tem de ser por ESCOPO, não por outro motivo qualquer.
	//
	// MEDIDO (E-347): antes desta correção, `Cancel("")` e `Reschedule("")` eram ACEITOS e só falhavam depois com
	// "agendamento não encontrado" — ou seja, com um identificador VÁLIDO operariam no escopo COMPARTILHADO. A categoria
	// "delegação" NÃO se sustentava para estes dois pontos de entrada; o motivo categórico era, na verdade, um furo.
	if _, err := store.Cancel("", "x", "y"); !errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("Cancel sem organização precisa recusar pelo MOTIVO de escopo: %v", err)
	}
	if _, err := store.Reschedule("", "x", agora.Add(2*time.Hour)); !errors.Is(err, ErrTenantScopeRequired) {
		t.Fatalf("Reschedule sem organização precisa recusar pelo MOTIVO de escopo: %v", err)
	}
	assertSemArquivoCompartilhado(t, raiz)
}

// --- 2. NÃO É ORGANIZAÇÃO: o identificador vem de NOME DE TAREFA e precisa ser único e não vazio ---

// nomesDeTarefaDaSuite devolve os nomes MEDIDOS das tarefas internas do benchmark.
func nomesDeTarefaDaSuite(agora time.Time) []string {
	resultados := []BenchmarkTaskResult{
		benchmarkBusinessJourney(agora),
		benchmarkIdempotentOrder(agora),
		benchmarkInboxSLAAndHandoff(agora),
		benchmarkOrganizationIsolation(agora),
		benchmarkRestartRecovery(agora),
	}
	nomes := make([]string, 0, len(resultados))
	for _, resultado := range resultados {
		nomes = append(nomes, resultado.Name)
	}
	return nomes
}

// validarNomesDeTarefa é a propriedade que substitui o escopo de tenant para identificadores INTERNOS.
//
// POR QUE ISTO, E NÃO `requireOrganization` (E-347): o identificador destas tarefas **não vem do usuário** — vem do
// código. O risco aqui **não** é vazamento entre organizações, é **COLISÃO DE CAMINHO**: dois nomes iguais ou um nome
// vazio fazem duas tarefas compartilharem o mesmo diretório de dados. *A propriedade certa muda com a origem do dado.*
func validarNomesDeTarefa(nomes []string) error {
	if len(nomes) == 0 {
		return errors.New("nenhuma tarefa declarada")
	}
	vistos := map[string]bool{}
	for _, nome := range nomes {
		if nome == "" {
			return errors.New("nome de tarefa vazio: o caminho de dados ficaria compartilhado")
		}
		if vistos[nome] {
			return errors.New("nome de tarefa duplicado: duas tarefas no mesmo diretório de dados: " + nome)
		}
		vistos[nome] = true
	}
	return nil
}

func TestTarefasDeBenchmarkTemNomeUnicoENaoVazio(t *testing.T) {
	agora := time.Now().UTC()
	nomes := nomesDeTarefaDaSuite(agora)
	if err := validarNomesDeTarefa(nomes); err != nil {
		t.Fatalf("as tarefas internas do benchmark precisam de identificador íntegro: %v (nomes=%v)", err, nomes)
	}
	if len(nomes) < 5 {
		t.Fatalf("a suíte precisa cobrir as tarefas conhecidas: %v", nomes)
	}
}

// A direção NEGATIVA: a propriedade só vale se REPROVAR nome vazio e nome duplicado.
func TestValidacaoDeNomeDeTarefaReprovaVazioEDuplicado(t *testing.T) {
	if err := validarNomesDeTarefa([]string{"a", ""}); err == nil {
		t.Fatal("ARMADILHA: nome vazio precisa reprovar (compartilharia o diretório de dados)")
	}
	if err := validarNomesDeTarefa([]string{"a", "b", "a"}); err == nil {
		t.Fatal("ARMADILHA: nome duplicado precisa reprovar (duas tarefas no mesmo diretório)")
	}
	if err := validarNomesDeTarefa(nil); err == nil {
		t.Fatal("lista vazia precisa reprovar")
	}
	if err := validarNomesDeTarefa([]string{"a", "b"}); err != nil {
		t.Fatalf("nomes íntegros precisam passar: %v", err)
	}
}
