package multillm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// INFERÊNCIA REAL POR PROVEDOR DE NUVEM — OPT-IN (E-210).
//
// POR QUE ESTE TESTE EXISTE
// -------------------------
// O estágio 4 exige "chamadas reais por provedor" como evidência. O inventário de provedores (E-209) mostrou que
// **19 provedores têm credencial presente**, mas `inference_verified` era `true` em **um só** — o Ollama local. Ou
// seja: o que faltava não era chave, era **inferência verificada**.
//
// POR QUE ELE É OPT-IN
// --------------------
// Uma chamada de verdade usa a **conta do operador** e pode gerar custo. Um CI hermético que dependesse disso ficaria
// vermelho por motivo alheio ao código — o mesmo princípio que aplicamos à sessão real de CLI (E-205). Então o padrão
// é **pular**, e a chamada só acontece quando o operador autoriza com `OLLAMA_AGENT_REAL_PROVIDER=<id do preset>`.
//
// O QUE ELE USA DO PRODUTO
// ------------------------
// O **preset do provedor** (`ProviderPresetByID`), o **mesmo caminho de resolução de credencial**
// (`CredentialValue(envName)`, que lê o ambiente e o diretório de credenciais do produto) e o **caminho de chat
// declarado no preset** (`Paths`). Não é um curl paralelo ao produto: é o produto dizendo onde e como falar.
//
// O QUE ELE **NÃO** FAZ
// ---------------------
// Não passa pelo proxy HTTP completo do `Gateway` (que exigiria subir o servidor). Isso está **declarado** para não
// confundir "inferência real com o preset do produto" com "inferência real através de todo o proxy".
func TestInferenciaRealPorProvedorEhOptIn(t *testing.T) {
	id := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_PROVIDER"))
	if id == "" {
		t.Skip("sem OLLAMA_AGENT_REAL_PROVIDER: a inferência real usa conta do operador e pode gerar custo")
	}

	preset, ok := ProviderPresetByID(id)
	if !ok {
		t.Fatalf("preset %q não existe no catálogo do produto", id)
	}
	t.Logf("preset: id=%s nome=%q base=%s chave_em=%q paths=%v local=%v", preset.ID, preset.Name, preset.BaseURL, preset.APIKeyEnv, preset.Paths, preset.Local)

	if preset.Local {
		t.Skipf("preset %q é local: não é uma chamada externa", id)
	}

	// Mesmo caminho de credencial do produto. A chave NUNCA é impressa.
	chave := CredentialValue(preset.APIKeyEnv)
	if strings.TrimSpace(chave) == "" {
		t.Skipf("BLOCKED_BY_EXTERNAL_DEPENDENCY: %s ausente para o preset %q", preset.APIKeyEnv, id)
	}
	t.Logf("credencial presente via %s (valor não impresso)", preset.APIKeyEnv)

	caminho := ""
	for _, p := range preset.Paths {
		if strings.Contains(p, "chat") {
			caminho = p
			break
		}
	}
	if caminho == "" {
		t.Fatalf("o preset %q não declara caminho de chat; paths=%v", id, preset.Paths)
	}

	modelo := strings.TrimSpace(os.Getenv("OLLAMA_AGENT_REAL_MODEL"))
	if modelo == "" {
		modelo = "llama-3.1-8b-instant" // modelo pequeno e barato, adequado a um teste de caminho
	}

	corpo := map[string]any{
		"model":      modelo,
		"messages":   []map[string]string{{"role": "user", "content": "Responda apenas: OK"}},
		"max_tokens": 8,
		"stream":     false,
	}
	bruto, err := json.Marshal(corpo)
	if err != nil {
		t.Fatal(err)
	}

	url := strings.TrimRight(preset.BaseURL, "/") + caminho
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bruto))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+chave)

	inicio := time.Now()
	resp, err := (&http.Client{Timeout: 55 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("chamada real ao provedor %q falhou: %v", id, err)
	}
	defer resp.Body.Close()
	latencia := time.Since(inicio)
	corpoResp, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<10))

	if resp.StatusCode != http.StatusOK {
		// A mensagem do provedor explica o motivo (401 = chave, 429 = limite); truncamos para o log não virar despejo.
		trecho := strings.TrimSpace(string(corpoResp))
		if len(trecho) > 240 {
			trecho = trecho[:240] + "..."
		}
		t.Fatalf("provedor %q respondeu HTTP %d em %s: %s", id, resp.StatusCode, latencia, trecho)
	}

	var conteudo struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(corpoResp, &conteudo); err != nil {
		t.Fatalf("resposta 200 mas não é o JSON esperado: %v", err)
	}
	if len(conteudo.Choices) == 0 || strings.TrimSpace(conteudo.Choices[0].Message.Content) == "" {
		t.Fatal("HTTP 200 sem conteúdo: não conta como inferência verificada")
	}

	// Evidência: provedor, modelo, latência e uso — sem a chave e sem despejar a resposta inteira.
	t.Logf("INFERÊNCIA REAL: provedor=%s modelo=%s latência=%s uso=%v resposta=%q",
		id, modelo, latencia.Round(time.Millisecond), conteudo.Usage, primeiros(conteudo.Choices[0].Message.Content, 40))
}

// primeiros corta o texto para o log, evitando despejo de conteúdo do modelo.
func primeiros(texto string, limite int) string {
	limpo := strings.Join(strings.Fields(texto), " ")
	if len(limpo) > limite {
		return fmt.Sprintf("%s…(%d chars)", limpo[:limite], len(limpo))
	}
	return limpo
}
