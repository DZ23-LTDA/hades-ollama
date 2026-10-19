// Registo de push -- `POST /notifications/register`.
//
// O TOKEN NAO VOLTA NA RESPOSTA, E ISSO E' ESTRUTURAL
//
// Em `internal/agent/push.go`:
//
//     Token string `json:"-"`
//
// O `json:"-"` nao e' um detalhe de serializacao: e' a garantia de que o token **nao tem como** sair
// no JSON. O handler ainda faz `subscription.Token = "redacted"` -- dupla proteccao.
//
//   *Um campo que nao se pode serializar nao precisa de ser redigido. Redigi-lo na mesma e' dizer que
//   quem escreveu o handler sabia que o campo existia -- e que nao confiou so' na tag.*
//
// O painel NAO mostra o token. Nao ha' nada para mostrar: o servidor nao o devolve.
//
// SEM SERVICO DE PUSH, A ROTA DIZ 501 E NAO FINGE
//
//     if a.push == nil { c.JSON(http.StatusNotImplemented, ...) }
//
// 501 e' "isto nao esta' implementado neste runtime" -- e' diferente de 503 ("a dependencia caiu").
// O painel distingue os dois: um manda configurar, o outro manda tentar de novo.
import { agentFetch } from "@/lib/agenticClient";

export interface SubscricaoDePush {
  id?: string;
  platform?: string;
  user_id?: string;
  organization_id?: string;
  created_at?: string;
  [k: string]: unknown;
}

/** O servico de push nao esta' configurado neste runtime (501). */
export class PushNaoImplementadoError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "PushNaoImplementadoError";
  }
}

/** O registo foi recusado: token vazio, plataforma invalida, sem sessao. */
export class PushRecusadoError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "PushRecusadoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  // 501: nao implementado. NAO e' o mesmo que 503 (dependencia caida).
  if (/501|not implemented|push service is not configured/i.test(mensagem)) {
    throw new PushNaoImplementadoError(
      "O serviço de push não está configurado neste runtime (501): não é uma falha temporária — é configuração em falta.",
    );
  }
  if (/400|401|403|obrigat[oó]ri|token|platform|invalid|n[aã]o autenticad/i.test(mensagem)) {
    throw new PushRecusadoError(mensagem);
  }
  throw causa;
}

const PLATAFORMAS = ["ios", "android", "web"] as const;
export type PlataformaDePush = (typeof PLATAFORMAS)[number];

/** Plataformas que o servidor reconhece. Uma plataforma inventada nao e' uma plataforma. */
export function plataformasConhecidas(): readonly string[] {
  return PLATAFORMAS;
}

/** O token tem de existir; a plataforma tem de ser uma das conhecidas. */
export function validarRegisto(token: string | undefined, plataforma: string | undefined): string | null {
  if ((token ?? "").trim() === "") return "falta o token do dispositivo";
  const p = (plataforma ?? "").trim().toLowerCase();
  if (p === "") return "falta a plataforma (ios, android ou web)";
  if (!(PLATAFORMAS as readonly string[]).includes(p)) {
    return `plataforma «${p}» não reconhecida: o servidor conhece ${PLATAFORMAS.join(", ")}`;
  }
  return null;
}

/** Diz por que o token nao pode ser mostrado -- e' uma garantia, nao uma omissao. */
export function descreverAusenciaDoToken(): string {
  return (
    "O token do dispositivo não aparece aqui porque o servidor não o pode devolver: o campo tem " +
    "`json:\"-\"` em `internal/agent/push.go`. Não é uma omissão do painel — é uma garantia do tipo."
  );
}

/** Registo confirmado, dito sem inventar. */
export function descreverSubscricao(s: SubscricaoDePush | null): string {
  if (!s) return "o servidor respondeu sem subscrição";
  const id = (s.id ?? "").trim() === "" ? "(sem id)" : s.id;
  const plataforma = (s.platform ?? "").trim() === "" ? "plataforma não declarada" : s.platform;
  return `subscrição ${id} · ${plataforma}`;
}

/** Regista o dispositivo. Devolve 201 e o token vem sempre ausente. */
export async function registarDispositivo(entrada: {
  token: string;
  platform: string;
}): Promise<SubscricaoDePush> {
  const erro = validarRegisto(entrada.token, entrada.platform);
  if (erro) throw new PushRecusadoError(erro);

  try {
    return await agentFetch<SubscricaoDePush>("/api/agent/v1/notifications/register", {
      method: "POST",
      body: JSON.stringify({ token: entrada.token.trim(), platform: entrada.platform.trim().toLowerCase() }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
