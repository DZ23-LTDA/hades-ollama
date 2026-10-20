package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O CASO QUE MOTIVA A PERSISTENCIA: parar, reiniciar, e o pare continua parado.
func TestPareSobreviveAoReinicio(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "pare.json")

	antes := NovoPareDeEmergencia()
	if err := antes.Acionar("paragem que tem de sobreviver", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	if err := antes.GuardarEm(caminho); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	// "Reinicio": instancia nova, que comeca desarmada.
	depois := NovoPareDeEmergencia()
	if depois.Bloqueia() {
		t.Fatal("a instancia nova devia comecar desarmada antes de carregar")
	}
	if err := depois.CarregarDe(caminho); err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !depois.Bloqueia() {
		t.Fatal("depois do reinicio o pare TEM de continuar parado")
	}
	if motivo := depois.Estado().Motivo; !strings.Contains(motivo, "sobreviver") {
		t.Fatalf("o motivo tem de sobreviver: %q", motivo)
	}
}

// Ficheiro inexistente e' o estado inicial, nao um erro.
func TestCarregarSemFicheiroNaoEProblema(t *testing.T) {
	pare := NovoPareDeEmergencia()
	caminho := filepath.Join(t.TempDir(), "nao-existe.json")
	if err := pare.CarregarDe(caminho); err != nil {
		t.Fatalf("ficheiro inexistente nao pode ser erro: %v", err)
	}
	if pare.Bloqueia() {
		t.Fatal("sem ficheiro o pare fica desarmado")
	}
}

// Um estado ILEGIVEL e' erro -- e o pare NAO pode passar a "desarmado" por causa disso.
func TestEstadoIlegivelNaoDesarmaOPare(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "pare.json")
	if err := os.WriteFile(caminho, []byte("{isto nao e' json"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}

	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("estava parado antes de ler o lixo", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	if err := pare.CarregarDe(caminho); err == nil {
		t.Fatal("estado ilegivel tem de ser erro")
	}
	if !pare.Bloqueia() {
		t.Fatal("um estado ilegivel NAO pode desarmar o pare")
	}
	if motivo := pare.Estado().Motivo; !strings.Contains(motivo, "lixo") {
		t.Fatalf("o estado anterior tem de ficar intacto: %q", motivo)
	}
}

// Guardar e' atomico na pratica: o ficheiro publicado e' JSON valido, e nao sobra temporario.
func TestGuardarPublicaJSONValidoSemTemporarios(t *testing.T) {
	directorio := t.TempDir()
	caminho := filepath.Join(directorio, "pare.json")

	pare := NovoPareDeEmergencia()
	if err := pare.Acionar("motivo", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	if err := pare.GuardarEm(caminho); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	bruto, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if !strings.Contains(string(bruto), "\"acionado\": true") {
		t.Fatalf("o ficheiro tem de conter o estado: %s", bruto)
	}
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(entradas) != 1 {
		nomes := make([]string, 0, len(entradas))
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Fatalf("nao pode sobrar temporario: %v", nomes)
	}
}

// Recusas explicitas: caminho vazio e pare nulo.
func TestGuardarRecusaCaminhoVazio(t *testing.T) {
	pare := NovoPareDeEmergencia()
	if err := pare.GuardarEm(""); err == nil {
		t.Fatal("caminho vazio tem de ser recusado")
	}
	var nulo *PareDeEmergencia
	if err := nulo.GuardarEm(filepath.Join(t.TempDir(), "x.json")); err == nil {
		t.Fatal("pare nulo tem de ser recusado")
	}
	if err := nulo.CarregarDe(filepath.Join(t.TempDir(), "x.json")); err == nil {
		t.Fatal("carregar num pare nulo tem de ser recusado")
	}
}
