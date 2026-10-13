package agent

import (
	"os"
	"path/filepath"
	"testing"
)

// E-281 — A PUBLICAÇÃO DECLARA O QUE ENTREGOU. Achado: PublishLocal apenas COPIA arquivos e grava
// Status = "published", sem backend/auth/banco — exatamente o que o master proíbe ("botão que aparenta funcionar").
//
// ESCOPO HONESTO: esta rodada NÃO gera backend. Ela faz a publicação DIZER o que entregou, porque copiar arquivos
// não é aplicação rodando.
func TestPublicacaoDeclaraOQueEntregouEmVezDeApenasDizerPublished(t *testing.T) {
	raiz := t.TempDir()

	// 1. Projeto SÓ com frontend: não é executável e o relatório NOMEIA o que falta.
	front := filepath.Join(raiz, "front")
	if err := os.MkdirAll(filepath.Join(front, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(front, "index.html"), []byte("<html><body>oi</body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(front, "src", "app.tsx"), []byte("export const App = () => null;"), 0o600); err != nil {
		t.Fatal(err)
	}
	relatorio := InspectBuilderPublish(front)
	if relatorio.FilesCopied != 2 {
		t.Fatalf("precisa contar os arquivos copiados: %d", relatorio.FilesCopied)
	}
	if relatorio.Runnable {
		t.Fatalf("sem backend NÃO pode ser executável: %+v", relatorio)
	}
	if relatorio.BackendIncluded {
		t.Fatalf("não há backend neste projeto: %+v", relatorio)
	}
	faltando := map[string]bool{}
	for _, item := range relatorio.Missing {
		faltando[item] = true
	}
	for _, esperado := range []string{"backend", "auth", "banco"} {
		if !faltando[esperado] {
			t.Fatalf("o relatório precisa NOMEAR o que falta (%s): %+v", esperado, relatorio.Missing)
		}
	}
	if relatorio.Note == "" {
		t.Fatal("o relatório precisa EXPLICAR por que não é executável")
	}

	// 2. Projeto COM backend de verdade: a detecção é MEDIDA, não fixa.
	comBackend := filepath.Join(raiz, "full")
	if err := os.MkdirAll(comBackend, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comBackend, "server.go"), []byte("package main\n// auth com database sqlite\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	completo := InspectBuilderPublish(comBackend)
	if !completo.BackendIncluded || !completo.Runnable {
		t.Fatalf("backend presente precisa ser detectado: %+v", completo)
	}
	if !completo.AuthIncluded || !completo.DatabaseIncluded {
		t.Fatalf("auth e banco presentes precisam ser detectados: %+v", completo)
	}

	// 3. Diretório VAZIO não vira "pronto".
	vazio := filepath.Join(raiz, "vazio")
	if err := os.MkdirAll(vazio, 0o700); err != nil {
		t.Fatal(err)
	}
	semNada := InspectBuilderPublish(vazio)
	if semNada.FilesCopied != 0 || semNada.Runnable || semNada.Note == "" {
		t.Fatalf("publicação sem arquivos precisa ser dita como tal: %+v", semNada)
	}
}
