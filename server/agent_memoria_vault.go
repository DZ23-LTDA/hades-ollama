package server

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ollama/ollama/internal/agent"
)

// ErrCorpoComCampoDesconhecido recusa um corpo que traga campos que o contrato nao declara.
var ErrCorpoComCampoDesconhecido = errors.New("body contains unknown fields")

// IdentidadeDaSessao devolve o ID do utilizador autenticado, lido do contexto que o `authMiddleware`
// preenche. E' a UNICA fonte de identidade: nao ha' parametro, cabecalho ou campo de corpo que a
// possa fornecer. Sem sessao, devolve ("", false) e o chamador recusa.
func IdentidadeDaSessao(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	valor, existe := c.Get("agent.user")
	if !existe {
		return "", false
	}
	utilizador, ok := valor.(agent.User)
	if !ok || strings.TrimSpace(utilizador.ID) == "" {
		return "", false
	}
	return strings.TrimSpace(utilizador.ID), true
}

// OrganizacaoDaSessao devolve o ID da organizacao do contexto. Tambem nunca vem do corpo: o
// `authMiddleware` verifica o cabecalho `X-Ollama-Organization` CONTRA a organizacao do token antes
// de o publicar aqui.
func OrganizacaoDaSessao(c *gin.Context) (string, bool) {
	if c == nil {
		return "", false
	}
	valor, existe := c.Get("agent.organization")
	if !existe {
		return "", false
	}
	organizacao, ok := valor.(agent.Organization)
	if !ok || strings.TrimSpace(organizacao.ID) == "" {
		return "", false
	}
	return strings.TrimSpace(organizacao.ID), true
}

// DecodificarCorpoEstrito descodifica o corpo recusando campos que o contrato nao declara.
//
// PORQUE E' ESTRITO: se um corpo pode trazer `identidade`, `user_id` ou `tenant`, mais cedo ou mais
// tarde um deles sera' lido como se fosse sessao. Ignorar o campo em silencio deixa o pedido passar
// com uma intencao que o servidor nao honrou; recusar diz ao chamador que o contrato nao tem esse
// campo. *Um campo ignorado em silencio e' uma porta que parece fechada.*
func DecodificarCorpoEstrito(corpo io.Reader, destino any) error {
	if corpo == nil {
		return ErrCorpoComCampoDesconhecido
	}
	descodificador := json.NewDecoder(corpo)
	descodificador.DisallowUnknownFields()
	if err := descodificador.Decode(destino); err != nil {
		return err
	}
	// Um segundo valor JSON no mesmo corpo tambem e' recusado.
	if err := descodificador.Decode(&struct{}{}); err != io.EOF {
		return ErrCorpoComCampoDesconhecido
	}
	return nil
}
