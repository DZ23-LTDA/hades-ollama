// Mídia -- `POST /media/ocr` e `POST /media/tone`.
//
// O CAMINHO DE ENTRADA E' CONTIDO NO WORKSPACE, E ISSO E' O QUE TORNA O OCR SEGURO
//
// O handler do OCR:
//
//     inputPath, err := containedPath(workspace, request.InputPath)
//
// O caminho nao e' usado como veio: e' **resolvido dentro do workspace da missao**. Um caminho que
// escapa -- `../../etc/passwd`, `/etc/shadow`, um symlink para fora -- e' recusado antes de se abrir
// o ficheiro.
//
//   *Um caminho que escapa do workspace nao e' um caminho: e' uma leitura arbitraria com nome de
//   parametro. O OCR le o ficheiro que lhe derem -- e e' por isso que quem decide qual ficheiro e' o
//   workspace, nao o cliente.*
//
// O painel NAO pode verificar isso, e nao tenta: manda o caminho relativo e mostra a recusa do servidor.
// Reimplementar `containedPath` no frontend seria a segunda verdade sobre o que e' "dentro".
//
// TODA A OPERACAO PERTENCE A UMA MISSAO
//
// `mission_id` e' obrigatorio nos dois: e' ele que escolhe o workspace, e o workspace e' o que limita o
// que pode ser lido e escrito. Sem missao nao ha' fronteira.
//
//   *Uma operacao de ficheiros sem missao nao tem raiz. E uma operacao de ficheiros sem raiz e' uma
//   operacao sobre o disco inteiro.*
//
// OS DOIS DEVOLVEM 201, E NAO E' 200
//
// Os dois **criam um Artifact**. 201 e' a resposta correcta: nao "correu bem", mas "existe uma coisa
// nova". Um 200 diria que nada foi criado -- e o Artifact e' o produto.
//
// O OCR PODE DEVOLVER 502, E ISSO NAO E' CULPA DO PEDIDO
//
//     writeAgentError(c, http.StatusBadGateway, err)
//
// 502 e' "a dependencia falhou", nao "o pedido esta' mal". A distincao importa: um 400 manda o
// utilizador corrigir o pedido; um 502 manda-o tentar outra vez ou ver o motor de OCR.
import { agentFetch } from "@/lib/agenticClient";

export interface ResultadoDeMidia {
  artifact?: {
    id?: string;
    mission_id?: string;
    kind?: string;
    path?: string;
    [k: string]: unknown;
  };
  [k: string]: unknown;
}

/** O runtime de midia nao esta' ligado neste runtime. */
export class MidiaAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MidiaAusenteError";
  }
}

/** O pedido foi recusado: missao em falta, caminho fora do workspace, parametro invalido. */
export class MidiaRecusadaError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MidiaRecusadaError";
  }
}

/** A dependencia de OCR falhou (502) -- o pedido estava bem, o motor e' que nao respondeu. */
export class MidiaIndisponivelError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "MidiaIndisponivelError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o\s+est[aá]['\u2019]?\s+ligado|n[aã]o\s+configurad/i.test(mensagem)) {
    throw new MidiaAusenteError("O runtime de mídia não está ligado neste runtime.");
  }
  // 502: o pedido estava correcto e a dependencia falhou. NAO e' erro do cliente.
  if (/502|bad gateway|ocr.*falh|motor de ocr/i.test(mensagem)) {
    throw new MidiaIndisponivelError(
      "O motor de OCR não respondeu (502): o pedido estava correcto — a dependência é que falhou.",
    );
  }
  if (/400|obrigat[oó]ri|fora do workspace|escape|inv[aá]lid|n[aã]o encontrad|mission/i.test(mensagem)) {
    throw new MidiaRecusadaError(mensagem);
  }
  throw causa;
}

/** Sem missao nao ha' workspace -- e sem workspace nao ha' fronteira. */
export function validarMissao(missionId: string | undefined): string | null {
  if ((missionId ?? "").trim() === "") {
    return "falta a missão (mission_id): é ela que escolhe o workspace e limita o que pode ser lido";
  }
  return null;
}

