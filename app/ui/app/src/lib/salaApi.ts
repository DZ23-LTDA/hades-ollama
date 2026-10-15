import type { MembroDaSala } from "../components/SalaMembros";

/**
 * Complemento 48 / FASE 2 — O CLIENTE DA SALA.
 *
 * **A REGRA QUE GOVERNA ESTE FICHEIRO:** *uma falha NUNCA pode virar lista vazia.* O componente `SalaMembros`
 * distingue «não consegui carregar» de «não há ninguém»; se o cliente engolir o erro e devolver `[]`, essa
 * distinção morre aqui e a tela volta a mentir — *com a agravante de o teste do componente continuar verde*.
 * Por isso aqui **falha lança**, e lista vazia só existe quando o servidor disse mesmo que está vazia.
 *
 * LIMITE DECLARADO (medido, não presumido): **o endpoint ainda não existe no servidor.** A matriz do complemento
 * tem a sala no domínio (`internal/agent/rooms.go`) e a superfície na UI, mas **não há rota HTTP** exposta para
 * os membros da sala. Este cliente fixa o **contrato** que a rota terá de cumprir; enquanto ela não existir, o
 * caminho real devolve `SalaIndisponivel` — *e isso é uma falha declarada, não uma lista vazia.*
 */

/** O que o servidor tem de devolver: os membros, com o mesmo vocabulário do domínio. */
export type RespostaDaSala = {
  membros: MembroDaSala[];
};

/** Falha de leitura da sala. **Sempre** um erro — nunca um `[]` disfarçado. */
export class SalaIndisponivel extends Error {
  // NOTA DE TIPO (aprendida com o `tsc`, não com a opinião): este projeto tem `erasableSyntaxOnly`, que proíbe
  // **parameter properties** (`constructor(private x: T)`). *Campos declarados à parte e atribuídos no corpo* —
  // e `tsc -b` é uma verificação diferente de «os testes passaram».
  readonly causa?: unknown;

  constructor(message: string, causa?: unknown) {
    super(message);
    this.name = "SalaIndisponivel";
    this.causa = causa;
  }
}

/** Assinatura mínima de `fetch` que o cliente usa — injectável para o teste não tocar na rede. */
export type Fetcher = (
  input: string,
  init?: { signal?: AbortSignal; headers?: Record<string, string> },
) => Promise<{
  ok: boolean;
  status: number;
  json: () => Promise<unknown>;
}>;

export function urlDosMembros(projectID: string, roomID: string): string {
  return `/api/agent/v1/projects/${encodeURIComponent(projectID)}/rooms/${encodeURIComponent(roomID)}/members`;
}

function ehMembro(valor: unknown): valor is MembroDaSala {
  if (typeof valor !== "object" || valor === null) return false;
  const candidato = valor as Record<string, unknown>;
  const tipo = candidato.tipo;
  return (
    typeof candidato.id === "string" &&
    (tipo === "humano" || tipo === "agente") &&
    (candidato.nome === undefined || typeof candidato.nome === "string")
  );
}

/**
 * `buscarMembrosDaSala` devolve os membros lidos do servidor.
 *
 * *Falha ⇒ lança `SalaIndisponivel`.* Corpo sem `membros` **não** é «sala vazia»: é resposta inválida — e uma
 * resposta inválida é uma falha, não uma ausência. *É esta a diferença que o `SalaMembros` testa na tela e que
 * aqui tem de ser preservada.*
 */
export async function buscarMembrosDaSala(opcoes: {
  fetcher: Fetcher;
  projectID: string;
  roomID: string;
  signal?: AbortSignal;
}): Promise<MembroDaSala[]> {
  const { fetcher, projectID, roomID, signal } = opcoes;
  let resposta: Awaited<ReturnType<Fetcher>>;
  try {
    resposta = await fetcher(urlDosMembros(projectID, roomID), {
      signal,
      headers: { accept: "application/json" },
    });
  } catch (causa) {
    throw new SalaIndisponivel("a leitura da sala falhou (rede)", causa);
  }

  if (!resposta.ok) {
    throw new SalaIndisponivel(`o servidor recusou a leitura da sala (HTTP ${resposta.status})`);
  }

  let corpo: unknown;
  try {
    corpo = await resposta.json();
  } catch (causa) {
    throw new SalaIndisponivel("a resposta da sala não é JSON válido", causa);
  }

  if (typeof corpo !== "object" || corpo === null || !("membros" in corpo)) {
    throw new SalaIndisponivel("a resposta da sala não trouxe o campo `membros`");
  }

  const membros = (corpo as RespostaDaSala).membros;
  if (!Array.isArray(membros)) {
    throw new SalaIndisponivel("o campo `membros` da sala não é uma lista");
  }
  for (const membro of membros) {
    if (!ehMembro(membro)) {
      throw new SalaIndisponivel("a sala trouxe um membro com forma desconhecida");
    }
  }
  return membros;
}
