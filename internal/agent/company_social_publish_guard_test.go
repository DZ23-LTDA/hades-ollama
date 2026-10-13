package agent

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// ESTÁGIO 36 / §11.7 — PUBLICAÇÃO SOCIAL: APROVAÇÃO ANTES DO EFEITO, EFEITO UMA VEZ SÓ, SEM ENXURRADA (E-302).
//
// O módulo já recusa publicar rascunho NÃO aprovado (`ErrCompanyApprovalRequiredForExternal`) — isso é medido aqui
// também, para travar a regressão. O que NÃO estava travado:
//
//  1. PUBLICAR O MESMO RASCUNHO DUAS VEZES. O §11.7 exige que "o mesmo efeito externo não ocorra duas vezes":
//     republicar é postar de novo na rede social. Se o segundo Publish for aceito, o Hades duplica publicação;
//  2. ENXURRADA (spam): publicar muitos rascunhos no mesmo intervalo, sem nenhum limite, é o risco "spam" que o
//     estágio 36 nomeia. Um teto local com motivo explícito é melhor que nenhum teto.
//
// LIMITE DECLARADO: o teto é POLÍTICA LOCAL do Hades — não é o rate limit da API da rede social (que só se mede com
// conta real, e continua BLOCKED_BY_EXTERNAL_DEPENDENCY). Não se finge que é o mesmo.
func TestPublicacaoSocialNaoDuplicaEfeitoNemDisparaEnxurrada(t *testing.T) {
	store, empresa, agora := socialPublishStore(t)

	// Trava a regra que JÁ existia: sem aprovação, não publica.
	rascunho := criarRascunhoSocial(t, store, empresa, "promoção de primavera")
	primeiroID := rascunho.SocialDrafts[0].ID
	if _, err := store.PublishSocialDraft(empresa, primeiroID); !errors.Is(err, ErrCompanyApprovalRequiredForExternal) {
		t.Fatalf("publicar SEM aprovação precisa ser recusado com ErrCompanyApprovalRequiredForExternal, veio: %v", err)
	}

	// Aprova e publica: o caminho felizanda uma vez.
	if _, err := store.ApproveSocialDraft(empresa, primeiroID); err != nil {
		t.Fatal(err)
	}
	publicada, err := store.PublishSocialDraft(empresa, primeiroID)
	if err != nil {
		t.Fatalf("publicação aprovada precisa funcionar: %v", err)
	}
	estado := publishedDraft(t, publicada, primeiroID)
	if estado.Status != "published_sandbox" || estado.PublishedAt == nil {
		t.Fatalf("publicação precisa registrar status e instante: %+v", estado)
	}

	// 1. PUBLICAR DE NOVO O MESMO RASCUNHO: efeito externo duplicado — tem de ser RECUSADO.
	if _, err := store.PublishSocialDraft(empresa, primeiroID); err == nil {
		t.Fatal("republicar o MESMO rascunho duplicaria o efeito externo e precisa ser recusado")
	}

	// 2. ENXURRADA: muitos rascunhos aprovados publicados em sequência.
	//    Alguns passam (é uso legítimo); a partir do teto, a recusa precisa ser EXPLÍCITA, não silenciosa.
	limite := CompanySocialPublishLimitPerWindow
	publicados, recusados := 1, 0
	var ultimaRecusa error
	for i := 0; i < limite+3; i++ {
		outro := criarRascunhoSocial(t, store, empresa, "conteúdo número "+strings.Repeat("x", 1))
		id := outro.SocialDrafts[len(outro.SocialDrafts)-1].ID
		if _, err := store.ApproveSocialDraft(empresa, id); err != nil {
			t.Fatal(err)
		}
		if _, err := store.PublishSocialDraft(empresa, id); err != nil {
			recusados++
			ultimaRecusa = err
			continue
		}
		publicados++
	}
	if recusados == 0 {
		t.Fatalf("publicar %d rascunhos em sequência precisa encontrar o teto de %d por janela (spam)", limite+3, limite)
	}
	if ultimaRecusa == nil || !errors.Is(ultimaRecusa, ErrCompanySocialPublishThrottled) {
		t.Fatalf("a recusa por enxurrada precisa ser EXPLÍCITA (ErrCompanySocialPublishThrottled), veio: %v", ultimaRecusa)
	}
	if publicados != limite {
		t.Fatalf("dentro da janela, o teto precisa ser exatamente %d publicações (mediu %d)", limite, publicados)
	}

	// A recusa por enxurrada NÃO pode ter corrompido o estado: o rascunho barrado continua aprovado e NÃO publicado.
	lista, err := store.Get(empresa)
	if err != nil {
		t.Fatal(err)
	}
	naoPublicados := 0
	for _, d := range lista.SocialDrafts {
		if d.PublishedAt == nil {
			naoPublicados++
		}
	}
	if naoPublicados == 0 {
		t.Fatal("o rascunho barrado pelo teto precisa permanecer aprovado e NÃO publicado (recusa, não perda)")
	}
	_ = agora
}

// socialPublishStore cria o store de empresa isolado para o teste.
func socialPublishStore(t *testing.T) (*CompanyStore, string, time.Time) {
	t.Helper()
	store, err := NewCompanyStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	criada, err := store.Create(Company{ID: "emp-social", OrganizationID: "org-a", Name: "Loja Social", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	// O store decide o ID: usar o ID DEVOLVIDO, não o que eu supus.
	return store, criada.ID, time.Now().UTC()
}

// criarRascunhoSocial cria UM rascunho e devolve o estado resultante.
func criarRascunhoSocial(t *testing.T, store *CompanyStore, empresa, titulo string) Company {
	t.Helper()
	atualizado, err := store.CreateSocialDraft(empresa, CompanySocialDraft{
		Provider: "instagram",
		Title:    titulo,
		Body:     "corpo do conteúdo aprovado",
	})
	if err != nil {
		t.Fatalf("criar rascunho: %v", err)
	}
	return atualizado
}

// publishedDraft devolve o rascunho pelo ID.
func publishedDraft(t *testing.T, company Company, id string) CompanySocialDraft {
	t.Helper()
	for _, d := range company.SocialDrafts {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("rascunho %s não encontrado", id)
	return CompanySocialDraft{}
}
