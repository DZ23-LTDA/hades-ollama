/**
 * E-538 (Estágio 46) — o Companion: **estado real, texto, onboarding e acessibilidade**.
 *
 * ***A REGRA QUE GOVERNA ESTE COMPONENTE.*** O complemento é literal: *«Não utilizar animações ou respostas falsas para
 * simular trabalho»*. ⇒ **este componente não inventa estado nenhum: ele recebe o estado já derivado de eventos
 * verificados** (`DerivarEstadoCompanion`, E-536, no servidor) — e a única coisa que faz com ele é **dizê-lo**.
 *
 * ***E É POR ISSO QUE NÃO HÁ UM `useEffect` A ANIMAR NADA AQUI.*** Um mascote que anima *sozinho* é um mascote que
 * **decide** que trabalho está a acontecer — e essa decisão não lhe pertence. *A animação é derivada do `estado`, nunca
 * de um temporizador.*
 *
 * ***O QUE ESTE FICHEIRO NÃO TEM, E PORQUE.***
 *
 *  - **Voz (STT/TTS)**: `BLOCKED_BY_EXTERNAL_DEPENDENCY` — *não há provider de fala ligado neste ambiente, e simular
 *    «a falar» com uma onda desenhada seria exactamente a animação falsa que o documento proíbe.* O componente expõe
 *    `podeFalar` e **declara** que está indisponível, em vez de fingir.
 *  - **Respostas**: o Companion **não responde** perguntas. *Um mascote que inventa uma resposta é um modelo de
 *    linguagem escondido atrás de um desenho, e sem o motor por trás seria uma alucinação com olhos.*
 */

import {
  PREFERENCIAS_PADRAO,
  type CompanionPreferencias,
  podeDesenharCompanion,
  razaoParaNaoAnimar,
} from "../lib/companionPreferencias";

export type CompanionEstado =
  | "ausente"
  | "a_planear"
  | "a_trabalhar"
  | "a_aprovar"
  | "bloqueado"
  | "concluido"
  | "falhou"
  | "cancelado";

export interface CompanionProps {
  /** O estado **já derivado** no servidor. *Este componente não o calcula.* */
  estado: CompanionEstado;
  /** A razão que sustenta o estado (o evento que o produziu). */
  razao: string;
  /** Nome do mascote, para o `aria-label`. */
  nome?: string;
  /** Se o provider de fala está ligado. `false` ⇒ o botão de voz diz que **não** faz, em vez de fingir. */
  podeFalar?: boolean;
  /** **Só** é chamado quando o humano carrega. *O Companion nunca inicia uma acção por si.* */
  onFalarPedido?: () => void;
  /**
   * `E-1085`: as preferências do UTILIZADOR (`visivel`/`silenciado`/`movimentoReduzido`).
   * *Ausente ⇒ `PREFERENCIAS_PADRAO`* — **um componente que muda de comportamento para quem não mudou
   * nada é uma regressão, não uma feature.**
   */
  preferencias?: CompanionPreferencias;
  /**
   * `E-1085`: o que o `matchMedia("(prefers-reduced-motion: reduce)")` diz. *Injectado para que o teste
   * possa exercitar os dois lados, e para que o componente não leia `window` por conta própria.*
   */
  sistemaPedeMenosMovimento?: boolean;
}

/** O texto de cada estado. *Cada frase é verificável contra a `razao` que vem do servidor.* */
const TEXTO_DO_ESTADO: Record<CompanionEstado, { titulo: string; explicacao: string }> = {
  ausente: {
    titulo: "Sem missão em curso",
    explicacao: "Não há eventos desta missão. Não estou a fazer nada — e digo-o em vez de rodar.",
  },
  a_planear: {
    titulo: "A planear",
    explicacao: "A missão foi criada e está a ser planeada.",
  },
  a_trabalhar: {
    titulo: "A trabalhar",
    explicacao: "Há um passo aberto. É o único estado em que posso dizer isto.",
  },
  a_aprovar: {
    titulo: "À espera da sua decisão",
    explicacao: "Há uma aprovação pedida e não resolvida. Não estou a avançar: estou à sua espera.",
  },
  bloqueado: {
    titulo: "Bloqueado",
    explicacao: "A missão declarou um bloqueio. Ele não se levanta sozinho.",
  },
  concluido: {
    titulo: "Concluído",
    explicacao: "A missão terminou com um evento de conclusão.",
  },
  falhou: {
    titulo: "Falhou",
    explicacao: "A missão declarou falha. O motivo está na razão, abaixo.",
  },
  cancelado: {
    titulo: "Cancelado",
    explicacao: "A missão foi cancelada.",
  },
};

/** A classe de movimento — **derivada do estado**, e nunca de um temporizador. */
function classeDeMovimento(estado: CompanionEstado): string {
  switch (estado) {
    case "a_trabalhar":
      return "companion-typing";
    case "a_planear":
    case "a_aprovar":
      return "companion-breath";
    case "bloqueado":
    case "falhou":
      return "companion-pulse";
    default:
      // ***`ausente`, `concluido` e `cancelado` NÃO ANIMAM.*** *Um mascote parado quando nada acontece é a
      // representação fiel do sistema — e não uma avaria.*
      return "companion-idle";
  }
}

