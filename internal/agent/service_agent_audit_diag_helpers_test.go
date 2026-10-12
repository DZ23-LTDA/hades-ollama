package agent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// Ferramentas de diagnóstico de DLP na trilha (E-193).
//
// Por que existem: `TestServiceAgentAuditKeepsContactTextOut` falhou UMA vez na CI e a mensagem antiga dizia apenas
// "o texto não pode ser gravado" — sem dizer QUAL campo. Uma falha que não nomeia o mecanismo não é investigável
// quando não se consegue reproduzir. Estes dois helpers fazem a falha entregar o campo e um identificador do texto
// **sem revelar** o conteúdo de terceiro no log.

// campoDaTrilhaComTexto devolve qual campo do primeiro registro carrega o texto, no formato `entries[i].campo`.
// Devolve "" quando a trilha está limpa (ou ilegível e sem o texto).
func campoDaTrilhaComTexto(conteudo, texto string) string {
	var arquivo serviceAgentAuditFile
	if err := json.Unmarshal([]byte(conteudo), &arquivo); err != nil {
		// Formato inesperado: não dá para nomear campo, mas se o texto está no arquivo bruto isso precisa aparecer —
		// silenciar aqui esconderia vazamento por mudança de formato.
		if contemTexto(conteudo, texto) {
			return "arquivo (formato não reconhecido)"
		}
		return ""
	}
	for indice, entrada := range arquivo.Entries {
		campos := []struct{ nome, valor string }{
			{"summary", entrada.Summary},
			{"reason", entrada.Reason},
			{"tool_name", entrada.ToolName},
			{"effect", entrada.Effect},
			{"status", entrada.Status},
			{"agent_id", entrada.AgentID},
			{"conversation_id", entrada.ConversationID},
			{"model_id", entrada.ModelID},
		}
		for _, campo := range campos {
			if contemTexto(campo.valor, texto) {
				return fmt.Sprintf("entries[%d].%s", indice, campo.nome)
			}
		}
	}
	return ""
}

// contemTexto cobre o texto completo E o prefixo de cartão usado nos fixtures: a invariante é "conteúdo de terceiro
// não entra", e o número de cartão é o caso concreto que os testes exercitam.
func contemTexto(valor, texto string) bool {
	return strings.Contains(valor, texto) || strings.Contains(valor, "4111")
}

// mascararTexto devolve comprimento + impressão digital, para localizar o campo no log SEM revelar o conteúdo.
func mascararTexto(texto string) string {
	soma := sha256.Sum256([]byte(texto))
	return fmt.Sprintf("%d caracteres, sha256:%x", len(texto), soma[:6])
}
