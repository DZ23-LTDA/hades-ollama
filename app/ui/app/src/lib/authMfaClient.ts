// MFA -- `POST /auth/mfa/enable`, `/auth/mfa/disable`, `/auth/mfa/recovery/generate`.
//
// 🚨 ACHADO DE SEGURANCA (documentado, nao corrigivel nesta fatia)
//
// Os tres handlers verificam APENAS `agent.user`:
//
//     value, ok := c.Get("agent.user")
//     user, userOK := value.(agent.User)
//     if !ok || !userOK { ... 401 }
//
// Nenhum pede senha, TOTP actual, codigo de recuperacao ou qualquer prova de posse do segundo factor.
// `disableMFA` e `generateRecoveryCodes` **nem corpo tem**: quem tiver um bearer token valido desativa o
// MFA e gera codigos de recuperacao novos com um pedido vazio.
//
//   *Um segundo factor que qualquer portador do primeiro pode remover nao e' um segundo factor: e' uma
//   cerimonia. A prova de posse do primeiro nao prova posse do segundo.*
//
// O PAINEL NAO FINGE UMA REAUTENTICACAO QUE O SERVIDOR NAO FAZ
//
// Pedir a senha no painel e envia-la para um servidor que a ignora seria **teatro de seguranca**: o
// utilizador ficaria com a impressao de uma barreira que nao existe. O painel, em vez disso:
//
//   1. exige uma **confirmacao explicita** do utilizador (defesa em profundidade honesta -- trava o
//      clique por engano, e nada mais);
//   2. **declara** que o servidor nao reautentica, num testid proprio;
//   3. reporta o achado como finding, nao como funcionalidade.
//
//   *Uma confirmacao local protege contra o engano, nao contra o ataque. Dizer qual dos dois e' e' a
//   diferenca entre um painel e um selo.*
//
// OS CODIGOS DE RECUPERACAO SAO MOSTRADOS UMA VEZ, E O SERVIDOR DIZ ISSO
//
//     warning: "store these codes securely; they are shown only once"
//
// O painel nao guarda, nao reenvia, nao persiste. Mostra e avisa. *Um codigo de recuperacao que o
// servidor pode voltar a mostrar nao e' um codigo: e' uma password.*
import { agentFetch } from "@/lib/agenticClient";

export interface ResultadoDeMfa {
  enabled?: boolean;
  issuer?: string;
  account?: string;
  [k: string]: unknown;
}

export interface ResultadoDeRecuperacao {
  user?: { email?: string; [k: string]: unknown };
  recovery_codes?: string[];
  warning?: string;
  [k: string]: unknown;
}

/** Nao ha' sessao autenticada (401). */
export class MfaSemSessaoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "MfaSemSessaoError";
  }
}

/** O segredo TOTP foi recusado, ou o pedido e' invalido. */
export class MfaRecusadoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "MfaRecusadoError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/401|authenticated user|sess[aã]o|n[aã]o autenticad/i.test(mensagem)) {
    throw new MfaSemSessaoError("Não há sessão autenticada (401): o MFA pertence sempre a um utilizador.");
  }
  if (/400|secret|inv[aá]lid|mfa|c[oó]digo/i.test(mensagem)) {
    throw new MfaRecusadoError(mensagem);
  }
  throw causa;
}

/** O segredo TOTP tem de ser base32 e nao vazio. Nao validamos o algoritmo: isso e' do servidor. */
export function validarSegredo(segredo: string | undefined): string | null {
  const s = (segredo ?? "").trim().replace(/\s+/g, "");
  if (s === "") return "falta o segredo TOTP (secret)";
  if (!/^[A-Z2-7]+=*$/i.test(s)) {
    return "o segredo TOTP tem de ser base32 (letras A–Z e dígitos 2–7)";
  }
  if (s.replace(/=+$/, "").length < 16) {
    return "o segredo TOTP tem menos de 16 caracteres: não é uma chave, é um lapso";
  }
  return null;
}

/** O achado, dito como achado -- nao como funcionalidade. */
export function descreverAchadoDeReautenticacao(): string {
  return (
    "ACHADO: nenhum dos três handlers de MFA exige reautenticação — verificam apenas a sessão. Quem " +
    "tiver um bearer token válido desativa o MFA ou gera códigos novos com um pedido vazio. Um segundo " +
    "factor que qualquer portador do primeiro pode remover não é um segundo factor: é uma cerimónia. " +
    "Esta confirmação local protege contra o engano, não contra o ataque."
  );
}

/** O servidor nao reautentica; o painel nao finge que sim. */
export function descreverLimiteDaConfirmacao(): string {
  return (
    "A confirmação abaixo é uma trava de clique, não uma barreira de segurança. O servidor não pede " +
    "senha nem TOTP para desativar o MFA nem para gerar códigos de recuperação."
  );
}

/** O estado do MFA, dito sem inventar. */
export function descreverEstado(r: ResultadoDeMfa | null): string {
  if (!r) return "o servidor respondeu sem estado";
  if (r.enabled === true) {
    return `MFA LIGADO${r.account ? ` · conta ${r.account}` : ""}${r.issuer ? ` · emissor ${r.issuer}` : ""}`;
  }
  if (r.enabled === false) return "MFA DESLIGADO";
  return "o servidor não declarou o estado do MFA";
}

/** Os codigos de recuperacao: mostrados uma vez, nunca guardados. */
export function descreverCodigos(r: ResultadoDeRecuperacao | null): string {
  if (!r) return "o servidor respondeu sem códigos";
  const n = (r.recovery_codes ?? []).length;
  if (n === 0) return "o servidor devolveu zero códigos — a operação não produziu o que prometeu";
  return `${n} código(s) de recuperação — mostrados UMA vez. O servidor não os volta a mostrar.`;
}

/** Liga o MFA com o segredo TOTP. Devolve 200. */
export async function ligarMfa(segredo: string): Promise<ResultadoDeMfa> {
  const erro = validarSegredo(segredo);
  if (erro) throw new MfaRecusadoError(erro);

  try {
    return await agentFetch<ResultadoDeMfa>("/api/agent/v1/auth/mfa/enable", {
      method: "POST",
      body: JSON.stringify({ secret: segredo.trim().replace(/\s+/g, "") }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Desliga o MFA. SEM corpo no pedido -- e e' isso que torna o achado grave. */
export async function desligarMfa(): Promise<ResultadoDeMfa> {
  try {
    return await agentFetch<ResultadoDeMfa>("/api/agent/v1/auth/mfa/disable", { method: "POST" });
  } catch (causa) {
    traduzir(causa);
  }
}

/** Gera codigos de recuperacao. Devolve 201, e os codigos so' aparecem uma vez. */
export async function gerarCodigosDeRecuperacao(): Promise<ResultadoDeRecuperacao> {
  try {
    return await agentFetch<ResultadoDeRecuperacao>("/api/agent/v1/auth/mfa/recovery/generate", {
      method: "POST",
    });
  } catch (causa) {
    traduzir(causa);
  }
}
