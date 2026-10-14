package agent

import (
	"sync"
	"testing"
	"time"
)

// E-424 / ADV-66 (CRÍTICO) — "USO ÚNICO" DA APROVAÇÃO TEM DE RESISTIR A CONCORRÊNCIA.
//
// **MEDIDO por auditoria independente**: `Consume` era leitura-modificação-escrita **sem trava**
// (`loadFile` → checa `Status` → `saveFile`) ⇒ *dois leitores veem `approved`, ambos gravam `consumed` e **ambos
// executam***. Sonda do auditor, com o MESMO pedido e **UMA** decisão humana:
//
//	8 turnos simultâneos  → executadas=3 escritas_no_CRM=3, todas com o MESMO ApprovalID
//	8 Consume simultâneos → 20 de 20 rodadas com MAIS DE UM consumo bem-sucedido (máximo 5)
//
// **E a defesa existente não podia pegar**: `TestServiceAgentApprovalIsSingleUse` é **sequencial**, e
// `grep 'sync.WaitGroup|go func'` em `*service_agent*_test.go` dava **0 ocorrências** — *nenhum teste do worker
// usava concorrência*. *Prova sequencial de uma propriedade concorrente é prova de outra coisa.*
//
// ***A janela que o comentário de `service_agent_tools.go` dizia ter fechado continuava aberta — e foram precisos
// oito turnos ao mesmo tempo para que ela aparecesse.***
func TestConsumoDeAprovacaoEResistenteAConcorrencia(t *testing.T) {
	loja := NewServiceAgentApprovalStore(t.TempDir())
	agora := time.Now().UTC()

	pedido, err := loja.Request("org-a", ServiceAgentApproval{
		OrganizationID: "org-a",
		AgentID:        "support",
		ToolName:       "crm.opportunity.write",
		Fingerprint:    "fp-concorrencia",
		RequestedAt:    agora,
	})
	if err != nil {
		t.Fatalf("pedido: %v", err)
	}
	if _, err := loja.Decide("org-a", pedido.ID, true, "gerente-ana", "aprovado no teste", agora); err != nil {
		t.Fatalf("decisão: %v", err)
	}

	// OITO consumos SIMULTÂNEOS da MESMA aprovação. Exatamente UM pode vencer.
	const turnos = 8
	var (
		mu         sync.Mutex
		vitoriosos []string
		prontos    sync.WaitGroup
	)
	largada := make(chan struct{})
	prontos.Add(turnos)
	for i := range turnos {
		go func(indice int) {
			defer prontos.Done()
			<-largada // todos partem juntos: é a janela que interessa
			consumido, err := loja.Consume("org-a", pedido.ID, agora)
			if err == nil {
				mu.Lock()
				vitoriosos = append(vitoriosos, consumido.ID)
				mu.Unlock()
			}
		}(i)
	}
	close(largada)
	prontos.Wait()

	if len(vitoriosos) != 1 {
		t.Fatalf("ARMADILHA: %d de %d consumos simultâneos venceram para a MESMA aprovação %s — *uma decisão humana "+
			"autorizou %d execuções*. O uso único só é verdade se consumir e gravar forem o MESMO passo indivisível "+
			"(E-424). Vencedores: %v", len(vitoriosos), turnos, pedido.ID, len(vitoriosos), vitoriosos)
	}

	// E o estado final no disco é `consumed` — a prova de que a gravação acompanhou a decisão.
	finais, err := loja.ListApprovals("org-a", "", agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(finais) != 1 || finais[0].Status != ServiceAgentApprovalConsumed {
		t.Fatalf("esperava UMA aprovação consumida, veio %+v", finais)
	}
}

// E-425 (ALTO) — A IMPRESSÃO DIGITAL TEM DE AMARRAR `reason` E `agent`, CADA UM SOZINHO.
//
// **MEDIDO por DUAS auditorias independentes**, e o defeito estava em **dois lugares**:
//   - o **código**: `serviceAgentApprovalFingerprint` amarrava doze partes e **não** `Reason` nem `AgentID` ⇒
//     *aprovação pedida com `reason="MOTIVO-APROVADO"` executava um turno com `"MOTIVO-NAO-APROVADO"`, e a
//     conversa gravava literalmente o motivo **não aprovado***; e um pedido de `support` era consumido por um
//     turno de `receptionist`;
//   - o **teste**: `TestServiceAgentApprovalIsBoundToArguments` varia **título E valor ao mesmo tempo** ⇒
//     *a impressão digital difere se guardar **qualquer um** dos dois — é um **OU**, não um **E***. Removendo
//     **um componente por vez**, **9 de 9** passavam com a suíte inteira verde.
//
// ***Um teste que varia duas coisas ao mesmo tempo não mede nenhuma das duas.*** Aqui cada parte varia **sozinha**.
func TestImpressaoDigitalAmarraReasonEAgentSeparadamente(t *testing.T) {
	base := ServiceAgentToolRequest{
		OrganizationID: "org-a",
		ConversationID: "cv-1",
		ContactID:      "ct-1",
		SessionID:      "sess-1",
		Title:          "Oportunidade",
		ValueCents:     10000,
		OwnerID:        "ana",
		Assignee:       "bruno",
		CompanyID:      "co-1",
		Provider:       "whatsapp",
		Body:           "corpo",
		AgentID:        "support",
		Reason:         "motivo aprovado",
	}
	referencia := serviceAgentApprovalFingerprint(base, "inbox.handoff")

	// CONTROLE: pedido idêntico ⇒ impressão idêntica. *Sem isto, um hash que mudasse sempre passaria nos dois
	// casos abaixo e não estaria amarrando nada.*
	if igual := serviceAgentApprovalFingerprint(base, "inbox.handoff"); igual != referencia {
		t.Fatal("o mesmo pedido precisa produzir a MESMA impressão digital — o teste abaixo não provaria nada")
	}

	casos := []struct {
		nome   string
		mudar  func(ServiceAgentToolRequest) ServiceAgentToolRequest
		porque string
	}{
		{
			nome:   "reason",
			mudar:  func(r ServiceAgentToolRequest) ServiceAgentToolRequest { r.Reason = "outro motivo"; return r },
			porque: "`Reason` É o efeito: `inbox.handoff` grava o motivo na conversa",
		},
		{
			nome:   "agent",
			mudar:  func(r ServiceAgentToolRequest) ServiceAgentToolRequest { r.AgentID = "receptionist"; return r },
			porque: "`inbox.handoff` é declarado por 6 dos 7 agentes: reuso cruzado não é caso de canto",
		},
		{
			nome:   "value",
			mudar:  func(r ServiceAgentToolRequest) ServiceAgentToolRequest { r.ValueCents = 1; return r },
			porque: "o componente que o auditor removeu com a suíte inteira verde",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			alterado := caso.mudar(base)
			// UMA parte varia; todo o resto fica idêntico.
			if fingerprint := serviceAgentApprovalFingerprint(alterado, "inbox.handoff"); fingerprint == referencia {
				t.Fatalf("ARMADILHA: mudar SÓ `%s` não mudou a impressão digital — *a aprovação não está amarrada a "+
					"esse parâmetro*, e uma decisão dada para um valor autoriza outro (%s)", caso.nome, caso.porque)
			}
		})
	}
}
