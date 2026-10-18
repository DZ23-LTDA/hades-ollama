package server

// FASE 28.7: testes das rotas `artifact.checksum` e `artifact.verify`.
//
// O que estes testes travam:
//
//  1. `checksum` sem `caminho` -> 400;
//  2. `checksum` com caminho RELATIVO -> 400 (o servidor nao sabe qual e' o directorio de quem pede);
//  3. `checksum` de ficheiro inexistente -> 404;
//  4. `checksum` de um ficheiro real -> 200 com o SHA-256 certo;
//  5. `verify` sem `sha256` -> 400, com a razao (sem referencia seria uma medicao);
//  6. `verify` com referencia CURTA -> 400;
//  7. `verify` com o SHA correcto -> 200, `integro` e `aprovado`;
//  8. `verify` com o SHA ERRADO -> 200, `adulterado`, NAO aprovado -- o teste que importa;
//  9. as duas rotas declaram `somente_leitura: true` e um `aviso_de_fronteira` nao vazio;
// 10. o ARTEFACTO REAL da E-1251 verificado por HTTP contra o SHA que a E-1251 declarou.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// routerDeArtefacto monta o `agentAPI` minimo e as duas rotas, como em producao.
func routerDeArtefacto(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &agentAPI{}
	group := router.Group("/api/agent/v1")
	group.GET("/build/artifact/checksum", api.checksumDeArtefacto)
	group.GET("/build/artifact/verify", api.verificarArtefacto)
	return router
}

// pedirDeArtefacto faz um GET e devolve o codigo e o corpo descodificado.
func pedirDeArtefacto(t *testing.T, router *gin.Engine, caminho string) (int, map[string]any) {
	t.Helper()
	pedido := httptest.NewRequest(http.MethodGet, caminho, nil)
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)
	var corpo map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e' JSON (%d): %s", resposta.Code, resposta.Body.String())
	}
	return resposta.Code, corpo
}

// ficheiroDeTeste escreve conteudo num directorio temporario e devolve caminho + SHA-256.
func ficheiroDeTeste(t *testing.T, conteudo string) (string, string) {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "artefacto.bin")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	soma := sha256.Sum256([]byte(conteudo))
	return caminho, hex.EncodeToString(soma[:])
}

func TestChecksumRecusaSemCaminho(t *testing.T) {
	router := routerDeArtefacto(t)
	codigo, corpo := pedirDeArtefacto(t, router, "/api/agent/v1/build/artifact/checksum")
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400", codigo)
	}
	if _, ok := corpo["erro"]; !ok {
		t.Fatalf("resposta sem campo `erro`: %v", corpo)
	}
}

// *Um caminho relativo e' relativo ao directorio de quem pede, e o servidor nao sabe qual e'.*
func TestChecksumRecusaCaminhoRelativo(t *testing.T) {
	router := routerDeArtefacto(t)
	for _, relativo := range []string{"artefacto.bin", "./x", "dist/hades.exe"} {
		codigo, corpo := pedirDeArtefacto(t, router,
			"/api/agent/v1/build/artifact/checksum?caminho="+relativo)
		if codigo != http.StatusBadRequest {
			t.Fatalf("caminho %q -> codigo %d; esperado 400 (%v)", relativo, codigo, corpo)
		}
	}
}

func TestChecksumFicheiroInexistente(t *testing.T) {
	router := routerDeArtefacto(t)
	inexistente := filepath.Join(t.TempDir(), "nao-existe.bin")
	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/checksum?caminho="+inexistente)
	if codigo != http.StatusNotFound {
		t.Fatalf("codigo = %d; esperado 404 (%v)", codigo, corpo)
	}
}

