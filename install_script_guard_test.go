package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Guarda do contrato de instalação, atualização, rollback e desinstalação.
//
// O gate intermediário de engenharia (estágio 22) exige install/upgrade/rollback. O
// script `scripts/install.ps1` entrega os quatro caminhos, e esta guarda impede que uma
// edição futura remova um deles **em silêncio** — o script voltaria a instalar sem
// volta, e ninguém notaria até precisar reverter uma atualização quebrada.
//
// A guarda é ESTÁTICA de propósito: ela verifica o contrato do script, não substitui a
// execução real do ciclo, que é exercitada e registrada como evidência (E-113).
func TestInstallScriptKeepsInstallUpgradeRollbackAndUninstall(t *testing.T) {
	path := filepath.Join("scripts", "install.ps1")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("o script de instalação precisa existir em %s: %v", path, err)
	}
	script := string(raw)

	required := []struct {
		name   string
		marker string
	}{
		{"desinstalação", "$Uninstall"},
		{"rollback", "$Rollback"},
		{"preservação do binário anterior", "$binaryName.previous"},
		{"manifesto com hash", "install-manifest.json"},
		{"hash SHA-256", "Get-FileHash"},
		{"build a partir do código", "go build"},
		{"flag de release", "$Ldflags"},
	}
	for _, item := range required {
		if !strings.Contains(script, item.marker) {
			t.Fatalf("o script de instalação perdeu o caminho de %s (marcador %q ausente)", item.name, item.marker)
		}
	}

	// A desinstalação NÃO pode remover dados do usuário: apagar modelos junto seria
	// perda de dados, não limpeza.
	if strings.Contains(script, "Remove-Item -Path $modelsDir") || strings.Contains(script, "OLLAMA_MODELS") {
		t.Fatal("a desinstalação não pode remover os dados do usuário (modelos)")
	}
	if !strings.Contains(script, "Os dados do usuário (modelos) foram preservados") {
		t.Fatal("a desinstalação precisa DECLARAR que preserva os dados do usuário")
	}

	// O rollback precisa ser reversível: o binário substituído é preservado, então dá
	// para voltar atrás e avançar de novo.
	if !strings.Contains(script, ".rolledback") {
		t.Fatal("o rollback precisa preservar o binário que estava instalado")
	}
}
