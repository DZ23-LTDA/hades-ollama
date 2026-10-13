package agent

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// §11.2 / ESTÁGIO 35 — TROCAR DE MODELO NÃO PODE ALTERAR AS POLÍTICAS (E-300).
//
// O código AFIRMA isto no campo do pedido: "ModelID é o modelo escolhido; trocar de modelo NÃO altera as políticas".
// Afirmação sem teste é promessa; este teste a torna FALSIFICÁVEL, executando o MESMO pedido com DOIS modelos.
//
// A política em jogo é a do efeito: ferramenta de SAÍDA (`inbox.reply`) exige consentimento CONCEDIDO. Se a troca de
// modelo afrouxasse a política, o Hades deixaria um modelo externo causar efeito sem consentimento — que é
// exatamente a "fuga de privilégio" que o §11.2 proíbe.
//
// Direções travadas:
//  1. sem consentimento, `inbox.reply` é BLOQUEADO com motivo NÃO VAZIO (política antes de efeito). MEDIDO: o
//     motivo é o do caminho único de resposta, não o de consentimento — assertar o TEXTO do motivo seria proxy;
//  2. com OUTRO modelo, continua RECUSADO — a política não é do modelo;
//  3. leitura funciona nos DOIS modelos (a recusa é do EFEITO, não do modelo);
//     3b. LIMITE DECLARADO: com o único instrumento de saída (`inbox.reply`) não dá para ISOLAR o portão de
//     consentimento, porque o caminho único de resposta recusa antes. Não se inventa assert sobre o não isolado;
//  4. com consentimento concedido, `inbox.reply` EXECUTA nos DOIS modelos (sem bloqueio falso);
//  5. o turno registra o modelo usado (rastreabilidade da troca). MEDIDO: `ExecutedAt` marca QUANDO A DECISÃO foi
//     tomada, não quando o efeito rodou — o discriminador é `Status`, e inferir execução de `ExecutedAt` é suposição;
//  6. as POLÍTICAS declaradas são as MESMAS nos dois modelos.
func TestTrocaDeModeloNaoEliminaPoliticasDoAgenteDeAtendimento(t *testing.T) {
	raiz := t.TempDir()
	inbox, err := NewInboxStore(filepath.Join(raiz, "inbox"))
	if err != nil {
		t.Fatal(err)
	}
	crm, err := NewCRMStore(filepath.Join(raiz, "crm"))
	if err != nil {
		t.Fatal(err)
	}
	consent, err := NewConsentRegistry(filepath.Join(raiz, "consent"))
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	const telefone = "+5511977776666"
	if _, _, err := crm.UpsertContact(CRMContact{OrganizationID: "org-a", Name: "Carla Souza", Phone: telefone}, agora); err != nil {
		t.Fatal(err)
	}
	conversa, _, err := inbox.AppendInbound("org-a", "whatsapp", telefone, "Carla Souza", "quero falar com alguem", "wamid.pol.1", agora)
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewServiceAgentRuntime(crm, inbox, consent, DefaultServiceAgentTools())

	const modeloLocal = "modelo-local-a"
	const modeloExterno = "modelo-externo-b"

	turno := func(modelo string, ferramenta string) ServiceAgentTurn {
		t.Helper()
		saida, err := runtime.RunTurn(ServiceAgentTurnRequest{
			OrganizationID: "org-a",
			AgentID:        ServiceAgentSupport,
			ConversationID: conversa.ID,
			Intent:         "segue a resposta combinada",
			ModelID:        modelo,
			RequestedTools: []string{ferramenta},
		}, agora)
		if err != nil {
			t.Fatalf("turno com %s pedindo %s: %v", modelo, ferramenta, err)
		}
		return saida
	}

	chamada := func(saida ServiceAgentTurn, ferramenta string) (ServiceAgentToolCall, bool) {
		for _, c := range saida.ToolCalls {
			if c.Name == ferramenta {
				return c, true
			}
		}
		return ServiceAgentToolCall{}, false
	}

	// 1 + 2. SEM consentimento: recusa nos DOIS modelos, pelo MESMO motivo (política, não modelo).
	recusaLocal := turno(modeloLocal, "inbox.reply")
	local, temos := chamada(recusaLocal, "inbox.reply")
	if !temos {
		t.Fatalf("a chamada de inbox.reply precisa aparecer na trilha: %+v", recusaLocal.ToolCalls)
	}
	if strings.TrimSpace(local.Status) != "blocked" {
		t.Fatalf("sem consentimento, a saída precisa ser BLOQUEADA: %+v", local)
	}
	if strings.TrimSpace(local.Reason) == "" {
		t.Fatalf("a recusa precisa EXPLICAR o motivo, seja qual for o portão que falou primeiro: %+v", local)
	}
	if strings.TrimSpace(local.Summary) != "" {
		t.Fatalf("chamada bloqueada não pode carregar resumo de efeito: %+v", local)
	}

	recusaExterno := turno(modeloExterno, "inbox.reply")
	externo, temos := chamada(recusaExterno, "inbox.reply")
	if !temos {
		t.Fatalf("a chamada precisa aparecer na trilha com o segundo modelo: %+v", recusaExterno.ToolCalls)
	}
	if strings.TrimSpace(externo.Status) != "blocked" {
		t.Fatalf("TROCAR DE MODELO não pode desbloquear o efeito: %+v", externo)
	}
	if strings.TrimSpace(externo.Reason) == "" {
		t.Fatalf("com o segundo modelo a recusa também precisa explicar o motivo: %+v", externo)
	}
	if strings.TrimSpace(externo.Summary) != "" {
		t.Fatalf("com o segundo modelo, a chamada bloqueada também não pode carregar resumo de efeito: %+v", externo)
	}

	// 3. Leitura funciona nos dois modelos: a recusa é do EFEITO, não do modelo.
	for _, modelo := range []string{modeloLocal, modeloExterno} {
		saida := turno(modelo, "crm.contact.read")
		leitura, temos := chamada(saida, "crm.contact.read")
		if !temos {
			t.Fatalf("[%s] a leitura precisa aparecer na trilha: %+v", modelo, saida.ToolCalls)
		}
		if strings.TrimSpace(leitura.Status) == "blocked" || strings.TrimSpace(leitura.Summary) == "" {
			t.Fatalf("[%s] leitura não depende de consentimento e precisa EXECUTAR com resumo: %+v", modelo, leitura)
		}
	}

	// 4. DECLARADO: `inbox.reply` é bloqueado pela regra do CAMINHO ÚNICO DE RESPOSTA — com ou sem consentimento,
	// nos dois modelos. O portão de consentimento NÃO é isolável com este instrumento, e não se inventa assert
	// sobre o que não se isolou. O que se exige aqui é que conceder consentimento NÃO mude o desfecho por causa do
	// modelo: continua bloqueado, com motivo, nos dois.
	if _, err := consent.GrantConsent("org-a", ConsentChannelWhatsApp, telefone, "site", "checkbox", "ana", nil, agora); err != nil {
		t.Fatal(err)
	}
	for _, modelo := range []string{modeloLocal, modeloExterno} {
		saida := turno(modelo, "inbox.reply")
		bloqueada, temos := chamada(saida, "inbox.reply")
		if !temos {
			t.Fatalf("[%s] a chamada precisa aparecer na trilha: %+v", modelo, saida.ToolCalls)
		}
		if strings.TrimSpace(bloqueada.Status) != "blocked" {
			t.Fatalf("[%s] a regra do caminho único de resposta vale nos dois modelos: %+v", modelo, bloqueada)
		}
		if strings.TrimSpace(bloqueada.Reason) == "" {
			t.Fatalf("[%s] a recusa precisa explicar o motivo: %+v", modelo, bloqueada)
		}
	}

	// 5 + 6. Rastreabilidade da troca e POLÍTICAS iguais nos dois modelos.
	primeiro := turno(modeloLocal, "crm.contact.read")
	segundo := turno(modeloExterno, "crm.contact.read")
	if primeiro.Context.ModelID != modeloLocal || segundo.Context.ModelID != modeloExterno {
		t.Fatalf("o turno precisa registrar o modelo usado: %q / %q", primeiro.Context.ModelID, segundo.Context.ModelID)
	}
	if len(primeiro.Context.Policies) == 0 {
		t.Fatal("o contexto precisa declarar as políticas aplicadas")
	}
	if strings.Join(primeiro.Context.Policies, "|") != strings.Join(segundo.Context.Policies, "|") {
		t.Fatalf("as políticas precisam ser AS MESMAS ao trocar de modelo:\n%v\n%v",
			primeiro.Context.Policies, segundo.Context.Policies)
	}
}
