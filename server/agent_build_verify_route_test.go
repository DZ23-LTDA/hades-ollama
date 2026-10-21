package server

// FASE 28.9: testes da rota `build.verify` (E-1262).
//
// O que estes testes travam:
//
//  1. um lote de artefactos integros e' aprovado;
//  2. UM artefacto adulterado reprova o lote inteiro;
//  3. um corpo sem artefactos NAO e' um 200 com lista vazia;
//  4. um artefacto sem `sha256` e' recusado antes de tocar no disco;
//  5. um artefacto sem `caminho` e' recusado;
//  6. o lote tem limite superior;
//  7. `nao_verificadas` diz a VERDADE sobre assinatura -- a rota nao tem adaptador;
//  8. o aviso de fronteira vai no CORPO.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func routerDeVerificacaoDeBuild(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &agentAPI{}
	router.POST("/api/agent/v1/build/verify", api.verificarBuild)
	return router
}

// artefactoParaCorpo escreve um ficheiro e devolve o `agent.Artefato` correspondente.
func artefactoParaCorpo(t *testing.T, nome, conteudo string) agent.Artefato {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	soma := sha256.Sum256([]byte(conteudo))
	return agent.Artefato{
		Caminho:      caminho,
		Nome:         nome,
		TamanhoBytes: int64(len(conteudo)),
		SHA256:       hex.EncodeToString(soma[:]),
		Alvo:         "windows",
	}
}

func postarVerificacao(t *testing.T, router *gin.Engine, corpo any) (int, map[string]any) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/build/verify", bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	resposta := httptest.NewRecorder()
	router.ServeHTTP(resposta, pedido)
	var descodificado map[string]any
	if err := json.Unmarshal(resposta.Body.Bytes(), &descodificado); err != nil {
		t.Fatalf("resposta nao e' JSON (%d): %s", resposta.Code, resposta.Body.String())
	}
	return resposta.Code, descodificado
}

