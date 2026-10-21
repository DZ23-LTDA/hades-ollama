package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrEvolutionQRUnavailable indica que a instancia nao devolveu QR nem esta' ligada.
var ErrEvolutionQRUnavailable = errors.New("evolution: sem QR e sem instancia ligada")

// EvolutionConnectState e' o estado de emparelhamento da instancia Evolution.
//
// I0b: o QR de emparelhamento e' gerado pelo SERVIDOR EVOLUTION do utilizador, nao pelo Hades.
// Antes disto o painel nao tinha como o mostrar, e a UI dizia apenas "nao configurado" -- o
// utilizador ficava sem saber onde clicar para ligar o WhatsApp.
type EvolutionConnectState struct {
	// State e' "open" (ligada), "connecting" (a aguardar leitura do QR) ou "close".
	State string `json:"state"`
	// QR e' o data-URL do QR (ex.: "data:image/png;base64,..."), vazio quando ja' esta' ligada.
	QR string `json:"qr,omitempty"`
	// PairingCode e' o codigo alternativo, quando a instancia o fornece.
	PairingCode string `json:"pairing_code,omitempty"`
}

// configuredDiz se as credenciais minimas existem. O webhook_secret NAO e' exigido aqui:
// emparelhar nao depende de receber webhooks.
func (e *EvolutionAdapter) configured() bool {
	return strings.TrimSpace(e.config.BaseURL) != "" &&
		strings.TrimSpace(e.config.APIKey) != "" &&
		strings.TrimSpace(e.config.Instance) != ""
}

// doRequest faz um GET autenticado na instancia Evolution e devolve o corpo.
// A API key so' viaja no cabecalho e NUNCA entra em mensagens de erro (nao vaza em logs/UI).
func (e *EvolutionAdapter) doRequest(ctx context.Context, caminho string) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(e.config.BaseURL), "/")
	alvo := base + caminho
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alvo, nil)
	if err != nil {
		return nil, fmt.Errorf("evolution: pedido invalido: %w", err)
	}
	req.Header.Set("apikey", e.config.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("evolution: instancia inacessivel: %w", err)
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("evolution: falha a ler resposta: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("evolution: instancia respondeu %d", resp.StatusCode)
	}
	return corpo, nil
}

// Connect consulta a instancia: se ja' estiver ligada devolve State=open; caso contrario pede
// o QR ao endpoint de conexao e devolve-o como data-URL.
func (e *EvolutionAdapter) Connect(ctx context.Context) (EvolutionConnectState, error) {
	if !e.configured() {
		return EvolutionConnectState{}, errors.New("evolution: base_url, api_key e instance sao obrigatorios")
	}
	inst := url.PathEscape(strings.TrimSpace(e.config.Instance))

	// 1. A instancia esta' ligada?
	if corpo, err := e.doRequest(ctx, "/instance/connectionState/"+inst); err == nil {
		var estado struct {
			Instance struct {
				State string `json:"state"`
			} `json:"instance"`
			State string `json:"state"`
		}
		if json.Unmarshal(corpo, &estado) == nil {
			s := strings.ToLower(strings.TrimSpace(estado.Instance.State))
			if s == "" {
				s = strings.ToLower(strings.TrimSpace(estado.State))
			}
			if s == "open" {
				return EvolutionConnectState{State: "open"}, nil
			}
		}
	}

	// 2. Pedir o QR.
	corpo, err := e.doRequest(ctx, "/instance/connect/"+inst)
	if err != nil {
		return EvolutionConnectState{}, err
	}
	var resposta struct {
		Base64      string `json:"base64"`
		Code        string `json:"code"`
		PairingCode string `json:"pairingCode"`
		Qrcode      struct {
			Base64 string `json:"base64"`
			Code   string `json:"code"`
		} `json:"qrcode"`
		Instance struct {
			State string `json:"state"`
		} `json:"instance"`
	}
	if err := json.Unmarshal(corpo, &resposta); err != nil {
		return EvolutionConnectState{}, fmt.Errorf("evolution: resposta do QR ilegivel: %w", err)
	}

	qr := strings.TrimSpace(resposta.Base64)
	if qr == "" {
		qr = strings.TrimSpace(resposta.Qrcode.Base64)
	}
	pairing := strings.TrimSpace(resposta.PairingCode)
	estado := strings.ToLower(strings.TrimSpace(resposta.Instance.State))
	if estado == "" {
		estado = "connecting"
	}

	if qr == "" {
		// QR vazio e' uma resposta incompleta: dizer isso, em vez de mostrar um quadrado vazio.
		return EvolutionConnectState{State: estado, PairingCode: pairing}, ErrEvolutionQRUnavailable
	}
	// A API devolve o base64 cru; a UI precisa de um data-URL para <img>.
	if !strings.HasPrefix(qr, "data:") {
		qr = "data:image/png;base64," + qr
	}
	return EvolutionConnectState{State: estado, QR: qr, PairingCode: pairing}, nil
}
