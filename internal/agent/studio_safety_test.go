package agent

// FASE 26.5 / 26.12: testes NEGATIVOS do validador de conteudo do Studio.
//
// Cada caso tem DOIS lados, e o segundo e' o que impede o validador de ser teatro:
//
//	o ataque e' recusado   E   o conteudo legitimo e' aceite
//
//	*Um validador que recusa tudo passa em todos os testes negativos e destroi o produto.*

import (
	"errors"
	"strings"
	"testing"
)

// === URLs ===

func TestValidarURLDeConteudoRecusaEsquemasPerigosos(t *testing.T) {
	casos := []struct{ nome, url string }{
		{"javascript simples", "javascript:alert(1)"},
		{"javascript com caixa mista", "JaVaScRiPt:alert(1)"},
		{"javascript com espacos", "java script:alert(1)"},
		{"vbscript", "vbscript:msgbox(1)"},
		{"data html", "data:text/html,<script>alert(1)</script>"},
		{"data base64", "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg=="},
		{"file", "file:///C:/Windows/System32/config/SAM"},
		{"blob", "blob:https://x/abc"},
		{"about", "about:blank"},
		{"esquema desconhecido", "gopher://x/"},
		{"protocolo relativa", "//evil.example.com/x.js"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if err := ValidarURLDeConteudo(c.url); err == nil {
				t.Fatalf("URL %q devia ser recusada", c.url)
			} else if !errors.Is(err, ErrConteudoInseguro) {
				t.Fatalf("erro nao e' ErrConteudoInseguro: %v", err)
			}
		})
	}
}

func TestValidarURLDeConteudoRecusaCaractereDeControlo(t *testing.T) {
	// `java\nscript:` e' o bypass classico: o parser do browser ignora o controlo e o da' como
	// `javascript:`.
	for _, u := range []string{"java\nscript:alert(1)", "java\tscript:x", "javascript:\x00"} {
		if err := ValidarURLDeConteudo(u); err == nil {
			t.Fatalf("URL com caractere de controlo %q devia ser recusada", u)
		}
	}
}

func TestValidarURLDeConteudoAceitaOLegitimo(t *testing.T) {
	casos := []string{
		"https://exemplo.com/imagem.png",
		"http://localhost:3000/api",
		"mailto:alguem@exemplo.com",
		"tel:+5511999999999",
		"/caminho/relativo",
		"caminho/relativo.png",
		"#ancora",
		"?consulta=1",
		"https://exemplo.com/a?b=c#d",
	}
	for _, u := range casos {
		if err := ValidarURLDeConteudo(u); err != nil {
			t.Fatalf("URL legitima %q recusada: %v", u, err)
		}
	}
}

func TestValidarURLDeConteudoRecusaVaziaEExigeHost(t *testing.T) {
	if err := ValidarURLDeConteudo(""); err == nil {
		t.Fatal("URL vazia devia ser recusada")
	}
	if err := ValidarURLDeConteudo("https:///sem-host"); err == nil {
		t.Fatal("https sem host devia ser recusado")
	}
}

// === Caminhos de asset ===

func TestValidarCaminhoDeAssetRecusaTravessia(t *testing.T) {
	casos := []struct{ nome, caminho string }{
		{"travessia simples", "../segredo.txt"},
		{"travessia dupla", "../../.ssh/id_rsa"},
		{"travessia no meio", "assets/../../fora.txt"},
		{"travessia com barra invertida", "assets\\..\\..\\fora.txt"},
		{"absoluto unix", "/etc/passwd"},
		{"absoluto windows", `C:\Windows\win.ini`},
		{"unidade relativa", "C:segredo.txt"},
		{"byte nulo", "assets/imagem.png\x00.txt"},
		{"apenas pontos", ".."},
		{"ponto", "."},
		{"vazio", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if err := ValidarCaminhoDeAsset(c.caminho); err == nil {
				t.Fatalf("caminho %q devia ser recusado", c.caminho)
			} else if !errors.Is(err, ErrConteudoInseguro) {
				t.Fatalf("erro nao e' ErrConteudoInseguro: %v", err)
			}
		})
	}
}

func TestValidarCaminhoDeAssetAceitaOLegitimo(t *testing.T) {
	casos := []string{
		"imagem.png",
		"assets/logo.svg",
		"assets/sub/pasta/ficheiro.webp",
		"a/b/c/d.txt",
		// Ponto no NOME do ficheiro nao e' travessia.
		"arquivo..nome.txt",
		"pasta.v2/ficheiro.png",
	}
	for _, c := range casos {
		if err := ValidarCaminhoDeAsset(c); err != nil {
			t.Fatalf("caminho legitimo %q recusado: %v", c, err)
		}
	}
}

// === HTML de conteudo ===

func TestValidarHTMLDeConteudoRecusaExecucao(t *testing.T) {
	casos := []struct{ nome, html string }{
		{"tag script", `<div><script>alert(1)</script></div>`},
		{"script com espaco", `<div>< script >alert(1)</ script></div>`},
		{"script caixa mista", `<SCRIPT>alert(1)</SCRIPT>`},
		{"handler onerror", `<img src=x onerror=alert(1)>`},
		{"handler onclick", `<button onclick="alert(1)">x</button>`},
		{"handler onload", `<body onload=alert(1)>`},
		{"javascript em href", `<a href="javascript:alert(1)">x</a>`},
		{"javascript espacado", `<a href="j a v a s c r i p t :alert(1)">x</a>`},
		{"iframe", `<iframe src="https://evil.example.com"></iframe>`},
		{"iframe srcdoc", `<iframe srcdoc="<script>alert(1)</script>"></iframe>`},
		{"object", `<object data="evil.swf"></object>`},
		{"embed", `<embed src="evil.swf">`},
		{"meta refresh", `<meta http-equiv="refresh" content="0;url=https://evil.example.com">`},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if err := ValidarHTMLDeConteudo(c.html); err == nil {
				t.Fatalf("HTML %q devia ser recusado", c.html)
			} else if !errors.Is(err, ErrConteudoInseguro) {
				t.Fatalf("erro nao e' ErrConteudoInseguro: %v", err)
			}
		})
	}
}

