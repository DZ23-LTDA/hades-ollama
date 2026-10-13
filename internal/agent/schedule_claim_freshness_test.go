package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 13/34 — O CLAIM DE AGENDAMENTO PRECISA DECIDIR SOBRE ESTADO FRESCO (E-291).
//
// Propriedade mandatória do §11.7: "missão longa retoma após matar e reiniciar worker; o MESMO EFEITO EXTERNO não
// ocorre duas vezes". A execução agendada é um efeito externo: ela envia mensagem, dispara webhook, publica.
//
// O RISCO MEDIDO: `ClaimDueSchedulesForOrganization` decide pela cópia EM MEMÓRIA e grava em seguida. Dois processos
// sobre a MESMA pasta de dados (o caso real de reinício com sobreposição, ou de runtime duplicado) mantêm cópias
// independentes: o segundo continua vendo o agendamento como "vencido" e DISPARA DE NOVO.
//
// Este teste é hermético e prova as duas direções:
//  1. depois que um store reivindica, o SEGUNDO store (com a cópia velha) NÃO pode reivindicar de novo;
//  2. um agendamento realmente vencido continua sendo reivindicado (a correção não pode travar o agendador).
func TestClaimDeAgendamentoNaoDisparaDuasVezesComDoisProcessos(t *testing.T) {
	raiz := t.TempDir()
	agora := time.Now().UTC()
	agendamento := Schedule{
		ID:              "sched-duplicado",
		OrganizationID:  LocalOrganizationID,
		Workspace:       filepath.Join(raiz, "ws"),
		Enabled:         true,
		IntervalSeconds: 60,
		NextRunAt:       agora.Add(-time.Minute), // vencido
		UpdatedAt:       agora.Add(-2 * time.Minute),
	}
	if err := os.MkdirAll(filepath.Join(raiz, "schedules"), 0o700); err != nil {
		t.Fatal(err)
	}
	dados, err := json.Marshal(agendamento)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(raiz, "schedules", agendamento.ID+".json"), json.RawMessage(dados)); err != nil {
		t.Fatal(err)
	}

	// DOIS processos sobre a MESMA pasta: cada um carrega a cópia vencida.
	primeiro, err := NewContextStore(raiz)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := NewContextStore(raiz)
	if err != nil {
		t.Fatal(err)
	}

	if reivindicados := primeiro.ClaimDueSchedulesForOrganization(LocalOrganizationID, agora); len(reivindicados) != 1 {
		t.Fatalf("o primeiro processo precisa reivindicar o agendamento vencido: %+v", reivindicados)
	}
	// O SEGUNDO ainda tem a cópia velha: reivindicar aqui seria DISPARAR O MESMO EFEITO DUAS VEZES.
	if reivindicados := segundo.ClaimDueSchedulesForOrganization(LocalOrganizationID, agora); len(reivindicados) != 0 {
		t.Fatalf("o segundo processo reivindicou de novo o MESMO agendamento (efeito externo duplicado): %+v",
			reivindicados)
	}
	// E um terceiro processo, criado DEPOIS do claim, também não pode reivindicar.
	terceiro, err := NewContextStore(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if reivindicados := terceiro.ClaimDueSchedulesForOrganization(LocalOrganizationID, agora); len(reivindicados) != 0 {
		t.Fatalf("um processo criado depois do claim não pode reivindicar: %+v", reivindicados)
	}

	// 2. O agendador NÃO pode ser travado pela correção: vencido de novo (o operador rearma), reivindica.
	depois := agora.Add(2 * time.Minute)
	if reivindicados := terceiro.ClaimDueSchedulesForOrganization(LocalOrganizationID, depois); len(reivindicados) != 1 {
		t.Fatalf("agendamento realmente vencido precisa continuar sendo reivindicado: %+v", reivindicados)
	}
}
