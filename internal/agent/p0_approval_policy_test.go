package agent

import (
	"context"
	"strings"
	"testing"
)

func TestCompanyCycleRiskUsesExplicitIntentTokens(t *testing.T) {
	for _, test := range []struct {
		name  string
		cycle CompanyCycle
		want  RiskClass
	}{
		{"transfer", CompanyCycle{Objective: "transferir saldo para o fornecedor"}, RiskDestructive},
		{"acquire", CompanyCycle{Objective: "acquire annual license"}, RiskDestructive},
		{"post", CompanyCycle{Objective: "postar campanha aprovada"}, RiskExternalSideEffect},
		{"launch", CompanyCycle{Objective: "lançar produto"}, RiskExternalSideEffect},
		{"substring-not-risk", CompanyCycle{Objective: "anadsorption deployment-notes"}, RiskRead},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := companyCycleRisk(test.cycle); got != test.want {
				t.Fatalf("risk=%q, want %q", got, test.want)
			}
		})
	}
}

func TestMissionApprovalRejectsRequesterSelfApproval(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "escrever arquivo protegido",
		OrganizationID: LocalOrganizationID,
		ActorID:        "requester-a",
		Capabilities:   []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mission.Approvals) == 0 || mission.Approvals[0].RequestedBy != "requester-a" {
		t.Fatalf("approval requester not persisted: %+v", mission.Approvals)
	}
	_, err = runtime.DecideApprovalForActorCAS(mission.ID, mission.Approvals[0].ID, true, "self", "requester-a", LocalOrganizationID, mission.Version, mission.Approvals[0].Nonce)
	if err == nil || !strings.Contains(err.Error(), "approval requester cannot approve") {
		t.Fatalf("self approval error=%v", err)
	}
}

// No modo local de usuário único todo request é atribuído ao ator sintético
// LocalActorID. Como há só uma pessoa (que é o aprovador humano), ela PRECISA
// conseguir aprovar as missões de escrita do próprio agente — senão o agente
// nunca executa nada com efeito secundário no desktop. A separação de funções
// continua valendo para atores reais/nomeados (ver o teste acima).
func TestMissionApprovalAllowsLocalSingleUserSelfApproval(t *testing.T) {
	runtime, _, _ := setupTestSupervisorRuntime(t)
	mission, err := runtime.CreateMission(context.Background(), CreateMissionRequest{
		Objective:      "escrever arquivo protegido",
		OrganizationID: LocalOrganizationID,
		ActorID:        LocalActorID,
		Capabilities:   []string{"workspace:write"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(mission.Approvals) == 0 || mission.Approvals[0].RequestedBy != LocalActorID {
		t.Fatalf("approval requester not persisted: %+v", mission.Approvals)
	}
	updated, err := runtime.DecideApprovalForActorCAS(mission.ID, mission.Approvals[0].ID, true, "aprovado pelo usuário local", LocalActorID, LocalOrganizationID, mission.Version, mission.Approvals[0].Nonce)
	if err != nil {
		t.Fatalf("local single-user self approval should be allowed, got error=%v", err)
	}
	if updated.Approvals[0].Status != ApprovalApproved {
		t.Fatalf("approval not marked approved: %+v", updated.Approvals)
	}
}
