package agent

import (
	"testing"
)

// E-431 / ALTO-2 — TODAS as 14 PARTES DA IMPRESSÃO DIGITAL, CADA UMA VARIANDO **SOZINHA**.
//
// **MEDIDO por DUAS auditorias independentes**: o contrato declarado (`service_agent_approvals.go:30-31`,
// *"A aprovação é AMARRADA aos PARÂMETROS da ação (impressão digital de ferramenta + escopo + valores)"*) era
// defendido em **4 de 14** componentes ✗. A campanha de mutação do auditor removeu **um componente por vez** e
// **10 passaram com `internal/agent` INTEIRO (431 s) e `server` INTEIRO (47,6 s) verdes** ✗:
//
//	toolName · org · conversation · contact · title · owner · assignee · company · provider · body
//
// ⇒ *A aprovação dizia "amarrada aos parâmetros" e **dez deles podiam desaparecer sem que nada acusasse***.
//
// **A CAUSA DO TESTE, e ela é a classe nº 7**: `TestServiceAgentApprovalIsBoundToArguments` varia **título E valor
// AO MESMO TEMPO** ⇒ *a impressão digital difere se guardar **qualquer um** dos dois. É um **OU**, não um **E***.
// E o teste que eu acrescentei em E-425 (`TestImpressaoDigitalAmarraReasonEAgentSeparadamente`) pinou **3** partes
// (reason, agent, value) — *mais do que antes, e ainda **3 de 14*** ✗.
//
// ***Um teste que varia duas coisas ao mesmo tempo não mede nenhuma das duas.*** Aqui cada parte varia **sozinha**,
// e o **controle** garante que a igualdade também é medida (*senão um hash que mudasse sempre passaria em todos*).
//
// **DUAS AUSÊNCIAS SÃO DELIBERADAS, e ficam declaradas**: `Channel` e `Intent` **não** entram na impressão digital.
// *`Intent` é o texto do contato — dado de terceiro, não confiável, que descreve a **entrada** e não a **ação***; e
// `Channel` não muda **o que** será executado (*a ferramenta já está na impressão digital*). Isto não é descuido:
// é uma decisão, e está aqui para poder ser contestada.
func TestTodasAsPartesDaImpressaoDigitalSaoAmarradas(t *testing.T) {
	const ferramenta = "inbox.handoff"
	base := ServiceAgentToolRequest{
		OrganizationID: "org-a",
		AgentID:        "support",
		ConversationID: "cv-1",
		ContactID:      "ct-1",
		Channel:        "whatsapp",
		Intent:         "quero falar com alguém",
		Title:          "Plano avançado",
		ValueCents:     79900,
		OwnerID:        "ana",
		Assignee:       "bruno",
		Reason:         "cliente pediu",
		CompanyID:      "co-1",
		SessionID:      "sess-1",
		Provider:       "whatsapp",
		Body:           "corpo da mensagem",
	}
	referencia := serviceAgentApprovalFingerprint(base, ferramenta)

	// CONTROLE: pedido idêntico ⇒ impressão idêntica. *Sem isto, um hash que mudasse sempre passaria em todos os
	// casos abaixo sem amarrar nada.*
	if repetido := serviceAgentApprovalFingerprint(base, ferramenta); repetido != referencia {
		t.Fatal("pedido idêntico precisa produzir a MESMA impressão — sem isso, os casos abaixo não provam nada")
	}

	// As 14 partes, na ordem em que aparecem em `serviceAgentApprovalFingerprint`. *Cada linha muda UM campo.*
	casos := []struct {
		parte   string
		porque  string
		alterar func(*ServiceAgentToolRequest)
		ferram  string
	}{
		{"toolName", "sem ele, a aprovação de `inbox.assign` autoriza `crm.opportunity.write`",
			func(*ServiceAgentToolRequest) {}, "crm.opportunity.write"},
		{"org", "sem ele, aprovação de uma organização vale noutra",
			func(r *ServiceAgentToolRequest) { r.OrganizationID = "org-b" }, ferramenta},
		{"conversation", "sem ele, a aprovação vale noutra conversa",
			func(r *ServiceAgentToolRequest) { r.ConversationID = "cv-2" }, ferramenta},
		{"contact", "sem ele, a aprovação vale para outro contato",
			func(r *ServiceAgentToolRequest) { r.ContactID = "ct-2" }, ferramenta},
		{"title", "o título é o rótulo da oportunidade criada",
			func(r *ServiceAgentToolRequest) { r.Title = "Outro plano" }, ferramenta},
		{"value", "aprovar 100 não pode autorizar 10.000 — é a promessa nº 2 do contrato",
			func(r *ServiceAgentToolRequest) { r.ValueCents = 1 }, ferramenta},
		{"owner", "o dono da oportunidade muda quem responde por ela",
			func(r *ServiceAgentToolRequest) { r.OwnerID = "outra-pessoa" }, ferramenta},
		{"assignee", "o responsável muda quem executa",
			func(r *ServiceAgentToolRequest) { r.Assignee = "outra-pessoa" }, ferramenta},
		{"company", "a empresa é o escopo de empresa/social/voz",
			func(r *ServiceAgentToolRequest) { r.CompanyID = "co-2" }, ferramenta},
		{"session", "uma aprovação para uma sessão de voz não pode valer para outra",
			func(r *ServiceAgentToolRequest) { r.SessionID = "sess-2" }, ferramenta},
		{"provider", "o canal de origem muda o que o provedor executa",
			func(r *ServiceAgentToolRequest) { r.Provider = "telegram" }, ferramenta},
		{"body", "o corpo é o conteúdo que será gravado",
			func(r *ServiceAgentToolRequest) { r.Body = "outro corpo" }, ferramenta},
		{"reason", "`inbox.handoff` grava o motivo literalmente na conversa (E-425)",
			func(r *ServiceAgentToolRequest) { r.Reason = "outro motivo" }, ferramenta},
		{"agent", "`inbox.handoff` é declarado por 6 dos 7 agentes: reuso cruzado não é caso de canto (E-425)",
			func(r *ServiceAgentToolRequest) { r.AgentID = "receptionist" }, ferramenta},
	}

	vistas := map[string]bool{}
	for _, caso := range casos {
		t.Run(caso.parte, func(t *testing.T) {
			alterado := base
			caso.alterar(&alterado)
			ferramentaDoCaso := caso.ferram
			impressao := serviceAgentApprovalFingerprint(alterado, ferramentaDoCaso)
			if impressao == referencia {
				t.Fatalf("ARMADILHA: mudar SÓ `%s` não mudou a impressão digital — *a aprovação NÃO está amarrada a "+
					"esse parâmetro*, e uma decisão dada para um valor autoriza outro. %s\n"+
					"Se este teste falhou, o componente `%s` saiu da lista de `serviceAgentApprovalFingerprint`.",
					caso.parte, caso.porque, caso.parte)
			}
			vistas[caso.parte] = true
		})
	}

	// E a catraca do próprio teste: *as 14 partes do código têm de ter um caso.* Se alguém acrescentar uma parte
	// nova à impressão digital sem caso aqui, este número acusa — **é esta asserção que impede a lista de voltar a
	// crescer sem defesa.**
	if len(vistas) != 14 {
		t.Fatalf("a impressão digital tem 14 partes e este teste cobre %d — *uma parte nova sem caso aqui volta a "+
			"ficar sem defesa*", len(vistas))
	}
}
