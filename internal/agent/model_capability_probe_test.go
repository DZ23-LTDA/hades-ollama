package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// SONDA DE CAPACIDADES — TESTES (E-326).
//
// Duas coisas são medidas aqui, além do caminho feliz: **a sonda é somente leitura** (o servidor de teste registra
// TODAS as rotas chamadas — se aparecer `/api/pull`, `/api/create` ou `/api/delete`, o teste falha) e **capacidade não
// declarada NÃO é registrada**.

func TestSondaLeCapacidadesDeclaradasERegistra(t *testing.T) {
	var mu sync.Mutex
	var rotas []string
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		rotas = append(rotas, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"capabilities":["completion","tools","vision"],` +
				`"model_info":{"llama.context_length":8192},"details":{"family":"llama"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer servidor.Close()

	registro := DefaultModelCapabilityRegistry()
	sonda := ModelCapabilityProbe{BaseURL: servidor.URL}
	capacidades, err := sonda.ProbeAndRegister(context.Background(), registro, "llama3.1:8b-probe")
	if err != nil {
		t.Fatalf("sonda: %v", err)
	}
	if !capacidades.ToolCalling || !capacidades.Vision || capacidades.ContextTokens != 8192 {
		t.Fatalf("capacidades declaradas não foram lidas: %+v", capacidades)
	}
	if capacidades.Source != CapabilitySourceMeasured {
		t.Fatalf("capacidade obtida por sonda precisa ter origem declarada: %+v", capacidades)
	}
	if !strings.Contains(capacidades.Note, "/api/show") {
		t.Fatalf("a nota precisa dizer DE ONDE veio o dado: %q", capacidades.Note)
	}
	if modo, _ := registro.ResolveToolMode("llama3.1:8b-probe"); modo != ToolModeNative {
		t.Fatalf("após a sonda o modo precisa ser nativo: %q", modo)
	}

	// SOMENTE LEITURA: nenhuma rota que crie, puxe ou remova modelo pode ter sido chamada.
	mu.Lock()
	defer mu.Unlock()
	for _, rota := range rotas {
		if !strings.HasSuffix(rota, "/api/show") {
			t.Fatalf("a sonda chamou rota que NÃO é descritiva: %s (rotas: %v)", rota, rotas)
		}
	}
	if len(rotas) != 1 {
		t.Fatalf("a sonda precisa fazer exatamente uma chamada descritiva: %v", rotas)
	}
}

func TestSondaSemDeclaracaoNaoRegistraNemInventa(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Endpoint que responde 200 mas NÃO declara capabilities (payload antigo ou incompleto).
		_, _ = w.Write([]byte(`{"model_info":{"llama.context_length":4096},"details":{"family":"llama"}}`))
	}))
	defer servidor.Close()

	registro := DefaultModelCapabilityRegistry()
	sonda := ModelCapabilityProbe{BaseURL: servidor.URL}
	if _, err := sonda.ProbeAndRegister(context.Background(), registro, "modelo-sem-declaracao"); !errors.Is(err, ErrModelCapabilityUnknown) {
		t.Fatalf("sem declaração a resposta honesta é NÃO MEDIDO: %v", err)
	}
	// ARMADILHA: o registro NÃO pode ter sido alterado — nem para "suporta", nem para "não suporta".
	if _, achou := registro.Lookup("modelo-sem-declaracao"); achou {
		t.Fatal("ARMADILHA: capacidade não declarada foi registrada — não se inventa suporte nem ausência")
	}
	if modo, _ := registro.ResolveToolMode("modelo-sem-declaracao"); modo != ToolModeUnknown {
		t.Fatalf("sem medição o modo precisa continuar UNKNOWN: %q", modo)
	}
}

func TestSondaFalhaFechadoEmErrosEEndpointFora(t *testing.T) {
	// 1. HTTP 500.
	erro500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer erro500.Close()
	registro := DefaultModelCapabilityRegistry()
	sonda := ModelCapabilityProbe{BaseURL: erro500.URL}
	if _, err := sonda.ProbeAndRegister(context.Background(), registro, "modelo-500"); !errors.Is(err, ErrCapabilityProbeUnavailable) {
		t.Fatalf("HTTP 500 precisa falhar fechado: %v", err)
	}
	if _, achou := registro.Lookup("modelo-500"); achou {
		t.Fatal("em falha de endpoint NADA pode ser registrado")
	}

	// 2. Endpoint inexistente (servidor já encerrado).
	fechado := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	urlFechada := fechado.URL
	fechado.Close()
	if _, err := (ModelCapabilityProbe{BaseURL: urlFechada}).Probe(context.Background(), "modelo-off"); !errors.Is(err, ErrCapabilityProbeUnavailable) {
		t.Fatalf("endpoint fora do ar precisa falhar fechado: %v", err)
	}

	// 3. Timeout respeitado.
	lento := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"capabilities":["tools"]}`))
	}))
	defer lento.Close()
	if _, err := (ModelCapabilityProbe{BaseURL: lento.URL, Timeout: 20 * time.Millisecond}).Probe(context.Background(), "modelo-lento"); !errors.Is(err, ErrCapabilityProbeUnavailable) {
		t.Fatalf("timeout precisa falhar fechado: %v", err)
	}

	// 4. Endpoint não configurado e modelo vazio também são erro explícito.
	if _, err := (ModelCapabilityProbe{}).Probe(context.Background(), "x"); !errors.Is(err, ErrCapabilityProbeUnavailable) {
		t.Fatalf("sem endpoint precisa falhar fechado: %v", err)
	}
	if _, err := (ModelCapabilityProbe{BaseURL: "http://127.0.0.1:1"}).Probe(context.Background(), "  "); !errors.Is(err, ErrCapabilityProbeUnavailable) {
		t.Fatalf("modelo vazio precisa falhar fechado: %v", err)
	}
}

func TestSondaSemToolsRebaixaParaModoCompativel(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"capabilities":["completion"]}`))
	}))
	defer servidor.Close()
	registro := DefaultModelCapabilityRegistry()
	sonda := ModelCapabilityProbe{BaseURL: servidor.URL}
	if _, err := sonda.ProbeAndRegister(context.Background(), registro, "modelo-so-texto"); err != nil {
		t.Fatalf("sonda: %v", err)
	}
	modo, aviso := registro.ResolveToolMode("modelo-so-texto")
	if modo != ToolModeCompatibleText {
		t.Fatalf("sem `tools` o modo precisa ser compatível: %q", modo)
	}
	// MEDIDO: eu comparei "não declara" e a mensagem é "Não declara" (caixa). O motor está certo; a asserção estava
	// rígida demais. Assertar por CONTEÚDO, não por caixa, é a lição que já estava nas minhas notas.
	if !strings.Contains(strings.ToLower(aviso), "não declara `tools`") {
		t.Fatalf("o aviso precisa explicar a limitação: %q", aviso)
	}
	if err := registro.RequireNativeToolCalling("modelo-so-texto"); !errors.Is(err, ErrModelWithoutNativeToolCalling) {
		t.Fatalf("pedido nativo precisa ser recusado: %v", err)
	}
}
