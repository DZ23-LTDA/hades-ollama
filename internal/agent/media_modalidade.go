package agent

import (
	"fmt"
	"strings"
)

// E-535 (Estagio 45, o `E-430` da matriz) - a recusa de modalidade na geracao de midia.
//
// ***POR QUE ISTO EXISTE, E POR QUE NAO E' PARANOIA.***
//
// O complemento e' literal: *«nao mandar pedido de video para modelo textual e afirmar sucesso»*. A versao anterior de
// `GenerateVideo` resolvia o modelo com `if model == "" { model = m.Provider.VideoModel }` e seguia para a rede. ⇒ *se
// o `VideoModel` nao estivesse configurado, a chamada saia com `{"model": ""}` e o provider decidia o que fazer - e o
// resultado passava a ser «video» por **AFIRMACAO**, nao por capacidade.*
//
// ***E O QUE ESTE FICHEIRO **NAO** FAZ - PORQUE NAO PODE.***
//
// Ele **nao** consulta um registry de capacidades, porque nao existe um para providers de midia neste codigo: o que o
// provider publica e' **`ImageModel`/`VideoModel`**, ou seja, **nomes**. ⇒ a recusa apoia-se no que existe: **o nome**.
// *Dizer isto e' obrigatorio - um instrumento que finge medir capacidades que nao ve e' pior do que um que se declara
// limitado ao nome.* (E' a mesma disciplina do `E-522` e do `media_pipeline.go:16`.)

// familiasTextuaisDeModelo sao fragmentos que declaram um modelo de **linguagem**, e nao um gerador de midia.
//
// *Nao e' uma lista de modelos: e' uma lista de **afirmacoes no nome**.* Um nome que contenha um destes fragmentos
// **afirma** ser de texto - e um gerador de imagem/video nao se chama `llama`, nem `gpt-4o`, nem `qwen`.
var familiasTextuaisDeModelo = []string{
	"llama", "qwen", "mistral", "mixtral", "gemma", "phi", "deepseek", "gpt-3", "gpt-4", "gpt-4o",
	"claude", "gemini", "command-r", "yi-", "falcon", "vicuna", "wizard", "solar", "granite",
	"smollm", "tinyllama", "codellama", "starcoder", "codestral", "nomic", "bge-", "e5-",
}

// familiasDeVideo sao fragmentos que afirmam ser gerador de **video**.
var familiasDeVideo = []string{"veo", "runway", "gen-3", "gen-4", "sora", "kling", "pika", "luma", "wan-", "hunyuanvideo"}

// familiasDeImagem sao fragmentos que afirmam ser gerador de **imagem**.
var familiasDeImagem = []string{"dall-e", "dalle", "stable-diffusion", "sdxl", "flux", "imagen", "midjourney", "firefly", "sd3"}

func contemAlgum(nome string, fragmentos []string) bool {
	for _, f := range fragmentos {
		if strings.Contains(nome, f) {
			return true
		}
	}
	return false
}

// recusarModalidadeDeMidia devolve erro quando a geracao de `modalidade` nao pode ser honestamente tentada com `modelo`.
//
// ***AS DUAS RECUSAS, NA ORDEM EM QUE FALHAM FECHADO.***
//
//  1. **Ausencia** (`modelo` vazio): `ErrMediaModelRequired`. *Nao ha caminho que chame o provider com um modelo vazio
//     «para ver no que da».*
//  2. **Incoerencia** (o nome afirma outra familia): `ErrMediaModalityMismatch`. *Um pedido de video a um modelo que se
//     chama `llama` nao e' um pedido arriscado: e' um pedido **impossivel** - e a resposta honesta e' a recusa.*
//
// *A ordem importa*: ausencia primeiro, porque um modelo vazio nao afirma nada, e `""` nao pode ser confundido com
// nenhuma familia.
func recusarModalidadeDeMidia(modalidade, modelo string) error {
	nome := strings.ToLower(strings.TrimSpace(modelo))
	if nome == "" {
		return fmt.Errorf("%w: modalidade %q sem modelo configurado - *o provider nao pode decidir no nosso lugar*",
			ErrMediaModelRequired, modalidade)
	}
	// (a) Um modelo de LINGUAGEM nunca gera imagem nem video, seja qual for a modalidade pedida.
	if contemAlgum(nome, familiasTextuaisDeModelo) {
		return fmt.Errorf("%w: modalidade %q com modelo %q - *um modelo de texto nao gera midia, e devolver sucesso "+
			"seria afirmar o que nao aconteceu*", ErrMediaModalityMismatch, modalidade, modelo)
	}
	// (b) E os dois sentidos cruzados: um gerador de VIDEO nao gera IMAGEM, e vice-versa.
	switch modalidade {
	case "image":
		if contemAlgum(nome, familiasDeVideo) {
			return fmt.Errorf("%w: modalidade %q com modelo %q - *um gerador de video nao gera imagem; a recusa "+
				"tambem vale neste sentido*", ErrMediaModalityMismatch, modalidade, modelo)
		}
	case "video":
		if contemAlgum(nome, familiasDeImagem) {
			return fmt.Errorf("%w: modalidade %q com modelo %q - *um gerador de imagem nao gera video*",
				ErrMediaModalityMismatch, modalidade, modelo)
		}
	}
	// ***O LIMITE, DECLARADO NO PROPRIO CAMINHO.*** Um nome fora destas familias **passa** - e passar nao e' uma
	// promessa de capacidade: e' a ausencia de um motivo para recusar. *A decisao de rota continua a ser do `Registry` (
	// `RouteCandidates`), e este ficheiro apenas impede o que o nome ja' desmente.*
	return nil
}
