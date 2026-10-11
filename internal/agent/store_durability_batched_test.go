package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// MODO EM LOTE (E-169): o meio-termo entre perder tudo desde a última sincronização e pagar um `fsync` por
// escrita. O que precisa ser provado:
//
//  1. a troca atômica e a legibilidade NÃO mudam;
//  2. a PRIMEIRA escrita sincroniza (a janela não começa aberta após trocar de modo);
//  3. escritas dentro da janela ficam PENDENTES e são contabilizadas (dá para VER a janela em aberto);
//  4. o flush fecha a janela e zera as pendências;
//  5. o modo inválido continua caindo para o seguro.
func TestWriteDurabilityBatchedKeepsAtomicWriteAndReadableFile(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(WriteDurabilityBatched); err != nil {
		t.Fatal(err)
	}
	if WriteDurability() != WriteDurabilityBatched {
		t.Fatalf("o modo em lote precisa valer quando pedido: %s", WriteDurability())
	}
	dir := t.TempDir()
	caminho := filepath.Join(dir, "lote.json")
	valor := amostraParaFormato()
	if err := writeJSONAtomic(caminho, valor); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	var lido amostraDeFormato
	if err := json.Unmarshal(dados, &lido); err != nil {
		t.Fatalf("o arquivo do modo em lote precisa ser JSON válido: %v", err)
	}
	if lido.Nome != valor.Nome {
		t.Fatalf("conteúdo divergente no modo em lote: %+v", lido)
	}
	sobras, err := filepath.Glob(filepath.Join(dir, ".agent-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sobras) != 0 {
		t.Fatalf("o modo em lote não pode deixar temporário para trás: %v", sobras)
	}
}

func TestWriteDurabilityBatchedSyncsFirstWriteAndCountsPending(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(WriteDurabilityBatched); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	alvo := filepath.Join(dir, "a.json")

	// Primeira escrita: sincroniza (a janela não começa aberta) e não deixa pendência.
	if err := writeJSONAtomic(alvo, amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	if pendentes := PendingDurableWrites(); pendentes != 0 {
		t.Fatalf("a primeira escrita precisa sincronizar, pendentes=%d", pendentes)
	}
	// Escrita seguinte DENTRO da janela: grava e fica pendente — é a perda possível, contabilizada.
	if err := writeJSONAtomic(filepath.Join(dir, "b.json"), amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	if pendentes := PendingDurableWrites(); pendentes != 1 {
		t.Fatalf("escrita dentro da janela precisa ficar pendente e contabilizada: %d", pendentes)
	}
	// Flush fecha a janela.
	if err := FlushDurableWrites(); err != nil {
		t.Fatal(err)
	}
	if pendentes := PendingDurableWrites(); pendentes != 0 {
		t.Fatalf("o flush precisa zerar as pendências: %d", pendentes)
	}
}

func TestWriteDurabilityBatchedIsNotTheDefault(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(""); err != nil {
		t.Fatal(err)
	}
	// O modo em lote é uma ESCOLHA: o padrão continua sendo o seguro.
	if modo := WriteDurability(); modo != WriteDurabilityDurable {
		t.Fatalf("o padrão não pode ser o modo em lote: %s", modo)
	}
	// E fora do modo em lote, ninguém fica "pendente".
	if err := writeJSONAtomic(filepath.Join(t.TempDir(), "c.json"), amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	if pendentes := PendingDurableWrites(); pendentes != 0 {
		t.Fatalf("fora do modo em lote não pode haver pendência: %d", pendentes)
	}
}

func TestWriteDurabilityBatchedRejectsUnknownValue(t *testing.T) {
	restoreDurability(t)
	modo, err := SetWriteDurability("lote-aberto")
	if err == nil || modo != WriteDurabilityDurable {
		t.Fatalf("valor desconhecido precisa recusar e cair para o seguro: %q (%v)", modo, err)
	}
}
