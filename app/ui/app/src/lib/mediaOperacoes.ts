/**
 * Catalogo das operacoes de midia (Estagio 45 / item 45 — Multimodal Studio).
 *
 * *Este ficheiro NAO chama nada.* Ele fixa, em codigo, o que o servidor REALMENTE aceita, medido em
 * `internal/agent/media_tool.go:40-84`:
 *
 *   tone.generate · audio.transcribe · image.analyze · image.generate · video.generate · speech.generate
 *   operacao fora desta lista ⇒ `media operation %q is not allowlisted`
 *   operacao que precise de provider, sem provider ⇒ `media provider is not configured`
 *   tudo corre dentro de um PASSO de missao (`validateStepID`) com workspace fixado
 *
 * *A meta e' explicita:* «Nao criar video ficticio como arquivo JSON nem tratar TTS sintetico como
 * provider real.» Por isso o catalogo distingue o que precisa de provider e **declara** a ausencia -
 * a UI nunca oferece um botao que finge gerar.
 *
 * A CHAMADA (qual rota conduz o passo) ainda **nao foi medida** e por isso **nao esta aqui**:
 * escrever um endpoint adivinhado seria inventar superficie.
 */

export type OperacaoDeMidia =
  | "tone.generate"
  | "audio.transcribe"
  | "image.analyze"
  | "image.generate"
  | "video.generate"
  | "speech.generate";

export type Modalidade = "audio" | "imagem" | "video";

export type DescritorDeOperacao = {
  operacao: OperacaoDeMidia;
  modalidade: Modalidade;
  rotulo: string;
  /** Campos que o servidor le^ deste `input` (medido no codigo, nao suposto). */
  campos: string[];
  /** `true` quando sem provider o servidor devolve «media provider is not configured». */
  exigeProvider: boolean;
  /** Limite de bytes do ficheiro de entrada, quando existe (medido: 100 MB / 25 MB). */
  limiteBytes?: number;
};

/** A allowlist do servidor, espelhada 1:1. *Se o servidor mudar, este teste tem de falhar.* */
export const OPERACOES_DE_MIDIA: readonly DescritorDeOperacao[] = Object.freeze(
  [
    {
      operacao: "tone.generate",
      modalidade: "audio",
      rotulo: "Tom de teste (local)",
      campos: ["frequency_hz", "duration_ms"],
      // *Unica operacao que NAO passa pelo provider: serve para provar o caminho sem configuracao.*
      exigeProvider: false,
    },
    {
      operacao: "audio.transcribe",
      modalidade: "audio",
      rotulo: "Transcrever audio (STT)",
      campos: ["file", "model"],
      exigeProvider: true,
      limiteBytes: 100 << 20,
    },
    {
      operacao: "image.analyze",
      modalidade: "imagem",
      rotulo: "Analisar imagem (visao)",
      campos: ["file", "prompt", "model"],
      exigeProvider: true,
      limiteBytes: 25 << 20,
    },
    {
      operacao: "image.generate",
      modalidade: "imagem",
      rotulo: "Gerar imagem",
      campos: ["prompt", "model"],
      exigeProvider: true,
    },
    {
      operacao: "video.generate",
      modalidade: "video",
      rotulo: "Gerar video",
      campos: ["prompt", "model"],
      exigeProvider: true,
    },
    {
      operacao: "speech.generate",
      modalidade: "audio",
      rotulo: "Gerar fala (TTS)",
      campos: ["text", "voice", "model"],
      exigeProvider: true,
    },
  ],
);

/** A operacao existe na allowlist do servidor? *Nada fora da allowlist chega a ser oferecido.* */
export function operacaoPermitida(
  operacao: string,
): operacao is OperacaoDeMidia {
  return OPERACOES_DE_MIDIA.some((d) => d.operacao === operacao);
}

/**
 * Disponibilidade honesta das operacoes.
 *
 * `true` ⇒ todas; `false` ⇒ so^ o que corre localmente; ***`undefined` ⇒ NAO SE SABE***.
 * No caso «nao sei» **nao se bloqueia nada**: *a autoridade e' o servidor, que responde com a
 * razao real («media provider is not configured») quando a missao e' lancada.*
 *
 * Motivo medido: o catalogo de conectores devolve sempre `status: "available"` (= «disponivel
 * para configurar») e o proprio ficheiro avisa que «nenhuma entrada afirma conexao pronta» -
 * logo a UI nao pode derivar «configurado» de la', nem fingir que sabe.
 */
export function operacoesDisponiveis(
  providerConfigurado: boolean | undefined,
): DescritorDeOperacao[] {
  if (providerConfigurado === undefined) {
    return OPERACOES_DE_MIDIA.slice();
  }
  return OPERACOES_DE_MIDIA.filter(
    (d) => providerConfigurado || !d.exigeProvider,
  );
}

/** Motivo exacto a mostrar quando a operacao nao pode correr - *a frase e' a do servidor.* */
export function motivoDeIndisponibilidade(
  operacao: string,
  providerConfigurado: boolean,
): string | null {
  if (!operacaoPermitida(operacao)) {
    return `A operacao "${operacao}" nao esta na allowlist do servidor.`;
  }
  const descritor = OPERACOES_DE_MIDIA.find((d) => d.operacao === operacao);
  if (descritor?.exigeProvider && !providerConfigurado) {
    // Texto literal do servidor: quem le^ o ecra^ ve^ a mesma razao que o log.
    return "media provider is not configured";
  }
  return null;
}

/** Mensagem de estado do estudio^ - nunca vazia, nunca prometendo o que nao existe. */
export function resumoDoEstudio(providerConfigurado: boolean): string {
  const disponiveis = operacoesDisponiveis(providerConfigurado);
  if (!providerConfigurado) {
    return (
      `Sem provider de midia configurado: ${disponiveis.length} de ${OPERACOES_DE_MIDIA.length} ` +
      "operacao(oes) disponivel(is) (tom local). Imagem, video, TTS e STT exigem provider " +
      "(Replicate/ElevenLabs estao em operator_setup_required)."
    );
  }
  return `${disponiveis.length} operacao(oes) de midia disponivel(is) com o provider configurado.`;
}
