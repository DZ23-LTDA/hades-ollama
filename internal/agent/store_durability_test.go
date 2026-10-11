package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// POLÍTICA DE DURABILIDADE (E-168). O `fsync` domina o custo da escrita (E-166/E-167), mas removê-lo é uma
// decisão sobre PERDER ESCRITAS numa queda abrupta — então o teste precisa travar as duas garantias:
//
//  1. o PADRÃO é seguro (sincroniza): ninguém ganha velocidade por acidente;
//  2. o modo `relaxed` é EXPLÍCITO, mantém a troca atômica e continua gravando arquivo legível;
//  3. um valor desconhecido cai para o seguro **e a divergência é declarada** (não vira "relaxed" por engano);
//  4. o modo efetivo é consultável — degradação silenciosa é o pior resultado possível.

func restoreDurability(t *testing.T) {
	t.Helper()
	anterior := WriteDurability()
	t.Cleanup(func() { _, _ = SetWriteDurability(anterior) })
}

func TestWriteDurabilityDefaultsToSafe(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(""); err != nil {
		t.Fatal(err)
	}
	if modo := WriteDurability(); modo != WriteDurabilityDurable {
		t.Fatalf("o padrão precisa ser o modo seguro: %s", modo)
	}
}

func TestWriteDurabilityUnknownValueFallsBackAndDeclares(t *testing.T) {
	restoreDurability(t)
	modo, err := SetWriteDurability("turbo")
	if err == nil {
		t.Fatal("valor desconhecido precisa ser declarado, não aceito em silêncio")
	}
	if modo != WriteDurabilityDurable || WriteDurability() != WriteDurabilityDurable {
		t.Fatalf("valor desconhecido precisa cair para o modo seguro: %s", WriteDurability())
	}
	if !strings.Contains(err.Error(), "desconhecido") {
		t.Fatalf("a mensagem precisa dizer que o valor não foi reconhecido: %v", err)
	}
}

func TestWriteDurabilityRelaxedKeepsAtomicWriteAndReadableFile(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(WriteDurabilityRelaxed); err != nil {
		t.Fatal(err)
	}
	if WriteDurability() != WriteDurabilityRelaxed {
		t.Fatalf("o modo relaxado precisa valer quando pedido: %s", WriteDurability())
	}
	dir := t.TempDir()
	caminho := filepath.Join(dir, "relaxado.json")
	valor := amostraParaFormato()
	if err := writeJSONAtomic(caminho, valor); err != nil {
		t.Fatalf("o modo relaxado continua gravando: %v", err)
	}
	// Mesmo sem `fsync`, a TROCA ATÔMICA e o conteúdo têm de continuar corretos — a única coisa que muda é a
	// garantia contra queda abrupta.
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	var lido amostraDeFormato
	if err := json.Unmarshal(dados, &lido); err != nil {
		t.Fatalf("o arquivo do modo relaxado precisa ser JSON válido: %v", err)
	}
	if lido.Nome != valor.Nome {
		t.Fatalf("conteúdo divergente no modo relaxado: %+v", lido)
	}
	// E não pode sobrar arquivo temporário: a troca atômica continua sendo feita.
	sobras, err := filepath.Glob(filepath.Join(dir, ".agent-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sobras) != 0 {
		t.Fatalf("o modo relaxado não pode deixar temporário para trás: %v", sobras)
	}
}

func TestWriteDurabilityFromEnvIsApplied(t *testing.T) {
	restoreDurability(t)
	t.Setenv("HADES_WRITE_DURABILITY", "relaxed")
	modo, err := ConfigureWriteDurabilityFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if modo != WriteDurabilityRelaxed || WriteDurability() != WriteDurabilityRelaxed {
		t.Fatalf("o ambiente precisa configurar o modo: %s", modo)
	}
	// Sem a variável, volta ao seguro.
	t.Setenv("HADES_WRITE_DURABILITY", "")
	if modo, err := ConfigureWriteDurabilityFromEnv(); err != nil || modo != WriteDurabilityDurable {
		t.Fatalf("sem configuração o modo precisa ser o seguro: %s (%v)", modo, err)
	}
}
