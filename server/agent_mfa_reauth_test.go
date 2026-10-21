// ACHADO P1 (E-1362): os handlers de MFA nao exigiam reautenticacao.
//
// Antes desta correcao, `disableMFA` e `generateRecoveryCodes` verificavam APENAS `agent.user` e nem corpo
// tinham: quem tivesse um bearer token valido desativava o MFA e gerava codigos novos com um pedido vazio.
//
//	*Um segundo factor que qualquer portador do primeiro pode remover nao e' um segundo factor: e' uma
//	cerimonia.*
//
// Este ficheiro prova as DUAS metades: que o pedido SEM prova e' recusado, e que o pedido COM prova passa.
// Um negativo sem o positivo ao lado prova que algo aconteceu, nao que aconteceu o que se diz.
package server

import (
	"encoding/base32"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// montarAPIComMFA devolve a api, o utilizador e dois codigos de recuperacao validos.
//
// Os codigos sao gerados DIRECTAMENTE na store, antes de a verificacao existir no caminho HTTP: usa-los
// pelo endpoint seria circular -- o endpoint e' exactamente o que esta a ser provado.
func montarAPIComMFA(t *testing.T) (*agentAPI, agent.User, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "p1-mfa-reauth-test-key-long-enough")

	auth, err := agent.NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.CreateUser("p1-mfa@example.com", "P1 MFA")
	if err != nil {
		t.Fatal(err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("0123456789abcdef"))
	if _, err := auth.EnableMFA(user.ID, secret); err != nil {
		t.Fatal(err)
	}
	// O `user` devolvido por CreateUser tem MFAEnabled=false. Quem o actualiza e' o EnableMFA -- e e'
	// esse User que o middleware poria no context. Usar o antigo faria o handler ler o estado errado.
	atualizado, codigos, err := auth.GenerateRecoveryCodes(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !atualizado.MFAEnabled {
		t.Fatal("MFA deveria estar ligado apos EnableMFA")
	}
	user = atualizado
	if len(codigos) < 2 {
		t.Fatalf("esperava >= 2 codigos de recuperacao, obtive %d", len(codigos))
	}
	return &agentAPI{auth: auth, authRequired: true}, user, codigos[0], codigos[1]
}

// pedidoMFA monta o contexto como o middleware de auth o deixaria: `agent.user` preenchido.
func pedidoMFA(user agent.User, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/agent/v1/auth/mfa", nil)
	ctx.Set("agent.user", user)
	ctx.Request.RemoteAddr = "127.0.0.1:4599"
	for chave, valor := range headers {
		ctx.Request.Header.Set(chave, valor)
	}
	return ctx, recorder
}

func TestDisableMFAExigeProvaDoSegundoFactor(t *testing.T) {
	api, user, codigoA, _ := montarAPIComMFA(t)

	casos := []struct {
		nome    string
		headers map[string]string
		quer    int
	}{
		// O DEFEITO: um pedido vazio desativava o MFA.
		{"sem header nenhum", nil, http.StatusForbidden},
		{"codigo totp errado", map[string]string{"X-Ollama-MFA-Code": "000000"}, http.StatusForbidden},
		{"recovery code inventado", map[string]string{"X-Ollama-MFA-Recovery-Code": "AAAA-BBBB"}, http.StatusForbidden},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			ctx, recorder := pedidoMFA(user, caso.headers)
			api.disableMFA(ctx)
			if recorder.Code != caso.quer {
				t.Fatalf("status=%d quer=%d aborted=%v body=%s", recorder.Code, caso.quer, ctx.IsAborted(), recorder.Body.String())
			}
		})
	}

	// Depois de tres recusas o MFA tem de continuar ligado: reler da store verdadeira.
	store := api.auth
	organizacao, _, err := store.CreateOrganization("P1 MFA Org", user)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.IssueToken(user.ID, organizacao.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	recarregado, _, _, err := store.Authenticate(token)
	if err != nil {
		t.Fatal(err)
	}
	if !recarregado.MFAEnabled {
		t.Fatal("as recusas NAO podem ter desligado o MFA")
	}

	// E com a prova certa, desliga.
	ctx, recorder := pedidoMFA(user, map[string]string{"X-Ollama-MFA-Recovery-Code": codigoA})
	api.disableMFA(ctx)
	if recorder.Code != http.StatusOK {
		t.Fatalf("com recovery code valido: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"enabled":false`) {
		t.Fatalf("resposta nao declara enabled=false: %s", recorder.Body.String())
	}
}

func TestGenerateRecoveryCodesExigeMFAEProva(t *testing.T) {
	api, user, _, codigoB := montarAPIComMFA(t)

	// Sem prova: recusado.
	ctx, recorder := pedidoMFA(user, nil)
	api.generateRecoveryCodes(ctx)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("sem prova: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	// Com a prova certa: 201 e codigos.
	ctx, recorder = pedidoMFA(user, map[string]string{"X-Ollama-MFA-Recovery-Code": codigoB})
	api.generateRecoveryCodes(ctx)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("com prova: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "recovery_codes") {
		t.Fatalf("resposta sem recovery_codes: %s", recorder.Body.String())
	}
}

func TestGenerateRecoveryCodesRecusaQuandoMFADesligado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "p1-mfa-disabled-test-key-long-enough")
	auth, err := agent.NewAuthStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	user, err := auth.CreateUser("sem-mfa@example.com", "Sem MFA")
	if err != nil {
		t.Fatal(err)
	}
	api := &agentAPI{auth: auth, authRequired: true}

	// Gerar codigos de um segundo factor DESLIGADO nao produz nada: 400, nao 201 silencioso.
	ctx, recorder := pedidoMFA(user, nil)
	api.generateRecoveryCodes(ctx)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("MFA desligado: status=%d quer=400 body=%s", recorder.Code, recorder.Body.String())
	}
}