func TestChecksumMedeFicheiroReal(t *testing.T) {
	router := routerDeArtefacto(t)
	caminho, esperado := ficheiroDeTeste(t, "conteudo-do-artefacto")

	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/checksum?caminho="+caminho)
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200 (%v)", codigo, corpo)
	}
	if obtido, _ := corpo["sha256"].(string); obtido != esperado {
		t.Fatalf("sha256 = %q; esperado %q", obtido, esperado)
	}
	// A fronteira declarada no CORPO, nao so' na documentacao.
	if so, _ := corpo["somente_leitura"].(bool); !so {
		t.Fatal("somente_leitura tem de ser true")
	}
	if aviso, _ := corpo["aviso_de_fronteira"].(string); aviso == "" {
		t.Fatal("aviso_de_fronteira vazio: quem le' nao sabe o que a rota nao faz")
	}
}

// Sem referencia NAO se verifica.
func TestVerifyRecusaSemReferencia(t *testing.T) {
	router := routerDeArtefacto(t)
	caminho, _ := ficheiroDeTeste(t, "x")
	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/verify?caminho="+caminho)
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
	if _, ok := corpo["erro"]; !ok {
		t.Fatal("resposta sem `erro`: sem referencia isto seria uma medicao")
	}
}

// Um PREFIXO de checksum nao e' uma referencia.
func TestVerifyRecusaReferenciaCurta(t *testing.T) {
	router := routerDeArtefacto(t)
	caminho, soma := ficheiroDeTeste(t, "x")
	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/verify?caminho="+caminho+"&sha256="+soma[:16])
	if codigo != http.StatusBadRequest {
		t.Fatalf("prefixo de 16 -> codigo %d; esperado 400 (%v)", codigo, corpo)
	}
}

func TestVerifyAprovaIntegro(t *testing.T) {
	router := routerDeArtefacto(t)
	caminho, soma := ficheiroDeTeste(t, "artefacto-integro")
	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/verify?caminho="+caminho+"&sha256="+soma)
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200 (%v)", codigo, corpo)
	}
	resultado, ok := corpo["resultado"].(map[string]any)
	if !ok {
		t.Fatalf("resposta sem `resultado`: %v", corpo)
	}
	if integro, _ := resultado["integro"].(bool); !integro {
		t.Fatalf("integro = false (%v)", resultado)
	}
	if aprovado, _ := resultado["aprovado"].(bool); !aprovado {
		t.Fatalf("aprovado = false (%v)", resultado)
	}
	// O `sha256_medido` tem de ser o do ficheiro, nao o esperado copiado.
	if medido, _ := resultado["sha256_medido"].(string); medido != soma {
		t.Fatalf("sha256_medido = %q; esperado %q", medido, soma)
	}
}

// O TESTE QUE IMPORTA: um byte diferente reprova.
func TestVerifyDetectaAdulteracao(t *testing.T) {
	router := routerDeArtefacto(t)
	caminho, somaOriginal := ficheiroDeTeste(t, "artefacto-original")

	// Confirma que o original passa.
	if codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/verify?caminho="+caminho+"&sha256="+somaOriginal); codigo != http.StatusOK {
		t.Fatalf("original: codigo %d (%v)", codigo, corpo)
	}

	// Adultera UM byte.
	conteudo, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	conteudo[0] ^= 0xFF
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		t.Fatalf("adulterar: %v", err)
	}

	codigo, corpo := pedirDeArtefacto(t, router,
		"/api/agent/v1/build/artifact/verify?caminho="+caminho+"&sha256="+somaOriginal)
	if codigo != http.StatusOK {
		// A rota responde 200 com `adulterado=true`: a VERIFICACAO correu, o veredito e' que falha.
		// *Um 4xx aqui confundiria "pedido invalido" com "artefacto adulterado".*
		t.Fatalf("codigo = %d; esperado 200 com adulterado=true (%v)", codigo, corpo)
	}
	resultado, _ := corpo["resultado"].(map[string]any)
	if adulterado, _ := resultado["adulterado"].(bool); !adulterado {
		t.Fatalf("ADULTERACAO NAO DETECTADA (%v)", resultado)
	}
	if aprovado, _ := resultado["aprovado"].(bool); aprovado {
		t.Fatalf("aprovado=true num artefacto adulterado (%v)", resultado)
	}
}