func TestVerificarBuildAprovaLoteIntegro(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	a := artefactoParaCorpo(t, "hades.exe", "binario um")
	b := artefactoParaCorpo(t, "outro.exe", "binario dois")

	codigo, corpo := postarVerificacao(t, router, map[string]any{
		"artefatos": []agent.Artefato{a, b},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200 (%v)", codigo, corpo)
	}
	verificacao, ok := corpo["verificacao"].(map[string]any)
	if !ok {
		t.Fatalf("resposta sem `verificacao`: %v", corpo)
	}
	if aprovada, _ := verificacao["aprovada"].(bool); !aprovada {
		t.Fatalf("aprovada = false com 2 integros (%v)", verificacao)
	}
	if total, _ := verificacao["total"].(float64); total != 2 {
		t.Fatalf("total = %v; esperado 2", total)
	}
	if integros, _ := verificacao["integros"].(float64); integros != 2 {
		t.Fatalf("integros = %v; esperado 2", integros)
	}
	// A rota NAO tem adaptador de assinatura: os dois ficam `NAO_VERIFICADA`, e isso e' dito.
	if naoVerificadas, _ := verificacao["nao_verificadas"].(float64); naoVerificadas != 2 {
		t.Fatalf("nao_verificadas = %v; esperado 2 (%v)", naoVerificadas, verificacao)
	}
	// E nao pode afirmar o contrario.
	if semAssinatura, _ := verificacao["sem_assinatura"].(float64); semAssinatura != 0 {
		t.Fatalf("sem_assinatura = %v; esperado 0", semAssinatura)
	}
}

// O TESTE QUE IMPORTA: um adulterado reprova o lote.
func TestVerificarBuildReprovaLoteComAdulterado(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	bom := artefactoParaCorpo(t, "bom.exe", "intacto")
	mau := artefactoParaCorpo(t, "mau.exe", "original")

	conteudo, err := os.ReadFile(mau.Caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	conteudo[0] ^= 0xFF
	if err := os.WriteFile(mau.Caminho, conteudo, 0o600); err != nil {
		t.Fatalf("adulterar: %v", err)
	}

	codigo, corpo := postarVerificacao(t, router, map[string]any{
		"artefatos": []agent.Artefato{bom, mau},
	})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d; esperado 200 -- a verificacao correu, o veredito e' que falha", codigo)
	}
	verificacao, _ := corpo["verificacao"].(map[string]any)
	if aprovada, _ := verificacao["aprovada"].(bool); aprovada {
		t.Fatalf("aprovada = true com um adulterado (%v)", verificacao)
	}
	if adulterados, _ := verificacao["adulterados"].(float64); adulterados != 1 {
		t.Fatalf("adulterados = %v; esperado 1", adulterados)
	}
	if explicacao, _ := verificacao["explicacao"].(string); !strings.Contains(explicacao, "ADULTERADOS") {
		t.Fatalf("explicacao nao declara a adulteracao: %q", explicacao)
	}
}

func TestVerificarBuildRecusaLoteVazio(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": []agent.Artefato{}})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
	// *Um plano vazio apresentado como sucesso e' a mesma mentira que um card clicavel que termina
	// em tela vazia.*
	if _, temVerificacao := corpo["verificacao"]; temVerificacao {
		t.Fatal("resposta com `verificacao` para um lote vazio")
	}
}

func TestVerificarBuildRecusaArtefactoSemReferencia(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	a := artefactoParaCorpo(t, "x.exe", "x")
	a.SHA256 = ""

	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": []agent.Artefato{a}})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
	if erro, _ := corpo["erro"].(string); !strings.Contains(erro, "sha256") {
		t.Fatalf("erro nao menciona a referencia: %q", erro)
	}
}

func TestVerificarBuildRecusaArtefactoSemCaminho(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	a := artefactoParaCorpo(t, "y.exe", "y")
	a.Caminho = ""

	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": []agent.Artefato{a}})
	if codigo != http.StatusBadRequest {
		t.Fatalf("codigo = %d; esperado 400 (%v)", codigo, corpo)
	}
}

// *Uma varredura sem limite e' uma pausa que o servidor nao pode recusar depois de comecar.*
func TestVerificarBuildRecusaLoteDemasiadoGrande(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	artefatos := make([]agent.Artefato, maxArtefactosPorVerificacao+1)
	for i := range artefatos {
		artefatos[i] = agent.Artefato{
			Caminho: fmt.Sprintf("C:\\falso\\%d.exe", i),
			SHA256:  strings.Repeat("0", 64),
		}
	}
	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": artefatos})
	if codigo != http.StatusRequestEntityTooLarge {
		t.Fatalf("codigo = %d; esperado 413 (%v)", codigo, corpo)
	}
}

func TestVerificarBuildDeclaraAFronteiraNoCorpo(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	a := artefactoParaCorpo(t, "z.exe", "z")

	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": []agent.Artefato{a}})
	if codigo != http.StatusOK {
		t.Fatalf("codigo = %d (%v)", codigo, corpo)
	}
	if so, _ := corpo["somente_leitura"].(bool); !so {
		t.Fatal("somente_leitura tem de ser true")
	}
	aviso, _ := corpo["aviso_de_fronteira"].(string)
	if aviso == "" {
		t.Fatal("aviso_de_fronteira vazio")
	}
	// O aviso tem de ensinar a ler o resultado, nao apenas negar accoes.
	if !strings.Contains(aviso, "nao significa que exista assinatura") {
		t.Fatalf("aviso nao explica o que `aprovada` NAO significa: %q", aviso)
	}
}

// Ficheiro apagado entre o build e a verificacao: 404, nao 500.
func TestVerificarBuildFicheiroApagado(t *testing.T) {
	router := routerDeVerificacaoDeBuild(t)
	a := artefactoParaCorpo(t, "vai-sumir.exe", "conteudo")
	if err := os.Remove(a.Caminho); err != nil {
		t.Fatalf("remover: %v", err)
	}

	codigo, corpo := postarVerificacao(t, router, map[string]any{"artefatos": []agent.Artefato{a}})
	if codigo != http.StatusNotFound {
		t.Fatalf("codigo = %d; esperado 404 (%v)", codigo, corpo)
	}
	// E nao pode ter produzido um veredito de adulteracao para um ficheiro que nao existe.
	if _, temVerificacao := corpo["verificacao"]; temVerificacao {
		t.Fatal("resposta com veredito para um ficheiro inexistente")
	}
}
