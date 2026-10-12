package agent

import (
	"path/filepath"
	"testing"
	"time"
)

// Estagio 31 (recuperacao) — a TRILHA ANALITICA do supervisor precisa sobreviver a um REINICIO REAL (E-258).
//
// O limite declarado era "o HISTORICO dos ciclos segue em memoria". Trilha que some no reinicio nao serve para
// recuperacao nem para o operador auditar o que o supervisor decidiu.
func TestTrilhaDoSupervisorSobreviveAoReinicio(t *testing.T) {
	runtime, dataRoot := persistentSupervisorRuntime(t)
	now := time.Now().UTC()
	organizacao := LocalOrganizationID
	// Cenario que gera acao pendente (estorno devido): garante que o ciclo tem o que decidir.
	geraEstornoDevido(t, runtime, organizacao, "chave-trilha", now)

	dir := filepath.Join(dataRoot, "business-supervisor")
	politica := SupervisorPolicy{MaxRefundDueCents: 1}

	primeiro, err := NewBusinessSupervisorAt(dir, runtime, politica)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := primeiro.BusinessTick(organizacao, now); err != nil {
		t.Fatalf("primeiro ciclo: %v", err)
	}
	antes := primeiro.History()
	if len(antes) == 0 {
		t.Fatal("o ciclo precisa deixar trilha")
	}

	// REINICIO REAL: instancia nova sobre o MESMO diretorio de dados.
	segundo, err := NewBusinessSupervisorAt(dir, runtime, politica)
	if err != nil {
		t.Fatal(err)
	}
	depois := segundo.History()
	if len(depois) != len(antes) {
		t.Fatalf("a trilha precisa sobreviver ao reinicio: antes=%d depois=%d", len(antes), len(depois))
	}

	// E continua CRESCENDO: novo ciclo acrescenta em vez de substituir a trilha carregada.
	if _, err := segundo.BusinessTick(organizacao, now.Add(time.Minute)); err != nil {
		t.Fatalf("segundo ciclo: %v", err)
	}
	if atual := len(segundo.History()); atual != len(depois)+1 {
		t.Fatalf("novo ciclo precisa acrescentar a trilha: %d -> %d", len(depois), atual)
	}
}

// Sem diretorio (modo em memoria) o supervisor continua funcionando: a persistencia e' opcional, nao obrigatoria.
func TestSupervisorEmMemoriaNaoTentaPersistir(t *testing.T) {
	runtime, _ := persistentSupervisorRuntime(t)
	supervisor := NewBusinessSupervisor(runtime, SupervisorPolicy{})
	agora := time.Now().UTC()
	if _, err := supervisor.BusinessTick(LocalOrganizationID, agora); err != nil {
		t.Fatalf("modo em memoria precisa continuar funcionando: %v", err)
	}
	if len(supervisor.History()) == 0 {
		t.Fatal("o modo em memoria tambem registra a trilha")
	}
}
