// Sessão de autenticação -- a leitura de `GET /auth/session`.
//
// `auth` tem 13 rotas. So' UMA e' leitura sem parametro: esta. As outras 12 sao escritas
// (logout, dev/token, MFA enable/disable/recovery, OAuth start/callback/refresh/revoke,
// SAML start/metadata/acs) -- acoes do utilizador, com consentimento e efeitos. O painel
// nao as chama e nao inventa botoes para elas.
//
// TRES COISAS QUE ESTE CONTRATO ENSINA:
//
//  1. **`Public()` E' A FRONTEIRA DOS SEGREDOS.** O handler chama `agent.User.Public()`:
//
//         func (u User) Public() User {
//             u.MFASecretCiphertext = ""
//             u.RecoveryCodesCiphertext = ""
//             return u
//         }
//
//     E os dois campos tem `omitempty`, portanto **desaparecem do JSON**. O painel nunca
//     ve um segredo de MFA nem um codigo de recuperacao -- nem cifrados.
//
//  2. **`omitempty` EM `mfa_enabled` FAZ «AUSENTE» SIGNIFICAR `false`.**
//
//         MFAEnabled bool `json:"mfa_enabled,omitempty"`
//
//     Quando o utilizador NAO tem MFA, o campo nao vem. E isso NAO e' «desconhecido»: e'
//     `false`. *Um campo booleano com `omitempty` que se leia como «nao sei» transforma
//     um «desligado» em duvida, e a duvida num alarme falso.*
//
//  3. **`mode: "local"` NAO E' ERRO.** Sem autenticacao exigida, o handler responde 200:
//
//         200 {"authenticated": false, "mode": "local"}
//
//     Isso e' o HADES a correr localmente sem login. Tratar isso como «sessao invalida»
//     faria o painel mentir sobre a instalacao mais comum.
import { agentFetch } from "@/lib/agenticClient";

export interface UtilizadorPublico {
  id: string;
  email: string;
  name: string;
  /** `omitempty`: ausente significa `false`, NUNCA «desconhecido». */
  mfa_enabled?: boolean;
  created_at: string;
}

export interface MembroDaOrganizacao {
  organization_id?: string;
  user_id?: string;
  role?: string;
  created_at?: string;
}

export interface SessaoDeAutenticacao {
  authenticated: boolean;
  /** `"local"` quando o HADES corre sem autenticacao exigida. */
  mode?: string;
  user?: UtilizadorPublico;
  organization?: Record<string, unknown>;
  membership?: MembroDaOrganizacao;
}

/**
 * Descreve o modo da sessao.
 *
 * `mode: "local"` com `authenticated: false` e' o modo local -- nao uma sessao expirada.
 */
export function descreverModo(sessao: SessaoDeAutenticacao | undefined): string {
  if (!sessao) return "sem informacao de sessao";
  if (sessao.mode === "local") {
    return "modo local -- sem autenticação exigida";
  }
  if (sessao.authenticated) return "sessão autenticada";
  return "sessão não autenticada";
}

/**
 * Le `mfa_enabled` sem confundir ausente com desconhecido.
 *
 * `omitempty` num booleano significa que `false` nao viaja. Ausente e' `false`.
 */
export function mfaLigado(utilizador: UtilizadorPublico | undefined): boolean {
  if (!utilizador) return false;
  return utilizador.mfa_enabled === true;
}

/** Descreve o estado de MFA, sem inventar incerteza. */
export function descreverMFA(utilizador: UtilizadorPublico | undefined): string {
  if (!utilizador) return "sem informação de MFA";
  return mfaLigado(utilizador)
    ? "MFA ligado neste utilizador"
    : "MFA desligado neste utilizador";
}

/** Nome legivel, com recurso ao email e depois ao id. */
export function nomeDoUtilizador(utilizador: UtilizadorPublico | undefined): string {
  if (!utilizador) return "utilizador não identificado";
  const nome = (utilizador.name ?? "").trim();
  if (nome !== "") return nome;
  const email = (utilizador.email ?? "").trim();
  if (email !== "") return email;
  const id = (utilizador.id ?? "").trim();
  if (id !== "") return id;
  return "utilizador sem identificação";
}

/** Papel na organizacao, quando declarado. */
export function descreverPapel(membro: MembroDaOrganizacao | undefined): string {
  const papel = (membro?.role ?? "").trim();
  if (papel === "") return "papel não declarado";
  return `papel: ${papel}`;
}

export async function obterSessao(init?: RequestInit): Promise<SessaoDeAutenticacao> {
  return await agentFetch<SessaoDeAutenticacao>("/api/agent/v1/auth/session", init);
}
