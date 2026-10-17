// BUSCA DO CONTEXTO DA MISSÃO, SEM DEPENDER DO REACT (D-9 / E-612).
//
// PORQUE ISTO EXISTE SEPARADO DO `App.tsx`: o `request` da app é declarado DENTRO do App.tsx e
// não é exportado, e o JSX daquele ficheiro é uma linha enorme. Pôr aqui a lógica permite
// TESTÁ-LA no harness da app (`node --experimental-strip-types`), sem renderização e sem rede —
// e deixa ao `App.tsx` uma chamada de uma linha.
//
// A REGRA QUE ESTE MÓDULO GARANTE: ***contexto indisponível nunca derruba a superfície***. A
// função NÃO lança: devolve uma vista vazia. O contexto é opcional — a missão continua a poder
// ser vista, executada e aprovada sem ele. É a mesma regra do servidor, onde contexto
// indisponível não bloqueia a conversa, e da UI do desktop, onde não esconde a entrega.

// @ts-expect-error Node's strip-types runner resolves the source extension directly.
// eslint-disable-next-line import/extensions
import { projectMissionContext, type MissionContextView } from "./missionContextPolicy.ts";

/** O caminho da rota canónica. Exportado para o teste o poder afirmar sem rede. */
export function missionContextPath(missionId: string): string {
  return `/api/agent/v1/missions/${encodeURIComponent(missionId)}/context`;
}

export type RespostaDeContexto = {
  /** Já projectada pela lista branca: só campos de trabalho. Vazia quando indisponível. */
  vista: MissionContextView;
  /** `undefined` quando correu bem; caso contrário, o motivo — para a superfície o poder dizer. */
  motivo?: string;
};

/**
 * fetchMissionContext busca e projecta o contexto. `fetchImpl` é injectado para que o teste não
 * precise de rede nem de React — e para que a app use o `fetch` que já tem.
 */
export async function fetchMissionContext(
  missionId: string,
  fetchImpl: typeof fetch,
  base: string,
  token?: string,
): Promise<RespostaDeContexto> {
  if (!missionId || !base) return { vista: {}, motivo: "missão ou servidor não declarado" };
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  try {
    const resposta = await fetchImpl(`${base.replace(/\/$/, "")}${missionContextPath(missionId)}`, { headers });
    if (!resposta || !resposta.ok) return { vista: {}, motivo: `HTTP ${resposta ? resposta.status : "sem resposta"}` };
    // O payload chega da rede: `unknown` é a verdade. Um tipo que prometesse a forma certa
    // estaria a confiar em quem responde.
    const payload: unknown = await resposta.json();
    return { vista: projectMissionContext(payload) };
  } catch (erro) {
    return { vista: {}, motivo: erro instanceof Error ? erro.message : "falha de rede" };
  }
}
