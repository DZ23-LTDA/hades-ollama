// FASE 17 + 23: CRM -- contactos, funil e consentimento.
//
// Décima primeira e décima segunda rotas do E-1319 a chegar ao produto:
// `/crm/contacts` e `/crm/pipeline`.
//
// `crm` era o maior domínio restante (**8 rotas**), com `crm.go` 18 KB e um store
// com consentimento por canal. Não havia componente `CRM*`.
//
//	*Uma rota que existe e não é consumida é uma promessa do backend, não uma
//	funcionalidade do produto.*
//
// OS CONTRATOS:
//
//	GET /crm/contacts?q=   -> 200 {contacts:[...], count:N}   503 CRM indisponivel
//	GET /crm/pipeline      -> 200 CRMPipeline (objecto directo)  503
//	GET /crm/contacts/:id  -> 200 CRMContactView {contact, consent, blocked_channels}
//
// TRÊS DECISÕES QUE O PAINEL NÃO PODE ERRAR:
//
//  1. **PII NÃO SE MOSTRA INTEIRA NUMA LISTA.** `phone` e `email` são dados
//     pessoais. O servidor devolve-os porque a API é do dono dos dados; uma lista
//     no ecrã é uma superfície diferente de uma resposta de API. O painel mascara
//     por omissão e diz que mascarou.
//
//		*Uma lista de contactos que mostra os contactos todos não é uma lista: é um
//		relatório que se pode fotografar.*
//
//  2. **CENTAVOS NÃO SÃO REAIS.** `value_cents`, `total_cents`,
//     `open_value_cents`, `won_value_cents` são inteiros de cêntimos.
//
//  3. **CONSENTIMENTO É POR CANAL E VEM DO REGISTRO.** O `blocked_channels` não é
//     um estado do contacto: é lido do `ConsentRegistry`. O painel mostra-o como
//     está -- sem o inferir, e sem o copiar para o contacto.
import { agentFetch } from "@/lib/agenticClient";
import { centavosParaReais } from "@/lib/commerceClient";

export interface ContactoDeCRM {
  id: string;
  organization_id: string;
  name: string;
  phone?: string;
  email?: string;
  tags?: string[];
  notes?: string;
  owner_id?: string;
  created_at: string;
  updated_at: string;
}

export interface OportunidadeDeCRM {
  id: string;
  organization_id: string;
  contact_id: string;
  title: string;
  value_cents: number;
  currency: string;
  stage: string;
  owner_id?: string;
  created_at: string;
  updated_at: string;
  stage_changed_at: string;
  stage_changed_by?: string;
  closed_at?: string;
}

export interface EtapaDoFunil {
  stage: string;
  count: number;
  total_cents: number;
}

export interface FunilDeCRM {
  organization_id: string;
  currency: string;
  stages: EtapaDoFunil[] | null;
  open_count: number;
  open_value_cents: number;
  won_value_cents: number;
  lost_count: number;
}

export interface RespostaDeContactos {
  contacts: ContactoDeCRM[] | null;
  count: number;
}

/** O CRM não está ligado neste runtime (`errCRMUnavailable`). */
export class CRMAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "CRMAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/CRM n[aã]o est[aá] dispon[íi]vel|503/i.test(mensagem)) {
    throw new CRMAusenteError(mensagem);
  }
  throw causa;
}

/**
 * Mascara um endereço: `ana.silva@exemplo.com` -> `a•••@exemplo.com`.
 *
 * Mantém o domínio (que é informação de negócio) e esconde a parte que identifica
 * a pessoa (que é dado pessoal).
 */
export function mascararEmail(email: string | undefined): string {
  const bruto = (email ?? "").trim();
  if (bruto === "") return "—";
  const arroba = bruto.lastIndexOf("@");
  if (arroba <= 0) return "•••";
  const local = bruto.slice(0, arroba);
  const dominio = bruto.slice(arroba + 1);
  const primeira = local.slice(0, 1);
  return `${primeira}•••@${dominio}`;
}

/**
 * Mascara um telefone: `+351912345678` -> `+351•••••678`.
 *
 * Mantém o indicativo (que diz de que país é) e os últimos dígitos (que ajudam a
 * distinguir dois números), e esconde o miolo.
 */
export function mascararTelefone(telefone: string | undefined): string {
  const bruto = (telefone ?? "").trim();
  if (bruto === "") return "—";
  const digitos = bruto.replace(/\D/g, "");
  if (digitos.length < 6) return "•••";
  const indicativo = bruto.startsWith("+") ? "+" + digitos.slice(0, 3) : "";
  const inicio = indicativo !== "" ? 3 : 0;
  const fim = digitos.slice(-3);
  const meio = "•".repeat(Math.max(3, digitos.length - inicio - 3));
  return `${indicativo}${meio}${fim}`;
}

/** Diz o que está bloqueado, com os canais em ordem. Não inventa o que não sabe. */
export function descreverCanaisBloqueados(bloqueados: Record<string, boolean> | undefined): string {
  if (!bloqueados) return "sem informação de consentimento";
  const canais = Object.keys(bloqueados)
    .filter((canal) => bloqueados[canal])
    .sort();
  if (canais.length === 0) return "nenhum canal bloqueado";
  return `bloqueado: ${canais.join(", ")}`;
}

/** O valor de uma oportunidade, em reais. Centavos não são reais. */
export function descreverValor(centavos: number | undefined, moeda = "BRL"): string {
  return centavosParaReais(centavos, moeda);
}

/** Etapa do funil dita de forma legível, sem inventar para o que não conhece. */
export function descreverEtapa(etapa: string | undefined): string {
  const bruto = (etapa ?? "").trim();
  if (bruto === "") return "sem etapa";
  return bruto.replace(/_/g, " ");
}

export async function listarContactos(
  consulta?: string,
  init?: RequestInit,
): Promise<RespostaDeContactos> {
  const limpo = (consulta ?? "").trim();
  const sufixo = limpo !== "" ? `?q=${encodeURIComponent(limpo)}` : "";
  try {
    // Prefixo `/api/agent/v1` explícito: o gate compara o path CRU (E-1321).
    return await agentFetch<RespostaDeContactos>(`/api/agent/v1/crm/contacts${sufixo}`, init);
  } catch (causa) {
    traduzir(causa);
  }
}

export async function funilDeCRM(init?: RequestInit): Promise<FunilDeCRM> {
  try {
    return await agentFetch<FunilDeCRM>("/api/agent/v1/crm/pipeline", init);
  } catch (causa) {
    traduzir(causa);
  }
}
