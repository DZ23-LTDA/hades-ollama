package agent

// E-1243 / FASE 26.5: testes da seguranca dos COMPONENTES VISUAIS, ligada ao pipeline.
//
// A LACUNA QUE ESTES TESTES FECHAM
//
// O `validateVisualComponents` existia mas perguntava so' ESTRUTURA: tem id? tem type? a arvore e'
// funda? Nao perguntava nada sobre o que o renderer do canvas ia EXECUTAR.
//
// E dois dos tres caminhos que chamam `renderVisualHTML` nem essa validacao estrutural faziam.
//
//	*Um componente com id e type e' um componente bem formado. Nao e' um componente seguro.*
//
// Cada caso tem os dois lados: ataque recusado E conteudo legitimo aceite.

import (
	"errors"
	"testing"
)

// componenteDeTeste monta um componente minimo valido.
func componenteDeTeste(id string) VisualComponent {
	return VisualComponent{ID: id, Type: "caixa"}
}

func TestValidarSegurancaDosComponentesRecusaEventoForaDaAllowlist(t *testing.T) {
	casos := []struct{ nome, evento, acao string }{
		{"onload", "onload", "navegar"},
		{"error", "error", "navegar"},
		{"custom", "meuEventoQualquer", "navegar"},
		{"maiusculas", "Click", "navegar"},
		{"vazio", "", "navegar"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			comp := componenteDeTeste("x")
			comp.Events = map[string]string{c.evento: c.acao}
			err := ValidarSegurancaDosComponentes([]VisualComponent{comp}, 0)
			if err == nil {
				t.Fatalf("evento %q devia ser recusado", c.evento)
			}
			if !errors.Is(err, ErrConteudoInseguro) {
				t.Fatalf("erro nao e' ErrConteudoInseguro: %v", err)
			}
		})
	}
}

func TestValidarSegurancaDosComponentesRecusaAcaoForaDaAllowlist(t *testing.T) {
	casos := []string{
		"executarScript",
		"fetchRemoto",
		"abrirUrlExterna",
		"eval",
		"javascript:alert(1)",
	}
	for _, acao := range casos {
		t.Run(acao, func(t *testing.T) {
			comp := componenteDeTeste("x")
			comp.Events = map[string]string{"click": acao}
			if err := ValidarSegurancaDosComponentes([]VisualComponent{comp}, 0); err == nil {
				t.Fatalf("accao %q devia ser recusada", acao)
			}
		})
	}
}

func TestValidarSegurancaDosComponentesRecusaBindingExecutavel(t *testing.T) {
	casos := []struct{ nome, valor string }{
		{"javascript", "javascript:alert(1)"},
		{"data html", "data:text/html,<script>x</script>"},
		{"script no valor", "items<script>alert(1)</script>"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			comp := componenteDeTeste("x")
			comp.Bindings = map[string]string{"items": c.valor}
			err := ValidarSegurancaDosComponentes([]VisualComponent{comp}, 0)
			if err == nil {
				t.Fatalf("binding %q devia ser recusado", c.valor)
			}
		})
	}
}

func TestValidarSegurancaDosComponentesRecusaPropsEStyleInseguros(t *testing.T) {
	// Prop com markup executavel.
	comp := componenteDeTeste("x")
	comp.Props = map[string]string{"texto": "<script>alert(1)</script>"}
	if err := ValidarSegurancaDosComponentes([]VisualComponent{comp}, 0); err == nil {
		t.Fatal("prop com <script> devia ser recusada")
	}

	// Style com expressao CSS/SVG.
	for _, valor := range []string{
		"width: expression(alert(1))",
		"background: url(javascript:alert(1))",
		"background: url(data:text/html,x)",
	} {
		c := componenteDeTeste("y")
		c.Style = map[string]string{"cor": valor}
		if err := ValidarSegurancaDosComponentes([]VisualComponent{c}, 0); err == nil {
			t.Fatalf("style %q devia ser recusado", valor)
		}
	}
}

func TestValidarSegurancaDosComponentesAceitaOLegitimo(t *testing.T) {
	comp := VisualComponent{
		ID:   "botao-entrar",
		Type: "botao",
		Props: map[string]string{
			"texto": "Entrar",
			"alt":   "Botão para entrar",
		},
		Style: map[string]string{
			"cor":           "#2563eb",
			"preenchimento": "8px 14px",
			"largura":       "100%",
		},
		Bindings: map[string]string{
			"items": "items",
			"nome":  "user.name",
			"total": "carrinho.total",
		},
		Events: map[string]string{
			"click": "enviarForm",
			"input": "filtrarLista",
		},
		Children: []VisualComponent{
			{ID: "filho", Type: "texto", Props: map[string]string{"texto": "ok"}},
		},
	}
	if err := ValidarSegurancaDosComponentes([]VisualComponent{comp}, 0); err != nil {
		t.Fatalf("componente legitimo recusado: %v", err)
	}
}

func TestValidarSegurancaDosComponentesRecusaArvoreProfunda(t *testing.T) {
	// Treze niveis excedem o limite de 12.
	raiz := componenteDeTeste("n0")
	atual := &raiz
	for i := 1; i <= 13; i++ {
		filho := componenteDeTeste("n" + string(rune('0'+i%10)))
		atual.Children = []VisualComponent{filho}
		atual = &atual.Children[0]
	}
	if err := ValidarSegurancaDosComponentes([]VisualComponent{raiz}, 0); err == nil {
		t.Fatal("arvore com 13+ niveis devia ser recusada")
	}
}

// O VALIDADOR ESTA' LIGADO AO PIPELINE
//
// Este teste e' o que fecha a lacuna da E-1241: la' o validador existia e nao era chamado por
// ninguem. Aqui prova-se que o caminho de CRIACAO do projection o aplica.
func TestCriacaoDeProjectoVisualRecusaComponenteInseguro(t *testing.T) {
	raiz := t.TempDir()
	servico, err := NewBuilderService(raiz)
	if err != nil {
		t.Fatalf("criar servico: %v", err)
	}
	malicioso := componenteDeTeste("mau")
	malicioso.Events = map[string]string{"click": "executarScript"}

	_, err = servico.Create(t.Context(), BuilderSpec{
		Name:       "projection-com-evento-proibido",
		Kind:       BuilderWebsite,
		Components: []VisualComponent{malicioso},
	})
	if err == nil {
		t.Fatal("criacao com evento proibido devia falhar: o validador nao esta' ligado ao pipeline")
	}
	if !errors.Is(err, ErrConteudoInseguro) {
		t.Fatalf("erro = %v; esperado ErrConteudoInseguro", err)
	}
}

// O caminho feliz continua a funcionar: ligar um validador nao pode quebrar o produto.
func TestCriacaoDeProjectoVisualAceitaComponenteSeguro(t *testing.T) {
	raiz := t.TempDir()
	servico, err := NewBuilderService(raiz)
	if err != nil {
		t.Fatalf("criar servico: %v", err)
	}
	bom := VisualComponent{
		ID:     "titulo",
		Type:   "texto",
		Props:  map[string]string{"texto": "Olá"},
		Style:  map[string]string{"cor": "#111"},
		Events: map[string]string{"click": "navegar"},
	}
	proj, err := servico.Create(t.Context(), BuilderSpec{
		Name:       "projection-seguro",
		Kind:       BuilderWebsite,
		Components: []VisualComponent{bom},
	})
	if err != nil {
		t.Fatalf("criacao legitima falhou: %v", err)
	}
	if proj.ID == "" {
		t.Fatal("projection criado sem id")
	}
}
