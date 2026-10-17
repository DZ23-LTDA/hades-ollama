package agent

import (
	"errors"
	"testing"
)

// Complemento 48 / FASE 2 — PRESENCA COM ORIGEM (item 4 da matriz, que estava `ADAPTER_ONLY`).
//
// O DEFEITO MEDIDO (2026-10-15): `Presence.Status` era **texto livre sem origem** ⇒ *um status forjado era
// indistinguivel de um real*. Estes testes fixam o comportamento NOVO: toda declaracao **tem autor**, a
// ausencia de autor e' **RECUSADA**, e a origem **persiste**. E fixam tambem o LIMITE: `SetPresence` e'
// AUTO-DECLARACAO, e este store **atribui**, nao **autentica** - quem autentica e' a sessao/credencial.

func TestPresencaAutoDeclaradaRegistraAOrigemEPersiste(t *testing.T) {
	raiz := t.TempDir()
	store, err := NewCollaborationStore(raiz)
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	presenca, err := store.SetPresence("proj-1", "agente:pesquisador", "ocupado")
	if err != nil {
		t.Fatalf("presenca: %v", err)
	}
	if presenca.Origem != "agente:pesquisador" {
		t.Fatalf("a auto-declaracao tem de gravar a origem do proprio declarante, veio %q", presenca.Origem)
	}
	// PERSISTENCIA: a origem tem de sobreviver a' releitura (sem isto ela seria so' um campo em memoria).
	reaberto, err := NewCollaborationStore(raiz)
	if err != nil {
		t.Fatalf("reabrir store: %v", err)
	}
	lista := reaberto.Presence("proj-1")
	if len(lista) != 1 || lista[0].Origem != "agente:pesquisador" {
		t.Fatalf("a origem NAO sobreviveu a' releitura: %+v", lista)
	}
}

func TestPresencaSemOrigemERecusadaSemGravarNada(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	_, err = store.SetPresenceWithOrigin("proj-1", "agente:pesquisador", "online", "   ")
	if !errors.Is(err, ErrPresencaSemOrigem) {
		t.Fatalf("presenca sem origem tem de ser RECUSADA, veio: %v", err)
	}
	// RECUSAR DEPOIS DE GRAVAR SERIA PIOR DO QUE NAO GRAVAR: o store tem de ficar intacto.
	if lista := store.Presence("proj-1"); len(lista) != 0 {
		t.Fatalf("a recusa nao pode gravar nada, mas ficaram %d presenca(s)", len(lista))
	}
}

func TestPresencaComOrigemDeOutroAtorFicaAtribuida(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	// Um observador (por exemplo o supervisor da empresa) declara a presenca de outro membro do time.
	presenca, err := store.SetPresenceWithOrigin("proj-1", "agente:pesquisador", "ausente", "supervisor:empresa")
	if err != nil {
		t.Fatalf("presenca declarada por terceiro: %v", err)
	}
	if presenca.UserID != "agente:pesquisador" || presenca.Origem != "supervisor:empresa" {
		t.Fatalf("sujeito e origem sao coisas DIFERENTES e tem de ficar separadas: %+v", presenca)
	}
	// LIMITE DECLARADO (fixado de proposito): o store ATRIBUI, nao AUTENTICA - um ator que declare a origem
	// de outro e' gravado como declarou. *Quando existir autenticacao de sessao, este teste muda* - e o
	// comentario existe para que ninguem confunda atribuicao com prova.
	mentiroso, err := store.SetPresenceWithOrigin("proj-1", "agente:pesquisador", "online", "supervisor:empresa")
	if err != nil {
		t.Fatalf("o store nao autentica a origem (e isso esta' declarado): %v", err)
	}
	if mentiroso.Origem != "supervisor:empresa" {
		t.Fatalf("a origem gravada e' a DECLARADA, nao uma inventada pelo store: %q", mentiroso.Origem)
	}
}

func TestPresencaSemSujeitoContinuaRecusada(t *testing.T) {
	store, err := NewCollaborationStore(t.TempDir())
	if err != nil {
		t.Fatalf("criar store: %v", err)
	}
	if _, err := store.SetPresence("proj-1", "  ", "online"); err == nil {
		t.Fatal("a checagem antiga (projeto e usuario obrigatorios) nao pode ter sido perdida na delegacao")
	}
}