/**
 * O caminho tem de ser RELATIVO ao workspace. Um caminho absoluto ou com `..` nao e' um caminho
 * dentro do workspace -- mas quem decide isso e' o servidor. O painel so' recusa o obvio.
 */
export function validarCaminhoDeEntrada(caminho: string | undefined): string | null {
  const c = (caminho ?? "").trim();
  if (c === "") return "falta o caminho de entrada (input_path)";
  if (c.startsWith("/") || /^[A-Za-z]:[\\/]/.test(c)) {
    return "o caminho tem de ser relativo ao workspace da missão, não absoluto";
  }
  if (c.split(/[\\/]+/).includes("..")) {
    return "o caminho não pode subir com `..`: quem decide o que está dentro é o servidor";
  }
  return null;
}

/** Frequencia e duracao de um tom. */
export function validarTom(frequencia: number | undefined, duracaoMs: number | undefined): string | null {
  if (frequencia === undefined || !Number.isFinite(frequencia)) return "a frequência é obrigatória (Hz)";
  if (frequencia <= 0) return "a frequência tem de ser positiva (Hz)";
  if (frequencia > 20000) return "a frequência acima de 20 kHz não é audível";
  if (duracaoMs === undefined || !Number.isInteger(duracaoMs)) return "a duração tem de ser inteira em milissegundos";
  if (duracaoMs <= 0) return "a duração tem de ser positiva";
  return null;
}

/** O caminho e' resolvido pelo SERVIDOR -- o painel nao reimplementa `containedPath`. */
export function descreverFronteiraDoCaminho(): string {
  return (
    "O caminho de entrada é resolvido DENTRO do workspace da missão pelo servidor. O painel não " +
    "reimplementa essa verificação: uma segunda verdade sobre o que está «dentro» seria uma segunda " +
    "fronteira, e a mais fraca é a que fica."
  );
}

/** O que veio, dito sem inventar. */
export function descreverResultado(r: ResultadoDeMidia | null): string {
  if (!r) return "o servidor respondeu sem resultado";
  const a = r.artifact;
  if (!a) return "o servidor respondeu sem artefato — a operação não deixou produto";
  const id = (a.id ?? "").trim() === "" ? "(sem id)" : a.id;
  const kind = (a.kind ?? "").trim() === "" ? "tipo não declarado" : a.kind;
  return `artefato ${id} · tipo: ${kind}`;
}

/** OCR local. Cria um Artifact e devolve 201. */
export async function executarOcr(entrada: {
  missionId: string;
  inputPath: string;
  language?: string;
}): Promise<ResultadoDeMidia> {
  const erroMissao = validarMissao(entrada.missionId);
  if (erroMissao) throw new MidiaRecusadaError(erroMissao);
  const erroCaminho = validarCaminhoDeEntrada(entrada.inputPath);
  if (erroCaminho) throw new MidiaRecusadaError(erroCaminho);

  const corpo: Record<string, unknown> = {
    mission_id: entrada.missionId.trim(),
    input_path: entrada.inputPath.trim(),
  };
  if (entrada.language && entrada.language.trim() !== "") corpo.language = entrada.language.trim();

  try {
    return await agentFetch<ResultadoDeMidia>("/api/agent/v1/media/ocr", {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Gera um tom. Cria um Artifact e devolve 201. */
export async function gerarTom(entrada: {
  missionId: string;
  frequency: number;
  durationMs: number;
}): Promise<ResultadoDeMidia> {
  const erroMissao = validarMissao(entrada.missionId);
  if (erroMissao) throw new MidiaRecusadaError(erroMissao);
  const erroTom = validarTom(entrada.frequency, entrada.durationMs);
  if (erroTom) throw new MidiaRecusadaError(erroTom);

  try {
    return await agentFetch<ResultadoDeMidia>("/api/agent/v1/media/tone", {
      method: "POST",
      body: JSON.stringify({
        mission_id: entrada.missionId.trim(),
        frequency: entrada.frequency,
        duration_ms: entrada.durationMs,
      }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
