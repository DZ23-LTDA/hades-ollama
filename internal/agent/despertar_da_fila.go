package agent

import "time"

// DESPERTAR OS AGENTES A PARTIR DA FILA (E-639).
//
// UMA SÓ IMPLEMENTAÇÃO, DOIS CHAMADORES: a rota (`POST /wakeups`, E-638) e o TIQUE do agendador
// (E-639). Duplicar a decisão de enfileirar em dois sítios seria a mesma classe de erro que o
// E-626 corrigiu no filtro de inquilino: *uma mudança num lado deixa o outro a agir a menos, e
// quem age a menos não se nota.*

// FalhaDeDespertar diz que uma missão devia ter ido à fila e não foi.
type FalhaDeDespertar struct {
	AgenteID string
	MissaoID string
	Erro     string
}

// ResultadoDoDespertar é o que aconteceu: quem foi despertado, o que foi enfileirado e o que falhou.
type ResultadoDoDespertar struct {
	Despertares  []Despertar
	Enfileiradas []string
	Falhas       []FalhaDeDespertar
}

// DespertarAgentes coalesce os despertares das missões dadas e enfileira as que precisam de
// execução, devolvendo o desfecho. Não esconde falha: uma missão que não entrou na fila aparece em
// `Falhas` — *sucesso parcial declarado não é sucesso.*
//
// Recebe as missões (em vez de as ler) para que cada chamador aplique o SEU escopo antes: a rota
// passa as que o chamador pode ver; o tique passa as da organização que ele serve. O escopo fica
// onde é decidido, não escondido aqui.
func (r *Runtime) DespertarAgentes(missoes []Mission, agora time.Time) ResultadoDoDespertar {
	despertares := DespertaresDasMissoes(missoes, agora)
	resultado := ResultadoDoDespertar{Despertares: despertares}
	porID := make(map[string]Mission, len(missoes))
	for _, missao := range missoes {
		porID[missao.ID] = missao
	}
	for _, despertar := range despertares {
		for _, missaoID := range MissoesAExecutar(despertar, porID) {
			if _, err := r.EnqueueMission(missaoID); err != nil {
				resultado.Falhas = append(resultado.Falhas, FalhaDeDespertar{
					AgenteID: despertar.AgenteID,
					MissaoID: missaoID,
					Erro:     err.Error(),
				})
				continue
			}
			resultado.Enfileiradas = append(resultado.Enfileiradas, missaoID)
		}
	}
	return resultado
}
