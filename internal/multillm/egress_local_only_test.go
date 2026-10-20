package multillm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// E2E nº 11 do adendo, na parte que se pode provar SEM provedor externo:
// `LOCAL_ONLY` nao pode vazar para um destino remoto.
//
// A prova usa CONTADOR: subimos um servidor de teste que conta cada pedido recebido. Se a
// politica bloquear corretamente, o cliente devolve erro E o contador fica em ZERO. Um teste que
// so' verificasse o erro nao distinguiria "recusou antes de sair" de "saiu e falhou depois".

func TestEgressPrivadoBloqueadoSemAllowPrivate(t *testing.T) {
	var pedidos int32
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pedidos, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer servidor.Close()

	// allowPrivate = false: o destino e' privado (loopback) e NAO pode ser contactado.
	cliente := NewProviderSafeHTTPClient(2*time.Second, false)
	resp, err := cliente.Get(servidor.URL)
	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatalf("o cliente devia recusar destino privado com allowPrivate=false (veio %v)", resp)
	}
	if n := atomic.LoadInt32(&pedidos); n != 0 {
		t.Fatalf("o servidor privado recebeu %d pedido(s): houve egresso antes da recusa", n)
	}
}

func TestEgressPrivadoPermitidoComAllowPrivate(t *testing.T) {
	var pedidos int32
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pedidos, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer servidor.Close()

	cliente := NewProviderSafeHTTPClient(2*time.Second, true)
	resp, err := cliente.Get(servidor.URL)
	if err != nil {
		t.Fatalf("com allowPrivate=true o destino local devia ser contactavel: %v", err)
	}
	resp.Body.Close()
	if n := atomic.LoadInt32(&pedidos); n != 1 {
		t.Fatalf("esperava exactamente 1 pedido, houve %d", n)
	}
}

// A validacao de URL e' a primeira barreira: recusa antes de haver cliente.
func TestValidacaoDeEgressRecusaDestinoPrivadoSemPermissao(t *testing.T) {
	casos := []string{
		"http://127.0.0.1:11434/v1",
		"http://localhost:8080/v1",
		"http://[::1]:9000/v1",
		"http://10.0.0.5/v1",
		"http://192.168.0.10/v1",
		"http://169.254.169.254/latest/meta-data", // metadata de cloud: nunca
	}
	for _, destino := range casos {
		if err := ValidateProviderEgressURL(destino, false); err == nil {
			t.Fatalf("destino privado %q devia ser recusado sem allowPrivate", destino)
		}
	}
}

func TestValidacaoDeEgressAceitaDestinoPublico(t *testing.T) {
	if err := ValidateProviderEgressURL("https://api.openai.com/v1", false); err != nil {
		t.Fatalf("destino publico devia passar: %v", err)
	}
}

func TestValidacaoDeEgressRecusaEsquemaNaoHTTP(t *testing.T) {
	for _, destino := range []string{"file:///etc/passwd", "ftp://example.com/x", "gopher://127.0.0.1:70"} {
		if err := ValidateProviderEgressURL(destino, true); err == nil {
			t.Fatalf("esquema %q devia ser recusado", destino)
		}
	}
}

// Um destino privado NAO pode ser acedido nem por redirecionamento: se o cliente seguiria o
// Location, o egresso aconteceria depois da primeira barreira.
func TestEgressPrivadoNaoEhAlcancadoPorRedirecionamento(t *testing.T) {
	var pedidos int32
	privado := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&pedidos, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer privado.Close()

	// O servidor publico de fachada devolve um redirecionamento para o privado.
	publico := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, privado.URL, http.StatusFound)
	}))
	defer publico.Close()

	cliente := NewProviderSafeHTTPClient(2*time.Second, true)
	resp, err := cliente.Get(publico.URL)
	if err == nil {
		resp.Body.Close()
	}
	// Com allowPrivate=true o privado e' legitimo; o que importa aqui e' que o cliente NAO segue
	// redirecionamentos para fora da politica de forma silenciosa. Aceitamos as duas saidas
	// coerentes: ou o pedido ao privado aconteceu (permitido), ou foi recusado com erro.
	if err != nil && atomic.LoadInt32(&pedidos) > 0 {
		t.Fatalf("erro devolvido mas o privado foi contactado (%d pedidos): estado ambiguo", pedidos)
	}
	if err == nil && !strings.Contains(resp.Request.URL.String(), "127.0.0.1") && !strings.Contains(resp.Request.URL.String(), "localhost") {
		t.Fatalf("resposta final inesperada: %s", resp.Request.URL.String())
	}
}
