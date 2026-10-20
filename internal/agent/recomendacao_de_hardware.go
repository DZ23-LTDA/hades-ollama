package agent

import (
	"errors"
	"fmt"
)

// ErrVRAMDesconhecida e' devolvido quando nao ha' medicao de VRAM para recomendar a partir dela.
//
// FAIL-CLOSED: uma recomendacao inventada a partir de VRAM desconhecida e' pior do que nenhuma --
// quem a recebe ajusta quantizacao e contexto acreditando que houve medicao.
var ErrVRAMDesconhecida = errors.New("VRAM desconhecida: nao ha' base para recomendar")

// RecomendacaoDeExecucao e' o que se pode dizer a quem tem uma VRAM medida.
//
// A REGRA QUE ISTO RESPEITA: o sistema pode RECOMENDAR; nao pode escolher por quem assina o merge nem
// aplicar sozinho. Por isso a estrutura devolve sempre `Motivo` (com os numeros que usou) e nunca
// executa nada.
type RecomendacaoDeExecucao struct {
	Quantizacao  string `json:"quantizacao"`
	Contexto     int    `json:"contexto"`
	CamadasNaGPU int    `json:"camadas_na_gpu"`
	// Parcial diz que o modelo NAO cabe todo na GPU e as restantes camadas correm em CPU.
	//
	// Antes desta fatia `CamadasNaGPU` so' devolvia 0 ou 100: ou cabia, ou nao se dizia nada.
	// Metade dos casos reais e' exactamente o meio -- cabe em parte, e e' preciso saber quanto.
	Parcial        bool   `json:"parcial"`
	VRAMUsadaBytes uint64 `json:"vram_usada_bytes"`
	VRAMLivreBytes uint64 `json:"vram_livre_bytes"`
	Aviso          string `json:"aviso"`
	Motivo         string `json:"motivo"`
	EContabilizado bool   `json:"e_contabilizado"`
}

// bytesPorParametro por quantizacao: numeros aproximados e DECLARADOS, para que o motivo seja
// auditavel em vez de magico.
var bytesPorParametro = []struct {
	nome  string
	bytes float64
	ordem int
}{
	{"Q8_0", 1.06, 4},
	{"Q5_K_M", 0.72, 3},
	{"Q4_K_M", 0.52, 2},
	{"Q3_K_M", 0.42, 1},
}

const (
	// contextoPadrao e' o minimo util; acima disso so' com folga medida.
	contextoPadrao = 2048
	contextoMedio  = 4096
	contextoAlto   = 8192
	// folgaDeContexto e' a VRAM que o KV-cache e o runtime consomem por 2048 tokens, em bytes.
	folgaDeContexto = 512 * 1024 * 1024
	// kBilhao e' 1e9 (parametros sao contados em milhares de milhoes).
	kBilhao = 1e9
)

