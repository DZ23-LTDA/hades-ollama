package server

// Testes de `credentials.status` (E-1270).
//
// O que estes testes travam:
//
//  1. sem `HADES_KIT_CREDENCIAIS` a rota responde 200 com `configurado: false` -- nao 404, nao 500;
//  2. com um kit real a rota devolve o inventario medido;
//  3. **a rota NAO devolve a raiz nem caminhos absolutos** -- o mapa do cofre nao sai pela rota;
//  4. **os segredos continuam nao lidos** (`conteudo_lido: false` para `.p12`/`.jks`);
//  5. o `aviso_de_fronteira` diz o que a rota nao faz;
//  6. a rota **nao aceita caminho pelo pedido** -- um `?raiz=` e' ignorado;
//  7. caminho configurado inexistente devolve 400 com motivo.

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// montarRotaDeCredenciais monta o router minimo com a rota.
func montarRotaDeCredenciais(t *testing.T, raizDoKit string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDoKitDeCredenciais, raizDoKit)
	router := gin.New()
	grupo := router.Group("/api/agent/v1")
	grupo.GET("/credentials/status", (&agentAPI{}).inventariarCredenciais)
	return router
}

// pedirCredenciais faz o GET e decodifica.
func pedirCredenciais(t *testing.T, router *gin.Engine, query string) (int, respostaDeCredenciais) {
	t.Helper()
	caminho := "/api/agent/v1/credentials/status"
	if query != "" {
		caminho += "?" + query
	}
	pedido := httptest.NewRequest(http.MethodGet, caminho, nil)
	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, pedido)
	var resposta respostaDeCredenciais
	if gravador.Code == http.StatusOK {
		if err := json.Unmarshal(gravador.Body.Bytes(), &resposta); err != nil {
			t.Fatalf("resposta invalida: %v (%s)", err, gravador.Body.String())
		}
	}
	return gravador.Code, resposta
}

// kitDeTeste monta um kit com um certificado publico e um segredo.
func kitDeTeste(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	// Certificado publico REAL (autoassinado, gerado aqui) -- o provider le'.
	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("gerar chave: %v", err)
	}
	modelo := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "HADES", Organization: []string{"DZ23 LTDA"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, modelo, modelo, &chave.PublicKey, chave)
	if err != nil {
		t.Fatalf("criar certificado: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(raiz, "apple"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "apple", "distribution.cer"), der, 0o600); err != nil {
		t.Fatalf("escrever certificado: %v", err)
	}
	// Segredo: NAO pode ser lido.
	if err := os.WriteFile(
		filepath.Join(raiz, "apple", "distribution.p12"),
		[]byte("segredo-que-nao-pode-ser-lido"), 0o600,
	); err != nil {
		t.Fatalf("escrever p12: %v", err)
	}
	return raiz
}

// Sem a variavel a rota diz que nao esta' configurada, e nao falha.
func TestCredenciaisSemConfiguracao(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDoKitDeCredenciais, "")
	router := gin.New()
	router.GET("/api/agent/v1/credentials/status", (&agentAPI{}).inventariarCredenciais)

	codigo, resposta := pedirCredenciais(t, router, "")
	if codigo != http.StatusOK {
		t.Fatalf("status = %d; esperado 200 (nao configurado nao e' erro)", codigo)
	}
	if resposta.Configurado {
		t.Fatal("configurado = true sem a variavel de ambiente")
	}
	if resposta.Total != 0 {
		t.Fatalf("total = %d sem kit", resposta.Total)
	}
	if !strings.Contains(resposta.Aviso, variavelDoKitDeCredenciais) {
		t.Fatalf("aviso nao diz qual variavel definir: %q", resposta.Aviso)
	}
}

func TestCredenciaisInventariaKit(t *testing.T) {
	router := montarRotaDeCredenciais(t, kitDeTeste(t))
	codigo, resposta := pedirCredenciais(t, router, "")
	if codigo != http.StatusOK {
		t.Fatalf("status = %d", codigo)
	}
	if !resposta.Configurado {
		t.Fatal("configurado = false com kit valido")
	}
	if resposta.Total != 2 {
		t.Fatalf("total = %d; esperado 2 (um .cer e um .p12)", resposta.Total)
	}
	if resposta.Certificados != 1 {
		t.Fatalf("certificados = %d; esperado 1", resposta.Certificados)
	}
}

