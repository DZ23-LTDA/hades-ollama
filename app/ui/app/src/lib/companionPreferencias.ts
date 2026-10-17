import { ESTADOS_CONHECIDOS } from "./companionApi";

/**
 * `E-1084` (§FASE-3-A, item 4) — as **preferências do Companion**, e o que elas NÃO são.
 *
 * ***PORQUE ESTE FICHEIRO EXISTE.*** O prompt exige que o pet possa ser **ocultado, silenciado e
 * restaurado**, e que a preferência **persista**, *«sem POST em loop»*. O `companionApi.ts` lê o
 * estado do servidor; ele **não** guarda decisão nenhuma do utilizador.
 *
 * ***E PORQUE `localStorage` E NÃO POST.*** A preferência é do **utilizador, neste dispositivo** —
 * não é estado da missão, não interessa ao servidor e não deve ser sincronizada. *Um POST por cada
 * mudança de preferência seria a «chamada em loop» que o prompt proíbe, e uma chamada de rede para
 * decidir se um desenho aparece.*
 *
 * ***O QUE ESTE FICHEIRO NÃO FAZ.*** Não lê o estado da missão, não desenha, não anima e não decide
 * o que aconteceu. *Ele guarda três booleanos e mais nada.*
 */

/** As preferências, e o **default de cada uma**: *o pet aparece, fala baixo e anima como os outros.* */
export interface CompanionPreferencias {
  /** `false` ⇒ o Companion não é desenhado. *Desligar não perde nenhuma funcionalidade central.* */
  visivel: boolean;
  /** `true` ⇒ o `aria-live` passa a `off` e as notificações do pet calam-se. *A acessibilidade não se perde: o texto continua na página.* */
  silenciado: boolean;
  /** `true` ⇒ nenhuma animação, independentemente de `prefers-reduced-motion`. */
  movimentoReduzido: boolean;
}

/** O default. *Um default é um contrato: mudá-lo muda o produto de quem nunca escolheu nada.* */
export const PREFERENCIAS_PADRAO: Readonly<CompanionPreferencias> = Object.freeze({
  visivel: true,
  silenciado: false,
  movimentoReduzido: false,
});

/** A chave no armazenamento. *Versionada, para que uma mudança de forma não leia lixo como se fosse válido.* */
export const CHAVE_DAS_PREFERENCIAS = "hades.companion.preferencias.v1";

/**
 * Valida o que veio do armazenamento — **e NÃO confia nele**.
 *
 * *O `localStorage` é escrito por qualquer script da mesma origem, sobrevive a versões e é editável
 * à mão.* ⇒ *um valor com a forma errada tem de cair no **default**, e não atravessar como se fosse
 * uma preferência legítima.* Medido no `ESTADOS_CONHECIDOS` do `companionApi.ts`: o defeito é o
 * mesmo, noutro sítio — **um estado fora do conjunto não pode atravessar a fronteira.**
 */
export function validarPreferencias(bruto: unknown): CompanionPreferencias {
  if (typeof bruto !== "object" || bruto === null) return { ...PREFERENCIAS_PADRAO };
  const o = bruto as Record<string, unknown>;
  const ler = (k: keyof CompanionPreferencias): boolean =>
    typeof o[k] === "boolean" ? (o[k] as boolean) : PREFERENCIAS_PADRAO[k];
  return {
    visivel: ler("visivel"),
    silenciado: ler("silenciado"),
    movimentoReduzido: ler("movimentoReduzido"),
  };
}

/**
 * Lê as preferências guardadas.
 *
 * *Devolve o **default** em qualquer falha* — sem `localStorage` (SSR, modo privado), com JSON
 * inválido, com a forma errada. ⇒ **uma preferência ilegível não é uma preferência: é o default.**
 */
export function carregarPreferencias(
  armazenamento: Pick<Storage, "getItem"> | null = typeof localStorage === "undefined"
    ? null
    : localStorage,
): CompanionPreferencias {
  if (!armazenamento) return { ...PREFERENCIAS_PADRAO };
  try {
    const cru = armazenamento.getItem(CHAVE_DAS_PREFERENCIAS);
    if (!cru) return { ...PREFERENCIAS_PADRAO };
    return validarPreferencias(JSON.parse(cru));
  } catch {
    return { ...PREFERENCIAS_PADRAO };
  }
}

