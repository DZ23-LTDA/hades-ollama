package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestMissaoMultiagenteProduzArtefatosReais prova a lacuna declarada do estágio 9: os testes anteriores usavam
// runner SINTÉTICO (resultado de mentira), então nada provava artefato real. Aqui cada agente ESCREVE no próprio
// workspace e o teste confere:
//  1. o artefato existe no disco e o SHA-256 registrado corresponde aos BYTES REAIS (não a uma string inventada);
//  2. o workspace de cada agente EXISTE (era o defeito: o orquestrador atribuía o caminho e não criava o diretório,
//     então o primeiro WriteFile do agente falhava com "no such file or directory");
//  3. a travessia para fora do workspace é RECUSADA, e caminho relativo legítimo é aceito, pela API real.
//
// MEDIÇÃO registrada (não é exigência): hoje os agentes compartilham o workspace do JOB. Se a intenção do estágio 9
// for isolamento POR AGENTE, isso é lacuna a confirmar contra a alegação de E-215 antes de mudar comportamento.
func TestMissaoMultiagenteProduzArtefatosReais(t *testing.T) {
	raiz := t.TempDir()
	var mu sync.Mutex
	escritos := make(map[AgentRole]string)

	runner := func(ctx context.Context, task AgentTask) (AgentResult, error) {
		conteudo := []byte("artefato real de " + string(task.Role))
		caminho := filepath.Join(task.Workspace, string(task.Role)+".txt")
		if err := os.WriteFile(caminho, conteudo, 0o644); err != nil {
			return AgentResult{}, err
		}
		soma := sha256.Sum256(conteudo)
		mu.Lock()
		escritos[task.Role] = caminho
		mu.Unlock()
		return AgentResult{
			Output:   "concluído: " + string(task.Role),
			Evidence: []AgentEvidence{{URL: "file://" + caminho, SHA256: hex.EncodeToString(soma[:]), Source: string(task.Role)}},
		}, nil
	}

	orquestrador, err := NewAgentOrchestrator(raiz, runner)
	if err != nil {
		t.Fatalf("criar orquestrador: %v", err)
	}
	trabalho, err := orquestrador.Plan(
		"produzir artefatos por especialista",
		filepath.Join(t.TempDir(), "missao"),
		"projeto-1",
		[]AgentRole{RoleResearch, RoleSecurity, RoleReview},
		AgentBudget{MaxAgents: 3, MaxRetries: 1, MaxSeconds: 15},
	)
	if err != nil {
		t.Fatalf("planejar: %v", err)
	}
	concluida, err := orquestrador.Run(context.Background(), trabalho.ID)
	if err != nil {
		t.Fatalf("executar missão multiagente: %v", err)
	}
	if concluida.State != OrchestrationCompleted {
		t.Fatalf("missão deveria concluir: state=%s conflitos=%v", concluida.State, concluida.Conflicts)
	}

	caminhos := make(map[string]bool, len(concluida.Tasks))
	artefatos := 0
	for _, tarefa := range concluida.Tasks {
		if strings.TrimSpace(tarefa.Workspace) == "" {
			t.Fatalf("agente %s rodou SEM workspace: %+v", tarefa.Role, tarefa)
		}
		caminhos[tarefa.Workspace] = true
		if _, err := os.Stat(tarefa.Workspace); err != nil {
			t.Fatalf("workspace do agente %s não existe no disco (%s): %v", tarefa.Role, tarefa.Workspace, err)
		}
		for _, evidencia := range tarefa.Evidence {
			if evidencia.SHA256 == "" || !strings.HasPrefix(evidencia.URL, "file://") {
				continue
			}
			caminho := strings.TrimPrefix(evidencia.URL, "file://")
			bytes, err := os.ReadFile(caminho)
			if err != nil {
				t.Fatalf("artefato declarado em %s não existe no disco: %v", caminho, err)
			}
			soma := sha256.Sum256(bytes)
			if hex.EncodeToString(soma[:]) != evidencia.SHA256 {
				t.Fatalf("SHA-256 do artefato %s NÃO corresponde aos bytes reais: registrado=%s real=%s",
					caminho, evidencia.SHA256, hex.EncodeToString(soma[:]))
			}
			artefatos++
		}
	}
	if artefatos != 3 {
		t.Fatalf("esperava 3 artefatos reais verificados por SHA-256, veio %d", artefatos)
	}
	for papel, caminho := range escritos {
		if _, err := os.Stat(caminho); err != nil {
			t.Fatalf("artefato de %s desapareceu: %v", papel, err)
		}
	}

	// TRAVESSIA recusada e caminho relativo aceito, pela API REAL de escrita.
	var algum string
	for caminho := range caminhos {
		algum = caminho
		break
	}
	if _, err := validateWorkspaceWritePath(algum, "../fora-do-workspace.txt"); err == nil {
		t.Fatal("travessia para fora do workspace precisa ser RECUSADA")
	}
	if _, err := validateWorkspaceWritePath(algum, "ok.txt"); err != nil {
		t.Fatalf("caminho relativo legítimo foi recusado: %v", err)
	}

	t.Logf("MULTIAGENTE: agentes=%d artefatos_reais=%d caminhos_de_workspace=%d travessia=recusada",
		len(concluida.Tasks), artefatos, len(caminhos))
}
