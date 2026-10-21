package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// POSITIVO: instancia a aguardar leitura -> devolve o QR como data-URL e envia a api key no header.
func TestEvolutionConnectDevolveQR(t *testing.T) {
	var viuAPIKey bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") == "chave-secreta" {
			viuAPIKey = true
		}
		switch {
		case strings.Contains(r.URL.Path, "/instance/connectionState/"):
			w.Write([]byte(`{"instance":{"state":"connecting"}}`))
		case strings.Contains(r.URL.Path, "/instance/connect/"):
			w.Write([]byte(`{"base64":"iVBORw0KGgo=","pairingCode":"ABCD-1234"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	ad := NewEvolutionAdapter(EvolutionConfig{
		BaseURL: srv.URL, APIKey: "chave-secreta", Instance: "hades",
	})
	estado, err := ad.Connect(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if !viuAPIKey {
		t.Fatal("a api key devia viajar no cabecalho apikey")
	}
	if !strings.HasPrefix(estado.QR, "data:image/png;base64,") {
		t.Fatalf("QR devia vir como data-URL, veio %q", estado.QR)
	}
	if estado.PairingCode != "ABCD-1234" {
		t.Fatalf("pairing code perdido: %q", estado.PairingCode)
	}
	if strings.Contains(estado.QR, "chave-secreta") {
		t.Fatal("a api key nao pode aparecer na resposta")
	}
}

// POSITIVO: instancia ja' ligada -> state=open e SEM QR (nao pede leitura).
func TestEvolutionConnectJaLigada(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"instance":{"state":"open"}}`))
	}))
	defer srv.Close()

	ad := NewEvolutionAdapter(EvolutionConfig{BaseURL: srv.URL, APIKey: "k", Instance: "hades"})
	estado, err := ad.Connect(context.Background())
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if estado.State != "open" {
		t.Fatalf("esperava state=open, veio %q", estado.State)
	}
	if estado.QR != "" {
		t.Fatalf("nao devia haver QR quando ja' esta ligada, veio %q", estado.QR)
	}
}

// NEGATIVO: instancia responde 500 -> erro, e nenhum QR falso.
func TestEvolutionConnectInstanciaComErro(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ad := NewEvolutionAdapter(EvolutionConfig{BaseURL: srv.URL, APIKey: "k", Instance: "hades"})
	estado, err := ad.Connect(context.Background())
	if err == nil {
		t.Fatal("DEFEITO: erro 500 da instancia devolveu sucesso")
	}
	if estado.QR != "" {
		t.Fatal("DEFEITO: devolveu QR quando a instancia falhou")
	}
}

// NEGATIVO: sem credenciais -> erro explicito, sem tentar a rede.
func TestEvolutionConnectSemCredenciais(t *testing.T) {
	ad := NewEvolutionAdapter(EvolutionConfig{})
	if _, err := ad.Connect(context.Background()); err == nil {
		t.Fatal("DEFEITO: sem credenciais devolveu sucesso")
	}
}

// NEGATIVO: instancia responde 200 mas sem QR -> ErrEvolutionQRUnavailable (nao um quadrado vazio).
func TestEvolutionConnectSemQRNaResposta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/instance/connect/") {
			w.Write([]byte(`{"code":"2@abc"}`))
			return
		}
		w.Write([]byte(`{"instance":{"state":"connecting"}}`))
	}))
	defer srv.Close()

	ad := NewEvolutionAdapter(EvolutionConfig{BaseURL: srv.URL, APIKey: "k", Instance: "hades"})
	_, err := ad.Connect(context.Background())
	if !errors.Is(err, ErrEvolutionQRUnavailable) {
		t.Fatalf("esperava ErrEvolutionQRUnavailable, veio %v", err)
	}
}
