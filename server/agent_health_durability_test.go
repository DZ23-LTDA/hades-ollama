package server

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ollama/ollama/internal/agent"
)

// POLÍTICA DE DURABILIDADE VISÍVEL NA SAÚDE (E-168). Trocar `fsync` por velocidade é uma decisão sobre perder
// escritas numa queda abrupta; se o modo não aparecer em lugar nenhum, a troca só seria descoberta por um
// arquivo perdido. Este teste trava a VISIBILIDADE — e o texto do risco, para a interface não virar um rótulo
// bonito sem consequência declarada.
func TestHealthReportsWriteDurabilityPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	anterior := agent.WriteDurability()
	t.Cleanup(func() { _, _ = agent.SetWriteDurability(anterior) })

	for _, caso := range []struct {
		modo      string
		esperado  string
		noDetalhe string
	}{
		{modo: "", esperado: agent.WriteDurabilityDurable, noDetalhe: "sincroniza"},
		{modo: agent.WriteDurabilityRelaxed, esperado: agent.WriteDurabilityRelaxed, noDetalhe: "pode ser perdido"},
	} {
		t.Setenv("HADES_WRITE_DURABILITY", caso.modo)
		runtime, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir(), DataRoot: t.TempDir()})
		if err != nil {
			t.Fatalf("modo %q: %v", caso.modo, err)
		}
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		(&agentAPI{runtime: runtime}).health(context)
		if recorder.Code != 200 {
			t.Fatalf("modo %q: status = %d, body = %s", caso.modo, recorder.Code, recorder.Body.String())
		}
		var corpo struct {
			Subsystems struct {
				Escrita struct {
					Status string `json:"status"`
					Detail string `json:"detail"`
				} `json:"escrita"`
			} `json:"subsystems"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &corpo); err != nil {
			t.Fatalf("modo %q: decode health: %v (body=%s)", caso.modo, err, recorder.Body.String())
		}
		if corpo.Subsystems.Escrita.Status != caso.esperado {
			t.Fatalf("modo %q: a saúde precisa relatar %q, veio %q", caso.modo, caso.esperado, corpo.Subsystems.Escrita.Status)
		}
		if !strings.Contains(corpo.Subsystems.Escrita.Detail, caso.noDetalhe) {
			t.Fatalf("modo %q: o detalhe precisa declarar a consequência (%q): %q", caso.modo, caso.noDetalhe, corpo.Subsystems.Escrita.Detail)
		}
	}
}
