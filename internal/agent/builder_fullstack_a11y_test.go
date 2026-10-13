package agent

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// TestAppFullstackGeradoAtendeContratoDeAcessibilidade cobre o limite declarado em E-248: o app gerado era
// responsivo e com landmarks, mas ninguem VERIFICAVA isso — um template que perde um "<label for>" passaria
// despercebido e o app nasceria inacessivel.
//
// O teste sobe o app de verdade (staging de E-248), busca o HTML servido e checa o contrato. E checa TAMBEM o
// sentido inverso: um HTML deliberadamente quebrado precisa ser ACUSADO, senao a verificacao seria decorativa.

// verificarAcessibilidade devolve os problemas encontrados. Lista vazia = contrato atendido.
func verificarAcessibilidade(documento string) []string {
	var problemas []string
	raiz, err := html.Parse(strings.NewReader(documento))
	if err != nil {
		return []string{"HTML nao pode ser interpretado: " + err.Error()}
	}

	var (
		totalH1     int
		totalTabela int
		rotulos     = map[string]bool{}
		controls    []*html.Node
		elementos   []*html.Node
	)
	var percorrer func(*html.Node)
	percorrer = func(no *html.Node) {
		if no.Type == html.ElementNode {
			elementos = append(elementos, no)
			switch no.Data {
			case "html":
				if atributo(no, "lang") == "" {
					problemas = append(problemas, "<html> sem atributo lang")
				}
			case "input":
				controls = append(controls, no)
			case "button":
				if textoDoNo(no) == "" && atributo(no, "aria-label") == "" {
					problemas = append(problemas, "<button> sem texto e sem aria-label")
				}
			case "h1":
				totalH1++
			case "table":
				totalTabela++
			case "label":
				if alvo := atributo(no, "for"); alvo != "" {
					rotulos[alvo] = true
				}
			}
		}
		for filho := no.FirstChild; filho != nil; filho = filho.NextSibling {
			percorrer(filho)
		}
	}
	percorrer(raiz)

	// Cada campo precisa de rotulo associado (label[for=id]) ou nome acessivel explicito.
	for _, controle := range controls {
		id := atributo(controle, "id")
		if id != "" && rotulos[id] {
			continue
		}
		if atributo(controle, "aria-label") != "" || atributo(controle, "aria-labelledby") != "" {
			continue
		}
		problemas = append(problemas, "<input> sem rotulo associado (id="+id+")")
	}

	// Cabecalho de tabela precisa de escopo, e a tabela precisa de legenda.
	for _, elemento := range elementos {
		if elemento.Data == "th" && atributo(elemento, "scope") == "" {
			problemas = append(problemas, "<th> sem atributo scope")
		}
		if elemento.Data == "table" && !temDescendente(elemento, "caption") {
			problemas = append(problemas, "<table> sem <caption>")
		}
	}

	if totalH1 != 1 {
		problemas = append(problemas, "a pagina precisa de exatamente um <h1>")
	}
	if totalTabela > 0 && totalH1 == 0 {
		problemas = append(problemas, "pagina com tabela e sem titulo principal")
	}
	// Requisitos que valem para PAGINA servida (nao para fragmento): viewport e foco visivel.
	if !strings.Contains(documento, `name="viewport"`) {
		problemas = append(problemas, "sem meta viewport (nao responde a tela pequena)")
	}
	if !strings.Contains(documento, ":focus-visible") {
		problemas = append(problemas, "sem estilo :focus-visible (navegacao por teclado sem indicacao)")
	}
	return problemas
}

func atributo(no *html.Node, chave string) string {
	for _, a := range no.Attr {
		if a.Key == chave {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}

func textoDoNo(no *html.Node) string {
	var construtor strings.Builder
	var percorrer func(*html.Node)
	percorrer = func(atual *html.Node) {
		if atual.Type == html.TextNode {
			construtor.WriteString(atual.Data)
		}
		for filho := atual.FirstChild; filho != nil; filho = filho.NextSibling {
			percorrer(filho)
		}
	}
	percorrer(no)
	return strings.TrimSpace(construtor.String())
}

func temDescendente(no *html.Node, nome string) bool {
	for filho := no.FirstChild; filho != nil; filho = filho.NextSibling {
		if filho.Type == html.ElementNode && filho.Data == nome {
			return true
		}
		if temDescendente(filho, nome) {
			return true
		}
	}
	return false
}

func TestVerificacaoDeAcessibilidadeAcusaHTMLQuebrado(t *testing.T) {
	// Sentido inverso: se a verificacao nao acusar um HTML ruim, ela nao serve para nada.
	ruim := `<!doctype html><html><head><title>x</title></head><body>
		<h1>a</h1><h1>b</h1>
		<input id="sem-rotulo">
		<button></button>
		<table><tr><th>Coluna</th></tr></table>
	</body></html>`
	problemas := verificarAcessibilidade(ruim)
	esperados := []string{"sem atributo lang", "sem rotulo associado", "<button> sem texto", "exatamente um <h1>", "<th> sem atributo scope", "sem <caption>", "sem meta viewport", "sem estilo :focus-visible"}
	for _, esperado := range esperados {
		encontrado := false
		for _, problema := range problemas {
			if strings.Contains(problema, esperado) {
				encontrado = true
				break
			}
		}
		if !encontrado {
			t.Fatalf("verificacao deveria acusar %q; acusou: %v", esperado, problemas)
		}
	}
	t.Logf("VERIFICACAO INVERSA: %d problemas acusados em HTML quebrado", len(problemas))
}

func TestAppFullstackGeradoAtendeContratoDeAcessibilidade(t *testing.T) {
	servico, err := NewBuilderService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	projeto, err := servico.Create(context.Background(), BuilderSpec{Name: "Gestao", Kind: BuilderFullstack})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := servico.PublishLocal(context.Background(), projeto.ID); err != nil {
		t.Fatal(err)
	}
	palco, err := servico.StageFullstack(context.Background(), projeto.ID, "senha-de-acessibilidade")
	if err != nil {
		t.Fatalf("staging falhou: %v", err)
	}
	defer palco.Stop()

	resposta, err := http.Get(palco.URL + "/")
	if err != nil {
		t.Fatalf("buscar a pagina do app gerado: %v", err)
	}
	defer resposta.Body.Close()
	if resposta.StatusCode != http.StatusOK {
		t.Fatalf("pagina do app gerado deveria responder 200, veio %d", resposta.StatusCode)
	}
	corpo, err := io.ReadAll(resposta.Body)
	if err != nil {
		t.Fatal(err)
	}

	problemas := verificarAcessibilidade(string(corpo))
	if len(problemas) > 0 {
		t.Fatalf("app gerado viola o contrato de acessibilidade: %v", problemas)
	}
	// O botao criado por JavaScript tambem precisa de nome acessivel.
	if !strings.Contains(string(corpo), "aria-label") {
		t.Fatal("a pagina nao declara nome acessivel para o controle criado por JavaScript")
	}
	t.Logf("ACESSIBILIDADE DO APP GERADO: 0 problemas · html_lang=ok viewport=ok rotulos=ok foco_visivel=ok")
}
