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

// Quando nao cabe INTEIRO, a resposta da' o parcial: quantas camadas cabem na GPU.
//
// ESTE TESTE MUDOU DE CONTRATO, e a mudanca e' o ponto: antes exigia `CamadasNaGPU == 0` porque a
// funcao so' sabia dizer "cabe" ou "nao cabe". Agora calcula o parcial -- *"nao cabe" e "nao cabe,
// mas cabem N de 100 camadas" sao respostas diferentes, e a segunda permite decidir.*
func TestQuandoNaoCabeInteiroDaOParcial(t *testing.T) {
	r, err := RecomendarParaVRAM(1*gb, 70)
	if err != nil {
		t.Fatalf("1 GB para 70 B devia responder, nao falhar: %v", err)
	}
	if r.EContabilizado {
		t.Fatalf("1 GB para 70 B nao pode estar contabilizado: %+v", r)
	}
	if !r.Parcial {
		t.Fatalf("devia ser um resultado PARCIAL: %+v", r)
	}
	if r.CamadasNaGPU < 1 || r.CamadasNaGPU > 99 {
		t.Fatalf("o parcial tem de ficar entre 1 e 99 camadas, veio %d", r.CamadasNaGPU)
	}
	if !strings.Contains(r.Aviso, "PARCIAL") || !strings.Contains(r.Aviso, "CPU") {
		t.Fatalf("o aviso tem de dizer parcial e que o resto corre em CPU: %q", r.Aviso)
	}
	if !strings.Contains(r.Motivo, "nao cabe inteiro") {
		t.Fatalf("o motivo tem de dizer que nao cabe inteiro: %q", r.Motivo)
	}
	if r.VRAMUsadaBytes+r.VRAMLivreBytes != 1*gb {
		t.Fatalf("a contabilidade tem de fechar: %d + %d != 1 GB", r.VRAMUsadaBytes, r.VRAMLivreBytes)
	}
}

// Quando nem o peso cabe (VRAM menor do que a folga minima de contexto), a resposta e' explicita e
// NAO ha' camadas na GPU -- porque uma camada na GPU sem VRAM para ela seria uma promessa.
func TestQuandoNemOPesoCabeNaoHaCamadas(t *testing.T) {
	r, err := RecomendarParaVRAM(256*1024*1024, 70) // 256 MB
	if err != nil {
		t.Fatalf("256 MB devia responder, nao falhar: %v", err)
	}
	if r.Parcial || r.EContabilizado {
		t.Fatalf("nao pode ser parcial nem contabilizado: %+v", r)
	}
	if r.CamadasNaGPU != 0 {
		t.Fatalf("sem VRAM para o peso nao ha' camadas na GPU, veio %d", r.CamadasNaGPU)
	}
	if !strings.Contains(r.Aviso, "nao cabe") {
		t.Fatalf("tem de dizer que nao cabe: %q", r.Aviso)
	}
}

// MONOTONICIDADE no parcial: mais VRAM nunca pode dar MENOS camadas na GPU.
func TestMaisVRAMNuncaDaMenosCamadasNoParcial(t *testing.T) {
	anterior := -1
	for _, vram := range []uint64{1 * gb, 2 * gb, 4 * gb, 6 * gb, 8 * gb, 10 * gb, 14 * gb, 20 * gb} {
		r, err := RecomendarParaVRAM(vram, 70) // 70 B: cabe parcial em toda esta gama
		if err != nil {
			t.Fatalf("%d GB: %v", vram/gb, err)
		}
		if anterior >= 0 && r.CamadasNaGPU < anterior {
			t.Fatalf("%d GB deu %d camadas, MENOS que %d com menos VRAM", vram/gb, r.CamadasNaGPU, anterior)
		}
		anterior = r.CamadasNaGPU
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
