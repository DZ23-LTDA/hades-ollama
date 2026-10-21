package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// E-464 — **O TOKEN DO WEBHOOK DO WHATSAPP ERA COMPARADO COM `==`, E O REPOSITÓRIO JÁ SABIA FAZER DIFERENTE.**
//
// **A ASSIMETRIA, MEDIDA**: `whatsapp_adapter.go` comparava o token com `token == c.config.VerifyToken` ✗ — *e o
// `ValidateWebhook` do **mesmo ficheiro** valida a assinatura com **HMAC*** ✓. ***E o repositório já usa
// `subtle.ConstantTimeCompare`/`hmac.Equal` em SEIS outros sítios*** (*auth TOTP, `auth.go:537`, `backup.go`,
// `mcp_remote_h5.go`, `omnichannel.go`, `postgres_key_rotation.go*) ⇒ ***era inconsistência, não desconhecimento.***
//
// **O QUE ESTE TESTE MEDE, e o que ele NÃO pode medir**: *a comparação de strings em Go sai no primeiro byte diferente,
// então o tempo revela quantos bytes o atacante acertou.* ***Medir TEMPO num teste unitário é ruído*** — *o que se mede
// aqui é o **CONTRATO** do helper: quem confere, quem não confere, e que um token vazio nunca autoriza.*
func TestTokenDeVerificacaoDoWhatsApp(t *testing.T) {
	// (1) O contrato do helper, nas três direcções.
	if !verifyTokenConfere("segredo-do-webhook", "segredo-do-webhook") {
		t.Fatal("o token correcto tem de conferir")
	}
	if verifyTokenConfere("segredo-do-webhoo", "segredo-do-webhook") {
		t.Fatal("um prefixo NAO pode conferir — *e e' exactamente isto que a comparacao byte a byte deixava medir*")
	}
	if verifyTokenConfere("outra-coisa", "segredo-do-webhook") {
		t.Fatal("um token diferente NAO pode conferir")
	}
	if verifyTokenConfere("", "segredo-do-webhook") || verifyTokenConfere("segredo-do-webhook", "") {
		t.Fatal("token VAZIO de qualquer lado NAO pode conferir — *um token nao configurado nao pode autorizar nada*")
	}
	// ⚠️ E-464: eu escrevi esta asserção ao CONTRÁRIO, a partir do comportamento ANTIGO (`token != ""` sem aparar). O
	// helper APARA os dois lados ⇒ *`"  segredo  "` **confere**, e isso é uma **melhoria** sobre o código anterior.*
	// ***Asserção escrita a partir do código velho mede o código velho.***
	if !verifyTokenConfere("  segredo-do-webhook  ", "segredo-do-webhook") {
		t.Fatal("espacos nas extremidades devem ser aparados dos DOIS lados — *e o codigo anterior recusava isto*")
	}

	// (2) E o ENDPOINT, não só o helper: *o helper podia estar certo e o chamador não usá-lo.*
	adaptador := &CloudAPIAdapter{config: CloudAPIConfig{
		PhoneNumberID: "phone-1", AccessToken: "token-de-acesso", AppSecret: "segredo",
		VerifyToken: "segredo-do-webhook",
	}}
	pedir := func(mode, token string) ([]byte, bool) {
		requisicao := httptest.NewRequest(http.MethodGet,
			"/webhook?hub.mode="+mode+"&hub.verify_token="+token+"&hub.challenge=desafio-123", nil)
		return adaptador.VerifyWebhook(requisicao)
	}

	if corpo, ok := pedir("subscribe", "segredo-do-webhook"); !ok || string(corpo) != "desafio-123" {
		t.Fatalf("o token correcto tinha de devolver o desafio; veio ok=%v corpo=%q", ok, string(corpo))
	}
	if _, ok := pedir("subscribe", "segredo-do-webhoo"); ok {
		t.Fatal("ARMADILHA: um PREFIXO do token foi aceite pelo endpoint")
	}
	if _, ok := pedir("subscribe", ""); ok {
		t.Fatal("ARMADILHA: token vazio foi aceite pelo endpoint")
	}
	if _, ok := pedir("unsubscribe", "segredo-do-webhook"); ok {
		t.Fatal("ARMADILHA: modo diferente de `subscribe` foi aceite")
	}

	// (3) E o adaptador SEM token configurado: *nao autoriza nada, seja qual for o pedido.*
	semToken := &CloudAPIAdapter{config: CloudAPIConfig{VerifyToken: ""}}
	requisicao := httptest.NewRequest(http.MethodGet,
		"/webhook?hub.mode=subscribe&hub.verify_token=&hub.challenge=desafio", nil)
	if _, ok := semToken.VerifyWebhook(requisicao); ok {
		t.Fatal("ARMADILHA: adaptador com VerifyToken vazio aceitou um pedido")
	}
}
