package server

// Testes de `publish.rollback` (E-1276).
//
// O que estes testes travam:
//
//  1. sem `HADES_ARTEFACTOS` a rota recusa com 412 e nomeia a variavel;
//  2. **o pedido NAO aceita `directorio`** -- um `directorio` no corpo e' ignorado, e o plano usa a
//     raiz do ambiente;
//  3. `tipo` vazio e' recusado (400) -- *o canal nao tem valor por omissao*;
//  4. canal sem credencial devolve **409**, nao 500 -- *um 500 confundiria "o servidor falhou" com
//     "o canal nao existe"*;
//  5. um directorio com instaladores produz um plano com candidatos e SHA-256 medido;
//  6. **`executado` e' SEMPRE `false`** e o aviso garante que nenhum ficheiro foi movido.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// montarRotaDeRollback monta o router com a rota e a raiz.
func montarRotaDeRollback(t *testing.T, raiz string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDosArtefactos, raiz)
	router := gin.New()
	router.POST("/api/agent/v1/publish/rollback", (&agentAPI{}).planearRollback)
	return router
}

// pedirRollback faz o POST e devolve o codigo e o corpo.
func pedirRollback(t *testing.T, router *gin.Engine, corpo any) (int, []byte) {
	t.Helper()
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pedido := httptest.NewRequest(http.MethodPost, "/api/agent/v1/publish/rollback",
		bytes.NewReader(bruto))
	pedido.Header.Set("Content-Type", "application/json")
	gravador := httptest.NewRecorder()
	router.ServeHTTP(gravador, pedido)
	return gravador.Code, gravador.Body.Bytes()
}

