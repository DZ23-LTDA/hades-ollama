// Execucao da avaliacao de personas contra o motor local (FASE 30.7).
//
// # O QUE ISTO LIGA
//
// O E-1369 provou o harness; o E-1375 provou-o contra o `qwen2.5:7b` real. Faltava o PRODUTO: a rota
// respondia `inference_available: false` porque nada no servidor sabia falar com o motor.
//
//	*Um backend completo e um produto incompleto nao sao a mesma coisa, e a distancia entre os dois e a
//	interface.*
//
// SEM PROMESSA: DETECCAO AO VIVO
//
// `inference_available` nao passa a ser `true` por ter sido escrito. E' perguntado ao motor no momento da
// leitura: `GET /api/tags` no Ollama configurado. Se nao responder, continua `false` com o motivo.
//
//	*Um estado que so o programador pode nomear valido nao e um estado: e um campo de texto.*
//
// # CUSTO E LIMITES
//
// Uma avaliacao real percorre a suite inteira contra o modelo: dezenas de segundos, CPU/GPU a serio. Por
// isso:
//
//   - a rota e' POST e o utilizador pede-a explicitamente -- nao corre ao abrir o painel;
//
//   - ha' um teto de casos e um timeout por chamada;
//
//   - o modelo e' configuravel mas TEM de existir, senao a rota recusa antes de gastar nada.
//
//     *Aprovar uma acao sem ver o custo e aprovar um numero que nao se leu.*
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/internal/agent"
)

// errMotorIndisponivel cobre o caso em que o Ollama nao responde ou nao tem o modelo.
var errMotorIndisponivel = errors.New("local inference engine is not available")

// evalRunnerTimeout e' o teto por chamada ao motor. Um caso que demore mais do que isto e' um caso que nao
// vai terminar; cortar cedo poupa o resto da suite.
const evalRunnerTimeout = 90 * time.Second

// evalRunnerMaxCasos limita quantos casos uma execucao pode correr. Acima disto a rota recusa: uma avaliacao
// que o utilizador nao ve terminar nao e' uma avaliacao, e' um timeout.
const evalRunnerMaxCasos = 8

// ollamaEndpoint devolve o endpoint configurado, com o mesmo default do resto do produto.
//
// `OLLAMA_HOST` pode vir como `host:porta` ou `http://host:porta`. Aceita os dois, sem inventar esquema.
func ollamaEndpoint() string {
	bruto := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
	if bruto == "" {
		return "http://127.0.0.1:11434"
	}
	if !strings.HasPrefix(bruto, "http://") && !strings.HasPrefix(bruto, "https://") {
		return "http://" + bruto
	}
	return strings.TrimSuffix(bruto, "/")
}

// motorLocal descreve o que o servidor conseguiu descobrir sobre o motor.
type motorLocal struct {
	Disponivel bool     `json:"available"`
	Endpoint   string   `json:"endpoint"`
	Modelo     string   `json:"model,omitempty"`
	Modelos    []string `json:"models,omitempty"`
	Motivo     string   `json:"reason,omitempty"`
}

// detectarMotor pergunta ao motor, ao vivo, se esta' la' e que modelos tem.
//
// Nao ha' cache: um resultado de ha' dez minutos nao prova saude agora.
// *Uma falha antiga nao pode banir um provedor para sempre e um sucesso antigo nao prova saude atual.*
func detectarMotor(ctx context.Context, modeloPedido string) motorLocal {
	endpoint := ollamaEndpoint()
	m := motorLocal{Endpoint: endpoint}

	cli := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/api/tags", nil)
	if err != nil {
		m.Motivo = "endpoint invalido: " + err.Error()
		return m
	}
	resp, err := cli.Do(req)
	if err != nil {
		m.Motivo = "o motor nao responde em " + endpoint
		return m
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		m.Motivo = fmt.Sprintf("o motor devolveu HTTP %d", resp.StatusCode)
		return m
	}
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		m.Motivo = "resposta do motor ilegivel"
		return m
	}
	nomes := make([]string, 0, len(tags.Models))
	for _, t := range tags.Models {
		nomes = append(nomes, t.Name)
	}
	m.Modelos = nomes
	if len(nomes) == 0 {
		m.Motivo = "o motor respondeu mas nao tem modelos instalados"
		return m
	}

	desejado := strings.TrimSpace(modeloPedido)
	if desejado == "" {
		desejado = nomes[0]
	}
	for _, n := range nomes {
		if n == desejado {
			m.Disponivel = true
			m.Modelo = desejado
			return m
		}
	}
	m.Motivo = fmt.Sprintf("o modelo %q nao esta' instalado (disponiveis: %s)", desejado, strings.Join(nomes, ", "))
	return m
}

// inferirNoMotor devolve um `agent.InferFunc` ligado ao motor local.
//
// Temperature 0: uma avaliacao que compara respostas nao pode comparar duas respostas diferentes a mesma
// pergunta.
func inferirNoMotor(modelo string) agent.InferFunc {
	endpoint := ollamaEndpoint()
	cli := &http.Client{Timeout: evalRunnerTimeout}
	return func(prompt string) (string, error) {
		corpo, err := json.Marshal(map[string]any{
			"model":   modelo,
			"prompt":  prompt,
			"stream":  false,
			"options": map[string]any{"temperature": 0.0},
		})
		if err != nil {
			return "", err
		}
		resp, err := cli.Post(endpoint+"/api/generate", "application/json", bytes.NewReader(corpo))
		if err != nil {
			return "", fmt.Errorf("motor local: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("motor local: HTTP %d", resp.StatusCode)
		}
		var saida struct {
			Response string `json:"response"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&saida); err != nil {
			return "", fmt.Errorf("motor local: resposta ilegivel: %w", err)
		}
		return saida.Response, nil
	}
}

// executarAvaliacao corre a suite de uma divisao contra a persona pedida.
//
// Recusa ANTES de gastar se o motor nao estiver disponivel ou o modelo nao existir. Recusar depois de correr
// seria gastar o tempo do utilizador para lhe dizer o que ja' se sabia.
func executarAvaliacao(ctx context.Context, personaID, divisao, modelo string) (agent.EvalReport, motorLocal, error) {
	m := detectarMotor(ctx, modelo)
	if !m.Disponivel {
		return agent.EvalReport{}, m, fmt.Errorf("%w: %s", errMotorIndisponivel, m.Motivo)
	}
	persona, ok := agent.CatalogLookup(strings.TrimSpace(personaID))
	if !ok {
		return agent.EvalReport{}, m, fmt.Errorf("%w: persona %q", agent.ErrPersonaInvalid, personaID)
	}
	suite, ok := agent.EvalSuiteFor(strings.TrimSpace(divisao))
	if !ok {
		return agent.EvalReport{}, m, fmt.Errorf("%w: division %q has no suite", agent.ErrEvalSuiteInvalida, divisao)
	}
	if len(suite.Cases) > evalRunnerMaxCasos {
		return agent.EvalReport{}, m, fmt.Errorf("%w: suite has %d cases (max %d)",
			agent.ErrEvalSuiteInvalida, len(suite.Cases), evalRunnerMaxCasos)
	}
	rel := agent.Evaluate(persona, suite, inferirNoMotor(m.Modelo))
	return rel, m, nil
}
