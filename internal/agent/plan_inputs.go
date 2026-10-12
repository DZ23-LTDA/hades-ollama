package agent

import (
	"fmt"
	"strings"
)

// ValidatePlanStepInputs recusa, NO PLANEJAMENTO, etapa cujo input não traz os campos que a
// própria ferramenta declara como obrigatórios.
//
// Por que existe: um modelo real pode devolver um plano sintaticamente válido e
// semanticamente inútil — observado em teste opt-in, `qwen2.5:7b` planejou `workspace.read`
// com `input` vazio. O runtime recusava corretamente (fail-closed), mas com a mensagem
// "anchored workspace path is required", que NÃO diz qual etapa nem qual campo faltou.
// Um plano inválido nunca deve virar erro obscuro na execução: deve ser recusado aqui,
// com a informação que o operador precisa para corrigir o prompt, a política ou o modelo.
func ValidatePlanStepInputs(step Step, descriptor ToolDescriptor) error {
	faltando := make([]string, 0, len(descriptor.RequiredInputs))
	for _, campo := range descriptor.RequiredInputs {
		if !planInputPresente(step.Input, campo) {
			faltando = append(faltando, campo)
		}
	}
	if len(faltando) == 0 {
		return nil
	}
	return fmt.Errorf("plano invalido: etapa %q (%s) exige input %s", step.ID, step.Kind, strings.Join(faltando, ", "))
}

// planInputPresente considera ausente o campo inexistente, nulo, ou string em branco: um
// caminho vazio não ancora nada, então tratá-lo como presente só empurraria o erro adiante.
func planInputPresente(input map[string]any, campo string) bool {
	valor, ok := input[campo]
	if !ok || valor == nil {
		return false
	}
	if texto, ok := valor.(string); ok {
		return strings.TrimSpace(texto) != ""
	}
	return true
}
