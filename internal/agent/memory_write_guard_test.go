package agent

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Estágio 35 — GUARDA DE DIVERGÊNCIA COM INTENÇÃO EXPLÍCITA (E-289, refaz E-288 pela causa-raiz).
//
// E-288 mostrou que uma guarda em TODOS os caminhos quebra operações legítimas: no arquivo, deleção legítima e
// sobrescrita por snapshot velho são indistinguíveis. A causa-raiz apontada foi a falta de INTENÇÃO DECLARADA — e
// aqui a intenção é declarada POR CAMINHO: só os não-destrutivos passam pela guarda.
//
// Propriedades travadas:
//  1. memória acrescentada por OUTRO escritor ⇒ gravação do caminho não-destrutivo RECUSADA e disco intacto;
//  2. gravação legítima sem divergência continua funcionando;
//  3. a guarda está LIGADA nos dois caminhos não-destrutivos (checagem de fiação no fonte).
func TestGuardaDeDivergenciaRecusaApagarMemoriaAcrescentadaPorOutro(t *testing.T) {
	raiz := t.TempDir()
	store := &ContextStore{root: raiz}
	projeto := "proj-guarda"
	caminho := filepath.Join(raiz, "memories", projeto+".json")
	if err := os.MkdirAll(filepath.Dir(caminho), 0o700); err != nil {
		t.Fatal(err)
	}
	agora := time.Now().UTC()
	minha := Memory{ID: "mem-minha", ProjectID: projeto, Content: "minha", CreatedAt: agora}

	// Baseline legítima do caminho não-destrutivo.
	if err := store.writeMemoriesFileGuardingAdditions(projeto, []Memory{minha}); err != nil {
		t.Fatalf("gravação inicial sem divergência precisa funcionar: %v", err)
	}

	// OUTRO ESCRITOR acrescenta uma memória ao arquivo (fora do store).
	outra := Memory{ID: "mem-de-fora", ProjectID: projeto, Content: "outro escritor", CreatedAt: agora}
	disco, err := memoriesForDisk([]Memory{minha, outra})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(caminho, disco); err != nil {
		t.Fatal(err)
	}

	// 1. O snapshot antigo (sem a memória de fora) gravaria APAGANDO o trabalho do outro ⇒ RECUSA.
	err = store.writeMemoriesFileGuardingAdditions(projeto, []Memory{minha})
	if !errors.Is(err, ErrMemoryChangedOutsideStore) {
		t.Fatalf("snapshot sem a memória acrescentada por outro precisa ser RECUSADO: %v", err)
	}
	// A recusa nomeia o que seria apagado — mensagem sem id não orienta correção.
	if erro := err.Error(); !strings.Contains(erro, "mem-de-fora") {
		t.Fatalf("a recusa precisa NOMEAR a memória ameaçada: %v", erro)
	}
	// O disco continua com as DUAS: a recusa é real, não cosmética.
	depois, err := store.readMemoriesForGuard(projeto)
	if err != nil {
		t.Fatal(err)
	}
	if len(depois) != 2 {
		t.Fatalf("o disco não pode ter sido alterado pela tentativa recusada: %+v", depois)
	}

	// 2. Com o snapshot ATUALIZADO (incluindo a memória de fora), a gravação é permitida.
	if err := store.writeMemoriesFileGuardingAdditions(projeto, []Memory{minha, outra}); err != nil {
		t.Fatalf("snapshot atualizado precisa poder gravar: %v", err)
	}

	// 3. A guarda está LIGADA nos caminhos NÃO-DESTRUTIVOS: sem isso, a guarda existiria e não protegeria nada.
	for arquivo, esperado := range map[string]string{
		"context.go":        "writeMemoriesFileGuardingAdditions(memory.ProjectID",
		"memory_version.go": "writeMemoriesFileGuardingAdditions(projectID",
	} {
		conteudo, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		if !strings.Contains(string(conteudo), esperado) {
			t.Fatalf("a guarda precisa estar LIGADA em %s (%s)", arquivo, esperado)
		}
	}
	// E os caminhos DESTRUTIVOS seguem no escritor simples, por INTENÇÃO declarada: ligar a guarda neles é o erro
	// que E-288 cometeu e que quebrou RemoveMemory, PruneMemories e trimMemoriesToLimit.
	for arquivo := range map[string]bool{
		"context_memory_governance.go": true,
		"memory_retention.go":          true,
	} {
		conteudo, err := os.ReadFile(arquivo)
		if err != nil {
			t.Fatalf("ler %s: %v", arquivo, err)
		}
		if strings.Contains(string(conteudo), "writeMemoriesFileGuardingAdditions") {
			t.Fatalf("caminho destrutivo NÃO pode usar a guarda (falso positivo de E-288): %s", arquivo)
		}
	}
}
