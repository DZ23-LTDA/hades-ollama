package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/internal/agent"
)

// O QUE ESTA FATIA FECHA: com a variavel de ambiente posta, o servidor ADOPTA o estado guardado.
func TestArranqueAdoptaOParePersistido(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "pare.json")

	guardado := agent.NovoPareDeEmergencia()
	if err := guardado.Acionar("paragem guardada antes do reinicio", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	if err := guardado.GuardarEm(caminho); err != nil {
		t.Fatalf("guardar: %v", err)
	}

	// Simular arranque: pare do servidor desarmado, variavel apontada ao ficheiro.
	anterior := pareDoServidor
	t.Cleanup(func() { pareDoServidor = anterior })
	pareDoServidor = agent.NovoPareDeEmergencia()
	t.Setenv(variavelDoCaminhoDoPare, caminho)

	if pareDoServidor.Bloqueia() {
		t.Fatal("o pare tem de comecar desarmado antes de adoptar")
	}
	adoptarParePersistido()
	if !pareDoServidor.Bloqueia() {
		t.Fatal("depois do arranque o pare TEM de continuar parado")
	}
	if motivo := pareDoServidor.Estado().Motivo; motivo == "" {
		t.Fatal("o motivo tem de vir do estado guardado")
	}
}

// SEM a variavel, a persistencia fica DESLIGADA -- desligada e' um estado explicito.
func TestArranqueSemVariavelNaoLeNada(t *testing.T) {
	anterior := pareDoServidor
	t.Cleanup(func() { pareDoServidor = anterior })
	pareDoServidor = agent.NovoPareDeEmergencia()

	caminho := filepath.Join(t.TempDir(), "pare.json")
	guardado := agent.NovoPareDeEmergencia()
	if err := guardado.Acionar("nao deve ser lido", "operador"); err != nil {
		t.Fatalf("acionar: %v", err)
	}
	if err := guardado.GuardarEm(caminho); err != nil {
		t.Fatalf("guardar: %v", err)
	}
	// Variavel deliberadamente NAO posta: `t.Setenv` com "" e' explicito.
	t.Setenv(variavelDoCaminhoDoPare, "")

	adoptarParePersistido()
	if pareDoServidor.Bloqueia() {
		t.Fatal("sem variavel o pare NAO pode ser adoptado")
	}
}

// E um ficheiro ilegivel nao impede o arranque nem desarma nada.
func TestArranqueComFicheiroIlegivelNaoRebenta(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "pare.json")
	if err := os.WriteFile(caminho, []byte("{lixo"), 0o600); err != nil {
		t.Fatalf("escrever: %v", err)
	}
	anterior := pareDoServidor
	t.Cleanup(func() { pareDoServidor = anterior })
	pareDoServidor = agent.NovoPareDeEmergencia()
	t.Setenv(variavelDoCaminhoDoPare, caminho)

	adoptarParePersistido() // nao pode entrar em panico
	if pareDoServidor.Bloqueia() {
		t.Fatal("lixo no ficheiro nao pode deixar o pare acionado por engano")
	}
}
