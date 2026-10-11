package agent

import (
	"strings"
	"testing"
)

// POLÍTICA DE DURABILIDADE NO RUNTIME (E-168). O runtime é quem aplica a configuração na prática — então é ele
// que precisa provar que: (a) o padrão é seguro; (b) o modo pedido vale; e (c) configuração inválida NÃO passa
// em silêncio (fail-closed), porque uma garantia de durabilidade trocada sem ninguém saber é pior que um erro
// de configuração.

func TestRuntimeDurabilityDefaultsToSafe(t *testing.T) {
	restoreDurability(t)
	t.Setenv("HADES_WRITE_DURABILITY", "")
	runtime, err := NewRuntime(RuntimeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if modo := runtime.WriteDurabilityMode(); modo != WriteDurabilityDurable {
		t.Fatalf("o padrão do runtime precisa ser o modo seguro: %s", modo)
	}
}

func TestRuntimeDurabilityAppliesConfiguredMode(t *testing.T) {
	restoreDurability(t)
	t.Setenv("HADES_WRITE_DURABILITY", WriteDurabilityRelaxed)
	runtime, err := NewRuntime(RuntimeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if modo := runtime.WriteDurabilityMode(); modo != WriteDurabilityRelaxed {
		t.Fatalf("o runtime precisa relatar o modo pedido: %s", modo)
	}
	if WriteDurability() != WriteDurabilityRelaxed {
		t.Fatalf("a política global precisa refletir o modo pedido: %s", WriteDurability())
	}
}

func TestRuntimeDurabilityInvalidValueFailsLoudly(t *testing.T) {
	restoreDurability(t)
	t.Setenv("HADES_WRITE_DURABILITY", "rapido-demais")
	runtime, err := NewRuntime(RuntimeConfig{})
	if err == nil {
		t.Fatal("configuração inválida de durabilidade precisa INTERROMPER a criação do runtime")
	}
	if runtime != nil {
		t.Fatalf("runtime não pode ser criado com configuração inválida: %+v", runtime)
	}
	if !strings.Contains(err.Error(), "desconhecido") {
		t.Fatalf("a mensagem precisa dizer que o valor não foi reconhecido: %v", err)
	}
	// E a política fica no modo SEGURO depois da recusa.
	if modo := WriteDurability(); modo != WriteDurabilityDurable {
		t.Fatalf("depois de recusar, o sistema precisa ficar no modo seguro: %s", modo)
	}
}
