package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// PROVA DO CAMINHO DE FALHA DO E-565 (F5 da auditoria de segurança).
//
// O E-565 corrigiu a ORDEM em `DeleteProject`: disco primeiro, memória depois — para
// que uma falha no disco deixasse a memória INTACTA (falha fechado) em vez de esquecer
// o projeto e deixar os ficheiros em disco. No commit desse PR eu declarei, com todas
// as letras, que o caminho de FALHA **não estava provado por execução**, porque o
// `os.Remove` é chamado diretamente e não havia ponto de injeção.
//
// Há ponto de injeção: o SISTEMA DE FICHEIROS. Substituindo o ficheiro do projeto por
// um DIRETÓRIO NÃO VAZIO com o mesmo nome, o `os.Remove` do caminho do ficheiro falha
// (ENOTEMPTY no Linux/macOS, "directory not empty" no Windows) — sem tocar no código de
// produção e sem mock nenhum.
//
// Se este teste passasse com a ordem ANTIGA, ele não provaria nada: com a ordem antiga
// o projeto já tinha sido esquecido e `ListProjects()` viria vazio.
func TestDeleteProjectNaoEsqueceQuandoODiscoFalha(t *testing.T) {
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := store.CreateProject("Workspace", "", "org_test")
	if err != nil {
		t.Fatal(err)
	}

	// INJEÇÃO: o caminho de destino passa a ser um diretório não vazio.
	alvo := filepath.Join(root, "projects", projeto.ID+".json")
	if err := os.Remove(alvo); err != nil {
		t.Fatalf("preparar a injeção (remover o ficheiro real): %v", err)
	}
	if err := os.MkdirAll(filepath.Join(alvo, "ocupado"), 0o700); err != nil {
		t.Fatalf("preparar a injeção (criar diretório): %v", err)
	}
	if err := os.WriteFile(filepath.Join(alvo, "ocupado", "x"), []byte("x"), 0o600); err != nil {
		t.Fatalf("preparar a injeção (tornar o diretório não vazio): %v", err)
	}

	err = store.DeleteProject(projeto.ID)
	if err == nil {
		// SKIP DECLARADO, não artificial: se a plataforma permitisse remover um
		// diretório não vazio, o caminho de falha não teria sido exercitado e o teste
		// não pode afirmar nada sobre ele.
		t.Skip("a plataforma removeu um diretório não vazio: o caminho de falha não foi exercitado")
	}

	// A PROPRIEDADE QUE IMPORTA: o erro é verdadeiro E a memória não mudou.
	if projetos := store.ListProjects(); len(projetos) != 1 || projetos[0].ID != projeto.ID {
		t.Fatalf("FALHA FECHADO QUEBRADA: o disco falhou e o projeto desapareceu da memória: %+v", projetos)
	}
}
