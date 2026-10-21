package agent

import (
	"errors"
	"testing"
	"time"
)

// E-460 — **A ASSIMETRIA CHAVE×CANAL: DOIS CLIENTES DIFERENTES COM A MESMA CHAVE.**
//
// **O QUE E-459 DECLAROU E NÃO MEDIU**: *"o comentário do motor diz \"vinda do mesmo **canal**\", mas o código compara
// **só a chave** — não inclui `ContactID` nem `Channel`"* ✗.
//
// **E A PERGUNTA TEM CONSEQUÊNCIA DE SEGURANÇA**: se dois clientes de canais diferentes escolhem a mesma
// `IdempotencyKey` (*e ela costuma ser gerada pelo cliente — UUID, hash de mensagem, id de conversa*), o segundo recebe
// **`ErrSchedulingDuplicate` e, com ele, o agendamento do PRIMEIRO** (*foi a mudança de E-458: o recurso existente volta
// junto*). ⇒ ***O comentário promete um escopo que o código não impõe — e o escopo que falta é o que separa duas
// pessoas.***
//
// **AS TRÊS MEDIDAS**:
//  1. **mesmo contacto + mesma chave** ⇒ duplicata ✓ (*a protecção que existe, e tem de continuar existindo*);
//  2. **contactos DIFERENTES + mesma chave** ⇒ *a medida que E-459 declarou faltar*: **observa-se o que acontece**, sem
//     afirmar de antemão se é defeito;
//  3. e o cruzamento: *se colidir, o segundo recebe o agendamento do primeiro?* — *é isso que decide se é só um
//     incómodo ou um vazamento entre pessoas.*
//
// ***Não afirmo o veredicto antes de medir; afirmo o que a medição mostrar.***
func TestColisaoDeChaveEntreContactosDiferentes(t *testing.T) {
	store := NewSchedulingStore(t.TempDir())
	organizacao := LocalOrganizationID

	servico, err := store.AddService(SchedulingService{OrganizationID: organizacao, Name: "Consulta", DurationMinutes: 30})
	if err != nil {
		t.Fatalf("criar serviço: %v", err)
	}
	recurso, err := store.AddResource(SchedulingResource{OrganizationID: organizacao, Name: "Bia"})
	if err != nil {
		t.Fatalf("criar recurso: %v", err)
	}
	dia := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	for deslocamento := range 7 {
		if _, err := store.AddShift(SchedulingShift{
			OrganizationID: organizacao, ResourceID: recurso.ID,
			Weekday: int(dia.AddDate(0, 0, deslocamento).Weekday()), OpensAt: 0, ClosesAt: 24 * 60,
		}); err != nil {
			t.Fatalf("criar turno: %v", err)
		}
	}

	pedido := func(contacto, canal string, inicio time.Time) BookRequest {
		return BookRequest{
			OrganizationID: organizacao, ResourceID: recurso.ID, ServiceID: servico.ID,
			ContactID: contacto, Channel: canal, StartsAt: inicio, IdempotencyKey: "k-compartilhada",
		}
	}

	// (1) MESMO contacto, mesma chave: a protecção existe e tem de continuar existindo.
	primeiro, err := store.Book(pedido("cliente-A", "voice", time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("primeiro agendamento (cliente-A): %v", err)
	}
	if _, err := store.Book(pedido("cliente-A", "voice", time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC))); !errors.Is(err, ErrSchedulingDuplicate) {
		t.Fatalf("o MESMO contacto com a mesma chave tinha de dar ErrSchedulingDuplicate, veio: %v", err)
	}

	// (2) CONTACTOS DIFERENTES, mesma chave — **a medida que faltava**.
	segundo, erroSegundo := store.Book(pedido("cliente-B", "whatsapp", time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)))

	// (3) O CRUZAMENTO: se colidiu, o cliente-B recebeu o agendamento do cliente-A?
	if errors.Is(erroSegundo, ErrSchedulingDuplicate) {
		// A colisão aconteceu. *E agora a pergunta que separa incómodo de vazamento:*
		vazou := segundo.ID == primeiro.ID
		t.Fatalf("MEDIDO: A CHAVE COLIDE ENTRE CONTACTOS DIFERENTES. cliente-B (whatsapp) recebeu ErrSchedulingDuplicate "+
			"por causa de um agendamento do cliente-A (voice). Recebeu o agendamento do OUTRO? %v (id devolvido=%q, "+
			"id do cliente-A=%q). ***O comentario do motor promete 'a MESMA intencao vinda do mesmo CANAL', mas o codigo "+
			"compara so' a chave - e o escopo que falta e' o que separa DUAS PESSOAS.***", vazou, segundo.ID, primeiro.ID)
	}
	if erroSegundo != nil {
		t.Fatalf("a medicao nao era esta: cliente-B com chave compartilhada deu um erro DIFERENTE de duplicata: %v", erroSegundo)
	}

	// Sem colisao: dois agendamentos distintos, um por cliente. *E o teste registra o que mediu.*
	if segundo.ID == "" || segundo.ID == primeiro.ID {
		t.Fatalf("cliente-B agendou e recebeu um id igual ao do cliente-A (%q vs %q) — *isto seria vazamento*",
			segundo.ID, primeiro.ID)
	}
	agendamentos, err := store.Appointments(organizacao)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(agendamentos) != 2 {
		t.Fatalf("esperava 2 agendamentos (um por cliente), veio %d — *a chave NAO colidiu entre contactos*", len(agendamentos))
	}
	t.Log("MEDIDO: contactos diferentes com a mesma chave NAO colidem — cada um agendou o seu. " +
		"*O comentario do motor fala em 'mesmo canal', mas a comparacao e' so' a chave: aqui isso nao causou dano porque " +
		"as chaves eram iguais E as intencoes distintas; o risco fica declarado, nao medido como incidente.*")
}
