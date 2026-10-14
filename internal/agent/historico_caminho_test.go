package agent

import (
	"path/filepath"
	"strings"
	"testing"
)

// Testes do caminho do histórico (E-581). O objecto é a RECUSA: estes identificadores
// chegam de um pedido HTTP, e concatená-los sem validar é travessia de directório.

func TestCaminhoDoHistoricoMontaDentroDaRaiz(t *testing.T) {
	raiz := t.TempDir()
	caminho, err := CaminhoDoHistorico(raiz, "org-a", "proj-1")
	if err != nil {
		t.Fatalf("recusou um caso legítimo: %v", err)
	}
	esperado := filepath.Join(raiz, "historico", "org-a", "proj-1.jsonl")
	if caminho != esperado {
		t.Fatalf("caminho = %q, esperado %q", caminho, esperado)
	}
	// E está DENTRO da raiz — a propriedade que interessa a quem for escrever.
	relativo, err := filepath.Rel(raiz, caminho)
	if err != nil || strings.HasPrefix(relativo, "..") {
		t.Fatalf("o caminho não está contido na raiz: %q", relativo)
	}
}

func TestCaminhoDoHistoricoRecusaTravessia(t *testing.T) {
	raiz := t.TempDir()
	maus := []struct {
		nome string
		org  string
		proj string
	}{
		{"projeto sobe um nível", "org-a", ".."},
		{"projeto relativo com travessia", "org-a", "../../etc/passwd"},
		{"organização com travessia", "../../etc", "proj-1"},
		{"separador unix no projeto", "org-a", "a/b"},
		{"separador windows no projeto", "org-a", `a\b`},
		{"caminho absoluto unix", "org-a", "/etc/passwd"},
		{"byte nulo", "org-a", "proj\x00-1"},
		{"dois-pontos (unidade/ADS no Windows)", "org-a", "proj:stream"},
		{"ponto único", ".", "proj-1"},
		{"organização vazia", "   ", "proj-1"},
		{"projeto vazio", "org-a", ""},
		{"projeto gigante", "org-a", strings.Repeat("x", LimiteDeIdentificadorNoCaminho+1)},
	}
	for _, caso := range maus {
		t.Run(caso.nome, func(t *testing.T) {
			if caminho, err := CaminhoDoHistorico(raiz, caso.org, caso.proj); err == nil {
				t.Fatalf("ACEITOU %q / %q e produziria %q", caso.org, caso.proj, caminho)
			}
		})
	}
	// Raiz vazia: sem raiz não há onde guardar — recusa em vez de escrever no cwd.
	if _, err := CaminhoDoHistorico("", "org-a", "proj-1"); err == nil {
		t.Fatal("aceitou raiz vazia")
	}
}

func TestCaminhoDoHistoricoNaoEhAfectadoPorPontoEmIdentificadorLegitimo(t *testing.T) {
	raiz := t.TempDir()
	// Um ponto DENTRO do nome é legítimo (`proj.v2`, `org.br`): recusar isto seria
	// excesso de zelo que quebraria identificadores reais. O que se recusa é o SEGMENTO
	// `..`, não a presença de pontos.
	for _, ok := range []struct{ org, proj string }{
		{"org.br", "proj.v2"},
		{"org_a", "proj-1.2.3"},
	} {
		if _, err := CaminhoDoHistorico(raiz, ok.org, ok.proj); err != nil {
			t.Fatalf("recusou identificadores legítimos %q/%q: %v", ok.org, ok.proj, err)
		}
	}
}
