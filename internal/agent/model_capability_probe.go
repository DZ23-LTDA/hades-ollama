package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SONDA DE CAPACIDADES NO ENDPOINT (E-326) — fecha o limite declarado em E-325.
//
// E-325 registrou o limite: *"não há sonda automática de endpoint nesta fatia; `WithCapabilities` existe para receber o
// resultado de uma sonda"*. Aqui a sonda existe, com três compromissos:
//
//  1. **SOMENTE LEITURA.** A sonda usa `POST /api/show`, que **descreve** um modelo. Ela **não** puxa, não copia, não
//     cria e não remove modelo nenhum: ***o Ollama local do usuário não é mexido para satisfazer uma descoberta.***
//     O teste prova isso registrando **todas** as rotas chamadas pelo servidor de teste.
//  2. **NÃO INVENTAR CAPACIDADE.** Se o endpoint não declarar `capabilities`, a resposta honesta é
//     `ErrModelCapabilityUnknown` — e **o registro não é alterado**. "Não declarado" não é "não suporta", e muito
//     menos "suporta".
//  3. **FALHAR FECHADO.** Endpoint fora do ar, 5xx, 404 ou timeout ⇒ erro explícito
//     (`ErrCapabilityProbeUnavailable`), sem capacidade inventada e sem registro alterado.

// ErrCapabilityProbeUnavailable: não foi possível obter a declaração do endpoint.
var ErrCapabilityProbeUnavailable = errors.New("model capability probe unavailable")

// ModelCapabilityProbe consulta um endpoint compatível com Ollama sobre um modelo.
type ModelCapabilityProbe struct {
	// BaseURL é a raiz do endpoint (ex.: http://127.0.0.1:11434).
	BaseURL string
	// Client permite injetar um cliente HTTP com timeout próprio; nil usa um padrão com timeout.
	Client *http.Client
	// Timeout vale quando Client é nil.
	Timeout time.Duration
	// Authorization, quando presente, é enviada como Bearer (Ollama Cloud). NUNCA é registrada em log nem em erro.
	Authorization string
}

func (p ModelCapabilityProbe) cliente() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

type ollamaShowRequest struct {
	Model string `json:"model"`
}

type ollamaShowResponse struct {
	// Capabilities é a declaração do endpoint (ex.: ["completion","tools","vision"]).
	Capabilities []string       `json:"capabilities"`
	ModelInfo    map[string]any `json:"model_info"`
	Details      struct {
		Family string `json:"family"`
	} `json:"details"`
}

// Probe consulta o endpoint e devolve as capacidades DECLARADAS por ele.
//
// Não escreve no registro: quem decide registrar é o chamador, com `WithCapabilities`.
func (p ModelCapabilityProbe) Probe(ctx context.Context, model string) (ModelCapabilities, error) {
	modelo := strings.TrimSpace(model)
	if modelo == "" {
		return ModelCapabilities{}, fmt.Errorf("%w: modelo vazio", ErrCapabilityProbeUnavailable)
	}
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return ModelCapabilities{}, fmt.Errorf("%w: endpoint não configurado", ErrCapabilityProbeUnavailable)
	}

	corpo, err := json.Marshal(ollamaShowRequest{Model: modelo})
	if err != nil {
		return ModelCapabilities{}, fmt.Errorf("%w: %v", ErrCapabilityProbeUnavailable, err)
	}
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/show", bytes.NewReader(corpo))
	if err != nil {
		return ModelCapabilities{}, fmt.Errorf("%w: %v", ErrCapabilityProbeUnavailable, err)
	}
	requisicao.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(p.Authorization) != "" {
		requisicao.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.Authorization))
	}

	resposta, err := p.cliente().Do(requisicao)
	if err != nil {
		// A mensagem do erro do cliente HTTP pode conter a URL; NÃO inclui credencial (nunca a colocamos na URL).
		return ModelCapabilities{}, fmt.Errorf("%w: %v", ErrCapabilityProbeUnavailable, err)
	}
	defer resposta.Body.Close()
	// Limite de leitura: um endpoint hostil não pode fazer o agente carregar um corpo enorme.
	conteudo, err := io.ReadAll(io.LimitReader(resposta.Body, 1<<20))
	if err != nil {
		return ModelCapabilities{}, fmt.Errorf("%w: %v", ErrCapabilityProbeUnavailable, err)
	}
	if resposta.StatusCode != http.StatusOK {
		return ModelCapabilities{}, fmt.Errorf("%w: HTTP %d", ErrCapabilityProbeUnavailable, resposta.StatusCode)
	}

	declarado := ollamaShowResponse{}
	if err := json.Unmarshal(conteudo, &declarado); err != nil {
		return ModelCapabilities{}, fmt.Errorf("%w: resposta ilegível: %v", ErrCapabilityProbeUnavailable, err)
	}
	if len(declarado.Capabilities) == 0 {
		// "Não declarado" NÃO é "não suporta": a resposta honesta é não saber.
		return ModelCapabilities{}, fmt.Errorf(
			"%w: o endpoint não declarou `capabilities` para %q — capacidade NÃO MEDIDA (não se inventa suporte nem ausência)",
			ErrModelCapabilityUnknown, modelo)
	}

	tem := func(alvo string) bool {
		for _, capacidade := range declarado.Capabilities {
			if strings.EqualFold(strings.TrimSpace(capacidade), alvo) {
				return true
			}
		}
		return false
	}
	capacidades := ModelCapabilities{
		Model:       modelo,
		ToolCalling: tem("tools"),
		Vision:      tem("vision"),
		Embeddings:  tem("embedding") || tem("embeddings"),
		Source:      CapabilitySourceMeasured,
	}
	for chave, valor := range declarado.ModelInfo {
		if strings.HasSuffix(strings.ToLower(chave), ".context_length") {
			if numero, ok := valor.(float64); ok && numero > 0 {
				capacidades.ContextTokens = int(numero)
			}
		}
	}
	capacidades.Note = fmt.Sprintf("declarado pelo endpoint em POST /api/show: [%s]", strings.Join(declarado.Capabilities, ", "))
	if !capacidades.ToolCalling {
		capacidades.Note += ". Não declara `tools`: usar em modo compatível (texto) ou escolher modelo com ferramentas."
	}
	return capacidades, nil
}

// ProbeAndRegister consulta o endpoint e registra a capacidade no registro, devolvendo o resultado.
//
// Só registra o que o endpoint DECLAROU: em qualquer dúvida, o registro fica como estava.
func (p ModelCapabilityProbe) ProbeAndRegister(ctx context.Context, registro *ModelCapabilityRegistry, model string) (ModelCapabilities, error) {
	if registro == nil {
		return ModelCapabilities{}, fmt.Errorf("%w: registro ausente", ErrCapabilityProbeUnavailable)
	}
	capacidades, err := p.Probe(ctx, model)
	if err != nil {
		return ModelCapabilities{}, err
	}
	if err := registro.WithCapabilities(capacidades); err != nil {
		return ModelCapabilities{}, err
	}
	return capacidades, nil
}
