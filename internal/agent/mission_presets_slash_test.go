package agent

// FASE 23 do adendo V3: a LIGACAO entre o catalogo de presets e o comando `/preset`.
//
// Sem esta ligacao o catalogo existiria e nao seria usado por ninguem. O teste que DISTINGUE a
// ligacao e' TestPresetComandoResolveOCatalogo: ele cobre o caminho completo
// `ParseSlashInvocation -> ResolvePresetInvocation -> MissionPreset`.

import (
	"errors"
	"strings"
	"testing"
)

// TestPresetComandoExisteEDisponivel prova que `/preset` esta' no registro e aponta para rota real.
func TestPresetComandoExisteEDisponivel(t *testing.T) {
	spec, ok := SlashCommandSpecByName("/preset")
	if !ok {
		t.Fatal("/preset nao existe no registro")
	}
	if !spec.Available {
		t.Fatalf("/preset esta' indisponivel: %q", spec.UnavailableReason)
	}
	if spec.Backend != "GET /api/agent/v1/presets" {
		t.Fatalf("/preset aponta para %q", spec.Backend)
	}
	if len(spec.Args) != 1 || spec.Args[0].Name != "id" || spec.Args[0].Required {
		t.Fatalf("/preset devia ter um argumento opcional 'id': %+v", spec.Args)
	}
}

// TestPresetComandoResolveOCatalogo e' o teste que DISTINGUE a ligacao.
func TestPresetComandoResolveOCatalogo(t *testing.T) {
	inv, err := ParseSlashInvocation("/preset briefing-diario")
	if err != nil {
		t.Fatalf("ParseSlashInvocation: %v", err)
	}
	preset, err := ResolvePresetInvocation(inv)
	if err != nil {
		t.Fatalf("ResolvePresetInvocation: %v", err)
	}
	if preset.ID != "briefing-diario" {
		t.Fatalf("preset errado: %q", preset.ID)
	}
	if preset.Schedule != "daily" {
		t.Fatalf("preset sem o schedule esperado: %q", preset.Schedule)
	}
	if len(preset.CompletionCriteria) == 0 {
		t.Fatal("preset resolvido sem criterio de conclusao")
	}
}

// TestPresetSemIDListaOsDisponiveis prova que a recusa nao e' um beco sem saida.
//
// *Um comando que recusa sem dizer o que aceita obriga a adivinhar.*
func TestPresetSemIDListaOsDisponiveis(t *testing.T) {
	inv, err := ParseSlashInvocation("/preset")
	if err != nil {
		t.Fatalf("ParseSlashInvocation: %v", err)
	}
	_, err = ResolvePresetInvocation(inv)
	if !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("esperado ErrPresetInvalido, obtido %v", err)
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Fatalf("a recusa devia listar os ids disponiveis: %v", err)
	}
	for _, id := range MissionPresetIDs() {
		if !strings.Contains(err.Error(), id) {
			t.Errorf("a recusa nao menciona o preset %q", id)
		}
	}
}

// TestPresetInexistenteERecusadoComLista prova que um id errado nao resolve para nada.
func TestPresetInexistenteERecusadoComLista(t *testing.T) {
	inv, err := ParseSlashInvocation("/preset nao-existe-este")
	if err != nil {
		t.Fatalf("ParseSlashInvocation: %v", err)
	}
	_, err = ResolvePresetInvocation(inv)
	if !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("esperado ErrPresetInvalido, obtido %v", err)
	}
	if !strings.Contains(err.Error(), "unknown preset") {
		t.Fatalf("o erro devia dizer que o preset e' desconhecido: %v", err)
	}
}

// TestResolveRecusaInvocacaoQueNaoEPreset prova que a funcao nao aceita qualquer comando.
func TestResolveRecusaInvocacaoQueNaoEPreset(t *testing.T) {
	inv, err := ParseSlashInvocation("/status missao-1")
	if err != nil {
		t.Fatalf("ParseSlashInvocation: %v", err)
	}
	if _, err := ResolvePresetInvocation(inv); !errors.Is(err, ErrPresetInvalido) {
		t.Fatalf("esperado ErrPresetInvalido, obtido %v", err)
	}
}

// TestTodosOsPresetsResolvemPeloComando prova que cada id do catalogo e' alcancavel pelo `/preset`.
//
// *Um catalogo onde metade dos presets nao e' invocavel e' um catalogo pela metade.*
func TestTodosOsPresetsResolvemPeloComando(t *testing.T) {
	for _, id := range MissionPresetIDs() {
		inv, err := ParseSlashInvocation("/preset " + id)
		if err != nil {
			t.Errorf("%s: ParseSlashInvocation: %v", id, err)
			continue
		}
		preset, err := ResolvePresetInvocation(inv)
		if err != nil {
			t.Errorf("%s: ResolvePresetInvocation: %v", id, err)
			continue
		}
		if preset.ID != id {
			t.Errorf("%s resolveu para %s", id, preset.ID)
		}
	}
}
