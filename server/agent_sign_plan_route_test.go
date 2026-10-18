package server

// FASE 28.7: testes da rota `sign.plan` (E-1264).
//
// O que estes testes travam:
//
//  1. um certificado COM EKU serve -> `apta: true`;
//  2. um certificado SEM o EKU e' recusado com o motivo;
//  3. um certificado da Apple NAO serve para Authenticode (o defeito que o ficheiro real expos);
//  4. sem `tipo` e' 400 -- sem ele nao se sabe qual EKU exigir;
//  5. uma identidade sem certificado e' 400;
//  6. ficheiro ilegivel e' 400 com o motivo, nao 500;
//  7. `aviso_de_fronteira` diz que NAO houve assinatura e que a cadeia nao foi validada;
//  8. `alvos_suportados` declara exactamente os dois formatos X.509.

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func routerDePlanoDeAssinatura(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &agentAPI{}
	router.POST("/api/agent/v1/sign/plan", api.planearAssinatura)
	return router
}

// certificadoPEMParaRota gera um certificado emitido por uma CA fictícia com o nome dado e devolve o
// caminho do PEM.
//
// O emissor tem de ir no `parent` de `x509.CreateCertificate`: o campo `Issuer` do `modelo` e'
// IGNORADO, e reescreve-lo depois em memoria nao muda o DER que fica no ficheiro.
//
//	*Um teste que reescreve o campo em memoria e verifica o ficheiro no disco testa a memoria.*
func certificadoPEMParaRota(t *testing.T, emissorCN string, ekus []x509.ExtKeyUsage) string {
	t.Helper()
	agora := time.Now()

	chaveCA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("chave CA: %v", err)
	}
	serieCA, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serie CA: %v", err)
	}
	ca := &x509.Certificate{
		SerialNumber:          serieCA,
		Subject:               pkix.Name{CommonName: emissorCN, Organization: []string{"CA de Teste"}},
		NotBefore:             agora.Add(-48 * time.Hour),
		NotAfter:              agora.Add(48 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	chave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("chave: %v", err)
	}
	serie, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatalf("serie: %v", err)
	}
	folha := &x509.Certificate{
		SerialNumber:          serie,
		Subject:               pkix.Name{CommonName: "HADES Rota de Teste"},
		NotBefore:             agora.Add(-time.Hour),
		NotAfter:              agora.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           ekus,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, folha, ca, &chave.PublicKey, chaveCA)
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	caminho := filepath.Join(t.TempDir(), "certificado.pem")
	bloco := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(caminho, bloco, 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	return caminho
}

func postarPlanoDeAssinatura(t *testing.T, router *gin.Engine, corpo any) (int, map[string]any) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sign/plan", bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)
	var descodificado map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &descodificado); err != nil {
		t.Fatalf("resposta nao e' JSON (%d): %s", resposta.Code, resposta.Body.String())
	}
	return resposta.Code, descodificado
}

func TestPlanoDeAssinaturaAprovaComEKU(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	caminho := certificadoPEMParaRota(t, "DigiCert Code Signing CA",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning})

	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo": "AUTHENTICODE",
		"identidade": map[string]any{
			"caminho_do_certificado": caminho,
			"rotulo":                 "certificado de teste",
			"ambiente":               "desenvolvimento",
		},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200 (%v)", codigo, corpo)
	}
	avaliacao, _ := corpo["avaliacao"].(map[string]any)
	if apta, _ := avaliacao["apta"].(bool); !apta {
		t.Fatalf("apta = false com EKU de Code Signing (%v)", avaliacao)
	}
	// A fronteira dita no CORPO.
	if so, _ := corpo["somente_leitura"].(bool); !so {
		t.Fatal("somente_leitura tem de ser true")
	}
	if nada, _ := corpo["nada_assinado"].(bool); !nada {
		t.Fatal("nada_assinado tem de ser true")
	}
	aviso, _ := corpo["aviso_de_fronteira"].(string)
	if !strings.Contains(aviso, "nao significa que o artefacto foi assinado") {
		t.Fatalf("aviso nao desmente a assinatura: %q", aviso)
	}
	// E o aviso tem de declarar o que NAO foi validado.
	if !strings.Contains(aviso, "cadeia criptografica e a revogacao NAO sao validadas") {
		t.Fatalf("aviso nao declara a validacao ausente: %q", aviso)
	}
}