// A ROTA NAO DEVOLVE O MAPA DO COFRE.
func TestCredenciaisNaoDevolveCaminhoAbsoluto(t *testing.T) {
	raiz := kitDeTeste(t)
	router := montarRotaDeCredenciais(t, raiz)
	codigo, resposta := pedirCredenciais(t, router, "")
	if codigo != http.StatusOK {
		t.Fatalf("status = %d", codigo)
	}
	for _, credencial := range resposta.Credenciais {
		if filepath.IsAbs(credencial.CaminhoRelativo) {
			t.Fatalf("caminho absoluto devolvido: %q", credencial.CaminhoRelativo)
		}
		if strings.Contains(credencial.CaminhoRelativo, raiz) {
			t.Fatalf("a raiz do kit vazou no caminho: %q", credencial.CaminhoRelativo)
		}
	}
	// E o JSON bruto nao pode conter a raiz em lugar nenhum.
	bruto, err := json.Marshal(resposta)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(bruto), raiz) {
		t.Fatal("a raiz do kit aparece na resposta")
	}
}

// Os segredos continuam nao lidos.
func TestCredenciaisNaoLeSegredos(t *testing.T) {
	router := montarRotaDeCredenciais(t, kitDeTeste(t))
	_, resposta := pedirCredenciais(t, router, "")
	var encontrouP12 bool
	for _, credencial := range resposta.Credenciais {
		if !strings.HasSuffix(credencial.CaminhoRelativo, ".p12") {
			continue
		}
		encontrouP12 = true
		if credencial.ConteudoLido {
			t.Fatal("conteudo_lido = true para um .p12")
		}
		if credencial.Fingerprint != "" {
			t.Fatal("fingerprint preenchido para um .p12: o ficheiro foi lido")
		}
		if !strings.Contains(credencial.Nota, "NAO lido") {
			t.Fatalf("nota do p12 nao declara que nao foi lido: %q", credencial.Nota)
		}
	}
	if !encontrouP12 {
		t.Fatal("o .p12 nao aparece no inventario")
	}
}

func TestCredenciaisAvisoDeFronteira(t *testing.T) {
	router := montarRotaDeCredenciais(t, kitDeTeste(t))
	_, resposta := pedirCredenciais(t, router, "")
	if !strings.Contains(resposta.AvisoDeFronteira, "NAO sao abertos") {
		t.Fatalf("aviso de fronteira nao declara que os segredos nao sao abertos: %q",
			resposta.AvisoDeFronteira)
	}
	if !strings.Contains(resposta.AvisoDeFronteira, "nunca do pedido") {
		t.Fatalf("aviso de fronteira nao declara que o caminho vem do ambiente: %q",
			resposta.AvisoDeFronteira)
	}
	if !resposta.SomenteLeitura {
		t.Fatal("somente_leitura = false")
	}
}

// A rota IGNORA um caminho vindo do pedido.
func TestCredenciaisIgnoraCaminhoDoPedido(t *testing.T) {
	raizDoKit := kitDeTeste(t)
	router := montarRotaDeCredenciais(t, raizDoKit)
	// Um `?raiz=` para outro sitio tem de ser ignorado: o total continua a ser o do kit.
	_, resposta := pedirCredenciais(t, router, "raiz="+t.TempDir())
	if resposta.Total != 2 {
		t.Fatalf("total = %d; o `?raiz=` devia ser ignorado e o kit tem 2 ficheiros", resposta.Total)
	}
}

func TestCredenciaisCaminhoInexistente(t *testing.T) {
	router := montarRotaDeCredenciais(t, filepath.Join(t.TempDir(), "nao-existe"))
	codigo, _ := pedirCredenciais(t, router, "")
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400 para um kit inexistente", codigo)
	}
}

// garantir que o tipo do provider continua a ser o unico usado.
var _ agent.ProviderDeCredenciais
