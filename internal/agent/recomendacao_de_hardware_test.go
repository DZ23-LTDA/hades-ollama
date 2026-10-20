package agent

import (
	"errors"
	"strings"
	"testing"
)

const gb = uint64(1024 * 1024 * 1024)

// NEGATIVO: sem medicao de VRAM nao ha' recomendacao. Inventar uma seria pior do que nao responder,
// porque quem a recebe ajusta quantizacao e contexto acreditando que houve medicao.
func TestRecomendarRecusaSemVRAMMedida(t *testing.T) {
	if _, err := RecomendarParaVRAM(0, 7); !errors.Is(err, ErrVRAMDesconhecida) {
		t.Fatalf("VRAM zero devia dar ErrVRAMDesconhecida, veio %v", err)
	}
	if _, err := RecomendarParaVRAM(8*gb, 0); !errors.Is(err, ErrVRAMDesconhecida) {
		t.Fatalf("modelo com 0 parametros devia dar ErrVRAMDesconhecida, veio %v", err)
	}
}

// POSITIVO: com VRAM medida, a recomendacao vem com motivo e numeros.
func TestRecomendarDevolveMotivoComNumeros(t *testing.T) {
	r, err := RecomendarParaVRAM(8*gb, 7)
	if err != nil {
		t.Fatalf("8 GB para 7 B devia recomendar: %v", err)
	}
	if !r.EContabilizado {
		t.Fatalf("8 GB para 7 B devia caber; aviso=%q", r.Aviso)
	}
	if r.Quantizacao == "" || r.Contexto < contextoPadrao {
		t.Fatalf("recomendacao incompleta: %+v", r)
	}
	if !strings.Contains(r.Motivo, "VRAM medida") || !strings.Contains(r.Motivo, "GB") {
		t.Fatalf("o motivo tem de citar a VRAM medida e uma unidade: %q", r.Motivo)
	}
	if r.VRAMUsadaBytes+r.VRAMLivreBytes != 8*gb {
		t.Fatalf("a contabilidade tem de fechar: %d + %d != 8 GB", r.VRAMUsadaBytes, r.VRAMLivreBytes)
	}
}

// MONOTONICIDADE: mais VRAM nunca pode dar quantizacao PIOR nem contexto MENOR.
//
//	*Uma recomendacao que piora quando o hardware melhora nao e' conservadora: e' errada.*
func TestMaisVRAMNuncaPioraARecomendacao(t *testing.T) {
	ordem := map[string]int{"Q3_K_M": 1, "Q4_K_M": 2, "Q5_K_M": 3, "Q8_0": 4}
	anterior := RecomendacaoDeExecucao{}
	temAnterior := false
	for _, vram := range []uint64{4 * gb, 6 * gb, 8 * gb, 12 * gb, 16 * gb, 24 * gb, 48 * gb} {
		r, err := RecomendarParaVRAM(vram, 7)
		if err != nil {
			t.Fatalf("%d GB: %v", vram/gb, err)
		}
		if !r.EContabilizado {
			t.Fatalf("%d GB devia caber", vram/gb)
		}
		if temAnterior {
			if ordem[r.Quantizacao] < ordem[anterior.Quantizacao] {
				t.Fatalf("%d GB deu %s, PIOR que %s com menos VRAM", vram/gb, r.Quantizacao, anterior.Quantizacao)
			}
			if r.Contexto < anterior.Contexto {
				t.Fatalf("%d GB deu contexto %d, MENOR que %d com menos VRAM", vram/gb, r.Contexto, anterior.Contexto)
			}
		}
		anterior, temAnterior = r, true
	}
}

// Quando nada cabe, a resposta diz isso com numeros e NAO se diz contabilizada.
func TestQuandoNaoCabeDizQueNaoCabe(t *testing.T) {
	r, err := RecomendarParaVRAM(1*gb, 70)
	if err != nil {
		t.Fatalf("1 GB para 70 B devia responder (nao cabe), nao falhar: %v", err)
	}
	if r.EContabilizado {
		t.Fatalf("1 GB para 70 B nao pode estar contabilizado: %+v", r)
	}
	if r.Aviso == "" || !strings.Contains(r.Motivo, "nao cabe") {
		t.Fatalf("tem de dizer que nao cabe: aviso=%q motivo=%q", r.Aviso, r.Motivo)
	}
	if r.CamadasNaGPU != 0 {
		t.Fatalf("sem VRAM suficiente nao ha' camadas na GPU, veio %d", r.CamadasNaGPU)
	}
}

// Determinismo: a mesma entrada da' a mesma saida (sem relogio, sem aleatorio, sem ambiente).
func TestRecomendacaoEDeterministica(t *testing.T) {
	a, _ := RecomendarParaVRAM(12*gb, 7)
	b, _ := RecomendarParaVRAM(12*gb, 7)
	if a != b {
		t.Fatalf("duas chamadas iguais deram resultados diferentes:\n %+v\n %+v", a, b)
	}
}
