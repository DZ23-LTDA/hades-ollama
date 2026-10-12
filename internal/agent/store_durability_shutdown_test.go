package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// GANCHO DE ENCERRAMENTO (E-250): o modo em lote existe desde E-169, mas o item ficou ABERTO — "o lote só fecha
// com OUTRA escrita depois da janela ou com FlushDurableWrites explícito; NÃO há hook de encerramento ligado ao
// flush". Ou seja: escrever uma vez e encerrar podia perder o último lote.
//
// Este teste NÃO usa outra escrita para fechar a janela: a ÚNICA coisa chamada depois da escrita é o gancho de
// encerramento — exatamente como o processo faz ao receber SIGINT/SIGTERM.
func TestFlushDurableWritesOnShutdownFechaJanelaSemOutraEscrita(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(WriteDurabilityBatched); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// A primeira escrita sincroniza e abre a janela; a segunda fica PENDENTE (a perda possível, contabilizada).
	if err := writeJSONAtomic(filepath.Join(dir, "primeira.json"), amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	alvo := filepath.Join(dir, "ultima.json")
	if err := writeJSONAtomic(alvo, amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	if pendentes := PendingDurableWrites(); pendentes != 1 {
		t.Fatalf("a escrita dentro da janela precisa ficar pendente: %d", pendentes)
	}

	sincronizados, err := FlushDurableWritesOnShutdown()
	if err != nil {
		t.Fatalf("o gancho de encerramento falhou: %v", err)
	}
	if sincronizados != 1 {
		t.Fatalf("o gancho precisa INFORMAR quantos itens sincronizou (sem isso a falha seria invisível): %d", sincronizados)
	}
	if pendentes := PendingDurableWrites(); pendentes != 0 {
		t.Fatalf("o gancho precisa fechar a janela: %d pendentes", pendentes)
	}

	// Durabilidade de verdade: o conteúdo precisa estar no disco e legível — contador zerado não prova nada.
	dados, err := os.ReadFile(alvo)
	if err != nil {
		t.Fatalf("o arquivo sincronizado precisa estar legível: %v", err)
	}
	var lido amostraDeFormato
	if err := json.Unmarshal(dados, &lido); err != nil {
		t.Fatalf("conteúdo inválido após o encerramento: %v", err)
	}
	if lido.Nome != amostraParaFormato().Nome {
		t.Fatalf("conteúdo divergente após o encerramento: %+v", lido)
	}

	// Segundo encerramento: no-op honesto (nada pendente, zero sincronizado).
	if deNovo, err := FlushDurableWritesOnShutdown(); err != nil || deNovo != 0 {
		t.Fatalf("segundo encerramento deveria ser no-op: itens=%d err=%v", deNovo, err)
	}
}

// O gancho não pode MENTIR fora do modo em lote: sem janela aberta ele informa zero.
func TestFlushDurableWritesOnShutdownNaoInventaPendencia(t *testing.T) {
	restoreDurability(t)
	if _, err := SetWriteDurability(WriteDurabilityDurable); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(filepath.Join(t.TempDir(), "seguro.json"), amostraParaFormato()); err != nil {
		t.Fatal(err)
	}
	sincronizados, err := FlushDurableWritesOnShutdown()
	if err != nil {
		t.Fatalf("no modo seguro o gancho não deve falhar: %v", err)
	}
	if sincronizados != 0 {
		t.Fatalf("no modo seguro não há pendência a sincronizar: %d", sincronizados)
	}
}
