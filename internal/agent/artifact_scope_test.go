package agent

import (
	"errors"
	"strings"
	"testing"
)

// E-532 (Estagio 47) - o teste do CONTRATO do modelo canonico de artefacto.
//
// ***NAO TESTA CAMPOS. TESTA REGRAS.*** *Um teste que repetisse a struct so' provaria que a copiei duas vezes.*

func TestValidateArtifactScopeRecusaArtefactoSemTenant(t *testing.T) {
	// (a) O caso que o Estagio 47 nao pode permitir: artefacto sem organizacao.
	semTenant := ArtifactManifest{ID: "art_1", MissionID: "m1", SHA256: "abc"}
	err := ValidateArtifactScope(semTenant)
	if !errors.Is(err, ErrArtifactSemTenant) {
		t.Fatalf("ARMADILHA: um artefacto SEM tenant foi aceite (err=%v) - *«sem tenant» e' exactamente o estado em "+
			"que uma busca atravessa organizacoes sem ninguem notar*", err)
	}

	// (b) O controlo: com tenant, id e sha256, passa.
	completo := ArtifactManifest{ID: "art_1", MissionID: "m1", TenantID: "acme", SHA256: "abc", Version: 1}
	if err := ValidateArtifactScope(completo); err != nil {
		t.Fatalf("manifesto legitimo recusado: %v", err)
	}
}

func TestValidateArtifactScopeRecusaVersaoOuRetencaoNegativas(t *testing.T) {
	base := ArtifactManifest{ID: "art_1", TenantID: "acme", SHA256: "abc"}

	negativa := base
	negativa.Version = -1
	if err := ValidateArtifactScope(negativa); err == nil {
		t.Fatal("ARMADILHA: versao negativa aceite")
	}

	retencao := base
	retencao.RetentionDays = -5
	if err := ValidateArtifactScope(retencao); err == nil {
		t.Fatal("ARMADILHA: retencao negativa aceite")
	}

	// *Zero e' aceite como «nao declarado» - e nao declarado NAO e' o mesmo que eterno: quem apaga tem de tratar o 0.
	zero := base
	if err := ValidateArtifactScope(zero); err != nil {
		t.Fatalf("zero devia ser aceite como «nao declarado»: %v", err)
	}
}

func TestValidateArtifactScopeExigeATipoEIdNaACL(t *testing.T) {
	base := ArtifactManifest{ID: "art_1", TenantID: "acme", SHA256: "abc"}

	validas := []string{"user:ana", "group:qa", "tenant:acme", "service:indexer:worker"}
	for _, acl := range validas {
		m := base
		m.ACL = []string{acl}
		if err := ValidateArtifactScope(m); err != nil {
			t.Fatalf("ACL legitima %q recusada: %v", acl, err)
		}
	}

	invalidas := []string{"ana", ":", ":ana", "user:", "   "}
	for _, acl := range invalidas {
		m := base
		m.ACL = []string{acl}
		if err := ValidateArtifactScope(m); !errors.Is(err, ErrArtifactACLInvalida) {
			t.Fatalf("ARMADILHA: ACL invalida %q aceite (err=%v)", acl, err)
		}
	}
}

func TestArtifactVisivelParaFalhaFechado(t *testing.T) {
	m := ArtifactManifest{
		ID: "art_1", TenantID: "acme", UserID: "ana", SHA256: "abc",
		ACL: []string{"group:qa"},
	}

	casos := []struct {
		nome     string
		tenant   string
		sujeito  string
		esperado bool
		porque   string
	}{
		{"o dono ve' o seu", "acme", "ana", true, "*regra 2: o dono ve' o seu, sem precisar de ACL*"},
		{"a ACL acrescenta", "acme", "qa", true, "*regra 3: a ACL so' ACRESCENTA*"},
		{"outro tenant NAO ve'", "outra", "ana", false, "*regra 1: tenant primeiro, e sempre*"},
		{"outro sujeito NAO ve'", "acme", "bob", false, "*nem o proprio id, se nao for dono nem ACL*"},
		{"sujeito vazio NAO ve'", "acme", "", false, "*vazio falha fechado*"},
		{"tenant vazio NAO ve'", "", "ana", false, "*tenant vazio falha fechado*"},
	}
	for _, c := range casos {
		if obtido := ArtifactVisivelPara(m, c.tenant, c.sujeito); obtido != c.esperado {
			t.Fatalf("%s: ArtifactVisivelPara(tenant=%q, sujeito=%q) = %v, esperado %v - %s",
				c.nome, c.tenant, c.sujeito, obtido, c.esperado, c.porque)
		}
	}

	// ***A CONTRAPROVA QUE IMPORTA***: ACL **vazia** significa «ninguem alem do dono» - NUNCA «todos».
	semACL := ArtifactManifest{ID: "art_2", TenantID: "acme", UserID: "ana", SHA256: "abc"}
	if ArtifactVisivelPara(semACL, "acme", "estranho") {
		t.Fatal("ARMADILHA: ACL VAZIA deixou um estranho ver o artefacto - *o sentido do vazio escolhe-se pelo lado " +
			"que falha fechado, e «todos» e' o lado errado*")
	}

	// E um artefacto SEM tenant nao e' visivel para ninguem - *ele tambem nao devia existir*.
	semTenant := ArtifactManifest{ID: "art_3", UserID: "ana", SHA256: "abc"}
	if ArtifactVisivelPara(semTenant, "acme", "ana") {
		t.Fatal("ARMADILHA: artefacto sem tenant ficou visivel - *nao e' visivel para ninguem, e e' o comportamento certo*")
	}
}

// TestArtifactManifestPreservaOsCamposAntigos e' a contraprova de que o E-532 **nao partiu** o que ja' existia:
// *o que se acrescenta nao pode quebrar o que ja' funciona.*
func TestArtifactManifestPreservaOsCamposAntigos(t *testing.T) {
	m := ArtifactManifest{
		ID: "art_1", MissionID: "m1", StepID: "s1", Name: "relatorio.pdf",
		Path: "out/relatorio.pdf", MediaType: "application/pdf", Size: 1234,
		SHA256: strings.Repeat("a", 64),
	}
	// Os campos antigos continuam a ser o que eram.
	if m.ID != "art_1" || m.MissionID != "m1" || m.StepID != "s1" || m.MediaType != "application/pdf" || m.Size != 1234 {
		t.Fatalf("campo antigo alterado: %+v", m)
	}
	// E os novos nascem no estado que falha fechado.
	if m.TenantID != "" || m.UserID != "" || m.Version != 0 || len(m.ACL) != 0 || m.MIMEVerified {
		t.Fatalf("campo novo nasceu com valor inesperado: %+v", m)
	}
	// *`MIMEVerified` falso e' o estado honesto: a auditoria mostrou que o MediaType e' inferido por EXTENSAO, e uma
	// extensao mente.*
	if m.MIMEVerified {
		t.Fatal("MIMEVerified nasceu verdadeiro sem ninguem ter confirmado o conteudo")
	}
}
