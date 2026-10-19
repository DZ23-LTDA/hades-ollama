package server

// FASE 28.7: testes das rotas de estado do build.
//
// O que estes testes travam, e porque cada um falharia com o defeito:
//
//   - `cancel` sobre build terminado TEM de dar 409. Se alguem trocar por 200, um painel mostra
//     "cancelado" sobre um build que ja' tinha acabado.
//   - `status` de ID inexistente TEM de dar 404 e nao um registo vazio. Um zero-value devolvido como
//     sucesso e' um build inventado.
//   - `registos` vazio TEM de ser 200 com lista vazia. Uma lista vazia e um erro nao sao a mesma coisa.
//   - `artifacts` de um build sem resultado TEM de ser lista vazia com `produziu:false`, nunca uma
//     lista de artefactos "esperados".

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

func newBuildStatusRouter(api *agentAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api/agent/v1")
	g.GET("/build/status/:id", api.estadoDeBuild)
	g.GET("/build/logs/:id", api.saidaDoBuild)
	g.GET("/build/artifacts/:id", api.artefactosDoBuild)
	g.GET("/build/registos", api.registosDeBuild)
	g.POST("/build/cancel/:id", api.cancelarBuild)
	return r
}

func fazerPedido(t *testing.T, r *gin.Engine, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, caminho, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestStatusDeBuildDesconhecido404 prova que um ID inexistente nao vira registo vazio.
func TestStatusDeBuildDesconhecido404(t *testing.T) {
	api := &agentAPI{}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/status/nao-existe")
	if w.Code != http.StatusNotFound {
		t.Fatalf("devia dar 404, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("corpo invalido: %v", err)
	}
	if corpo["erro"] != "build desconhecido" {
		t.Fatalf("erro inesperado: %v", corpo["erro"])
	}
}

// TestStatusDeBuildExistenteDizAFronteira prova que a fronteira vai na resposta.
func TestStatusDeBuildExistenteDizAFronteira(t *testing.T) {
	api := &agentAPI{}
	id, err := api.storeDeBuilds().Registar("windows", "/r", "/d")
	if err != nil {
		t.Fatalf("registar: %v", err)
	}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/status/"+id)
	if w.Code != http.StatusOK {
		t.Fatalf("devia dar 200, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	// As invariantes tem de vir explicitas: um cliente nao tem de as conhecer de cor.
	if corpo["nada_assinado"] != true || corpo["nada_publicado"] != true || corpo["nada_instalado"] != true {
		t.Fatalf("as invariantes de fronteira deviam vir na resposta: %v", corpo)
	}
	if corpo["aviso_de_fronteira"] == nil || corpo["aviso_de_fronteira"] == "" {
		t.Fatal("a resposta devia trazer o aviso de fronteira")
	}
}

// TestCancelarBuildTerminado409 e' o teste central destas rotas.
//
//	*Um `cancelar` que devolve 200 sobre um build ja' terminado e' um botao que mente -- e o
//	utilizador so' descobre quando pergunta pelo resultado.*
func TestCancelarBuildTerminado409(t *testing.T) {
	api := &agentAPI{}
	id, _ := api.storeDeBuilds().Registar("windows", "/r", "/d")
	_, _ = api.storeDeBuilds().Iniciar(id)
	resultado := resultadoVazioParaTeste()
	if err := api.storeDeBuilds().Terminar(id, &resultado, nil, "ok"); err != nil {
		t.Fatalf("terminar: %v", err)
	}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "POST", "/api/agent/v1/build/cancel/"+id)
	if w.Code != http.StatusConflict {
		t.Fatalf("cancelar build terminado devia dar 409, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	if corpo["cancelado"] != false {
		t.Fatalf("o corpo tem de dizer `cancelado: false`: %v", corpo)
	}
	if corpo["estado"] == nil || corpo["estado"] == "" {
		t.Fatal("o corpo devia dizer o estado final real")
	}
}

// TestCancelarBuildEmCurso200 prova o caminho positivo.
func TestCancelarBuildEmCurso200(t *testing.T) {
	api := &agentAPI{}
	id, _ := api.storeDeBuilds().Registar("windows", "/r", "/d")
	_, _ = api.storeDeBuilds().Iniciar(id)
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "POST", "/api/agent/v1/build/cancel/"+id)
	if w.Code != http.StatusOK {
		t.Fatalf("devia dar 200, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	if corpo["cancelado"] != true {
		t.Fatalf("devia dizer cancelado: %v", corpo)
	}
	if corpo["estado"] != "CANCELADO" {
		t.Fatalf("estado devia ser CANCELADO: %v", corpo["estado"])
	}
}

// TestCancelarBuildDesconhecido404 prova o 404 no cancelamento.
func TestCancelarBuildDesconhecido404(t *testing.T) {
	api := &agentAPI{}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "POST", "/api/agent/v1/build/cancel/nao-existe")
	if w.Code != http.StatusNotFound {
		t.Fatalf("devia dar 404, deu %d", w.Code)
	}
}

// TestRegistosDeBuildListaVaziaNaoEhErro fixa a distincao.
//
//	*Uma lista vazia e um erro nao sao a mesma coisa: um painel que mostra "falhou" quando ainda
//	nada foi construido ensina o utilizador a desconfiar do painel.*
func TestRegistosDeBuildListaVaziaNaoEhErro(t *testing.T) {
	api := &agentAPI{}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/registos")
	if w.Code != http.StatusOK {
		t.Fatalf("devia dar 200 com lista vazia, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	if corpo["total"] != float64(0) {
		t.Fatalf("total devia ser 0: %v", corpo["total"])
	}
	builds, ok := corpo["builds"].([]any)
	if !ok || len(builds) != 0 {
		t.Fatalf("builds devia ser lista vazia: %v", corpo["builds"])
	}
}

// TestArtefactosSemResultadoListaVazia prova que nao se inventam artefactos "esperados".
func TestArtefactosSemResultadoListaVazia(t *testing.T) {
	api := &agentAPI{}
	id, _ := api.storeDeBuilds().Registar("windows", "/r", "/d")
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/artifacts/"+id)
	if w.Code != http.StatusOK {
		t.Fatalf("devia dar 200, deu %d: %s", w.Code, w.Body.String())
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	if corpo["produziu"] != false {
		t.Fatalf("sem resultado, `produziu` tem de ser false: %v", corpo)
	}
	arts, ok := corpo["artefactos"].([]any)
	if !ok || len(arts) != 0 {
		t.Fatalf("artefactos devia ser lista vazia: %v", corpo["artefactos"])
	}
	if corpo["nada_assinado"] != true {
		t.Fatal("a resposta de artefactos tem de dizer que nada foi assinado")
	}
}

// TestLogsDeBuildDesconhecido404 prova o 404 dos logs.
func TestLogsDeBuildDesconhecido404(t *testing.T) {
	api := &agentAPI{}
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/logs/nao-existe")
	if w.Code != http.StatusNotFound {
		t.Fatalf("devia dar 404, deu %d", w.Code)
	}
}

// TestLogsDeBuildDizSeTruncou prova que o truncamento e' declarado.
//
//	*Um corte silencioso esconde exactamente o resto que se procurava.*
func TestLogsDeBuildDizSeTruncou(t *testing.T) {
	api := &agentAPI{}
	id, _ := api.storeDeBuilds().Registar("windows", "/r", "/d")
	_, _ = api.storeDeBuilds().Iniciar(id)
	_ = api.storeDeBuilds().Terminar(id, nil, nil, "linha curta")
	r := newBuildStatusRouter(api)
	w := fazerPedido(t, r, "GET", "/api/agent/v1/build/logs/"+id)
	if w.Code != http.StatusOK {
		t.Fatalf("devia dar 200, deu %d", w.Code)
	}
	var corpo map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &corpo)
	if corpo["truncada"] != false {
		t.Fatalf("saida curta nao devia dizer truncada: %v", corpo["truncada"])
	}
	if corpo["saida"] != "linha curta" {
		t.Fatalf("saida inesperada: %v", corpo["saida"])
	}
}

// TestEstadoDeBuildPrecisaDeID cobre o param vazio (defensivo: gin nao casa a rota sem id, mas o
// handler nao pode confiar nisso).
// TestArranqueDoStoreAplicaRetencao prova que a retencao corre SEM ninguem a chamar.
//
// O `podar()` so' corre quando um build NOVO entra. Um servidor que arranca com ficheiros a mais e
// fica parado nunca volta a chamar `podar` -- e o disco nunca desce.
//
//	*Uma funcao de limpeza sem chamador e' a mesma familia do store que persistia e nunca escrevia:
//	codigo que existe e nao acontece.*
func TestArranqueDoStoreAplicaRetencao(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "builds")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Escreve mais ficheiros do que o tecto, todos terminais.
	for i := range agent.MaxBuildsGuardadosParaTeste() + 12 {
		corpo := `{"id":"x` + itoaDeTeste(i) + `","alvo":"w","estado":"CONCLUIDO","iniciado_em":"2026-01-01T00:00:00Z"}`
		if err := os.WriteFile(filepath.Join(sub, "x"+itoaDeTeste(i)+".json"), []byte(corpo), 0o644); err != nil {
			t.Fatalf("escrever %d: %v", i, err)
		}
	}
	antes, _ := os.ReadDir(sub)
	if len(antes) <= agent.MaxBuildsGuardadosParaTeste() {
		t.Fatalf("o teste devia comecar acima do tecto, tem %d", len(antes))
	}

	a := &agentAPI{}
	a.builds = agent.NovoBuildStoreComDirectorio(sub)
	_, _ = a.builds.CarregarDoDisco()
	removidos := a.builds.PodarDisco()
	if removidos == 0 {
		t.Fatal("o arranque devia ter podado")
	}
	depois, _ := os.ReadDir(sub)
	if len(depois) > agent.MaxBuildsGuardadosParaTeste() {
		t.Fatalf("depois da poda o disco devia ficar dentro do tecto, tem %d", len(depois))
	}
}

func itoaDeTeste(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestEstadoDeBuildPrecisaDeID(t *testing.T) {
	api := &agentAPI{}
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{}
	api.estadoDeBuild(c)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("sem id devia dar 400, deu %d", w.Code)
	}
}

// resultadoVazioParaTeste evita repetir a construcao nos testes.
func resultadoVazioParaTeste() agent.ResultadoDeBuild {
	return agent.ResultadoDeBuild{Explicacao: "concluido no teste"}
}
