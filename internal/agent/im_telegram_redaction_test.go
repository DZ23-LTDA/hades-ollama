package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// §11.4/§11.7 — O TOKEN DO BOT NÃO PODE VAZAR EM ERRO, LOG OU AUDITORIA (E-296).
//
// O token do Telegram vive NO CAMINHO DA URL (`/bot<TOKEN>/sendMessage`), então qualquer erro que carregue a URL
// carrega a credencial. A implementação já tenta redigir, mas a redação só troca campos que contêm `/bot`:
// um erro que ecoe o **token puro** (por exemplo, uma resposta do provedor) passaria intacto.
//
// Este teste afirma a propriedade em DUAS direções e NÃO tolera nenhuma delas:
//  1. URL completa no erro ⇒ redigida;
//  2. TOKEN PURO no erro (sem `/bot`) ⇒ também redigido;
//  3. texto legítimo NÃO é alterado (sem falso positivo na redação);
//  4. ponta a ponta: erro devolvido pela chamada real não contém o token.
const tokenDeTeste = "123456789:AAFakeTokenForRedactionTest_0123456789"

func TestTokenDoBotNaoVazaEmErroNemEmAuditoria(t *testing.T) {
	// 1 + 2. Redação nos dois casos: URL com /bot e token puro.
	casos := map[string]string{
		"url completa":      fmt.Sprintf("Post \"https://api.telegram.org/bot%s/sendMessage\": dial tcp: timeout", tokenDeTeste),
		"token puro":        fmt.Sprintf("provider rejected credential %s", tokenDeTeste),
		"só o id e segredo": fmt.Sprintf("token=%s inválido", tokenDeTeste),
	}
	for nome, entrada := range casos {
		saida := redactTelegramSecret(entrada, tokenDeTeste)
		if strings.Contains(saida, tokenDeTeste) {
			t.Fatalf("[%s] o token VAZOU na redação: %q", nome, saida)
		}
	}

	// 3. Sem falso positivo: texto legítimo passa intacto.
	legitimo := "canal de IM aprovado; dlp_findings=0"
	if saida := redactTelegramSecret(legitimo, tokenDeTeste); saida != legitimo {
		t.Fatalf("texto legítimo foi alterado pela redação: %q", saida)
	}

	// 4. Auditoria: o destino registrado nunca carrega o token.
	base, err := url.Parse(telegramAPIBaseURL)
	if err != nil {
		t.Fatal(err)
	}
	canal := newTelegramChannel()
	destino := canal.auditDestination(base, "sendMessage")
	if strings.Contains(destino, tokenDeTeste) || !strings.Contains(destino, "redigido") {
		t.Fatalf("destino de auditoria sem redação explícita: %q", destino)
	}

	// 4b. Ponta a ponta: servidor devolve erro ECOANDO o token puro; a chamada real não pode propagá-lo.
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		// Ecoa o caminho (que contém o token) E o token puro: os dois caminhos de vazamento de uma vez.
		fmt.Fprintf(w, `{"ok":false,"error_code":401,"description":"unauthorized for %s in %s"}`, tokenDeTeste, r.URL.Path)
	}))
	defer servidor.Close()

	canalHTTP := newTelegramChannel()
	canalHTTP.baseURL = servidor.URL
	canalHTTP.credential = func(string) string { return tokenDeTeste }
	_, err = canalHTTP.call(context.Background(), tokenDeTeste, "sendMessage", map[string]any{"chat_id": "1", "text": "oi"})
	if err == nil {
		t.Fatal("resposta 401 precisa virar erro")
	}
	if !errors.Is(err, ErrTelegramCallFailed) {
		t.Fatalf("erro inesperado: %v", err)
	}
	if strings.Contains(err.Error(), tokenDeTeste) {
		t.Fatalf("o TOKEN VAZOU no erro devolvido pela chamada: %v", err)
	}
}
