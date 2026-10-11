package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Atividade do estágio 15: CRIPTOGRAFIA da memória em repouso.
//
// As propriedades travadas aqui:
//  1. com chave configurada, o TEXTO CLARO NÃO APARECE no arquivo;
//  2. o round-trip sobrevive ao reinício do store;
//  3. sem a chave, ler memória cifrada FALHA com erro explícito — nunca devolve
//     vazio, porque memória vazia em silêncio é perda de dado disfarçada;
//  4. TODOS os caminhos de escrita cifram: adicionar, atualizar (versionar),
//     remover e aplicar retenção. Proteger alguns deixaria o "irmão esquecido"
//     gravando texto claro.
const testMemoryKey = "chave-de-teste-com-32-caracteres-ok"

func readMemoryFileRaw(t *testing.T, root, projectID string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "memories", projectID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMemoryIsEncryptedAtRestAndRoundTrips(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", testMemoryKey)
	if status := MemoryEncryption(); !status.Configured || !status.AtRest || status.Algorithm != MemoryEncryptionAlgorithm {
		t.Fatalf("o status precisa declarar a criptografia: %+v", status)
	}
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto cifrado", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	segredo := "anotação que não pode ficar em texto claro"
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: segredo,
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	raw := readMemoryFileRaw(t, root, project.ID)
	if strings.Contains(raw, segredo) {
		t.Fatalf("o texto claro VAZOU para o disco: %s", raw)
	}
	if !strings.Contains(raw, "encrypted_content") {
		t.Fatalf("o arquivo precisa carregar o conteúdo cifrado: %s", raw)
	}

	// Round-trip: o store reiniciado devolve o conteúdo original.
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	export, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 || export.Memories[0].Content != segredo {
		t.Fatalf("o round-trip precisa devolver o texto original: %+v", export.Memories)
	}
}

func TestMemoryEncryptedReadWithoutKeyFailsClosed(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", testMemoryKey)
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto sem chave", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "segredo persistido",
	}, "ana"); err != nil {
		t.Fatal(err)
	}

	// Sem a chave, abrir o store precisa FALHAR e dizer por quê.
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "")
	if _, err := NewContextStore(root); err == nil {
		t.Fatal("ler memória cifrada sem chave precisa falhar, não devolver vazio")
	} else if !strings.Contains(err.Error(), "key is unavailable") {
		t.Fatalf("o erro precisa explicar a chave ausente, veio: %v", err)
	}
	if status := MemoryEncryption(); status.Configured {
		t.Fatalf("sem chave o status precisa ser honesto: %+v", status)
	}
}

func TestMemoryWithoutKeyStaysPlainAndSaysSo(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", "")
	if status := MemoryEncryption(); status.Configured || status.AtRest || status.Guidance == "" {
		t.Fatalf("sem chave, o status precisa declarar o texto claro e orientar: %+v", status)
	}
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto claro", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "conteúdo sem chave",
	}, "ana"); err != nil {
		t.Fatal(err)
	}
	raw := readMemoryFileRaw(t, root, project.ID)
	if !strings.Contains(raw, "conteúdo sem chave") {
		t.Fatalf("sem chave o comportamento anterior precisa continuar: %s", raw)
	}
	// E o store abre normalmente: opt-in não pode quebrar instalação existente.
	if _, err := NewContextStore(root); err != nil {
		t.Fatal(err)
	}
}

// O teste que fecha o "irmão esquecido" na minha própria mudança: TODOS os
// caminhos que gravam o arquivo precisam cifrar.
func TestEveryMemoryWritePathEncryptsAtRest(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_CREDENTIAL_KEY", testMemoryKey)
	root := t.TempDir()
	store, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject("projeto caminhos", t.TempDir(), "org-a")
	if err != nil {
		t.Fatal(err)
	}
	segredo := "segredo em todos os caminhos"
	created, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: segredo,
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	assertEncrypted := func(step string) {
		t.Helper()
		raw := readMemoryFileRaw(t, root, project.ID)
		if strings.Contains(raw, segredo) {
			t.Fatalf("%s: o texto claro apareceu no disco: %s", step, raw)
		}
	}
	assertEncrypted("adicionar")

	// Caminho da ATUALIZAÇÃO (versionamento).
	if _, err := store.UpdateMemoryForActor(context.Background(), project.ID, created.ID, "ana", segredo+" revisado", "revisão"); err != nil {
		t.Fatal(err)
	}
	assertEncrypted("atualizar")

	// Caminho da SEGUNDA memória + REMOÇÃO (governança).
	other, err := store.AddMemoryForActor(context.Background(), Memory{
		ProjectID: project.ID, Kind: "note", Content: "outra memória cifrada",
	}, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveMemory(project.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	assertEncrypted("remover")

	// Caminho da RETENÇÃO por contagem.
	for index := range 3 {
		if _, err := store.AddMemoryForActor(context.Background(), Memory{
			ProjectID: project.ID, Kind: "note", Content: "retenção " + string(rune('a'+index)),
		}, "ana"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.SetRetentionPolicyForOrganization(MemoryRetentionPolicy{OrganizationID: "org-a", MaxAgeDays: 30, MaxMemoriesPerProject: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ApplyRetentionForOrganization("org-a", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	raw := readMemoryFileRaw(t, root, project.ID)
	for _, leaked := range []string{segredo, "outra memória cifrada", "retenção a"} {
		if strings.Contains(raw, leaked) {
			t.Fatalf("retenção: o texto claro %q apareceu no disco: %s", leaked, raw)
		}
	}
	// E o conteúdo continua legível pelo store após todas as operações.
	restarted, err := NewContextStore(root)
	if err != nil {
		t.Fatal(err)
	}
	export, err := restarted.ExportMemoriesForActor(project.ID, "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if export.Count != 1 {
		t.Fatalf("a retenção precisa manter uma memória: %d", export.Count)
	}
}
