package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Estágio 12 — O RELATÓRIO DE PUBLICAÇÃO ESTÁ LIGADO À REALIDADE (E-283).
//
// ACHADO DA AUDITORIA desta rodada: o gerador de backend JÁ EXISTE — o tipo `BuilderFullstack` produz `go.mod` e
// `main.go` de verdade (o comentário no código registra o incidente de CI que obrigou o módulo a ser autossuficiente).
// O que faltava NÃO era o gerador: era EVIDÊNCIA de que o relatório de E-281/E-282 diz a verdade.
//
// As duas direções são provadas aqui:
//  1. `kind: app`        => Runnable FALSO (só frontend);
//  2. `kind: fullstack`  => Runnable VERDADEIRO (backend presente);
//  3. o backend publicado COMPILA de fato (`go build ./...`) — código que compila é código, não enfeite.
func TestRelatorioDePublicacaoDizAVerdadeNasDuasDirecoes(t *testing.T) {
	ctx := context.Background()
	raiz := t.TempDir()
	servico, err := NewBuilderService(filepath.Join(raiz, "builder"))
	if err != nil {
		t.Fatal(err)
	}

	// 1. Só frontend: NÃO é executável.
	front, err := servico.Create(ctx, BuilderSpec{Name: "Loja Vitrine", Kind: BuilderApp})
	if err != nil {
		t.Fatalf("criar app: %v", err)
	}
	publicadoFront, caminhoFront, err := servico.PublishLocal(ctx, front.ID)
	if err != nil {
		t.Fatalf("publicar app: %v", err)
	}
	if publicadoFront.PublishReport.Runnable {
		t.Fatalf("app só-frontend NÃO pode ser executável: %+v", publicadoFront.PublishReport)
	}
	if publicadoFront.PublishReport.BackendIncluded {
		t.Fatalf("app só-frontend não tem backend: %+v", publicadoFront.PublishReport)
	}
	if publicadoFront.PublishReport.FilesCopied == 0 {
		t.Fatal("a publicação precisa ter copiado arquivos")
	}

	// 2. Fullstack: backend presente => executável.
	completo, err := servico.Create(ctx, BuilderSpec{Name: "Gestão de Estoque", Kind: BuilderFullstack})
	if err != nil {
		t.Fatalf("criar fullstack: %v", err)
	}
	publicadoFull, caminhoFull, err := servico.PublishLocal(ctx, completo.ID)
	if err != nil {
		t.Fatalf("publicar fullstack: %v", err)
	}
	relatorio := publicadoFull.PublishReport
	if !relatorio.BackendIncluded || !relatorio.Runnable {
		t.Fatalf("fullstack precisa ser detectado como executável: %+v (arquivos em %s)", relatorio, caminhoFull)
	}
	if relatorio.FilesCopied < 3 {
		t.Fatalf("fullstack precisa publicar módulo, código e página: %d arquivo(s)", relatorio.FilesCopied)
	}
	// O relatório NÃO promete mais do que entrega: mesmo executável, o aviso de rodar/testar continua.
	if relatorio.Note == "" {
		t.Fatal("o relatório precisa continuar declarando o próximo passo")
	}
	_ = caminhoFront

	// 3. COMPILA DE VERDADE. Sem toolchain disponível o teste DECLARA o pulo em vez de fingir sucesso.
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("toolchain Go indisponível: compilação do backend gerado NÃO foi medida")
	}
	construir := exec.Command("go", "build", "./...")
	construir.Dir = caminhoFull
	construir.Env = os.Environ()
	saida, err := construir.CombinedOutput()
	if err != nil {
		t.Fatalf("backend gerado NÃO compila (código decorativo seria pior que ausente): %v\n%s", err, string(saida))
	}

	// 4. O módulo gerado é AUTOSSUFICIENTE: sem go.mod, o build passaria só por estar dentro do módulo do Hades.
	if _, err := os.Stat(filepath.Join(caminhoFull, "go.mod")); err != nil {
		t.Fatalf("o projeto publicado precisa ter go.mod próprio: %v", err)
	}
	// E o relatório sobrevive ao reinício do serviço (é persistido com o projeto, não só devolvido na chamada).
	reaberto, err := NewBuilderService(filepath.Join(raiz, "builder"))
	if err != nil {
		t.Fatal(err)
	}
	recuperado, err := reaberto.Get(completo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !recuperado.PublishReport.Runnable {
		t.Fatalf("o relatório precisa sobreviver ao reinício: %+v", recuperado.PublishReport)
	}
	if recuperado.PublishReport.FilesCopied != relatorio.FilesCopied {
		t.Fatalf("contagem de arquivos precisa ser estável: %d != %d", recuperado.PublishReport.FilesCopied, relatorio.FilesCopied)
	}
	// Data de publicação não pode ser zero: o projeto foi realmente publicado.
	if recuperado.UpdatedAt.IsZero() || recuperado.UpdatedAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("data de atualização suspeita: %v", recuperado.UpdatedAt)
	}
}
