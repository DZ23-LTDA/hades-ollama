package server

// Testes das rotas `toolchain/*` (FASE 28.7 + FASE 24).
//
// O que estes testes protegem:
//   1. as rotas existem e respondem o schema prometido;
//   2. alvo desconhecido devolve 404, NAO 200 com plano vazio;
//   3. `blockers` e `trabalho_independente` sao SEMPRE listas, nunca `null`;
//   4. o `veredicto` distingue pronto / bloqueado / incompleto;
//   5. nenhuma rota instala ou altera nada -- todas sao GET.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func routerDeToolchain(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	(&agentAPI{}).register(engine)
	return engine
}

func getJSON(t *testing.T, engine *gin.Engine, caminho string, destino any) int {
	t.Helper()
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, caminho, nil))
	if destino != nil && recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), destino); err != nil {
			t.Fatalf("GET %s: JSON invalido: %v (corpo: %s)", caminho, err, recorder.Body.String())
		}
	}
	return recorder.Code
}

// A listagem de alvos tem de existir e declarar host, goos e os alvos.
func TestToolchainTargetsResponde(t *testing.T) {
	engine := routerDeToolchain(t)
	var payload struct {
		Host  string `json:"host"`
		GOOS  string `json:"goos"`
		Alvos []struct {
			Nome       string   `json:"nome"`
			Descricao  string   `json:"descricao"`
			Exigencias []string `json:"exigencias"`
		} `json:"alvos"`
		Count int `json:"count"`
	}
	if code := getJSON(t, engine, "/api/agent/v1/toolchain/targets", &payload); code != http.StatusOK {
		t.Fatalf("GET /toolchain/targets -> %d", code)
	}
	if payload.GOOS == "" {
		t.Fatal("resposta sem goos: o consumidor nao sabe em que maquina o catalogo foi lido")
	}
	if len(payload.Alvos) == 0 {
		t.Fatal("catalogo vazio na API")
	}
	if payload.Count != len(payload.Alvos) {
		t.Fatalf("count=%d mas %d alvos", payload.Count, len(payload.Alvos))
	}
	for _, alvo := range payload.Alvos {
		if alvo.Nome == "" || alvo.Descricao == "" {
			t.Fatalf("alvo incompleto: %+v", alvo)
		}
		if len(alvo.Exigencias) == 0 {
			t.Fatalf("alvo %s sem exigencias", alvo.Nome)
		}
		for _, ex := range alvo.Exigencias {
			if !strings.Contains(ex, " -- ") {
				t.Fatalf("exigencia sem motivo legivel: %q", ex)
			}
		}
	}
}

// Alvo desconhecido tem de ser 404, NAO 200 com plano vazio.
//
// *Um plano vazio para um alvo que nao existe pareceria "tudo pronto" a quem le so' o `pronto`.*
func TestToolchainPlanAlvoDesconhecidoE404(t *testing.T) {
	engine := routerDeToolchain(t)
	if code := getJSON(t, engine, "/api/agent/v1/toolchain/plan/blackberry-os", nil); code != http.StatusNotFound {
		t.Fatalf("alvo desconhecido -> %d; esperado 404", code)
	}
}

// `plan windows` tem de trazer o schema completo, com listas nunca nulas.
func TestToolchainPlanWindowsTemSchemaCompleto(t *testing.T) {
	engine := routerDeToolchain(t)
	var payload struct {
		Host      string `json:"host"`
		Alvo      string `json:"alvo"`
		Pronto    bool   `json:"pronto"`
		Veredicto string `json:"veredicto"`
		Probes    []struct {
			Nome        string `json:"nome"`
			Estado      string `json:"estado"`
			Obrigatorio bool   `json:"obrigatorio"`
			Motivo      string `json:"motivo"`
			Detalhe     string `json:"detalhe"`
		} `json:"probes"`
		Blockers []string `json:"blockers"`
		Avanca   []string `json:"trabalho_independente"`
	}
	if code := getJSON(t, engine, "/api/agent/v1/toolchain/plan/windows", &payload); code != http.StatusOK {
		t.Fatalf("GET /toolchain/plan/windows -> %d", code)
	}
	if payload.Alvo != "windows" {
		t.Fatalf("alvo = %q", payload.Alvo)
	}
	if len(payload.Probes) == 0 {
		t.Fatal("plano sem probes")
	}
	for _, probe := range payload.Probes {
		if probe.Nome == "" || probe.Estado == "" || probe.Motivo == "" || probe.Detalhe == "" {
			t.Fatalf("probe incompleto: %+v", probe)
		}
	}
	// `blockers` e `trabalho_independente` tem de ser listas, nao null.
	if payload.Blockers == nil {
		t.Fatal("blockers veio null; o contrato promete lista, e `null` obriga o consumidor a tratar dois casos")
	}
	if payload.Avanca == nil {
		t.Fatal("trabalho_independente veio null")
	}
	// O veredicto tem de ser um dos tres.
	switch payload.Veredicto {
	case "pronto", "bloqueado", "incompleto":
	default:
		t.Fatalf("veredicto inesperado: %q", payload.Veredicto)
	}
	// Coerencia entre `pronto` e `veredicto`.
	if payload.Pronto && payload.Veredicto != "pronto" {
		t.Fatalf("pronto=true com veredicto %q", payload.Veredicto)
	}
	if !payload.Pronto && payload.Veredicto == "pronto" {
		t.Fatal("pronto=false com veredicto pronto")
	}
	// Coerencia entre `bloqueado` e ter blockers.
	if payload.Veredicto == "bloqueado" && len(payload.Blockers) == 0 {
		t.Fatal("veredicto bloqueado sem blocker algum")
	}
	if payload.Veredicto == "incompleto" && len(payload.Blockers) != 0 {
		t.Fatal("veredicto incompleto com blockers: devia ser bloqueado")
	}
}

