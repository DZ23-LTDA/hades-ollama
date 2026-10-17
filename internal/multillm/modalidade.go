// Package multillm - E-431 / ADR-004: a MATRIZ DE MODALIDADE x ENGINE.
//
// ***POR QUE ISTO EXISTE, E POR QUE E' AQUI (e nao na UI).***
//
// O Estagio 45 (Multimodal Production Studio) precisa de saber *quem gera imagem, video, audio e documento*; o Estagio 43
// (Adaptive Router) precisa de saber *a quem pedir*. O ADR-004 decide que isso e' **uma matriz unica**, com
// **classificacao por capacidade**, alimentando o `Registry` - e **NAO** uma lista paralela na interface.
//
// A justificativa e' a mesma que se repete nesta missao: *duas fontes de verdade sobre o que o sistema sabe fazer
// divergem, e a que o utilizador ve' e' sempre a que mente mais devagar.*
//
// ***O QUE A MATRIZ **NAO** FAZ, E ISTO E' DELIBERADO.***
//
// Ela nao escolhe modelo, nao pontua, nao tem estado. Quem decide continua a ser o `Registry` (`RouteCandidates`), e a
// comparacao continua a ser a que ja' existe: `supports(m Model, required []string)` - **match EXACTO de strings** em
// `Model.Capabilities` (registry.go:395). A matriz apenas **liga uma modalidade** (o que o utilizador pede) **a capacidade
// que o Registry compara** (o que o modelo declara). *E' uma tabela de traducao, e uma tabela de traducao testa-se.*
package multillm

import "strings"

// Modalidade e' o tipo de midia que o sistema produz ou consome. E' o vocabulario do UTILIZADOR e dos estagios 45/47 -
// nao e' o vocabulario do modelo. A traducao entre os dois e' exactamente o que esta matriz faz.
type Modalidade string

const (
	ModalidadeTexto     Modalidade = "text"
	ModalidadeImagem    Modalidade = "image"
	ModalidadeVideo     Modalidade = "video"
	ModalidadeAudio     Modalidade = "audio"
	ModalidadeDocumento Modalidade = "document"
)

// EntradaDeModalidade liga uma modalidade a' capacidade que o `Registry` compara e aos motores que a produzem.
//
// `Capacidade` tem de ser uma string que apareca em `Model.Capabilities` - *senao a matriz promete um motor que o
// Registry nunca encontra, e a promessa falha em silencio, que e' o pior modo de falhar.*
type EntradaDeModalidade struct {
	Modalidade Modalidade
	// Capacidade e' o que o `supports()` compara (registry.go:395). **Match exacto.**
	Capacidade string
	// Motores sao os adaptadores/engines que produzem esta modalidade. Nomes declarativos: a matriz diz QUEM PODE,
	// nao QUEM ESTA DISPONIVEL - *disponibilidade e' estado, e estado nao se escreve numa tabela constante.*
	Motores []string
	// Observacao e' o limite honesto desta linha. *Uma matriz sem limites e' publicidade.*
	Observacao string
}

// matrizDeModalidades e' a **unica** fonte de verdade sobre "quem gera o que".
//
// ***LEIA A COLUNA `Observacao` ANTES DE CONFIAR NUMA LINHA.***
func matrizDeModalidades() []EntradaDeModalidade {
	return []EntradaDeModalidade{
		{
			Modalidade: ModalidadeTexto,
			Capacidade: "text",
			Motores:    []string{"ollama", "ollama-cloud", "anthropic", "openai", "gemini", "deepseek"},
			Observacao: "e' o caminho universal: *todo* modelo fala texto. **Nao medi** um modelo que recuse texto.",
		},
		{
			Modalidade: ModalidadeImagem,
			Capacidade: "vision",
			Motores:    []string{"ollama", "ollama-cloud", "anthropic", "openai", "gemini"},
			Observacao: "**ENTRADA** (ler imagem). A **SAIDA** (gerar imagem) tem capacidade propria - ver `image`. " +
				"*Confundir as duas e' o defeito que o E-430 existe para impedir.*",
		},
		{
			Modalidade: ModalidadeImagem,
			Capacidade: "image",
			Motores:    []string{"stable-diffusion", "flux", "dall-e", "imagen"},
			Observacao: "**SAIDA** (gerar imagem). *Nenhum destes motores esta' medido nesta maquina* - a lista e' " +
				"**declarativa**, e a homologacao fica `BLOCKED_BY_EXTERNAL_DEPENDENCY` ate' haver provider autorizado.",
		},
		{
			Modalidade: ModalidadeVideo,
			Capacidade: "video",
			Motores:    []string{"ffmpeg", "runway", "veo"},
			Observacao: "**BLOQUEADO**: *FFmpeg **nao** esta' instalado nesta maquina* (medido), e nao ha' provider de " +
				"video autorizado. ***O Estagio 45 exige: 'nao simular geracao de video; exibir bloqueio verificavel'.***",
		},
		{
			Modalidade: ModalidadeAudio,
			Capacidade: "audio",
			Motores:    []string{"whisper", "piper", "elevenlabs", "openai-tts"},
			Observacao: "STT/TTS. *Nenhum medido* - depende de credencial ou de modelo local instalado. " +
				"`BLOCKED_BY_EXTERNAL_DEPENDENCY` para a homologacao.",
		},
		{
			Modalidade: ModalidadeDocumento,
			Capacidade: "document",
			Motores:    []string{"python-docx", "openpyxl", "python-pptx", "pandoc"},
			Observacao: "PDF/DOCX/XLSX/PPTX com **fontes**. *E' a modalidade com maior probabilidade de estar " +
				"disponivel localmente* - e a matriz **nao** afirma que esta'.",
		},
	}
}

