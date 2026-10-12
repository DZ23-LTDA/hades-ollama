package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// E2E DO BUILDER: PUBLICAÇÃO SERVIDA E NAVEGÁVEL (E-178).
//
// O item da matriz de aceite "O Builder entrega aplicações utilizáveis" estava **NAO_COMPROVADO**: existiam
// contrato e testes herméticos, mas nenhuma prova de que a aplicação publicada **abre e renderiza**. Este teste
// fecha a parte que dá para provar dentro do repositório, de forma reprodutível:
//
//  1. cria um projeto pelo Builder REAL;
//  2. aplica componentes visuais (o canvas do Studio);
//  3. **PUBLICA** (`PublishLocal`) e confere os arquivos publicados;
//  4. **SERVE** o diretório publicado por HTTP de verdade;
//  5. busca a página como um navegador faria e confere que o HTML entregue contém o canvas e o dado visual.
//
// A navegação com NAVEGADOR REAL é o passo seguinte e fica registrada na evidência (o caminho publicado é escrito
// em `_tmp/builder-e2e-path.txt` para quem for abrir no navegador).
func TestBuilderPublishedAppIsServedAndNavigable(t *testing.T) {
	// Por padrão o teste é HERMÉTICO (diretório temporário, apagado no fim). Quando
	// `HADES_BUILDER_E2E_ROOT` está definido, a publicação vai para um diretório ESTÁVEL — é o que permite o passo
	// seguinte, de NAVEGADOR REAL, abrir a MESMA aplicação publicada que este teste verificou.
	raiz := os.Getenv("HADES_BUILDER_E2E_ROOT")
	if raiz == "" {
		raiz = t.TempDir()
	} else if err := os.MkdirAll(raiz, 0o700); err != nil {
		t.Fatalf("não foi possível preparar o diretório estável: %v", err)
	}
	servico, err := NewBuilderService(raiz)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	projeto, err := servico.Create(ctx, BuilderSpec{
		Name:           "Estoque E2E",
		Kind:           BuilderApp,
		OrganizationID: "org-e2e-builder",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servico.ApplyVisualComponents(ctx, projeto.ID, []VisualComponent{
		{ID: "c1", Type: "heading", Props: map[string]string{"text": "Estoque E2E"}},
		{ID: "c2", Type: "table", Props: map[string]string{"text": "Produtos"}},
		{ID: "c3", Type: "button", Props: map[string]string{"text": "Novo produto"}},
	}); err != nil {
		t.Fatal(err)
	}

	// PUBLICA: é o que transforma o rascunho em aplicação entregável.
	publicado, caminho, err := servico.PublishLocal(ctx, projeto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if caminho == "" {
		t.Fatal("PublishLocal precisa devolver o caminho publicado")
	}
	indice := filepath.Join(caminho, "index.html")
	if _, err := os.Stat(indice); err != nil {
		t.Fatalf("a aplicação publicada precisa ter index.html: %v", err)
	}
	if publicado.PublishedPath != caminho {
		t.Fatalf("o projeto publicado precisa registrar o caminho: %q != %q", publicado.PublishedPath, caminho)
	}

	// SERVE por HTTP real, como um host estático faria.
	servidor := httptest.NewServer(http.FileServer(http.Dir(caminho)))
	defer servidor.Close()

	resposta, err := http.Get(servidor.URL + "/index.html")
	if err != nil {
		t.Fatal(err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("a aplicação publicada precisa responder 200: %d", resposta.StatusCode)
	}
	corpo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatal(err)
	}
	pagina := string(corpo)
	for _, esperado := range []string{"<!doctype html>", "dz23-canvas", "dz23-visual", "Estoque E2E"} {
		if !strings.Contains(pagina, esperado) {
			t.Fatalf("o HTML servido precisa conter %q: %s", esperado, pagina[:min(len(pagina), 240)])
		}
	}
	// O dado visual precisa estar EMBUTIDO (é o que o navegador desenha): não basta a casca.
	if !strings.Contains(pagina, "Novo produto") || !strings.Contains(pagina, "Produtos") {
		t.Fatalf("os componentes visuais precisam estar no HTML servido: %s", pagina[:min(len(pagina), 400)])
	}

	// O caminho fica registrado para o passo de NAVEGADOR REAL (que roda fora do teste).
	if err := os.WriteFile(filepath.Join("..", "..", "_tmp", "builder-e2e-path.txt"), []byte(caminho), 0o600); err != nil {
		t.Logf("não foi possível registrar o caminho para o passo de navegador: %v", err)
	}
}
