package server

// Testes de `sbom.generate` (E-1281).
//
// O que estes testes travam:
//
//  1. sem `HADES_RAIZ_SBOM` a rota recusa com 412 e nomeia a variavel;
//  2. uma raiz que nao e' directorio tambem recusa com 412;
//  3. **a raiz NAO vem do pedido** -- um `raiz` no corpo e' ignorado;
//  4. **`assinado` e `provenance` sao SEMPRE `false`** e o aviso declara as limitacoes;
//  5. um projecto Go real produz componentes e a assinatura do inventario;
//  6. um directorio VAZIO produz `total: 0` com `nao_produzido` a declarar as fontes ausentes --
//     **nao um erro**.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// montarRotaDeSBOM monta o router com a raiz.
func montarRotaDeSBOM(t *testing.T, raiz string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDaRaizDoSBOM, raiz)
	router := gin.New()
	router.POST("/api/agent/v1/sbom/generate", (&agentAPI{}).gerarSBOM)
	return router
}

// pedirSBOM faz o POST e devolve o codigo e o corpo.
func pedirSBOM(t *testing.T, router *gin.Engine, corpo any) (int, []byte) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/sbom/generate",
		bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, pedido)
	return gravador.Code, gravador.Body.Bytes()
}

// projectoGoComDependencias monta um projecto Go minimo.
func projectoGoComDependencias(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	goMod := "module exemplo.test/projecto\n\ngo 1.24\n\nrequire (\n" +
		"\tgithub.com/gin-gonic/gin v1.10.0\n" +
		"\tgolang.org/x/text v0.14.0 // indirect\n" +
		")\n"
	if err := os.WriteFile(filepath.Join(raiz, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	return raiz
}

// Sem a raiz configurada a rota recusa.
func TestSBOMSemRaizConfigurada(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDaRaizDoSBOM, "")
	router := gin.New()
	router.POST("/api/agent/v1/sbom/generate", (&agentAPI{}).gerarSBOM)

	codigo, corpo := pedirSBOM(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; esperado 412: %s", codigo, corpo)
	}
	if !bytes.Contains(corpo, []byte(variavelDaRaizDoSBOM)) {
		t.Fatalf("erro nao nomeia a variavel: %s", corpo)
	}
}

// Uma raiz que nao e' directorio recusa.
func TestSBOMRaizNaoEDirectorio(t *testing.T) {
	ficheiro := filepath.Join(t.TempDir(), "nao-e-directorio.txt")
	if err := os.WriteFile(ficheiro, []byte("x"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	router := montarRotaDeSBOM(t, ficheiro)
	codigo, corpo := pedirSBOM(t, router, map[string]any{})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; esperado 412: %s", codigo, corpo)
	}
	if !bytes.Contains(corpo, []byte("nao e' um directorio")) {
		t.Fatalf("erro nao explica que nao e' directorio: %s", corpo)
	}
}

// A RAIZ NAO VEM DO PEDIDO.
func TestSBOMIgnoraRaizDoPedido(t *testing.T) {
	raiz := projectoGoComDependencias(t)
	router := montarRotaDeSBOM(t, raiz)

	// Um corpo com `raiz` a apontar para outro sitio: tem de ser ignorado.
	outro := t.TempDir()
	codigo, corpo := pedirSBOM(t, router, map[string]any{"raiz": outro})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d: %s", codigo, corpo)
	}
	var resposta respostaDeSBOM
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	// Se o `raiz` do pedido tivesse sido usado, `outro` (vazio) daria total 0. A raiz do AMBIENTE
	// tem 2 componentes.
	if resposta.Inventario.Total == 0 {
		t.Fatal("o `raiz` do pedido foi usado: o inventario do directorio vazio deu total 0")
	}
}

// `assinado` e `provenance` sao sempre false.
func TestSBOMNuncaAssinaNemAtesta(t *testing.T) {
	router := montarRotaDeSBOM(t, projectoGoComDependencias(t))
	codigo, corpo := pedirSBOM(t, router, map[string]any{})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d: %s", codigo, corpo)
	}
	var resposta respostaDeSBOM
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if resposta.Assinado {
		t.Fatal("assinado = true: esta rota nao assina")
	}
	if resposta.Provenance {
		t.Fatal("provenance = true: esta rota nao emite atestacao")
	}
	if !resposta.Gerado {
		t.Fatal("gerado = false com 200")
	}
	// E o aviso tem de declarar as limitacoes.
	for _, exigido := range []string{"MINIMO", "NAO inclui", "sempre false", "nunca do pedido"} {
		if !bytes.Contains(corpo, []byte(exigido)) {
			t.Fatalf("aviso nao declara %q: %s", exigido, resposta.AvisoDeFronteira)
		}
	}
}

// Um projecto Go real produz componentes e assinatura.
func TestSBOMInventariaProjectoGo(t *testing.T) {
	router := montarRotaDeSBOM(t, projectoGoComDependencias(t))
	codigo, corpo := pedirSBOM(t, router, map[string]any{})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d: %s", codigo, corpo)
	}
	var resposta respostaDeSBOM
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if resposta.Inventario.Total != 2 {
		t.Fatalf("total = %d; esperado 2 (gin directo, x/text indirecto)",
			resposta.Inventario.Total)
	}
	// A assinatura do inventario tem de ter linhas -- e nao pode estar vazia.
	if len(resposta.AssinaturaDoInventario) == 0 {
		t.Fatal("assinatura_do_inventario vazia com 2 componentes")
	}
	// E `ficheiros_lidos` tem de citar o go.mod.
	encontrouGoMod := false
	for _, ficheiro := range resposta.Inventario.FicheirosLidos {
		if filepath.Base(ficheiro) == "go.mod" {
			encontrouGoMod = true
		}
	}
	if !encontrouGoMod {
		t.Fatalf("ficheiros_lidos nao cita go.mod: %v", resposta.Inventario.FicheirosLidos)
	}
}

// Um directorio VAZIO nao e' um erro: e' um inventario de zero com as fontes declaradas ausentes.
func TestSBOMDirectorioVazioNaoEERRO(t *testing.T) {
	router := montarRotaDeSBOM(t, t.TempDir())
	codigo, corpo := pedirSBOM(t, router, map[string]any{})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d; um directorio vazio nao e' erro de servidor: %s", codigo, corpo)
	}
	var resposta respostaDeSBOM
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if resposta.Inventario.Total != 0 {
		t.Fatalf("total = %d com directorio vazio", resposta.Inventario.Total)
	}
	// E tem de DIZER que nao encontrou fontes, em vez de fingir um SBOM vazio e valido.
	if len(resposta.Inventario.NaoProduzido) == 0 {
		t.Fatal("nao_produzido vazio num directorio sem fontes de dependencia")
	}
}