// MatrizDeModalidades devolve uma COPIA da matriz. *Devolver a fatia interna deixaria qualquer chamador reescrever a
// fonte de verdade - e uma fonte de verdade que o consumidor edita nao e' fonte de verdade.*
func MatrizDeModalidades() []EntradaDeModalidade {
	origem := matrizDeModalidades()
	copia := make([]EntradaDeModalidade, len(origem))
	for i, e := range origem {
		copia[i] = EntradaDeModalidade{
			Modalidade: e.Modalidade,
			Capacidade: e.Capacidade,
			Motores:    append([]string(nil), e.Motores...),
			Observacao: e.Observacao,
		}
	}
	return copia
}

// CapacidadesParaModalidade devolve **todas** as capacidades que satisfazem uma modalidade, em ordem determinista.
//
// *Uma modalidade pode ter mais de uma capacidade* - e' o caso de `image`, que e' ao mesmo tempo **entrada** (`vision`)
// e **saida** (`image`). Devolver so' a primeira seria escolher por conta propria qual dos dois sentidos o chamador
// queria, e essa escolha nao nos pertence.
func CapacidadesParaModalidade(m Modalidade) ([]string, bool) {
	alvo := Modalidade(strings.ToLower(strings.TrimSpace(string(m))))
	vistas := make(map[string]bool)
	var fora []string
	for _, e := range matrizDeModalidades() {
		if e.Modalidade != alvo || vistas[e.Capacidade] {
			continue
		}
		vistas[e.Capacidade] = true
		fora = append(fora, e.Capacidade)
	}
	if len(fora) == 0 {
		return nil, false
	}
	return fora, true
}

// MotoresParaModalidade devolve os motores declarados para uma capacidade concreta.
func MotoresParaModalidade(m Modalidade, capacidade string) ([]string, bool) {
	alvo := Modalidade(strings.ToLower(strings.TrimSpace(string(m))))
	capAlvo := strings.ToLower(strings.TrimSpace(capacidade))
	for _, e := range matrizDeModalidades() {
		if e.Modalidade == alvo && strings.EqualFold(e.Capacidade, capAlvo) {
			return append([]string(nil), e.Motores...), true
		}
	}
	return nil, false
}

// ModalidadeCompativel diz se uma modalidade pedida pode ser satisfeita por um modelo **concreto**, usando a MESMA
// comparacao que o `Registry` usa (`supports`).
//
// ***E' AQUI QUE O E-430 VIVE.*** Se o modelo nao declara nenhuma das capacidades da modalidade, a resposta e' `false` -
// e quem chamar **tem** de transformar isso em falha verificavel. *Um pedido de video atendido por um modelo textual, com
// sucesso devolvido, e' a mesma traicao de uma aprovacao que nunca foi dada - so' muda o objecto.*
//
// ***E AQUI ESTA' UM ERRO QUE EU COMETI AO ESCREVER ISTO, E QUE FICA REGISTADO.***
//
// A primeira versao chamava `supports(modelo, capacidades)` - e o `supports` (registry.go:395) e' **AND**: exige
// *todas* as capacidades pedidas. Mas uma modalidade satisfaz-se com **QUALQUER UMA** das suas capacidades: `image` e'
// ao mesmo tempo **entrada** (`vision`) e **saida** (`image`), e um modelo de visao **nao** gera imagem. Com o `AND`,
// `ModalidadeCompativel(image, …)` so' seria verdadeiro para um modelo que fizesse **as duas coisas** - ou seja, para
// **nenhum** - e a matriz prometeria um motor que o `Registry` nunca encontra.
//
// ⇒ *a matriz tem de perguntar `OU`, e a comparacao por capacidade continua a ser a do `Registry`.* **A semantica muda;
// a comparacao, nao.**
func ModalidadeCompativel(m Modalidade, modelo Model) bool {
	capacidades, ok := CapacidadesParaModalidade(m)
	if !ok {
		return false
	}
	return supportsAny(modelo, capacidades)
}

// supportsAny e' o `supports` com semantica **OU**: basta que o modelo declare **uma** das capacidades pedidas.
//
// *Existe separado - em vez de um parametro em `supports` - para que a diferenca entre os dois seja **visivel no nome**,
// e nao um `bool` que quem le tem de ir procurar. Um `true`/`false` a mais no meio de uma assinatura e' o sitio exacto
// onde a proxima pessoa escolhe mal.*
func supportsAny(m Model, alternativas []string) bool {
	for _, alt := range alternativas {
		for _, tem := range m.Capabilities {
			if tem == alt {
				return true
			}
		}
	}
	return false
}
