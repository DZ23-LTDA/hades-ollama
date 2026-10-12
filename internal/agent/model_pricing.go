package agent

// Regras de preço do catálogo, isoladas aqui para serem testáveis sem importar o pacote
// de provedores no núcleo agentico.
//
// A distinção que importa: "custo zero" e "preço não declarado" NÃO são a mesma coisa.
// Tratar catálogo sem preço como zero faria o orçamento mentir para baixo — exatamente o
// erro que o master proíbe.

// LocalModelPrice declara o preço de um modelo LOCAL: gratuito POR CONSTRUÇÃO.
// É zero conhecido, não desconhecido.
func LocalModelPrice() ModelPrice {
	return ModelPrice{Known: true}
}

// ModelPriceFromCatalog declara o preço a partir dos campos do catálogo de provedores.
// Só é conhecido quando o catálogo traz alguma informação de custo (tarifa ou etiqueta).
// Sem tarifa e sem etiqueta, devolve Known=false — ausência de preço, nunca zero.
func ModelPriceFromCatalog(inputCentsPer1K, outputCentsPer1K int64, costTag string) ModelPrice {
	if inputCentsPer1K <= 0 && outputCentsPer1K <= 0 && costTag == "" {
		return ModelPrice{}
	}
	return ModelPrice{
		InputCentsPer1K:  inputCentsPer1K,
		OutputCentsPer1K: outputCentsPer1K,
		Known:            true,
	}
}
