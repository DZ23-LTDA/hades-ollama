package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
)

// Classes de erro do embedder. Elas existem para separar duas situações que NÃO são a mesma coisa:
//
//   - TRANSPORTE indisponível (serviço fora, modelo não carregado): o chamador pode cair numa reserva
//     LITERAL e DIZER que caiu — melhor que devolver "nenhuma memória" e esconder que o caminho semântico
//     morreu;
//   - resposta/configuração INVÁLIDA (sem cliente, sem modelo, vetor vazio): isso é defeito, não
//     indisponibilidade, e não deve ser mascarado por reserva silenciosa.
var (
	ErrEmbedderUnavailable = errors.New("embedder indisponível")
	ErrEmbedderInvalid     = errors.New("embedder devolveu resposta inesperada")
)

type OllamaEmbedder struct {
	Client *api.Client
	Model  string
}

func (e OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if e.Client == nil || strings.TrimSpace(e.Model) == "" {
		return nil, fmt.Errorf("%w: cliente ou modelo não configurado", ErrEmbedderInvalid)
	}
	response, err := e.Client.Embed(ctx, &api.EmbedRequest{Model: e.Model, Input: text, Truncate: boolPointer(true)})
	if err != nil {
		// O erro cru do cliente carrega endereço e detalhe de transporte; a classe é o que o chamador usa.
		return nil, fmt.Errorf("%w: destino não respondeu", ErrEmbedderUnavailable)
	}
	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("%w: nenhum vetor", ErrEmbedderInvalid)
	}
	return response.Embeddings[0], nil
}

func boolPointer(value bool) *bool { return &value }
