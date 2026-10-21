package agent

import (
	"strings"
	"testing"
)

// Com o numero REAL de camadas, o parcial deixa de ser uma percentagem disfarcada: 12 de 32, nao 12 de 100.
func TestParcialUsaOCamposReaisDeCamadas(t *testing.T) {
	r, err := RecomendarParaVRAMComCamadas(4*gb, 70, 32)
	if err != nil {
		t.Fatalf("4 GB / 70 B / 32 camadas: %v", err)
	}
	if r.CamadasTotais != 32 {
		t.Fatalf("CamadasTotais devia ser 32, veio %d", r.CamadasTotais)
	}
	if !r.Parcial {
		t.Fatalf("4 GB para 70 B devia ser parcial: %+v", r)
	}
	if r.CamadasNaGPU < 1 || r.CamadasNaGPU > 31 {
		t.Fatalf("parcial tem de ficar 1..31 (uma camada tem de correr em CPU), veio %d", r.CamadasNaGPU)
	}
	if !strings.Contains(r.Aviso, "de 32 camadas") {
		t.Fatalf("o aviso tem de dizer 'de 32 camadas': %q", r.Aviso)
	}
}

// Quando cabe inteiro com contagem real, devolve EXACTAMENTE o total -- nem 100, nem 99.
func TestCabeInteiroDevolveOTotalReal(t *testing.T) {
	r, err := RecomendarParaVRAMComCamadas(24*gb, 7, 32)
	if err != nil {
		t.Fatalf("24 GB / 7 B / 32 camadas: %v", err)
	}
	if !r.EContabilizado {
		t.Fatalf("24 GB para 7 B devia caber inteiro: %+v", r)
	}
	if r.CamadasNaGPU != 32 || r.CamadasTotais != 32 {
		t.Fatalf("devia ser 32 de 32, veio %d de %d", r.CamadasNaGPU, r.CamadasTotais)
	}
}

// Contagem de camadas invalida e' fail-closed, como a VRAM.
func TestCamadasInvalidasSaoRecusadas(t *testing.T) {
	if _, err := RecomendarParaVRAMComCamadas(8*gb, 7, 0); err == nil {
		t.Fatal("0 camadas devia ser recusado")
	}
	if _, err := RecomendarParaVRAMComCamadas(8*gb, 7, -3); err == nil {
		t.Fatal("camadas negativas deviam ser recusadas")
	}
}