export function CompanionMascote({
  estado,
  razao,
  nome = "Hades",
  podeFalar = false,
  onFalarPedido,
  preferencias = PREFERENCIAS_PADRAO,
  sistemaPedeMenosMovimento = false,
}: CompanionProps) {
  const texto = TEXTO_DO_ESTADO[estado];
  const aprovacaoPendente = estado === "a_aprovar";

  // ***`E-1085` — AS PREFERÊNCIAS DECIDEM, E A DECISÃO FICA DECLARADA.*** *O `data-companion-animacao`
  // diz QUAL razão parou a animação (`pedido`/`sistema`/`nenhum`), em vez de a parar em silêncio: um
  // efeito visível sem causa visível é indistinguível de uma avaria.*
  const razaoDeParar = razaoParaNaoAnimar(preferencias, sistemaPedeMenosMovimento);
  const movimento = razaoDeParar === "nenhum" ? classeDeMovimento(estado) : "companion-idle";

  // ***A GUARDA FINAL.*** *O `E-1084` já a tinha escrito para ser usada: visível **E** estado conhecido.
  // Se o utilizador desligou o pet, não se desenha **nada** — nem um contentor vazio a ocupar espaço.*
  if (!podeDesenharCompanion(preferencias, estado)) {
    return null;
  }

  return (
    <section
      className="companion-state-transition flex flex-col gap-3 rounded-2xl border border-neutral-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"
      aria-label={`Companion ${nome}`}
      data-state={estado}
      data-companion-animacao={razaoDeParar}
    >
      <div className="flex items-center gap-3">
        {/* O mascote: o `aria-hidden` é deliberado — *o estado é anunciado pelo texto ao lado, e um leitor de ecrã não
            precisa de ouvir duas vezes a mesma coisa.* */}
        <span
          aria-hidden="true"
          className={`${movimento} inline-flex h-10 w-10 items-center justify-center rounded-full bg-neutral-100 text-lg dark:bg-neutral-800`}
        >
          {estado === "concluido" ? "✓" : estado === "falhou" ? "!" : estado === "bloqueado" ? "⏸" : "◆"}
        </span>

        <div className="min-w-0">
          <p className="section-title">{texto.titulo}</p>
          <p className="page-description !mt-0">{texto.explicacao}</p>
        </div>
      </div>

      {/* ***A RAZÃO, SEMPRE VISÍVEL.*** *Um estado sem procedência é indistinguível de um estado inventado — e o
          E-536 devolve a razão precisamente para que ela possa ser mostrada aqui.* */}
      {razao ? (
        <p className="text-xs text-neutral-500 dark:text-neutral-400">
          Procedência: <code>{razao}</code>
        </p>
      ) : null}

      {/* ***`aria-live="polite"` E NÃO `"assertive"`.*** O estado muda com frequência, e interromper a leitura a cada
          mudança é pior do que anunciar quando o leitor chegar ao fim do que estava a ler. *O `assertive` fica
          reservado a aprovações — que são o único caso em que a acção é do humano.* */}
      <p
        className="sr-only"
        aria-live={preferencias.silenciado ? "off" : aprovacaoPendente ? "assertive" : "polite"}
        aria-atomic="true"
      >
        {`Companion ${nome}: ${texto.titulo}. ${texto.explicacao}`}
      </p>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm font-medium disabled:opacity-60 dark:border-neutral-700"
          disabled={!podeFalar}
          onClick={onFalarPedido}
          aria-disabled={!podeFalar}
        >
          {podeFalar ? "Falar com o Companion" : "Voz indisponível"}
        </button>

        {/* ***A AUSÊNCIA, DECLARADA.*** *A mesma disciplina do `ErrMediaTranscoderUnavailable` e do
            `media_pipeline.go`: quando a capacidade não existe, di-lo — não a simules.* */}
        {!podeFalar ? (
          <span className="text-xs text-neutral-500 dark:text-neutral-400">
            Sem provider de fala ligado neste ambiente.
          </span>
        ) : null}
      </div>
    </section>
  );
}

/**
 * O texto de **onboarding**: *o que o Companion faz, e — mais importante — o que **não** faz.*
 *
 * ***PORQUE A SEGUNDA LISTA EXISTE.*** *Um onboarding que só promete é publicidade. Este diz o que está ausente, para
 * que ninguém espere uma capacidade que não existe — e para que a ausência seja um facto declarado, e não uma surpresa.*
 */
export const ONBOARDING_DO_COMPANION: ReadonlyArray<{ titulo: string; corpo: string; ausente?: boolean }> = [
  {
    titulo: "Mostro o estado real da missão",
    corpo:
      "O que vê vem de eventos verificados. Se não houver eventos, digo «sem missão em curso» em vez de animar.",
  },
  {
    titulo: "Paro quando é a sua vez",
    corpo:
      "Quando há uma aprovação pendente, deixo de narrar progresso e passo a pedir a decisão. Não é trabalho meu.",
  },
  {
    titulo: "Digo de onde vem cada estado",
    corpo: "Cada estado traz a razão (o evento que o produziu). Um estado sem procedência seria indistinguível de um inventado.",
  },
  {
    titulo: "Respeito o movimento reduzido",
    corpo:
      "Se o seu sistema pede menos movimento, as animações param. O estado continua a ser lido e anunciado.",
  },
  {
    titulo: "Voz: indisponível neste ambiente",
    corpo:
      "Não há provider de fala ligado. O botão diz que não faz, em vez de fingir que faz (BLOCKED_BY_EXTERNAL_DEPENDENCY).",
    ausente: true,
  },
  {
    titulo: "Não respondo perguntas",
    corpo: "Não sou um chat. Um mascote que inventa respostas seria uma alucinação com olhos.",
    ausente: true,
  },
];
