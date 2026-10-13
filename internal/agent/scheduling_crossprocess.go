package agent

import (
	"os"
	"path/filepath"
	"time"
)

// AGENDAMENTO — FATIA 4: CORREÇÃO ENTRE PROCESSOS (V2.2 incremento I / E-323).
//
// O QUE ESTAVA FALTANDO (e eu havia declarado como limite nas fatias anteriores): o núcleo protegia contra
// concorrência **dentro do mesmo processo** (mutex). Dois processos — ou duas instâncias do servidor — com o mesmo
// diretório de dados **não** se viam: cada um tinha o seu cache em memória e a sua própria trava. O V2.2 exige
// *"transações e locks/constraints contra double booking sob concorrência"*, e ***um cliente confirmado em horário já
// ocupado por OUTRO processo é o mesmo defeito***.
//
// A CORREÇÃO TEM DUAS PARTES, e a segunda é a que costuma faltar:
//
//  1. **LOCK DE ARQUIVO por organização** — reusa `withFileLock`, o mesmo primitivo que o outbox de push já usa, com
//     implementação real em Windows e Unix. Nada de lock novo inventado.
//  2. **RECARREGAR SOB O LOCK** — não basta travar: é preciso **relê o estado do disco depois de obter a trava**,
//     porque o outro processo pode ter gravado entre a nossa última leitura e agora. ***Lock sem releitura protege
//     contra escrita simultânea, mas não contra decisão tomada com dado velho.***
//
// ORDEM DE TRAVA (consistente em todos os caminhos de escrita): **arquivo → mutex**. Inverter em um só caminho
// abriria deadlock; por isso as funções de leitura não tomam o lock de arquivo.
//
// LIMITE DECLARADO: **escrita** é serializada e reconferida entre processos; **leitura** é eventualmente consistente
// (uma consulta pode ver o estado anterior até que a próxima escrita recarregue). Para disponibilidade de horários em
// tempo real entre instâncias seria preciso um lock por consulta ou um banco com transação — decisão de arquitetura,
// não um detalhe escondido.

// withOrganizationLock executa `run` com a trava de arquivo da organização e com o estado RECARREGADO do disco.
func (s *SchedulingStore) withOrganizationLock(organizationID string, run func() error) error {
	if s == nil {
		return ErrSchedulingInvalid
	}
	if s.root == "" {
		// Sem persistência configurada não existe estado compartilhado a proteger (e não se inventa arquivo de lock).
		return run()
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	// `withFileLock` tem a MESMA assinatura em Windows e Unix: (path, func() error) error.
	return withFileLock(filepath.Join(s.root, sanitizeIDComponent(organizationID)+".lock"), func() error {
		// RECARREGAR sob a trava: o processo vizinho pode ter gravado desde a nossa última leitura.
		s.invalidateCache(organizationID)
		return run()
	})
}

// invalidateCache descarta o cache em memória da organização, forçando releitura no próximo acesso.
func (s *SchedulingStore) invalidateCache(organizationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded[organizationID] = false
}

// Book cria o agendamento recusando sobreposição no mesmo recurso, **inclusive contra outra instância do servidor**.
//
// A validação e a gravação acontecem sob a trava de arquivo da organização, com o estado recarregado do disco — é
// isto que impede que dois processos confirmem o mesmo horário.
func (s *SchedulingStore) Book(request BookRequest) (SchedulingAppointment, error) {
	var agendamento SchedulingAppointment
	err := s.withOrganizationLock(consentScope(request.OrganizationID), func() error {
		var errInterno error
		agendamento, errInterno = s.bookLocked(request)
		return errInterno
	})
	return agendamento, err
}

// Cancel liberta o horário sob a mesma trava (e recarregando o estado do disco).
func (s *SchedulingStore) Cancel(organizationID, appointmentID, reason string) (SchedulingAppointment, error) {
	var cancelado SchedulingAppointment
	err := s.withOrganizationLock(consentScope(organizationID), func() error {
		var errInterno error
		cancelado, errInterno = s.cancelLocked(organizationID, appointmentID, reason)
		return errInterno
	})
	return cancelado, err
}

// Reschedule move um agendamento sob a mesma trava de arquivo da organização.
func (s *SchedulingStore) Reschedule(organizationID, appointmentID string, newStart time.Time) (SchedulingAppointment, error) {
	var movido SchedulingAppointment
	err := s.withOrganizationLock(consentScope(organizationID), func() error {
		var errInterno error
		movido, errInterno = s.rescheduleLocked(organizationID, appointmentID, newStart)
		return errInterno
	})
	return movido, err
}
