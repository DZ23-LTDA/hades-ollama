package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Estágio 11/10 — RETOMADA: reimportar o MESMO projeto (ou retomar uma importação
// interrompida) NÃO pode falhar.
//
// DEFEITO REAL corrigido aqui: `initializeImportedRepository` exigia exit 0 do passo
// `git commit`, e o git retorna 1 com "nothing to commit" quando o snapshot JÁ estava
// commitado. A segunda chamada quebrava a importação — justamente o caminho de retomada.
// Este teste é a guarda: a inicialização é IDEMPOTENTE e, quando há mudanças de verdade,
// ainda produz um novo commit.
func TestInitializeImportedRepositoryIsIdempotent(t *testing.T) {
	root := shortTestRoot(t)
	file := filepath.Join(root, "README.md")
	if err := os.WriteFile(file, []byte("documentação do projeto\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// Primeira chamada: cria o repositório e commita o snapshot.
	if err := initializeImportedRepository(ctx, root); err != nil {
		t.Fatalf("primeira inicialização precisa funcionar: %v", err)
	}
	first := importedRepoHead(t, root)

	// SEGUNDA chamada com a MESMA árvore: antes falhava com "nothing to commit".
	if err := initializeImportedRepository(ctx, root); err != nil {
		t.Fatalf("reinicializar sem mudanças precisa ser idempotente, veio: %v", err)
	}
	if again := importedRepoHead(t, root); again != first {
		t.Fatalf("sem mudanças o commit não pode mudar: %s vs %s", first, again)
	}

	// Terceira chamada COM mudança real: precisa commitar de novo.
	if err := os.WriteFile(file, []byte("documentação revisada\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := initializeImportedRepository(ctx, root); err != nil {
		t.Fatalf("com mudanças a inicialização precisa commitar: %v", err)
	}
	if changed := importedRepoHead(t, root); changed == first {
		t.Fatal("com mudança real o snapshot precisa gerar um novo commit")
	}
	// A árvore fica LIMPA: o snapshot está commitado.
	if dirty := importedRepoDirty(t, root); dirty {
		t.Fatal("depois de inicializar a árvore precisa estar limpa")
	}
}

// importedRepoHead devolve o HEAD atual, ou falha se o repositório não tiver commit.
func importedRepoHead(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("o repositório importado precisa ter commit: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func importedRepoDirty(t *testing.T, root string) bool {
	t.Helper()
	pending, err := importedRepositoryHasPendingChanges(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	return pending
}

// TestProjectImportReportsLongPathClearly verifica que caminho acima do limite do Windows
// produz erro TIPADO e ACIONÁVEL, em vez de `exit status 128` sem explicação. O produto
// precisa disso porque o caminho interno do git (`.git/objects/...`) estoura MAX_PATH em
// workspaces longos, e o usuário tem de saber o que fazer.
func TestProjectImportReportsLongPathClearly(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("limite clássico de caminho é específico do Windows")
	}
	base := t.TempDir()
	long := filepath.Join(base, strings.Repeat("nivel-", 20), strings.Repeat("fundo-", 20), "worktree")
	if err := os.MkdirAll(long, 0o700); err != nil {
		t.Skipf("não foi possível criar o caminho longo: %v", err)
	}
	if len(long) < windowsMaxPath {
		t.Skipf("caminho de %d caracteres não atinge o limite de %d", len(long), windowsMaxPath)
	}
	err := initializeImportedRepository(context.Background(), long)
	if err == nil {
		t.Fatalf("caminho de %d caracteres deveria falhar de forma explícita", len(long))
	}
	if !errors.Is(err, ErrImportPathTooLong) {
		t.Fatalf("o erro precisa ser tipado como caminho longo, veio: %v", err)
	}
	message := err.Error()
	for _, expected := range []string{"limite é 260", "diretório mais curto", "LongPathsEnabled"} {
		if !strings.Contains(message, expected) {
			t.Fatalf("a mensagem precisa ser acionável (%q ausente): %s", expected, message)
		}
	}
}
