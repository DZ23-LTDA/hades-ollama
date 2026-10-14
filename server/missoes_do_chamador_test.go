package server

import (
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// ISOLAMENTO ENTRE INQUILINOS, PROVADO SEM HARNESS DE SERVIDOR (E-627).
//
// A regra foi separada da ligação HTTP exactamente para isto: um teste de rota exigiria montar um
// `agentAPI` com runtime e organização no contexto, e essa montagem nunca foi escrita — o que
// deixaria a garantia por provar. Aqui a regra é pura e o teste é directo.
//
// O QUE ESTE TESTE IMPEDE: que a rota `/performance` (ou qualquer outra que use a regra) conte
// missões de outro inquilino. *Uma rota que filtra a menos não se nota: continua a responder 200.*

func missoesDeTeste() []agent.Mission {
	return []agent.Mission{
		{ID: "mis_a1", OrganizationID: "org-a"},
		{ID: "mis_b1", OrganizationID: "org-b"},
		{ID: "mis_a2", OrganizationID: "org-a"},
		{ID: "mis_sem_org"},
	}
}

func TestMissoesDaOrganizacaoNaoVazaEntreInquilinos(t *testing.T) {
	daA := missoesDaOrganizacao(missoesDeTeste(), "org-a")
	if len(daA) != 2 {
		t.Fatalf("org-a devia ver 2 missões, viu %d", len(daA))
	}
	for _, mission := range daA {
		if mission.OrganizationID != "org-a" {
			t.Fatalf("*** vazou missão de outro inquilino: %s (%s) ***", mission.ID, mission.OrganizationID)
		}
	}
	// E o inverso: org-b não vê as de org-a.
	daB := missoesDaOrganizacao(missoesDeTeste(), "org-b")
	if len(daB) != 1 || daB[0].ID != "mis_b1" {
		t.Fatalf("org-b devia ver apenas a sua missão: %+v", daB)
	}
	// Uma organização sem missões vê vazio — e não «tudo», que seria o pior desfecho possível.
	if vazio := missoesDaOrganizacao(missoesDeTeste(), "org-c"); len(vazio) != 0 {
		t.Fatalf("organização sem missões devia ver vazio, viu %d", len(vazio))
	}
}

// O COMPORTAMENTO ANTERIOR, FIXADO: sem organização na sessão, a regra devolve tudo. Isto não é
// uma escolha nova — é o que o `missions` já fazia antes da extracção, e fica preso por teste para
// que a refactorização não o mude por acidente.
func TestMissoesDaOrganizacaoSemOrganizacaoDevolveTudo(t *testing.T) {
	todas := missoesDaOrganizacao(missoesDeTeste(), "")
	if len(todas) != 4 {
		t.Fatalf("sem organização a regra devia devolver tudo (4), devolveu %d", len(todas))
	}
}

// NÃO DEVOLVE O SLICE DE ENTRADA: a versão anterior escrevia por cima dele (`missions[:0]`), e
// quem chamasse ficava com o slice original alterado. Aqui prova-se que a entrada fica intacta.
func TestMissoesDaOrganizacaoNaoAlteraOSliceDeEntrada(t *testing.T) {
	entrada := missoesDeTeste()
	_ = missoesDaOrganizacao(entrada, "org-a")
	if len(entrada) != 4 || entrada[0].ID != "mis_a1" || entrada[1].ID != "mis_b1" {
		t.Fatalf("a regra alterou o slice de entrada: %+v", entrada)
	}
}
