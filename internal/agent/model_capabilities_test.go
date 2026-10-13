package agent

import (
	"errors"
	"strings"
	"testing"
)

// CAPACIDADES DE MODELO — TESTES (E-325; V2.2 §11.2 / adendo P0-D).
//
// O caso crítico: modelos bons para texto/código que **não** emitem tool calls nativos. O produto **não pode** prometer
// tool calling nativo para eles, e **não pode** rebaixar um pedido explícito em silêncio.

func TestModeloComToolCallingMedidoUsaModoNativo(t *testing.T) {
	registro := DefaultModelCapabilityRegistry()
	for _, modelo := range []string{"qwen2.5:7b", "llama3.1:8b", "llama3.2:3b"} {
		modo, aviso := registro.ResolveToolMode(modelo)
		if modo != ToolModeNative {
			t.Fatalf("%s tem tool calling medido e precisa operar em modo NATIVO, veio %q", modelo, modo)
		}
		if aviso != "" {
			t.Fatalf("modelo nativo não precisa de aviso de limitação: %q", aviso)
		}
		if err := registro.RequireNativeToolCalling(modelo); err != nil {
			t.Fatalf("pedido nativo para %s precisa ser aceito: %v", modelo, err)
		}
	}
}

func TestModeloSemToolCallNativoNaoFingeEFalaEmPortugues(t *testing.T) {
	registro := DefaultModelCapabilityRegistry()
	for _, modelo := range []string{"qwen2.5-coder:7b", "deepseek-r1:8b"} {
		modo, aviso := registro.ResolveToolMode(modelo)
		if modo == ToolModeNative {
			t.Fatalf("ARMADILHA: %s NÃO emite tool calls nativos — não pode ser apresentado como nativo", modelo)
		}
		if modo != ToolModeCompatibleText {
			t.Fatalf("%s precisa cair em modo compatível, veio %q", modelo, modo)
		}
		if !strings.Contains(aviso, "não emite tool calls nativos") {
			t.Fatalf("o aviso precisa EXPLICAR a limitação (V2.2 §11.2), veio %q", aviso)
		}
		// PEDIDO EXPLÍCITO de nativo: RECUSADO, não rebaixado em silêncio.
		if err := registro.RequireNativeToolCalling(modelo); !errors.Is(err, ErrModelWithoutNativeToolCalling) {
			t.Fatalf("pedido de tool call nativo para %s precisa ser RECUSADO: %v", modelo, err)
		}
	}
}

func TestModeloDesconhecidoEhNaoMedidoENaoInventaCapacidade(t *testing.T) {
	registro := DefaultModelCapabilityRegistry()
	// 1. Lookup de modelo desconhecido devolve "não sei" (e não "não suporta" nem "suporta").
	if _, achou := registro.Lookup("modelo-novo:latest"); achou {
		t.Fatal("modelo desconhecido não pode ter capacidade inventada no registro")
	}
	// 2. Modo: compatível com aviso EXPLÍCITO de que não foi medido (falhar fechado).
	modo, aviso := registro.ResolveToolMode("modelo-novo:latest")
	if modo != ToolModeUnknown {
		t.Fatalf("modelo não medido precisa ser declarado como UNKNOWN, veio %q", modo)
	}
	if !strings.Contains(aviso, "NÃO MEDIDA") {
		t.Fatalf("o aviso precisa dizer que a capacidade não foi medida, veio %q", aviso)
	}
	// 3. Pedido de nativo: recusado por FALTA DE MEDIÇÃO (erro distinto, dono da causa diferente).
	if err := registro.RequireNativeToolCalling("modelo-novo:latest"); !errors.Is(err, ErrModelCapabilityUnknown) {
		t.Fatalf("modelo não medido precisa recusar nativo como NÃO MEDIDO, não como 'não suporta': %v", err)
	}
	// 4. Sem nome nenhum também não vira capacidade.
	if _, achou := registro.Lookup(""); achou {
		t.Fatal("modelo vazio não pode ter capacidade")
	}
}

func TestCapacidadeSemOrigemNaoEntraNoRegistro(t *testing.T) {
	registro := DefaultModelCapabilityRegistry()
	err := registro.WithCapabilities(ModelCapabilities{Model: "modelo-x:1b", ToolCalling: true})
	if err == nil {
		t.Fatal("capacidade SEM ORIGEM não pode ser registrada: é exatamente o que o V2.2 proíbe")
	}
	if !strings.Contains(err.Error(), "sem origem") {
		t.Fatalf("o erro precisa dizer que falta procedência: %v", err)
	}
	// Com origem declarada, entra — e o modo passa a nativo porque a capacidade FOI declarada.
	if err := registro.WithCapabilities(ModelCapabilities{
		Model: "modelo-x:1b", ToolCalling: true,
		Source: CapabilitySourceDeclared,
	}); err != nil {
		t.Fatalf("capacidade com origem declarada precisa ser aceita: %v", err)
	}
	modo, _ := registro.ResolveToolMode("modelo-x:1b")
	if modo != ToolModeNative {
		t.Fatalf("após registrar capacidade com origem, o modo precisa refletir a declaração: %q", modo)
	}
	// E uma capacidade declarada como SEM ferramentas rebaixa corretamente (o inverso também é medido).
	if err := registro.WithCapabilities(ModelCapabilities{
		Model: "modelo-y:1b", ToolCalling: false,
		Source: CapabilitySourceDeclared, Note: "declarado sem ferramentas",
	}); err != nil {
		t.Fatal(err)
	}
	if modo, _ := registro.ResolveToolMode("modelo-y:1b"); modo != ToolModeCompatibleText {
		t.Fatalf("capacidade declarada sem ferramentas precisa cair em compatível: %q", modo)
	}
}

func TestResumoDeCapacidadesEhEstavelEPreservaOrigem(t *testing.T) {
	registro := DefaultModelCapabilityRegistry()
	primeiro := registro.CapabilitySummary()
	segundo := registro.CapabilitySummary()
	if len(primeiro) != len(segundo) {
		t.Fatalf("o resumo precisa ser estável: %d != %d", len(primeiro), len(segundo))
	}
	for i := range primeiro {
		if primeiro[i].Model != segundo[i].Model {
			t.Fatalf("a ordem do resumo mudou entre chamadas: %q != %q", primeiro[i].Model, segundo[i].Model)
		}
		if strings.TrimSpace(primeiro[i].Source) == "" {
			t.Fatalf("toda capacidade exibida precisa declarar a origem: %+v", primeiro[i])
		}
		if primeiro[i].ToolCalling == false && strings.TrimSpace(primeiro[i].Note) == "" {
			t.Fatalf("capacidade ausente precisa vir com explicação para a interface: %+v", primeiro[i])
		}
	}
}
