package agent

import (
	"testing"
	"time"
)

// E-426 — O CONTROLE DE TENANT DENTRO DO ARQUIVO: DEFESA EM PROFUNDIDADE SIMÉTRICA.
//
// **MEDIDO por auditoria independente**: `Decide`, `Consume`, `FindApproved` e `FindDenied` **nunca** comparavam
// `approval.OrganizationID` — *enquanto `ListApprovals` e `AuditTrail` comparavam*. **A assimetria denuncia o
// descuido**: *defesa em profundidade que só existe num dos lados não é defesa, é sorte.*
//
// **Alcance honesto, declarado pelo próprio auditor**: o único criador de organização no repositório é
// `auth.go` (`"org_"+uuid`), sem os caracteres que `sanitizeIDComponent` colapsa ⇒ *a colisão de caminho
// (`org a` ≡ `org-a`) **não é alcançável hoje***. O defeito é (i) o controle **não existir** no ponto que decide e
// (ii) o teste que alegava isolamento entre tenants (`service_agent_approvals_test.go:194`) ser satisfeito **pela
// separação de arquivo** — *passaria com `Decide` sem qualquer noção de organização*.
//
// ***Este teste não depende da colisão de caminho:*** ele grava um registro **de outra organização** DENTRO do
// arquivo da primeira (*o que a colisão produziria*) e exige que a busca **não o devolva**. Se a checagem de
// `OrganizationID` for removida, este teste **falha**.
func TestBuscaDeAprovacaoNaoDevolveRegistroDeOutraOrganizacao(t *testing.T) {
	loja := NewServiceAgentApprovalStore(t.TempDir())
	agora := time.Now().UTC()

	// Um arquivo do "org-a" contendo, por dentro, um registro que se declara do "org-b" — exatamente o estado que
	// a colisão de `sanitizeIDComponent` produziria se fosse alcançável.
	if err := loja.Save("org-a", []ServiceAgentApproval{
		{
			ID:             "apr-intrusa",
			OrganizationID: "org-b",
			ToolName:       "crm.opportunity.write",
			Fingerprint:    "fp-intrusa",
			Status:         ServiceAgentApprovalApproved,
			RequestedAt:    agora,
		},
		{
			ID:             "apr-legitima",
			OrganizationID: "org-a",
			ToolName:       "crm.opportunity.write",
			Fingerprint:    "fp-legitima",
			Status:         ServiceAgentApprovalApproved,
			RequestedAt:    agora,
		},
	}); err != nil {
		t.Fatal(err)
	}

	// (a) `FindApproved` do org-a NÃO pode devolver o registro do org-b.
	achada, ok, err := loja.FindApproved("org-a", "fp-intrusa", agora)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatalf("ARMADILHA: `FindApproved(\"org-a\")` devolveu a aprovação %q, que se declara da organização %q — "+
			"*a busca não confere o tenant dentro do arquivo*", achada.ID, achada.OrganizationID)
	}

	// (b) CONTROLE: a legítima TEM de ser encontrada — *senão o teste acima passaria com a busca sempre vazia, que
	// é um defeito diferente e igualmente grave.*
	legitima, ok, err := loja.FindApproved("org-a", "fp-legitima", agora)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || legitima.OrganizationID != "org-a" {
		t.Fatalf("a aprovação legítima da própria organização precisa ser encontrada: ok=%v %+v", ok, legitima)
	}

	// (c) E o mesmo para a negativa.
	if err := loja.Save("org-a", []ServiceAgentApproval{{
		ID:             "apr-negada-intrusa",
		OrganizationID: "org-b",
		ToolName:       "crm.opportunity.write",
		Fingerprint:    "fp-negada-intrusa",
		Status:         ServiceAgentApprovalDenied,
		RequestedAt:    agora,
	}}); err != nil {
		t.Fatal(err)
	}
	if negada, ok, err := loja.FindDenied("org-a", "fp-negada-intrusa"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatalf("ARMADILHA: `FindDenied(\"org-a\")` devolveu a negativa %q, que se declara da organização %q",
			negada.ID, negada.OrganizationID)
	}
}
