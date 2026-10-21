package server

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// I1b parte 3: o cliente tem de DISTINGUIR os motivos de recusa. Um 500 para tudo obrigaria a
// adivinhar; estes casos fixam o contrato.
func TestStatusDoErroDeEdicao(t *testing.T) {
	casos := []struct {
		nome string
		err  error
		quer int
	}{
		{"CAS: alguem escreveu entretanto", agent.ErrRestauroConflitante, http.StatusConflict},
		{"conteudo identico", agent.ErrConteudoArtefactoInalterado, http.StatusConflict},
		{"grande demais", agent.ErrConteudoArtefactoGrande, http.StatusRequestEntityTooLarge},
		{"vazio", agent.ErrConteudoArtefactoVazio, http.StatusBadRequest},
		{"caminho fora do workspace", agent.ErrCaminhoForaDoWorkspace, http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			if got := statusDoErroDeEdicao(caso.err); got != caso.quer {
				t.Fatalf("status = %d, queria %d", got, caso.quer)
			}
			// Envolvido num fmt.Errorf com %w continua a ser reconhecido.
			envolvido := errors.Join(caso.err, errors.New("contexto"))
			if got := statusDoErroDeEdicao(envolvido); got != caso.quer {
				t.Fatalf("erro envolvido: status = %d, queria %d", got, caso.quer)
			}
		})
	}
}
