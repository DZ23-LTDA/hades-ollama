/**
 * Fila de envio do composer (defeito D).
 *
 * Antes: `ChatForm.handleSubmit` fazia `if (... || isStreaming) return;` -- a mensagem era
 * DESCARTADA em silencio enquanto a IA respondia. Aqui a fila e' pura e testavel: enfileirar,
 * retirar o proximo e limpar (parar esvazia).
 */

/** Enfileira texto nao vazio. Texto vazio/espacos NAO entra (evita "envios fantasmas"). */
export function enfileirar(fila: readonly string[], texto: string): string[] {
  const limpo = texto.trim();
  if (limpo === "") return [...fila];
  return [...fila, limpo];
}

/** Retira o primeiro item; devolve null quando a fila esta' vazia. */
export function tirarProximo(fila: readonly string[]): { proximo: string | null; resto: string[] } {
  if (fila.length === 0) return { proximo: null, resto: [] };
  const [proximo, ...resto] = fila;
  return { proximo, resto };
}

export const FILA_VAZIA: readonly string[] = [];