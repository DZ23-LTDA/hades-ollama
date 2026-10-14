package agent

import (
	"fmt"
	"time"
)

// AGENDAMENTO — FATIA 5: INTERVALOS DENTRO DO TURNO E TURNO QUE CRUZA A MEIA-NOITE (E-328).
//
// Dois **limites declarados** nas fatias anteriores são fechados aqui:
//
//   - *"sem intervalos de almoço dentro do turno"* — um profissional que atende 09:00–18:00 **não** atende às 12:30, e
//     o sistema não pode oferecer esse horário ao cliente. ***Oferecer horário que a empresa não tem é a mesma classe
//     de erro que inventar expediente.***
//   - *"turnos que cruzam a meia-noite não são representados"* — barbearia que abre 22:00 e fecha 02:00 é rotina. Sem
//     isso, o motor teria de recusar todo agendamento de madrugada.
//
// O QUE ESTA FATIA **NÃO** FAZ (declarado): intervalos **não** cruzam a meia-noite (são janelas dentro do dia local);
// turno que cruza meia-noite **não** pode ser combinado com intervalo que passe da meia-noite — combinação que seria
// ambígua e por isso é recusada em vez de adivinhada.

// ErrSchedulingOnBreak recusa agendamento que cai dentro de um intervalo (almoço, descanso, manutenção).
var ErrSchedulingOnBreak = fmt.Errorf("%w: horário dentro de um intervalo do recurso", ErrSchedulingInvalid)

// SchedulingBreak é um intervalo dentro do turno. `ResourceID` vazio vale para todos os recursos.
type SchedulingBreak struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	ResourceID     string `json:"resource_id,omitempty"`
	Weekday        int    `json:"weekday"`
	StartsAtMinute int    `json:"starts_at_minute"`
	EndsAtMinute   int    `json:"ends_at_minute"`
	Label          string `json:"label,omitempty"`
}

