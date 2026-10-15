package server

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// E-668 (§9, eixo "tool") — A ROTA DE CAPACIDADES TEM DE DECLARAR AS FERRAMENTAS QUE **ESTE** RUNTIME REGISTROU.
//
// As capacidades de MODELO que a rota ja' publicava vem de um registro estatico. O que este processo consegue
// **executar** sao as ferramentas registradas no runtime — e elas dependem da injecao de dependencia (`model.infer`
// so' existe com cliente de inferencia). *Publicar so' os modelos daria a impressao de que a ferramenta esta'
// disponivel quando o kind nem existe neste processo.*
//
// ⚠️ A DIRECAO QUE COSTUMA FALTAR e' a segunda: **sem runtime**, o campo nao pode vir como lista vazia silenciosa —
// *lista vazia e' "nao medido", nunca "nenhuma ferramenta".*
func TestRotaDeCapacidadesDeclaraFerramentasRegistradas(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// (1) Runtime LIGADO, com cliente de inferencia: o executor aparece na lista.
	t.Setenv("OLLAMA_AGENT_ROOT", t.TempDir())
	t.Setenv("OLLAMA_AGENT_STORE", t.TempDir())
	t.Setenv("OLLAMA_AGENT_DATABASE_URL", "")
	t.Setenv("OLLAMA_AGENT_REDIS_URL", "")
	t.Setenv("OLLAMA_AGENT_MODEL", "qwen2.5:7b")
	runtime, err := newDefaultAgentRuntime()
	if err != nil {
		t.Fatalf("construir o runtime de producao: %v", err)
	}
	ferramentas := blocoDeFerramentas(t, &agentAPI{runtime: runtime})
	registradas, ok := ferramentas["registered"].([]any)
	if !ok {
		t.Fatalf("`tools.registered` veio como %T — *o eixo das ferramentas tem de ser publicavel*", ferramentas["registered"])
	}
	encontrado := false
	for _, kind := range registradas {
		if kind == "model.infer" {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatalf("`model.infer` tinha de constar das ferramentas registradas; veio %v", registradas)
	}
	if motivo, _ := ferramentas["unavailable"].(string); motivo != "" {
		t.Fatalf("com runtime ligado e ferramentas registradas nao ha' motivo de indisponibilidade, veio %q", motivo)
	}

	// (2) Runtime AUSENTE: NAO MEDIDO, com motivo declarado — nunca lista vazia calada.
	ferramentas = blocoDeFerramentas(t, &agentAPI{})
	if ferramentas["registered"] != nil {
		t.Fatalf("sem runtime nao pode haver lista de ferramentas (veio %v): *nao medido nao e' nenhuma*",
			ferramentas["registered"])
	}
	motivo, _ := ferramentas["unavailable"].(string)
	if motivo == "" {
		t.Fatal("sem runtime o motivo da nao-medicao tem de ser DECLARADO")
	}
	t.Logf("declarado sem runtime: %s", motivo)
}

// blocoDeFerramentas chama a rota REAL de capacidades e devolve o bloco `tools` da resposta.
func blocoDeFerramentas(t *testing.T, api *agentAPI) map[string]any {
	t.Helper()
	gravador := httptest.NewRecorder()
	contexto, _ := gin.CreateTestContext(gravador)
	contexto.Request = httptest.NewRequest("GET", "/api/agent/capabilities", nil)
	api.capabilities(contexto)
	if gravador.Code != 200 {
		t.Fatalf("a rota de capacidades devolveu %d", gravador.Code)
	}
	var corpo map[string]any
	if err := json.Unmarshal(gravador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta nao e' JSON legivel: %v", err)
	}
	ferramentas, ok := corpo["tools"].(map[string]any)
	if !ok {
		t.Fatalf("a resposta tem de trazer o bloco `tools` (veio %T)", corpo["tools"])
	}
	return ferramentas
}
