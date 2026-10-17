package server

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// ESTAGIO 47 — a superficie de versoes: as duas verdades que a rota tem de respeitar.
//
// *Estes testes existem porque a correcao do mapa de erros (round 115) estava `IMPLEMENTED`
// e sem prova, e porque `artefactoDaMissao` e' o que impede a rota de servir um artefacto
// que nao pertence a' missao do chamador.*

func TestStatusForAgentErrorMapeiaErrosDeVersao(t *testing.T) {
	if got := statusForAgentError(agent.ErrVersaoInexistente); got != http.StatusNotFound {
		t.Fatalf("versao inexistente devia ser 404, veio %d - *um 500 aqui esconde do cliente que o recurso nao existe*", got)
	}
	if got := statusForAgentError(agent.ErrRestauroConflitante); got != http.StatusConflict {
		t.Fatalf("conflito de restauro devia ser 409, veio %d - *o CAS falhou, o servidor nao avariou*", got)
	}
}

func TestStatusForAgentErrorVeOErroEmbrulhado(t *testing.T) {
	// O dominio embrulha com contexto (`fmt.Errorf("%w: versao %d de %q", ...)`): se o mapa
	// nao usasse `errors.Is`, o cliente receberia 500 por um erro que E' de versao.
	embrulhado := fmt.Errorf("artefacto %q: %w", "doc.md", agent.ErrVersaoInexistente)
	if got := statusForAgentError(embrulhado); got != http.StatusNotFound {
		t.Fatalf("erro embrulhado devia ser 404 (errors.Is), veio %d", got)
	}
	outro := fmt.Errorf("restauro: %w", agent.ErrRestauroConflitante)
	if got := statusForAgentError(outro); got != http.StatusConflict {
		t.Fatalf("erro embrulhado devia ser 409 (errors.Is), veio %d", got)
	}
}

func TestArtefactoDaMissaoEncontraPeloID(t *testing.T) {
	artefactos := []agent.ArtifactManifest{
		{ID: "art_1", MissionID: "m1", TenantID: "acme", Path: "a.md", Version: 1},
		{ID: "art_2", MissionID: "m1", TenantID: "acme", Path: "b.md", Version: 3},
	}
	achado, ok := artefactoDaMissao(artefactos, "art_2")
	if !ok || achado.ID != "art_2" || achado.Path != "b.md" {
		t.Fatalf("devia encontrar art_2 (b.md), veio %+v ok=%v", achado, ok)
	}
}

func TestArtefactoDaMissaoNaoInventaQuandoAusente(t *testing.T) {
	// *Um artefacto de OUTRA missao nao esta' nesta lista* - e a lista e' a da missao do
	// chamador. Devolver `ok=true` aqui seria servir dados de outra missao.
	_, ok := artefactoDaMissao(nil, "art_de_outra_missao")
	if ok {
		t.Fatal("ARMADILHA: artefacto ausente foi dado como encontrado")
	}
	_, ok = artefactoDaMissao([]agent.ArtifactManifest{{ID: "art_1"}}, "art_9")
	if ok {
		t.Fatal("ARMADILHA: ID inexistente foi dado como encontrado")
	}
}

func TestErroDeVersaoNaoEVermelhoDeServidor(t *testing.T) {
	// Guarda contra a regressao que motivou tudo: se alguem remover o mapeamento, estes
	// dois erros voltam a cair no `default` (500) e o cliente perde a distincao.
	for _, caso := range []struct {
		err  error
		quer int
	}{
		{agent.ErrVersaoInexistente, http.StatusNotFound},
		{agent.ErrRestauroConflitante, http.StatusConflict},
	} {
		if got := statusForAgentError(caso.err); got >= 500 {
			t.Fatalf("%v caiu em %d (5xx): o mapeamento foi perdido", caso.err, got)
		}
		if got := statusForAgentError(caso.err); got != caso.quer {
			t.Fatalf("%v devia ser %d, veio %d", caso.err, caso.quer, got)
		}
	}
	if !errors.Is(agent.ErrVersaoInexistente, agent.ErrVersaoInexistente) {
		t.Fatal("sanidade: errors.Is sobre o proprio erro")
	}
}