// AddBreak registra um intervalo. Intervalo que cruza a meia-noite é RECUSADO (não se adivinha o dia).
func (s *SchedulingStore) AddBreak(br SchedulingBreak) (SchedulingBreak, error) {
	if s == nil {
		return SchedulingBreak{}, ErrSchedulingInvalid
	}
	if err := requireOrganization(br.OrganizationID); err != nil {
		return SchedulingBreak{}, fmt.Errorf("%w: %v", ErrSchedulingInvalid, err)
	}
	br.OrganizationID = consentScope(br.OrganizationID)
	if br.OrganizationID == "" || br.Weekday < 0 || br.Weekday > 6 {
		return SchedulingBreak{}, fmt.Errorf("%w: organização e dia da semana (0..6) são obrigatórios", ErrSchedulingInvalid)
	}
	if br.StartsAtMinute < 0 || br.EndsAtMinute > 24*60 || br.EndsAtMinute <= br.StartsAtMinute {
		return SchedulingBreak{}, fmt.Errorf(
			"%w: intervalo inválido (%d..%d) — intervalo que cruza a meia-noite não é representado e por isso é recusado em vez de adivinhado",
			ErrSchedulingInvalid, br.StartsAtMinute, br.EndsAtMinute)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(br.OrganizationID); err != nil {
		return SchedulingBreak{}, err
	}
	if br.ID == "" {
		br.ID = "brk_" + sanitizeIDComponent(fmt.Sprintf("%d-%d-%d-%s", br.Weekday, br.StartsAtMinute, br.EndsAtMinute, br.ResourceID))
	}
	org := s.byOrg[br.OrganizationID]
	org.Breaks = append(org.Breaks, br)
	if err := s.saveLocked(br.OrganizationID); err != nil {
		org.Breaks = org.Breaks[:len(org.Breaks)-1]
		return SchedulingBreak{}, err
	}
	return br, nil
}

// Breaks devolve os intervalos da organização (para exibição e para o próprio motor).
func (s *SchedulingStore) Breaks(organizationID string) ([]SchedulingBreak, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	organizationID = consentScope(organizationID)
	if err := s.loadLocked(organizationID); err != nil {
		return nil, err
	}
	return append([]SchedulingBreak(nil), s.byOrg[organizationID].Breaks...), nil
}

// breakCovers diz se o intervalo local [inicio, inicio+duracao) colide com algum intervalo configurado para o recurso.
//
// Colisão PARCIAL também conta: um atendimento que começa 11:50 e dura 30min invade o almoço das 12:00 — e é isso que
// faria o cliente chegar e esperar.
func breakCovers(org *schedulingOrg, resourceID string, weekday time.Weekday, inicioMinuto, duracaoMinutos int) (SchedulingBreak, bool) {
	fimMinuto := inicioMinuto + duracaoMinutos
	for _, br := range org.Breaks {
		if br.Weekday != int(weekday) {
			continue
		}
		if br.ResourceID != "" && br.ResourceID != resourceID {
			continue
		}
		if inicioMinuto < br.EndsAtMinute && br.StartsAtMinute < fimMinuto {
			return br, true
		}
	}
	return SchedulingBreak{}, false
}

// shiftDurationMinutes devolve a duração da janela em minutos, aceitando turno que cruza a meia-noite.
//
// Regra explícita: `ClosesAt > OpensAt` é janela no mesmo dia; `ClosesAt < OpensAt` atravessa a meia-noite. Igualdade é
// recusada no registro (janela de duração zero não é janela).
func shiftDurationMinutes(shift SchedulingShift) int {
	if shift.ClosesAt > shift.OpensAt {
		return shift.ClosesAt - shift.OpensAt
	}
	// Virada de meia-noite: do horário de abertura até o fim do dia, mais o começo do dia seguinte.
	return (24*60 - shift.OpensAt) + shift.ClosesAt
}

// shiftStartInstant converte "abertura + deslocamento" no INSTANTE local real, cuidando da virada de meia-noite.
//
// Sem isto, um turno 22:00→02:00 geraria 01:00 do MESMO dia — ou seja, um horário no passado do próprio turno.
func shiftStartInstant(dia time.Time, shift SchedulingShift, deslocamentoMinutos int, loc *time.Location) time.Time {
	total := shift.OpensAt + deslocamentoMinutos
	offsetDia := total / (24 * 60)
	minutoNoDia := total % (24 * 60)
	return time.Date(dia.Year(), dia.Month(), dia.Day(), 0, minutoNoDia, 0, 0, loc).AddDate(0, 0, offsetDia)
}

// shiftCoversMinute diz se um horário local está DENTRO do turno, considerando virada de meia-noite.
//
// 🚨 ADV-61 (6ª auditoria independente, MEDIDO): o ramo da CAUDA (`fimMinuto <= shift.ClosesAt`) era aplicado
// **sem distinguir o dia** ⇒ *01:00 de segunda era aceito pelo turno de segunda que só abre às 22:00* ✗, e o
// `Book` confirmava um cliente com a empresa fechada — divergindo da própria lista de `OpenSlots`
// (`Book(segunda 01:00)` aceito; `OpenSlots` não oferecia nada nesse instante).
//
// **A informação do dia já está no parâmetro**: o chamador `shiftCovering` procura primeiro o turno do dia
// corrente (`minuto`) e, se não cobrir, o do dia ANTERIOR que atravessa a meia-noite (**`minuto + 24*60`**,
// `scheduling_breaks.go:139/145`). ⇒ *a cauda só pode valer para o minuto do dia seguinte.*
func shiftCoversMinute(shift SchedulingShift, inicioMinuto, duracaoMinutos int) bool {
	fimMinuto := inicioMinuto + duracaoMinutos
	if shift.ClosesAt > shift.OpensAt {
		return inicioMinuto >= shift.OpensAt && fimMinuto <= shift.ClosesAt
	}
	// Turno que cruza a meia-noite.
	if inicioMinuto >= 24*60 {
		// Minuto do dia SEGUINTE (o chamador somou 1440): aqui vale a cauda — terminar antes do fechamento.
		return fimMinuto <= shift.ClosesAt+24*60
	}
	// Minuto do dia do PRÓPRIO turno: a cauda dele é amanhã, não hoje. Só a abertura conta.
	// *Sem esta linha, 01:00 de segunda caía na cauda do turno de segunda que abre às 22:00.*
	return inicioMinuto >= shift.OpensAt
}

// shiftCovering encontra o turno que COBRE um instante local, considerando a virada de meia-noite.
//
// Procura primeiro o turno do próprio dia; se não cobrir, procura o turno do dia ANTERIOR que atravessa a meia-noite.
// Exemplo: 01:00 de terça pode ser coberto pelo turno de segunda 22:00→02:00.
//
// 🚨 ADV-61 (6ª auditoria independente, MEDIDO): devolve **diaAnterior** porque *só esta função sabe de que dia veio o
// turno* — e é essa informação que o chamador precisa para converter o minuto local no minuto do turno. Sem ela,
// `shiftCoversMinute(MondayShift, 60, …)` não tem como distinguir "01:00 de segunda, fora do turno que abre às 22:00"
// de "01:00 de TERÇA, cauda legítima do turno de segunda". *Duas situações com o mesmo minuto e respostas opostas.*
func shiftCovering(org *schedulingOrg, resourceID string, local time.Time) (SchedulingShift, bool, bool) {
	minuto := local.Hour()*60 + local.Minute()
	if shift, achou := shiftFor(org, resourceID, local.Weekday()); achou && shiftCoversMinute(shift, minuto, 0) {
		return shift, false, true
	}
	anterior := local.AddDate(0, 0, -1)
	if shift, achou := shiftFor(org, resourceID, anterior.Weekday()); achou && shift.ClosesAt <= shift.OpensAt {
		// Turno que cruza: o minuto relevante é o do dia anterior (minuto + 1440) para comparar com a janela.
		if shiftCoversMinute(shift, minuto+(24*60), 0) {
			return shift, true, true
		}
	}
	return SchedulingShift{}, false, false
}