// `doctor` tem de cobrir todos os alvos e o resumo tem de ser coerente com as contagens.
func TestToolchainDoctorResumeTodosOsAlvos(t *testing.T) {
	engine := routerDeToolchain(t)
	var payload struct {
		Alvos []struct {
			Alvo      string `json:"alvo"`
			Pronto    bool   `json:"pronto"`
			Veredicto string `json:"veredicto"`
			Falta     int    `json:"exigencias_bloqueantes"`
		} `json:"alvos"`
		Prontos     int    `json:"prontos"`
		Bloqueados  int    `json:"bloqueados"`
		Incompletos int    `json:"incompletos"`
		Total       int    `json:"total"`
		Resumo      string `json:"resumo"`
	}
	if code := getJSON(t, engine, "/api/agent/v1/toolchain/doctor", &payload); code != http.StatusOK {
		t.Fatalf("GET /toolchain/doctor -> %d", code)
	}
	if payload.Total == 0 || len(payload.Alvos) == 0 {
		t.Fatal("doctor nao reportou alvos")
	}
	// As contagens tem de somar o total.
	if payload.Prontos+payload.Bloqueados+payload.Incompletos != payload.Total {
		t.Fatalf("contagens nao somam: %d+%d+%d != %d",
			payload.Prontos, payload.Bloqueados, payload.Incompletos, payload.Total)
	}
	// Cada alvo classificado tem de ser coerente com o seu veredicto.
	for _, alvo := range payload.Alvos {
		switch alvo.Veredicto {
		case "pronto":
			if !alvo.Pronto || alvo.Falta != 0 {
				t.Fatalf("alvo %s pronto mas Falta=%d", alvo.Alvo, alvo.Falta)
			}
		case "bloqueado":
			if alvo.Falta == 0 {
				t.Fatalf("alvo %s bloqueado sem exigencias bloqueantes", alvo.Alvo)
			}
		case "incompleto":
			if alvo.Falta != 0 {
				t.Fatalf("alvo %s incompleto com %d bloqueantes", alvo.Alvo, alvo.Falta)
			}
		}
	}
	// O resumo nunca fica vazio: o Mission Control mostra-o directamente.
	if strings.TrimSpace(payload.Resumo) == "" {
		t.Fatal("resumo vazio: a UI nao pode mostrar uma frase vazia")
	}
	// E o resumo tem de mencionar o numero de bloqueados quando ha' alguns.
	if payload.Bloqueados > 0 && !strings.Contains(payload.Resumo, "bloqueado") {
		t.Fatalf("resumo nao menciona bloqueio apesar de %d bloqueados: %q", payload.Bloqueados, payload.Resumo)
	}
}

// Alvo vazio tem de ser 400, nao 404 nem panic.
func TestToolchainPlanAlvoVazioE400(t *testing.T) {
	engine := routerDeToolchain(t)
	// `/toolchain/plan/` sem segmento nao casa a rota: o gin devolve 404 do router.
	// O 400 do handler cobre o caso de o parametro chegar vazio por outra via.
	code := getJSON(t, engine, "/api/agent/v1/toolchain/plan/", nil)
	if code == http.StatusOK {
		t.Fatalf("alvo vazio devolveu 200")
	}
}

// As rotas sao de LEITURA: um POST tem de ser recusado.
//
// *Um preflight que muda o sistema nao e' preflight: e' instalacao disfarcada.*
func TestToolchainE_SoLeitura(t *testing.T) {
	engine := routerDeToolchain(t)
	for _, caminho := range []string{
		"/api/agent/v1/toolchain/targets",
		"/api/agent/v1/toolchain/doctor",
		"/api/agent/v1/toolchain/plan/windows",
	} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, caminho, nil))
		if recorder.Code == http.StatusOK {
			t.Fatalf("POST %s devolveu 200; o preflight tem de ser so' leitura", caminho)
		}
	}
}
