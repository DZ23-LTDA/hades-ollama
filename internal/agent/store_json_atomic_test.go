package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A otimização de E-165 troca `json.MarshalIndent` (buffer inteiro em memória) por um encoder que escreve direto
// no arquivo temporário. Trocar o caminho de escrita **sem teste de formato** seria arriscado: os arquivos do
// Hades são lidos por outras instâncias e por ferramentas externas. Estes testes travam o contrato.

type amostraDeFormato struct {
	Nome     string   `json:"nome"`
	Numero   int      `json:"numero"`
	Itens    []string `json:"itens"`
	Aninhado struct {
		Verdadeiro bool `json:"verdadeiro"`
	} `json:"aninhado"`
}

func amostraParaFormato() amostraDeFormato {
	valor := amostraDeFormato{Nome: "acentuação e <tags> & \"aspas\"", Numero: 42, Itens: []string{"um", "dois"}}
	valor.Aninhado.Verdadeiro = true
	return valor
}

func TestWriteJSONAtomicKeepsTheExactFormat(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "amostra.json")
	valor := amostraParaFormato()
	if err := writeJSONAtomic(caminho, valor); err != nil {
		t.Fatal(err)
	}
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatal(err)
	}
	// O formato de referência é o que o código anterior gravava: indentação de dois espaços + quebra final.
	esperado, err := json.MarshalIndent(valor, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	esperado = append(esperado, '\n')
	if string(dados) != string(esperado) {
		t.Fatalf("o formato do arquivo mudou:\n--- gravado ---\n%s\n--- esperado ---\n%s", dados, esperado)
	}
	// E o conteúdo precisa continuar LEGÍVEL como JSON.
	var lido amostraDeFormato
	if err := json.Unmarshal(dados, &lido); err != nil {
		t.Fatalf("o arquivo precisa continuar sendo JSON válido: %v", err)
	}
	if lido.Nome != valor.Nome || lido.Numero != valor.Numero {
		t.Fatalf("ida e volta divergente: %+v", lido)
	}
}

func TestWriteJSONAtomicBoundedRefusesOversizedValue(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, "grande.json")
	grande := map[string]string{"conteudo": strings.Repeat("x", 4096)}
	err := writeJSONAtomicBounded(caminho, grande, 256, "JSON file")
	if err == nil {
		t.Fatal("valor acima do limite precisa ser recusado")
	}
	// A MENSAGEM é contrato: quem já tratava esse erro não pode passar a ver outro texto.
	if !strings.Contains(err.Error(), "exceeds persistence limit") {
		t.Fatalf("a mensagem do limite precisa ser a mesma de antes: %v", err)
	}
	// E nada pode ter sido gravado no caminho final.
	if _, err := os.Stat(caminho); !os.IsNotExist(err) {
		t.Fatalf("valor recusado não pode aparecer no caminho final: %v", err)
	}
	// Um valor DENTRO do limite continua sendo gravado.
	pequeno := map[string]string{"conteudo": "ok"}
	if err := writeJSONAtomicBounded(caminho, pequeno, 256, "JSON file"); err != nil {
		t.Fatalf("valor dentro do limite precisa ser gravado: %v", err)
	}
	if _, err := os.Stat(caminho); err != nil {
		t.Fatalf("o arquivo precisa existir depois da gravação válida: %v", err)
	}
}
