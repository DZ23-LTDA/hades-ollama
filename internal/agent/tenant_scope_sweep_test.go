package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// VARREDURA SISTÊMICA DE ESCOPO VAZIO (E-337) — follow-up do ADV-11.
//
// E-336 achou que `sanitizeIDComponent("")` devolve **`geral`** e que `consentScope("")` também tem fallback. Isso
// significa que um armazenamento que não valida a organização **crua** pode gravar tudo numa pasta **compartilhada**
// — *um dado "sem tenant" viraria dado de todos*. A varredura abaixo ataca cada armazenamento desta missão e exige
// **duas** coisas: erro explícito E ausência de arquivo compartilhado.
//
// ESCOPO DECLARADO: cobre os armazenamentos cujos caminhos eu conheço e uso (`scheduling`, `waitlist`, `live`, fluxo
// de `workflow`, comércio, empresa). Os demais usos de `sanitizeIDComponent` (≈20 arquivos, medido: 39 usos) seguem
// **não varridos** e ficam registrados como ataque declarado — *a varredura não vira "tudo auditado" por decreto*.

// assertSemPastaCompartilhada falha se o armazenamento tiver criado o arquivo de fallback "geral".
func assertSemArquivoCompartilhado(t *testing.T, raiz string) {
	t.Helper()
	for _, candidato := range []string{"geral.json", filepath.Join("geral", "geral.json")} {
		caminho := filepath.Join(raiz, candidato)
		if _, err := os.Stat(caminho); err == nil {
			t.Fatalf("ARMADILHA: organização vazia criou arquivo COMPARTILHADO em %q "+
				"(um dado sem tenant vira dado de todos)", caminho)
		}
	}
}

func TestEscopoVazioNaoCompartilhaArmazenamento(t *testing.T) {
	agora := time.Now().UTC()

	t.Run("scheduling", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "sched")
		store := NewSchedulingStore(raiz)
		if _, err := store.AddService(SchedulingService{OrganizationID: "", Name: "x", DurationMinutes: 30}); err == nil {
			t.Fatal("serviço sem organização precisa ser recusado")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("waitlist", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "sched")
		store := NewSchedulingStore(raiz)
		if _, err := store.AddToWaitlist(SchedulingWaitlistEntry{
			OrganizationID: "", ResourceID: "r", ServiceID: "s",
			From: agora, To: agora.Add(time.Hour),
		}); err == nil {
			t.Fatal("fila sem organização precisa ser recusada")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("live_commerce", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "live")
		store := NewLiveCommerceStore(raiz)
		if _, err := store.CreateLiveSession(LiveSession{
			OrganizationID: "", ProjectID: "p",
			AuthorizedUntil: agora.Add(time.Hour),
		}); err == nil {
			t.Fatal("sessão de live sem organização precisa ser recusada")
		}
		if _, _, err := store.IngestLiveCommerceEvent(CommerceEvent{
			OrganizationID: "", ExternalID: "e1",
			Kind: "cart_added", SignatureVerified: true,
		}); err == nil {
			t.Fatal("evento de comércio sem organização precisa ser recusado")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("workflow", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "wf")
		motor := NewWorkflowEngine(raiz, nil)
		if _, err := motor.Register(WorkflowDefinition{
			OrganizationID: "", Name: "x", StartNode: "a",
			Nodes: []WorkflowNode{{ID: "a", Kind: WorkflowNodeTrigger}},
		}); err == nil {
			t.Fatal("workflow sem organização precisa ser recusado")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("commerce", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "commerce")
		store, err := NewCommerceStoreAt(raiz)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.UpsertProduct(CommerceProduct{OrganizationID: "", SKU: "S", Name: "n", PriceCents: 1}, agora); err == nil {
			t.Fatal("produto sem organização precisa ser recusado")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("company", func(t *testing.T) {
		raiz := filepath.Join(t.TempDir(), "company")
		// MEDIDO por `go vet`: NewCompanyStore devolve DOIS valores.
		store, err := NewCompanyStore(raiz)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.CreateRequest(CompanyCreateRequest{Name: "x"}, ""); err == nil {
			t.Fatal("empresa sem organização precisa ser recusada")
		}
		assertSemArquivoCompartilhado(t, raiz)
	})

	t.Run("media_org_scope", func(t *testing.T) {
		// A mídia já foi corrigida em E-336; a varredura confirma que continua valendo.
		if _, err := MediaOrgRoot(t.TempDir(), ""); err == nil {
			t.Fatal("raiz de mídia sem organização precisa ser recusada")
		}
	})
}

// A varredura também confirma o INVERSO: com organização válida, o armazenamento funciona (não é um bloqueio geral).
func TestEscopoValidoContinuaFuncionando(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "sched")
	store := NewSchedulingStore(raiz)
	if _, err := store.AddService(SchedulingService{OrganizationID: "org-a", Name: "Corte", DurationMinutes: 30}); err != nil {
		t.Fatalf("organização válida precisa funcionar: %v", err)
	}
	if _, err := store.AddService(SchedulingService{OrganizationID: "org-b", Name: "Corte", DurationMinutes: 30}); err != nil {
		t.Fatalf("segunda organização válida precisa funcionar: %v", err)
	}
	// E os dois escopos são arquivos diferentes (isolamento real, não compartilhado).
	a := filepath.Join(raiz, sanitizeIDComponent("org-a")+".json")
	b := filepath.Join(raiz, sanitizeIDComponent("org-b")+".json")
	if a == b {
		t.Fatal("organizações diferentes não podem compartilhar arquivo")
	}
	for _, caminho := range []string{a, b} {
		if _, err := os.Stat(caminho); err != nil {
			t.Fatalf("cada organização precisa ter o próprio arquivo (%s): %v", caminho, err)
		}
	}
}