// raizComInstaladores monta uma raiz com dois instaladores de datas diferentes.
func raizComInstaladores(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	for indice, nome := range []string{"HadesSetup-antigo.exe", "HadesSetup-novo.exe"} {
		caminho := filepath.Join(raiz, nome)
		if err := os.WriteFile(caminho, []byte("instalador "+nome), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
		instante := time.Now().Add(time.Duration(indice-2) * time.Hour)
		if err := os.Chtimes(caminho, instante, instante); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	return raiz
}

// Sem a raiz configurada a rota recusa.
func TestRollbackSemRaizConfigurada(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(variavelDosArtefactos, "")
	router := gin.New()
	router.POST("/api/agent/v1/publish/rollback", (&agentAPI{}).planearRollback)

	codigo, corpo := pedirRollback(t, router, pedidoDeRollback{
		Tipo: agent.RollbackWindowsInstaller,
	})
	if codigo != http.StatusPreconditionFailed {
		t.Fatalf("status = %d; esperado 412", codigo)
	}
	if !bytes.Contains(corpo, []byte(variavelDosArtefactos)) {
		t.Fatalf("erro nao nomeia a variavel: %s", corpo)
	}
}

// O PEDIDO NAO ESCOLHE O DIRECTORIO.
func TestRollbackIgnoraDirectorioDoPedido(t *testing.T) {
	raiz := raizComInstaladores(t)
	router := montarRotaDeRollback(t, raiz)

	// Um corpo com `directorio` a apontar para outro sitio: o campo nao existe no tipo do pedido, por
	// isso e' ignorado. O plano tem de usar a raiz do AMBIENTE (2 instaladores), nao o outro directorio.
	outro := t.TempDir()
	corpoComDirectorio := map[string]any{
		"tipo":       string(agent.RollbackWindowsInstaller),
		"directorio": outro,
	}
	codigo, corpo := pedirRollback(t, router, corpoComDirectorio)
	if codigo != http.StatusOK {
		t.Fatalf("status = %d: %s", codigo, corpo)
	}
	var resposta respostaDeRollback
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if resposta.TotalCandidatos != 2 {
		t.Fatalf("candidatos = %d; o `directorio` do pedido devia ser ignorado e a raiz tem 2",
			resposta.TotalCandidatos)
	}
}

// `tipo` vazio e' recusado: o canal nao tem valor por omissao.
func TestRollbackRecusaTipoVazio(t *testing.T) {
	router := montarRotaDeRollback(t, raizComInstaladores(t))
	codigo, corpo := pedirRollback(t, router, pedidoDeRollback{})
	if codigo != http.StatusBadRequest {
		t.Fatalf("status = %d; esperado 400: %s", codigo, corpo)
	}
	if !bytes.Contains(corpo, []byte("obrigatorio")) {
		t.Fatalf("erro nao declara que o tipo e' obrigatorio: %s", corpo)
	}
}

// Canal sem credencial devolve 409, nao 500.
func TestRollbackCanalNaoSuportadoDaConflito(t *testing.T) {
	router := montarRotaDeRollback(t, raizComInstaladores(t))
	codigo, corpo := pedirRollback(t, router, pedidoDeRollback{
		Tipo: agent.RollbackGooglePlay,
	})
	if codigo != http.StatusConflict {
		t.Fatalf("status = %d; esperado 409 (o servidor esta' bem, o canal e' que nao da'): %s",
			codigo, corpo)
	}
	if !bytes.Contains(corpo, []byte("executado")) {
		t.Fatalf("resposta nao declara `executado`: %s", corpo)
	}
	// E o corpo tem de dizer que nada foi executado.
	if !bytes.Contains(corpo, []byte("false")) {
		t.Fatalf("resposta nao tem `executado: false`: %s", corpo)
	}
}

// Um plano com candidatos, SHA-256 medido e `executado: false`.
func TestRollbackPlaneiaComCandidatos(t *testing.T) {
	router := montarRotaDeRollback(t, raizComInstaladores(t))
	codigo, corpo := pedirRollback(t, router, pedidoDeRollback{
		Tipo: agent.RollbackWindowsInstaller,
	})
	if codigo != http.StatusOK {
		t.Fatalf("status = %d: %s", codigo, corpo)
	}
	var resposta respostaDeRollback
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if resposta.Executado {
		t.Fatal("executado = true: a rota nao pode restaurar")
	}
	if !resposta.Suportado {
		t.Fatal("suportado = false para WINDOWS_INSTALLER")
	}
	if resposta.Alvo == nil {
		t.Fatal("alvo = nil com 2 candidatos")
	}
	if len(resposta.Alvo.SHA256) != 64 {
		t.Fatalf("sha256 do alvo com %d caracteres", len(resposta.Alvo.SHA256))
	}
	if resposta.Alvo.Nome != "HadesSetup-novo.exe" {
		t.Fatalf("alvo = %q; esperado o mais recente", resposta.Alvo.Nome)
	}
}

// O aviso garante que nenhum ficheiro foi movido.
func TestRollbackAvisoDeFronteira(t *testing.T) {
	router := montarRotaDeRollback(t, raizComInstaladores(t))
	_, corpo := pedirRollback(t, router, pedidoDeRollback{
		Tipo: agent.RollbackWindowsInstaller,
	})
	var resposta respostaDeRollback
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		t.Fatalf("resposta invalida: %v", err)
	}
	if !strings.Contains(resposta.AvisoDeFronteira, "executado") {
		t.Fatalf("aviso nao declara `executado`: %q", resposta.AvisoDeFronteira)
	}
	if !strings.Contains(resposta.AvisoDeFronteira, "nenhum ficheiro foi movido") {
		t.Fatalf("aviso nao garante que nenhum ficheiro mudou: %q", resposta.AvisoDeFronteira)
	}
	if !strings.Contains(resposta.AvisoDeFronteira, "nunca do pedido") {
		t.Fatalf("aviso nao declara que o directorio vem do ambiente: %q",
			resposta.AvisoDeFronteira)
	}
}

// garantir que a constante de SO continua a ser usada (o teste nao depende dela, mas o import sim).
var _ = runtime.GOOS
