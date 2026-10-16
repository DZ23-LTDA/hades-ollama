package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// CATRACA DO E-560 / E-561.
//
// *POR QUE ISTO EXISTE:* ate' 2026-10-15 o `POST /api/v1/settings` deste servidor
// respondia **200 ecoando o corpo** e **nao gravava nada**, e o `GET` seguinte
// devolvia um objeto hard-coded (`"SelectedModel": ""`). A UI concluia que tinha
// gravado, `selectedModel` ficava nulo e o envio abortava com "No model selected"
// => **nao havia chat nenhum contra este servidor**, e a falha era **silenciosa**.
//
// Nada no CI podia apanhar isso, e a razao e' subtil: a suite E2E corre contra
// `app/ui/app/e2e/first-run-backend.mjs`, que **persiste de verdade** (`:67` estado
// mutavel, `:125-131` merge no POST, `:143-153` devolve o estado gravado). Ou seja,
// *um fixture mais capaz que o servidor real esconde exactamente o defeito do
// servidor real* - o verde do E2E era silencioso quanto a este P0.
//
// Esta catraca fecha essa lacuna. Ela **nao** exige persistencia (que neste host
// esta' bloqueada por CGO: `app/store` usa `github.com/mattn/go-sqlite3`, o
// `CGO_ENABLED` e' 0 e o `gcc` esta' ausente - medido no E-561). Ela exige
// **HONESTIDADE**: enquanto nao houver persistencia, o POST tem de dizer que nao
// atende, em vez de devolver um sucesso que nao aconteceu.
//
// *** SE ESTA CATRACA CAIR, LEIA ANTES DE "CORRIGIR" O TESTE ***
// Cair pode significar uma de duas coisas, e so' uma delas e' boa:
//
//	(a) BOM: alguem ligou a persistencia de verdade. Entao o POST deve passar a
//	    200/204 **e** o `GET` seguinte deve devolver `"SelectedModel"` com o valor
//	    que foi enviado. Nesse caso: **apagar esta catraca**, registar a evidencia
//	    nova e correr o teste de aceitacao do `E-560` (o round-trip POST -> GET).
//	(b) MAU: alguem voltou a fazer o POST ecoar o corpo com 200 sem gravar. Nesse
//	    caso o teste esta' certo e o codigo esta' errado: **nao relaxar a catraca**.
//
// Um teste que aceitasse as duas coisas nao seria uma catraca.
func TestDesktopLocalSettingsNaoFingePersistencia(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// `agentAPI` nil e' suficiente: as rotas de settings sao registadas antes do
	// grupo de personalizacao, que devolve cedo quando nao ha middleware.
	s := &Server{}
	r := gin.New()
	s.registerDesktopLocalRoutes(r, nil)

	// (1) O POST tem de ser honesto: 501, e nunca 200 com eco do corpo.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/settings",
		strings.NewReader(`{"SelectedModel":"llama3.2:3b"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("POST /api/v1/settings devolveu %d (corpo=%q). Enquanto nao houver persistencia a resposta "+
			"honesta e' 501; um 200 aqui significa que o servidor esta' a fingir que gravou, que e' o defeito do E-560.",
			rec.Code, strings.TrimSpace(rec.Body.String()))
	}

	// A resposta de erro nao pode trazer um `settings` com ar de estado gravado.
	var resposta map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resposta); err != nil {
		t.Fatalf("o corpo do 501 nao e' JSON valido: %v (corpo=%q)", err, rec.Body.String())
	}
	if v, tem := resposta["settings"]; tem {
		t.Errorf("o 501 trouxe um campo `settings` (%v): um erro nao pode devolver um objeto com ar de estado gravado", v)
	}
	if _, tem := resposta["error"]; !tem {
		t.Errorf("o 501 nao trouxe `error`: a UI so' mostra o motivo se o campo existir (corpo=%q)", rec.Body.String())
	}

	// (2) O GET nao pode devolver o modelo que acabou de ser enviado. Se devolvesse,
	//     o POST acima teria gravado - e ele nao gravou.
	recGet := httptest.NewRecorder()
	r.ServeHTTP(recGet, httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	if recGet.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/settings devolveu %d, esperado 200", recGet.Code)
	}

	var corpo struct {
		Settings struct {
			SelectedModel     string `json:"SelectedModel"`
			OnboardingVersion int    `json:"OnboardingVersion"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(recGet.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("o corpo do GET nao e' JSON valido: %v (corpo=%q)", err, recGet.Body.String())
	}
	if corpo.Settings.SelectedModel != "" {
		t.Fatalf("o GET devolveu SelectedModel=%q sem que este servidor persista: o POST passou a gravar de facto, "+
			"logo esta catraca envelheceu - ver o cabecalho deste ficheiro, caso (a).", corpo.Settings.SelectedModel)
	}

	// (3) O onboarding tem de continuar intacto: `OnboardingVersion` e' o campo que
	//     a jornada de primeira execucao usa, e a correcao do E-560 nao o pode partir.
	if corpo.Settings.OnboardingVersion == 0 {
		t.Errorf("`OnboardingVersion` veio 0: o GET perdeu o campo que o gate de onboarding usa")
	}
}
