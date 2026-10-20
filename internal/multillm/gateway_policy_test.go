package multillm

import "testing"

// O DEFEITO: o gateway construia `Policy{Path: ...}` e nunca definia `LocalOnly`. A restricao
// existia na estrutura e NAO era alcancavel a partir do pedido -- uma politica que ninguem
// consegue ligar nao e' uma protecao, e' um campo.
func TestPoliticaDoPedidoPropagaLocalOnly(t *testing.T) {
	semRestricao := &Gateway{}
	if p := semRestricao.politicaDoPedido("/v1/chat/completions"); p.LocalOnly {
		t.Fatal("gateway sem LocalOnly nao pode produzir politica local-only")
	}

	comRestricao := &Gateway{LocalOnly: true}
	p := comRestricao.politicaDoPedido("/v1/chat/completions")
	if !p.LocalOnly {
		t.Fatal("gateway com LocalOnly=true tem de propagar a restricao para a politica")
	}
	if p.Path != "/v1/chat/completions" {
		t.Fatalf("o caminho perdeu-se: %q", p.Path)
	}
}

// E a politica produzida, aplicada ao registry, recusa remoto e aceita local: a ligacao serve
// para alguma coisa.
func TestPoliticaDoPedidoAplicadaAoRegistry(t *testing.T) {
	r := montarRegistryLocal(true, "local")
	g := &Gateway{LocalOnly: true}
	pol := g.politicaDoPedido("/v1/chat/completions")

	if _, ok := r.Resolve("remoto/modelo", pol); ok {
		t.Fatal("com LocalOnly o registry nao pode resolver o remoto, mesmo com AllowPrivate")
	}
	if _, ok := r.Resolve("local/modelo", pol); !ok {
		t.Fatal("com LocalOnly o registry tem de resolver o local")
	}
}

// nil-receiver nao entra em panico: o middleware trata o gateway como opcional nalguns testes.
func TestPoliticaDoPedidoAguentaGatewayNulo(t *testing.T) {
	var g *Gateway
	if p := g.politicaDoPedido("/v1/x"); p.LocalOnly {
		t.Fatal("gateway nulo nao pode produzir politica local-only")
	}
}
