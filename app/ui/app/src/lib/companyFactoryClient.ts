// Relatorio da fabrica de empresa -- `GET /company/factory/report/:id`.
//
// `company` tem 2 rotas: `factory/launch` e `factory/report/:id`. **So' a segunda e' leitura.**
//
// DOIS CAMPOS QUE NAO SAO NUMEROS: SAO AFIRMACOES
//
// O servidor declara, no proprio tipo:
//
//     // CustomersCreated e' sempre 0: a fabrica NAO inventa clientes.
//     CustomersCreated int `json:"customers_created"`
//
// E no topo das rotas:
//
//     // O lancamento materializa artefactos reais (plano, marca, precificacao e site) e
//     // devolve o checklist com o que depende de terceiros -- a fabrica NAO cria cliente
//     // nem emite registo legal.
//
// **`customers_created: 0` nao e' «zero clientes»: e' «esta fabrica nunca cria clientes».** Um
// painel que mostre «0 clientes» ao lado de «3 produtos» esta' a sugerir uma empresa que
// arrancou sem clientes -- quando o que aconteceu foi a fabrica ter feito tudo o que **nao
// depende de um humano**. *Zero informado e zero cobrado sao o mesmo numero e duas afirmacoes
// diferentes -- e aqui o zero e' uma fronteira, nao uma contagem.*
//
// `CatalogValueCents` e' a soma do valor de TABELA do estoque inicial -- nao e' dinheiro
// recebido, nao e' receita e nao e' avaliacao da empresa. *Um valor de tabela e' o preco
// pedido, nao o preco pago.*
import { centavosParaReais } from "@/lib/commerceClient";
import { agentFetch } from "@/lib/agenticClient";

export interface ItemDoChecklist {
  id?: string;
  label?: string;
  title?: string;
  description?: string;
  status?: string;
  /** Depende de terceiro (banco, contador, registo, cliente) e nao da fabrica. */
  depends_on_third_party?: boolean;
  external?: boolean;
  done?: boolean;
}

export interface ArtefactoDaFabrica {
  id?: string;
  kind?: string;
  name?: string;
  title?: string;
  url?: string;
  path?: string;
}

export interface RelatorioDaFabrica {
  company_id: string;
  organization_id: string;
  launched_at: string;
  company?: Record<string, unknown>;
  products?: unknown[] | null;
  artifacts?: ArtefactoDaFabrica[] | null;
  checklist?: ItemDoChecklist[] | null;
  /** Soma do valor de TABELA do estoque inicial, em centavos. Nao e' receita. */
  catalog_value_cents: number;
  /** Sempre 0: a fabrica NAO inventa clientes. */
  customers_created: number;
}

/** A fabrica de empresa nao esta' ligada neste runtime. */
export class FabricaAusenteError extends Error {
  constructor(mensagem: string) {
    super(mensagem);
    this.name = "FabricaAusenteError";
  }
}

function traduzir(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/company|empresa|n[aã]o est[aá] ligado|n[aã]o encontrad|503|404/i.test(mensagem)) {
    throw new FabricaAusenteError(mensagem);
  }
  throw causa;
}

/**
 * `customers_created` e' sempre 0 -- e isso e' uma AFIRMACAO, nao uma contagem.
 *
 * Devolve o texto que diz o que o zero significa.
 */
export function descreverClientesCriados(valor: number | undefined): string {
  if (valor === undefined || valor === null) return "clientes criados: não declarado";
  if (valor === 0) return "0 clientes criados — a fábrica não inventa clientes";
  return `${valor} cliente(s) criado(s)`;
}

/** O valor de TABELA do estoque, dito como valor de tabela. */
export function descreverValorDeCatalogo(centavos: number | undefined): string {
  const valor = Math.max(0, centavos ?? 0);
  if (valor === 0) return "estoque inicial sem valor de tabela declarado";
  return `valor de tabela do estoque: ${centavosParaReais(valor)} (não é receita)`;
}

/** Um item do checklist que depende de terceiro. */
export function dependeDeTerceiro(item: ItemDoChecklist): boolean {
  if (item.depends_on_third_party === true) return true;
  if (item.external === true) return true;
  const s = (item.status ?? "").trim().toLowerCase();
  return s === "external" || s === "third_party" || s === "blocked_external";
}

/** Separa o que a fabrica fez do que ficou para um humano. */
export function resumirChecklist(itens: ItemDoChecklist[] | null | undefined): string {
  const lista = itens ?? [];
  if (lista.length === 0) return "sem checklist declarado";
  const externos = lista.filter(dependeDeTerceiro).length;
  const internos = lista.length - externos;
  if (externos === 0) return `${internos} item(ns), todos ao alcance da fábrica`;
  return `${internos} feito(s) pela fábrica · ${externos} à espera de terceiro`;
}

/** Rotulo de um item, sem inventar. */
export function rotuloDoItem(item: ItemDoChecklist): string {
  const rotulo = (item.label ?? item.title ?? "").trim();
  if (rotulo !== "") return rotulo;
  const id = (item.id ?? "").trim();
  return id === "" ? "item sem rótulo" : id;
}

export async function obterRelatorioDaFabrica(id: string): Promise<RelatorioDaFabrica> {
  const alvo = (id ?? "").trim();
  if (alvo === "") {
    throw new FabricaAusenteError("sem id de empresa: nao ha' relatorio a pedir");
  }
  try {
    return await agentFetch<RelatorioDaFabrica>(
      `/api/agent/v1/company/factory/report/${encodeURIComponent(alvo)}`,
    );
  } catch (causa) {
    traduzir(causa);
  }
}