func TestValidarHTMLDeConteudoAceitaOLegitimo(t *testing.T) {
	casos := []string{
		"<div>texto simples</div>",
		`<a href="https://exemplo.com">link seguro</a>`,
		`<img src="assets/logo.png" alt="logo">`,
		`<h1 class="titulo">Titulo</h1>`,
		// Palavras que COMECAM por "on" mas nao sao handlers.
		`<p>online, once, apenas, ontem</p>`,
		// "javascript" como TEXTO e nao como esquema.
		`<p>a linguagem javascript e popular</p>`,
		// O proprio texto da regra nao deve disparar nada.
		`<code>&lt;script&gt;</code>`,
	}
	for _, c := range casos {
		if err := ValidarHTMLDeConteudo(c); err != nil {
			t.Fatalf("HTML legitimo %q recusado: %v", c, err)
		}
	}
}

func TestValidarHTMLDeConteudoVazioEAceite(t *testing.T) {
	for _, h := range []string{"", "   ", "\n\t"} {
		if err := ValidarHTMLDeConteudo(h); err != nil {
			t.Fatalf("HTML vazio %q devia ser aceite: %v", h, err)
		}
	}
}

// O diagnostico devolve TODAS as violacoes, nao so' a primeira.
func TestDiagnosticarHTMLDeConteudoDevolveTodas(t *testing.T) {
	html := `<div><script>a</script><img src=x onerror=b><a href="javascript:c">d</a></div>`
	recusas := DiagnosticarHTMLDeConteudo(html)
	if len(recusas) < 3 {
		t.Fatalf("recusas = %d; esperado >=3 (script, onerror, javascript:)", len(recusas))
	}
	padroes := map[string]bool{}
	for _, r := range recusas {
		if r.Motivo == "" {
			t.Fatalf("recusa sem motivo: %+v", r)
		}
		padroes[r.Padrao] = true
	}
	for _, esperado := range []string{"<script", "on<evento>=", "javascript:"} {
		if !padroes[esperado] {
			t.Fatalf("padrao %q nao detetado; veio %v", esperado, padroes)
		}
	}
}

func TestDiagnosticarHTMLDeConteudoLimpoNaoAcusa(t *testing.T) {
	if r := DiagnosticarHTMLDeConteudo(`<div><p>ok</p></div>`); len(r) != 0 {
		t.Fatalf("HTML limpo acusou %d violacoes: %+v", len(r), r)
	}
}

// A lista exposta ao Mission Control tem de coincidir com as regras aplicadas.
//
//	*Uma lista de padroes que nao corresponde ao codigo e' documentacao a mentir.*
func TestPadroesProibidosCorrespondemAoValidador(t *testing.T) {
	padroes := PadroesProibidosEmHTML()
	if len(padroes) == 0 {
		t.Fatal("lista de padroes vazia")
	}
	// Cada padrao nomeado tem de ter pelo menos um HTML que o dispara.
	amostras := map[string]string{
		"<script":                  `<script>x</script>`,
		"on<evento>=":              `<img onerror=x>`,
		"javascript:":              `<a href="javascript:x">`,
		"<iframe":                  `<iframe></iframe>`,
		"<object":                  `<object></object>`,
		"<embed":                   `<embed>`,
		"srcdoc=":                  `<iframe srcdoc="x">`,
		"<meta http-equiv=refresh": `<meta http-equiv=refresh>`,
	}
	for _, p := range padroes {
		amostra, ok := amostras[p]
		if !ok {
			t.Fatalf("padrao %q nao tem amostra no teste", p)
		}
		if err := ValidarHTMLDeConteudo(amostra); err == nil {
			t.Fatalf("padrao %q listado mas o validador NAO recusa %q", p, amostra)
		}
	}
	if len(amostras) != len(padroes) {
		t.Fatalf("lista tem %d padroes, teste cobre %d", len(padroes), len(amostras))
	}
}

// O validador e' DETERMINISTICO e idempotente: validar duas vezes da' o mesmo veredicto.
func TestValidadoresSaoDeterministicos(t *testing.T) {
	htmlSujo := `<script>a</script>`
	for i := range 3 {
		if err := ValidarHTMLDeConteudo(htmlSujo); err == nil {
			t.Fatalf("execucao %d aceitou HTML sujo", i)
		}
	}
	urlLimpa := "https://exemplo.com/x"
	for i := range 3 {
		if err := ValidarURLDeConteudo(urlLimpa); err != nil {
			t.Fatalf("execucao %d recusou URL limpa: %v", i, err)
		}
	}
	// A mensagem de erro tem de ser estavel (allowlist em ordem fixa).
	primeira := ValidarURLDeConteudo("gopher://x").Error()
	segunda := ValidarURLDeConteudo("gopher://x").Error()
	if primeira != segunda {
		t.Fatalf("mensagem instavel:\n%s\n%s", primeira, segunda)
	}
	if !strings.Contains(primeira, "http") {
		t.Fatalf("mensagem nao diz a allowlist: %s", primeira)
	}
}
