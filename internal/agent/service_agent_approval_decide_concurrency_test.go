package agent

import (
	"sync"
	"testing"
)

// E-448 (MÉDIO-2) — **A TRAVA DO `Decide`, E O DANO QUE AGORA SEI MEDIR.**
//
// **POR QUE ESTE TESTE NAO EXISTIA**: eu escrevi a trava do `Decide` em **E-426** e **nao** escrevi teste, porque nao
// havia medido **qual dano a raca produz** - *e um teste sem dano medido mede nada*. O proprio codigo declara o dano
// (`service_agent_approvals.go:557-560`):
//
//	"duas decisoes simultaneas que leem a lista antes da gravacao fazem uma PERDER, e o quorum fica SUB-CONTADO,
//	 que FALHA FECHADO — mas e' a mesma classe, e uma delas ja' custou um CRITICO."
//
// ⇒ ***O dano observavel e': N decisoes concorrentes tem de produzir EXATAMENTE N decisoes registradas. Decisao perdida
// e' quorum sub-contado, e quorum sub-contado recusa uma aprovacao legitima.*** *Isso e' mensuravel - e medido, o teste
// deixa de ser decorativo.*
//
// **O QUE ESTE TESTE NAO PROVA**: nao prova que a trava e' a UNICA coisa que impede a perda ✗ (*se `loadFile`/`saveFile`
// passassem a travar por dentro, este teste continuaria verde - e e' por isso que a contraprova por mutacao, no registo
// E-448, importa*). *Verde aqui e' condicao necessaria, nao suficiente.*
func TestDecisoesConcorrentesNaoPerdemRegistro(t *testing.T) {
	const quantas = 8
	raiz := t.TempDir()
	store := NewServiceAgentApprovalStore(raiz)
	agora := instanteDeTeste(t)

	// Oito aprovacoes PENDENTES, cada uma distinta. *Distintas de proposito: decidir a MESMA duas vezes mediria
	// idempotencia, nao a perda de registro.*
	ids := make([]string, 0, quantas)
	for i := 0; i < quantas; i++ {
		pedido := ServiceAgentApproval{
			AgentID:     "agente-concorrente",
			Fingerprint: "impressao-" + string(rune('a'+i)),
			Status:      ServiceAgentApprovalPending,
		}
		salvo, err := store.Request("org-a", pedido)
		if err != nil {
			t.Fatalf("criar a aprovacao %d: %v", i, err)
		}
		ids = append(ids, salvo.ID)
	}

	// Todas decidem ao mesmo tempo. *Se a trava nao estiver no caminho, duas leem a mesma lista e uma gravacao
	// sobrescreve a outra: a decisao perdida nao aparece.*
	var grupo sync.WaitGroup
	erros := make([]error, quantas)
	for i, id := range ids {
		grupo.Add(1)
		go func(indice int, identificador string) {
			defer grupo.Done()
			_, erros[indice] = store.Decide("org-a", identificador, true, "ator-da-sessao", "aprovado no teste", agora)
		}(i, id)
	}
	grupo.Wait()

	for indice, err := range erros {
		if err != nil {
			t.Fatalf("a decisao %d falhou (a trava nao pode REJEITAR decisao legitima): %v", indice, err)
		}
	}

	// **A MEDIDA**: nenhuma das 8 pode ter sido perdida.
	//
	// ⚠️ E-448, ERRO MEU DE SETUP (medido ao rodar): eu usei `FindApproved(org, id, quando)` ✗ — *mas o segundo
	// parâmetro e' a **IMPRESSAO DIGITAL**, nao o ID* (`service_agent_approvals.go:656`, e o corpo compara
	// `approval.Fingerprint == fingerprint`). Com o ID, nunca achava nada.
	//
	// A medida ficou **mais simples e mais forte**: carregar a organizacao e contar as APROVADAS. *Nao depende de eu
	// acertar qual campo e' a chave de busca - e e' o mesmo dado que o quorum le.*
	registros, err := store.Load("org-a")
	if err != nil {
		t.Fatalf("carregar os registros: %v", err)
	}
	aprovadas := 0
	for _, registro := range registros {
		if registro.Status == ServiceAgentApprovalApproved {
			aprovadas++
		}
	}
	if aprovadas != quantas {
		t.Fatalf("ARMADILHA: %d de %d decisoes sobreviveram — *decisao PERDIDA e' quorum SUB-CONTADO, e quorum "+
			"sub-contado recusa aprovacao legitima. Este e' o dano que a trava do `Decide` (E-426) impede, e que o "+
			"proprio codigo declara em service_agent_approvals.go:557-560*", aprovadas, quantas)
	}

	// E o controle do CONTROLE: *a organizacao que NAO decidiu nada tem de ter ZERO aprovadas - senao a medida acima
	// estaria contando registro de outro tenant.*
	daVizinha, err := store.Load("org-b")
	if err != nil {
		t.Fatalf("carregar os registros da vizinha: %v", err)
	}
	for _, registro := range daVizinha {
		if registro.Status == ServiceAgentApprovalApproved {
			t.Fatalf("a organizacao org-b tem aprovacao propria (%q) — *a medida esta' medindo o tenant errado*",
				registro.ID)
		}
	}
}
