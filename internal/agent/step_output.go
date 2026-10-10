package agent

import "strings"

// maxStepOutputExcerptRunes limita o trecho de saída de ferramenta publicado no
// evento step.succeeded. O teto fica ordens de grandeza abaixo de
// maxJSONStoreEventPayloadBytes (1 MiB): a trilha de execução precisa continuar
// pequena, legível e persistível em JSON.
const maxStepOutputExcerptRunes = 2000

// stepOutputTruncationMarker encerra um trecho cortado. O marcador é explícito
// para que a interface nunca apresente uma saída parcial como se fosse
// completa.
const stepOutputTruncationMarker = "\n[saída truncada]"

// stepOutputTextKeys define, em ordem fixa, as chaves textuais reconhecidas nas
// saídas de ferramenta (CLI governada, terminal, workspace e afins). A ordem é
// fixa porque a iteração de mapas em Go é aleatória: sem ela, o payload do
// evento variaria entre execuções idênticas da mesma missão.
var stepOutputTextKeys = []string{"stdout", "stderr", "output", "message", "summary", "text", "result"}

// stepOutputExcerpt devolve o trecho textual e limitado da saída de uma
// ferramenta para ser publicado no evento step.succeeded, tornando a saída
// visível na trilha de execução do chat.
//
// O valor recebido já passou por RedactValue (step.Result em runtime.go) e o
// próprio evento aplica RedactValue de novo antes de persistir: a redação por
// DLP é preservada nas duas camadas. Saídas exclusivamente estruturadas (por
// exemplo um screenshot em base64) não geram trecho, para não poluir a trilha
// com conteúdo binário nem com JSON interno.
func stepOutputExcerpt(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return truncateStepOutput(strings.TrimSpace(typed))
	case map[string]any:
		parts := make([]string, 0, len(stepOutputTextKeys))
		for _, key := range stepOutputTextKeys {
			text, ok := typed[key].(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(text); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		return truncateStepOutput(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

// truncateStepOutput corta o trecho por runas, e não por bytes, para não partir
// caracteres acentuados no meio. O marcador é anexado depois do corte,
// portanto não consome o orçamento do trecho.
func truncateStepOutput(text string) string {
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxStepOutputExcerptRunes {
		return text
	}
	return string(runes[:maxStepOutputExcerptRunes]) + stepOutputTruncationMarker
}