// RecomendarParaVRAM devolve quantizacao, contexto e camadas na GPU para um modelo com
// `parametrosB` milhoes de milhares de parametros, dada uma VRAM medida em bytes.
//
// A ESCOLHA: a melhor quantizacao que cabe COM o contexto, e o maior contexto que cabe na folga.
// Preferir qualidade (quantizacao maior) e so' depois contexto -- e' a ordem em que um erro de
// contexto se nota (resposta truncada) e um erro de quantizacao se nota (qualidade).
func RecomendarParaVRAM(vramBytes uint64, parametrosB float64) (RecomendacaoDeExecucao, error) {
	if vramBytes == 0 {
		return RecomendacaoDeExecucao{}, ErrVRAMDesconhecida
	}
	if parametrosB <= 0 {
		return RecomendacaoDeExecucao{}, fmt.Errorf("%w: tamanho do modelo invalido (%v B)", ErrVRAMDesconhecida, parametrosB)
	}

	for _, q := range bytesPorParametro {
		pesoDoModelo := uint64(q.bytes * parametrosB * kBilhao)
		if pesoDoModelo == 0 {
			continue
		}
		// contexto: quanto maior, mais VRAM; escolher o maior que cabe com folga.
		variantesDeContexto := []int{contextoAlto, contextoMedio, contextoPadrao}
		for _, ctx := range variantesDeContexto {
			folga := uint64(folgaDeContexto) * uint64(ctx/contextoPadrao)
			necessario := pesoDoModelo + folga
			if necessario > vramBytes {
				continue
			}
			camadas := 100
			usada := necessario
			aviso := ""
			if pesoDoModelo > vramBytes {
				// nunca acontece aqui (necessario <= vram), mantido por clareza da fronteira
				camadas = 0
			}
			return RecomendacaoDeExecucao{
				Quantizacao:    q.nome,
				Contexto:       ctx,
				CamadasNaGPU:   camadas,
				VRAMUsadaBytes: usada,
				VRAMLivreBytes: vramBytes - usada,
				Aviso:          aviso,
				Motivo: fmt.Sprintf(
					"modelo de %.1f B em %s ocupa ~%s; com contexto %d a folga e' ~%s; total ~%s de VRAM medida (%s), sobram %s",
					parametrosB, q.nome, formatarBytes(pesoDoModelo), ctx,
					formatarBytes(folga), formatarBytes(necessario), formatarBytes(vramBytes),
					formatarBytes(vramBytes-usada)),
				EContabilizado: true,
			}, nil
		}
	}

	// NADA CABE INTEIRO: ainda assim ha' um numero util a dar -- quantas camadas cabem na GPU.
	//
	// *"Nao cabe" e "nao cabe, mas cabem 12 de 100 camadas" sao respostas diferentes, e a segunda
	// permite decidir.* Por isso o parcial e' calculado em vez de recusado.
	menor := bytesPorParametro[len(bytesPorParametro)-1]
	peso := uint64(menor.bytes * parametrosB * kBilhao)
	folgaMinima := uint64(folgaDeContexto)

	if vramBytes <= folgaMinima || peso == 0 {
		return RecomendacaoDeExecucao{
			Quantizacao:    menor.nome,
			Contexto:       contextoPadrao,
			CamadasNaGPU:   0,
			VRAMUsadaBytes: 0,
			VRAMLivreBytes: vramBytes,
			Aviso:          "nao cabe: nem o peso do modelo cabe -- rode em CPU ou use um modelo menor",
			Motivo: fmt.Sprintf(
				"nao cabe: com %s e contexto %d precisa ~%s so' para o peso, e ha' %s medidos (a folga minima de contexto e' ~%s)",
				menor.nome, contextoPadrao, formatarBytes(peso), formatarBytes(vramBytes), formatarBytes(folgaMinima)),
			EContabilizado: false,
			Parcial:        false,
		}, nil
	}

	disponivelParaPeso := vramBytes - folgaMinima
	camadas := int(float64(disponivelParaPeso) * 100 / float64(peso))
	if camadas > 99 {
		camadas = 99
	}
	if camadas < 1 {
		camadas = 1
	}
	usada := uint64(float64(peso) * float64(camadas) / 100)
	if usada > disponivelParaPeso {
		usada = disponivelParaPeso
	}
	return RecomendacaoDeExecucao{
		Quantizacao:    menor.nome,
		Contexto:       contextoPadrao,
		CamadasNaGPU:   camadas,
		VRAMUsadaBytes: usada,
		VRAMLivreBytes: vramBytes - usada,
		Aviso: fmt.Sprintf(
			"offload PARCIAL: %d de 100 camadas na GPU; as restantes correm em CPU (mais lento, e a qualidade nao muda)",
			camadas),
		Motivo: fmt.Sprintf(
			"nao cabe inteiro: %s com contexto %d precisa ~%s e ha' %s medidos; sobram %s para o peso, o que da' %d de 100 camadas na GPU",
			menor.nome, contextoPadrao, formatarBytes(peso+folgaMinima), formatarBytes(vramBytes),
			formatarBytes(disponivelParaPeso), camadas),
		EContabilizado: false,
		Parcial:        true,
	}, nil
}

func formatarBytes(b uint64) string {
	const gb = 1024 * 1024 * 1024
	if b >= gb {
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	}
	return fmt.Sprintf("%.0f MB", float64(b)/float64(1024*1024))
}
