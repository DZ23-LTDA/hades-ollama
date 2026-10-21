package agent

import (
	"strings"
	"testing"
)

// FATIA 4 / T3. Os quatro nomes abaixo foram INVENTADOS pelo modelo no chat do dono e nao existem
// no binario. O teste prende essa alucinacao: se algum dia alguem os "documentar" como reais, isto
// falha.
func TestListaFechadaRecusaComandosInventados(t *testing.T) {
	r := NewRegistry()

	for _, inventado := range []string{"create-dir", "create-file", "list-files", "open-browser"} {
		if r.FerramentaExiste(inventado) {
			t.Fatalf("%q nao existe no registo, mas foi aceite como ferramenta", inventado)
		}
	}

	// Vazio e espacos tambem nao sao ferramenta.
	if r.FerramentaExiste("") || r.FerramentaExiste("   ") {
		t.Fatal("nome vazio nao pode ser aceite como ferramenta")
	}

	// POSITIVO: tudo o que o registo anuncia tem de ser reconhecido pela mesma validacao.
	nomes := r.NomesFerramentas()
	if len(nomes) == 0 {
		t.Fatal("o registo nao anunciou nenhuma ferramenta")
	}
	for _, nome := range nomes {
		if !r.FerramentaExiste(nome) {
			t.Fatalf("FerramentaExiste recusou %q, que o proprio registo anuncia", nome)
		}
		if strings.TrimSpace(nome) != nome {
			t.Fatalf("nome com espacos no registo: %q", nome)
		}
	}

	// A lista vem ordenada (serve para o prompt e para comparacao estavel).
	for i := 1; i < len(nomes); i++ {
		if nomes[i-1] > nomes[i] {
			t.Fatalf("lista nao ordenada: %q > %q", nomes[i-1], nomes[i])
		}
	}
}