/**
 * Guarda as preferências. **Devolve `true` se conseguiu**, e `false` se não.
 *
 * *O `false` não é decoração:* quem chama tem de poder dizer ao utilizador que a escolha **não**
 * persistiu, em vez de deixar a UI a prometer uma persistência que não houve. **É a mesma regra do
 * `estadoIndisponivel`: a ausência declara-se.**
 */
export function guardarPreferencias(
  prefs: CompanionPreferencias,
  armazenamento: Pick<Storage, "setItem"> | null = typeof localStorage === "undefined"
    ? null
    : localStorage,
): boolean {
  if (!armazenamento) return false;
  try {
    armazenamento.setItem(CHAVE_DAS_PREFERENCIAS, JSON.stringify(validarPreferencias(prefs)));
    return true;
  } catch {
    // *Quota cheia, modo privado, ou o armazenamento desligado por política: o chamador decide.*
    return false;
  }
}

/** Restaura o default — *«restaurar defaults» é uma operação, não a ausência de uma.* */
export function restaurarPreferencias(
  armazenamento: Pick<Storage, "setItem"> | null = typeof localStorage === "undefined"
    ? null
    : localStorage,
): boolean {
  return guardarPreferencias({ ...PREFERENCIAS_PADRAO }, armazenamento);
}

/**
 * A decisão de **animar ou não**, com a **precedência** explícita.
 *
 * *São DUAS razões independentes para não animar, e a ordem importa para o `data-*` que a UI expõe:*
 *  1. o utilizador pediu **movimento reduzido** ⇒ `"pedido"`;
 *  2. o sistema pediu **`prefers-reduced-motion`** ⇒ `"sistema"`;
 *  3. nenhum ⇒ `"nenhum"`.
 *
 * ⇒ *o `"pedido"` vence o `"sistema"` porque um pedido explícito é mais específico que uma definição
 * do sistema operativo — mas **os dois param a animação**, e é isso que o teste discrimina.*
 */
export type RazaoDeNaoAnimar = "pedido" | "sistema" | "nenhum";

export function razaoParaNaoAnimar(
  prefs: CompanionPreferencias,
  sistemaPedeMenosMovimento: boolean,
): RazaoDeNaoAnimar {
  if (prefs.movimentoReduzido) return "pedido";
  if (sistemaPedeMenosMovimento) return "sistema";
  return "nenhum";
}

/**
 * Lê a preferência do **sistema** (`prefers-reduced-motion`), com guarda para ambientes sem `matchMedia`.
 *
 * *Devolve `false` quando não é possível saber* — e `false` significa «anima». ⇒ *a ausência de
 * informação **não** desliga a animação de quem não pediu nada; mas o `movimentoReduzido` do
 * utilizador continua a vencer, porque não depende de `matchMedia`.*
 */
export function sistemaPedeMenosMovimento(
  matchMedia: ((q: string) => { matches: boolean }) | null = typeof window === "undefined"
    ? null
    : window.matchMedia,
): boolean {
  if (!matchMedia) return false;
  try {
    return matchMedia("(prefers-reduced-motion: reduce)").matches;
  } catch {
    return false;
  }
}

/**
 * A guarda final: **o Companion pode ser desenhado?**
 *
 * *Duas condições, e as duas têm de valer:* a preferência `visivel` **e** um estado que o servidor
 * conhece. ⇒ *não basta o utilizador querer vê-lo: é preciso que o servidor tenha afirmado um estado
 * legítimo. Um pet desenhado sobre um estado que ninguém afirmou é a animação falsa.*
 */
export function podeDesenharCompanion(
  prefs: CompanionPreferencias,
  estado: string,
): boolean {
  return prefs.visivel && (ESTADOS_CONHECIDOS as readonly string[]).includes(estado);
}
