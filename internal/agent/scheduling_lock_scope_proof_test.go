package agent

import (
	"fmt"
	"testing"
	"time"
)

// E-416 / ADV-63 — A TRAVA É POR ORGANIZAÇÃO: PROVA DIRETA, COM O MESMO CAMINHO QUE O PRODUTO USA.
//
// **O histórico, porque ele é o ponto**: o teste antigo (`scheduling_crossprocess_test.go:112-139`) agenda na
// organização vizinha **depois** de as operações da primeira terminarem ⇒ *prova **isolamento de dados**, não
// **concorrência de travas*** (*mutação para trava global ⇒ 0 falhas em 20 execuções*). Eu então escrevi uma
// prova direta — e ela também **não** pegou a mutação, porque calculava o caminho **replicando a fórmula do
// produto** (E-415). ***Cópia não muda quando o original muda.***
//
// Este teste usa **`loja.lockPath(...)`** — o **mesmo** método que o produto usa em `withOrganizationLock`. Com
// isso, mutar a fórmula muda o **input do próprio teste**, e a asserção dispara.
func TestTravaDeAgendamentoEporOrganizacaoNaoGlobal(t *testing.T) {
	raiz := t.TempDir()
	loja := NewSchedulingStore(raiz)

	// (a) O CAMINHO VEM DO PRODUTO. Duas organizações ⇒ dois arquivos.
	caminhoA := loja.lockPath("org-a")
	caminhoB := loja.lockPath("org-b")
	if caminhoA == caminhoB {
		t.Fatalf("ARMADILHA: as organizações A e B compartilham o caminho de trava %q — a trava é GLOBAL, e o "+
			"comentário de scheduling_crossprocess.go promete o contrário", caminhoA)
	}

	// Cenário da organização B: serviço, recurso e expediente de 24h, para que o único obstáculo seja a trava.
	servico, err := loja.AddService(SchedulingService{OrganizationID: "org-b", Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := loja.AddResource(SchedulingResource{OrganizationID: "org-b", Name: "Profissional Bia"})
	if err != nil {
		t.Fatal(err)
	}
	for dia := range 7 {
		if _, err := loja.AddShift(SchedulingShift{OrganizationID: "org-b", Weekday: dia, OpensAt: 0, ClosesAt: 24 * 60}); err != nil {
			t.Fatal(err)
		}
	}
	alvo := time.Now().UTC().Add(2 * time.Hour).Truncate(30 * time.Minute)

	// (b) A PROVA DIRETA: reter a trava da organização A (pelo caminho do PRODUTO) e exigir que B complete.
	liberarA := make(chan struct{})
	travaRetida := make(chan struct{})
	go func() {
		_ = withFileLock(caminhoA, func() error {
			close(travaRetida)
			<-liberarA
			return nil
		})
	}()
	select {
	case <-travaRetida:
	case <-time.After(5 * time.Second):
		t.Fatal("não consegui reter a trava da organização A — o cenário não foi montado")
	}

	pronto := make(chan error, 1)
	go func() {
		_, err := loja.Book(BookRequest{
			OrganizationID: "org-b", ResourceID: recurso.ID, ServiceID: servico.ID,
			ContactID: "+5511955556666", Channel: "web", StartsAt: alvo, IdempotencyKey: "orgb-trava",
		})
		pronto <- err
	}()

	select {
	case err := <-pronto:
		if err != nil {
			t.Fatalf("a organização B não pode falhar por causa da trava da A: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ARMADILHA: com a trava da organização A RETIDA, o `Book` da organização B não completou em 10s — " +
			"*a trava está bloqueando outra organização*, ou seja é GLOBAL, e o comentário de " +
			"scheduling_crossprocess.go promete trava por organização")
	}
	defer close(liberarA)

	// (c) E o dado da B persistiu, lido por uma instância NOVA — a prova de disco.
	depois, err := NewSchedulingStore(raiz).Appointments("org-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(depois) != 1 {
		t.Fatalf("esperava 1 agendamento da org-b, veio %d (%s)", len(depois), fmt.Sprint(depois))
	}
}