func TestPlanoDeAssinaturaRecusaSemEKU(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	caminho := certificadoPEMParaRota(t, "CA de Email",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageEmailProtection})

	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo":       "AUTHENTICODE",
		"identidade": map[string]any{"caminho_do_certificado": caminho},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; a avaliacao correu, o veredicto e' que recusa (%v)", codigo, corpo)
	}
	avaliacao, _ := corpo["avaliacao"].(map[string]any)
	if apta, _ := avaliacao["apta"].(bool); apta {
		t.Fatal("apta = true sem o EKU de Code Signing")
	}
	if aptidao, _ := avaliacao["aptidao"].(string); aptidao != "SEM_EKU_DE_ASSINATURA" {
		t.Fatalf("aptidao = %q; esperado SEM_EKU_DE_ASSINATURA", aptidao)
	}
}

// O certificado da Apple TEM Code Signing -- nao pode servir para Authenticode.
func TestPlanoDeAssinaturaRecusaAppleParaAuthenticode(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	caminho := certificadoPEMParaRota(t,
		"Apple Worldwide Developer Relations Certification Authority",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning})

	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo":       "AUTHENTICODE",
		"identidade": map[string]any{"caminho_do_certificado": caminho},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d (%v)", codigo, corpo)
	}
	avaliacao, _ := corpo["avaliacao"].(map[string]any)
	if apta, _ := avaliacao["apta"].(bool); apta {
		t.Fatal("um certificado Apple declarado apto a assinar Authenticode")
	}
	if aptidao, _ := avaliacao["aptidao"].(string); aptidao != "EMISSOR_INESPERADO" {
		t.Fatalf("aptidao = %q; esperado EMISSOR_INESPERADO", aptidao)
	}
}

func TestPlanoDeAssinaturaRecusaSemTipo(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"identidade": map[string]any{"caminho_do_certificado": "C:\\qualquer.pem"},
	})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
	if erro, _ := corpo["erro"].(string); !strings.Contains(erro, "tipo") {
		t.Fatalf("erro nao menciona `tipo`: %q", erro)
	}
}

func TestPlanoDeAssinaturaRecusaIdentidadeSemCertificado(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo":       "AUTHENTICODE",
		"identidade": map[string]any{"rotulo": "A1 DZ23 LTDA"},
	})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
}

func TestPlanoDeAssinaturaFicheiroIlegivel(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	caminho := filepath.Join(t.TempDir(), "nao-e-certificado.pem")
	if err := os.WriteFile(caminho, []byte("isto nao e' um certificado"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	codigo, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo":       "AUTHENTICODE",
		"identidade": map[string]any{"caminho_do_certificado": caminho},
	})
	// 400 e nao 500: o pedido aponta para um ficheiro que nao serve, e isso e' do pedido.
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
}

func TestPlanoDeAssinaturaDeclaraAlvos(t *testing.T) {
	router := routerDePlanoDeAssinatura(t)
	caminho := certificadoPEMParaRota(t, "DigiCert Code Signing CA",
		[]x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning})
	_, corpo := postarPlanoDeAssinatura(t, router, map[string]any{
		"tipo":       "AUTHENTICODE",
		"identidade": map[string]any{"caminho_do_certificado": caminho},
	})
	alvos, ok := corpo["alvos_suportados"].([]any)
	if !ok {
		t.Fatalf("resposta sem `alvos_suportados`: %v", corpo)
	}
	// *Um provider que existe no catalogo e nunca correu e' uma promessa com nome de modulo.*
	// A rota declara exactamente o que sabe avaliar, e nada mais.
	if len(alvos) != 2 {
		t.Fatalf("alvos = %v; esperado 2", alvos)
	}
	conjunto := map[string]bool{}
	for _, alvo := range alvos {
		conjunto[alvo.(string)] = true
	}
	if !conjunto["AUTHENTICODE"] || !conjunto["APPLE_DEVELOPER_ID"] {
		t.Fatalf("alvos = %v; faltam os dois formatos X.509", alvos)
	}
	_ = context.Background()
}
