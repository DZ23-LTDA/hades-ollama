package agent

// Cobertura ENTRE PROCESSOS do store de aprovacoes -- a lacuna que a `E-1168` nomeou.
//
// O teste concorrente existente (`TestConsumeConcorrenteComEscritoresNaoDuplicaExecucao`) prova a
// propriedade DENTRO de um processo, com goroutines e `sync.WaitGroup`. Este prova a mesma
// propriedade ACROSS processos: N subprocessos do proprio binario de teste tentam consumir a MESMA
// aprovacao aprovada, e exactamente UM deve conseguir.
//
// *A propriedade nao se demonstra por analogia: um `sync.Mutex` protege threads, nao processos, e
// a serializacao entre processos vem da escrita atomica (`builder.go`), nao do mutex. Ate' aqui a
// infra existia e ninguem a tinha exercitado neste store.*

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	envAprovRoot  = "HADES_APROV_CROSSPROC_ROOT"
	envAprovChild = "HADES_APROV_CROSSPROC_CHILD"
	envAprovID    = "HADES_APROV_CROSSPROC_ID"
)

// TestHelperConsumeCrossProcess e' o corpo do subprocesso. Nao corre no processo principal.
func TestHelperConsumeCrossProcess(t *testing.T) {
	if os.Getenv(envAprovChild) != "1" {
		t.Skip("nao e' o subprocesso")
	}
	store := NewServiceAgentApprovalStore(os.Getenv(envAprovRoot))
	_, err := store.Consume("org-crossproc", os.Getenv(envAprovID), time.Now().UTC())
	res := map[string]any{"ok": err == nil}
	if err != nil {
		res["erro"] = err.Error()
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	destino := filepath.Join(os.Getenv(envAprovRoot), "resultado-"+os.Getenv("HADES_APROV_CHILD_ID")+".json")
	if err := os.WriteFile(destino, b, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestConsumeEntreProcessosNaoDuplicaExecucao lanca 4 processos reais e exige exactamente 1 sucesso.
func TestConsumeEntreProcessosNaoDuplicaExecucao(t *testing.T) {
	if testing.Short() {
		t.Skip("lanca subprocessos: nao corre em -short")
	}
	root := t.TempDir()
	store := NewServiceAgentApprovalStore(root)
	agora := time.Now().UTC()

	pedido, err := store.Request("org-crossproc", ServiceAgentApproval{Fingerprint: "fp-crossproc", ValueCents: 500})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if _, err := store.Decide("org-crossproc", pedido.ID, true, "humano-crossproc", "ok", agora); err != nil {
		t.Fatalf("Decide: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	const quantos = 4
	cmds := make([]*exec.Cmd, 0, quantos)
	for i := 0; i < quantos; i++ {
		c := exec.Command(exe, "-test.run=TestHelperConsumeCrossProcess")
		c.Env = append(os.Environ(),
			envAprovChild+"=1",
			envAprovRoot+"="+root,
			envAprovID+"="+pedido.ID,
			"HADES_APROV_CHILD_ID="+string(rune('a'+i)),
		)
		if err := c.Start(); err != nil {
			t.Fatalf("Start %d: %v", i, err)
		}
		cmds = append(cmds, c)
	}
	for i, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Logf("subprocesso %d terminou com %v", i, err)
		}
	}

	sucessos := 0
	for i := 0; i < quantos; i++ {
		b, err := os.ReadFile(filepath.Join(root, "resultado-"+string(rune('a'+i))+".json"))
		if err != nil {
			t.Logf("sem resultado do filho %d: %v", i, err)
			continue
		}
		var res map[string]any
		if err := json.Unmarshal(b, &res); err != nil {
			t.Logf("resultado ilegivel do filho %d: %v", i, err)
			continue
		}
		if ok, _ := res["ok"].(bool); ok {
			sucessos++
		}
	}

	if sucessos != 1 {
		t.Errorf("CONSUME ENTRE PROCESSOS: esperado exactamente 1 sucesso em %d, obtido %d", quantos, sucessos)
	}
}
