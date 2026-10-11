package agent

import (
	"testing"
	"time"
)

// waitBudget devolve o orçamento de espera para eventos ASSÍNCRONOS em testes.
//
// O valor não é um limiar de correção: o teste afirma que o evento ACONTECE (o worker reivindica o job, a
// ferramenta começa, a notificação termina), não que ele acontece em 2 ou 3 segundos. Limites curtos
// transformam a velocidade da máquina — contêiner carregado, CI compartilhado, disco ocupado — em resultado
// de teste, e foi essa a família do defeito de E-142 (o caso que falhava 3 s antes da virada da hora) e da
// falha não reproduzida registrada em E-154.
//
// O orçamento é LIMITADO de propósito: uma espera que nunca termina precisa falhar rápido o bastante para
// dizer QUAL evento não aconteceu, em vez de estourar o timeout do pacote e esconder o nome do teste.
func waitBudget(t *testing.T) time.Duration {
	t.Helper()
	const (
		minimo = 5 * time.Second
		maximo = 30 * time.Second
		folga  = 20 * time.Second
	)
	orcamento := maximo
	if deadline, ok := t.Deadline(); ok {
		restante := time.Until(deadline) - folga
		if restante < orcamento {
			orcamento = restante
		}
	}
	if orcamento < minimo {
		orcamento = minimo
	}
	return orcamento
}
