// Cockpit da empresa -- `POST /business/certify` e `POST /company/factory/launch`.
//
// Sao dois atos do MESMO ciclo de vida: a fabrica LANCA a empresa (cria artefatos), a certificacao
// PROVA que a jornada inteira funciona. Um sem o outro e' uma promessa ou um exame sem aluno.
//
// A FABRICA NAO CRIA CLIENTE NEM EMITE REGISTRO LEGAL
//
// O comentario do handler e' explicito:
//
//     O lançamento materializa artefatos reais (plano, marca, precificação e site) e devolve o
//     checklist com o que depende de terceiros — a fábrica NÃO cria cliente nem emite registro legal.
//
//   *Um botao que promete «abrir a empresa» e entrega um ZIP de marca nao abriu a empresa: entregou o
//   prospecto. O painel tem de dizer isso com as palavras do servidor, nao com as do marketing.*
//
// A CERTIFICACAO DEVOLVE O RELATORIO MESMO QUANDO FALHA -- E ESSE E' O PONTO
//
//     if err != nil {
//         // O relatório VAI junto mesmo em falha: é ele que diz qual etapa quebrou.
//         c.JSON(http.StatusUnprocessableEntity, gin.H{"report": report, "error": err.Error()})
//         return
//     }
//
// **422 com `report` E `error`.** O relatorio e' o produto; o erro e' a conclusao. Devolver so' o erro
// esconderia as etapas que passaram; devolver so' o relatorio esconderia que a jornada quebrou.
//
//   *Um relatorio que so' aparece quando tudo corre bem e' um premio, nao um diagnostico. O diagnostico
//   e' o que se le' quando algo falhou.*
//
// E A PROVA DE ISOLAMENTO SEMPRE PRECISA DE UMA SEGUNDA ORGANIZACAO
//
//     if otherOrganizationID == "" {
//         otherOrganizationID = "certificacao-outra-organizacao"
//     }
//
// *Um teste de isolamento com uma so' organizacao prova que uma organizacao se ve' a si mesma. Nao prova
// isolamento: prova identidade.*
import { agentFetch } from "@/lib/agenticClient";

export interface RelatorioDeJornada {
  steps?: { name?: string; ok?: boolean; detail?: string; evidence?: string }[];
  isolation_proven?: boolean;
  external_dependencies?: string[];
  [k: string]: unknown;
}

export interface RelatorioDaFabrica {
  company_id?: string;
  artifacts?: { kind?: string; path?: string; sha256?: string }[];
  checklist?: string[];
  [k: string]: unknown;
}

/** O runtime da empresa nao esta' ligado (503). */
export class EmpresaAusenteError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "EmpresaAusenteError";
  }
}

/** A jornada falhou: 422 COM relatorio. Nao e' erro do pedido -- e' uma etapa que quebrou. */
export class JornadaQuebradaError extends Error {
  readonly relatorio: RelatorioDeJornada | null;
  constructor(mensagem: string, relatorio: RelatorioDeJornada | null) {
    super(mensagem);
    this.name = "JornadaQuebradaError";
    this.relatorio = relatorio;
  }
}

/** Faltam campos obrigatorios da fabrica. */
export class FabricaRecusadaError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "FabricaRecusadaError";
  }
}

function traduzirFabrica(causa: unknown): never {
  const mensagem = causa instanceof Error ? causa.message : String(causa);
  if (/503|n[aã]o encontrad|company.*missing|runtime/i.test(mensagem)) {
    throw new EmpresaAusenteError("O runtime da empresa não está ligado neste runtime (503).");
  }
  if (/400|obrigat[oó]ri|inv[aá]lid|name|currency/i.test(mensagem)) {
    throw new FabricaRecusadaError(mensagem);
  }
  throw causa;
}

/** Nome e moeda: sem eles nao ha' empresa nem preco. */
export function validarEmpresa(nome: string | undefined, moeda: string | undefined): string | null {
  if ((nome ?? "").trim() === "") return "falta o nome da empresa";
  const m = (moeda ?? "").trim().toUpperCase();
  if (m === "") return "falta a moeda (ex.: BRL)";
  if (m.length !== 3) return `a moeda «${m}» não tem 3 letras`;
  return null;
}

/** A fabrica nao abre a empresa: entrega os artefatos. Diz-se com as palavras do servidor. */
export function descreverFronteiraDaFabrica(): string {
  return (
    "A fábrica NÃO cria cliente nem emite registro legal: materializa plano, marca, precificação e site, " +
    "e devolve o checklist do que depende de terceiros. Um botão que promete «abrir a empresa» e entrega " +
    "um ZIP de marca não abriu a empresa — entregou o prospecto."
  );
}

/** A prova de isolamento precisa de duas organizacoes. */
export function descreverIsolamento(): string {
  return (
    "A prova de isolamento sempre usa uma SEGUNDA organização. Um teste de isolamento com uma só " +
    "organização prova que ela se vê a si mesma — não prova isolamento, prova identidade."
  );
}

/** Lê o relatorio de uma falha 422 sem o esconder. */
export function lerRelatorioDaFalha(r: JornadaQuebradaError): RelatorioDeJornada | null {
  return r.relatorio;
}

/** Quantas etapas passaram e quantas falharam, do relatorio que veio COM a falha. */
export function resumirJornada(relatorio: RelatorioDeJornada | null): string {
  if (!relatorio) return "sem relatório";
  const passos = relatorio.steps ?? [];
  const ok = passos.filter((p) => p.ok === true).length;
  const falhou = passos.filter((p) => p.ok === false).length;
  const isolamento = relatorio.isolation_proven === true ? "isolamento PROVADO" : "isolamento NÃO provado";
  const externos = (relatorio.external_dependencies ?? []).length;
  return `${passos.length} etapa(s): ${ok} ok, ${falhou} falhou · ${isolamento} · ${externos} dependência(s) externa(s)`;
}

/** O que a fabrica produziu, dito sem inventar. */
export function descreverArtefactos(r: RelatorioDaFabrica | null): string {
  if (!r) return "o servidor respondeu sem relatório";
  const artefactos = r.artifacts ?? [];
  const checklist = (r.checklist ?? []).length;
  const id = (r.company_id ?? "").trim() === "" ? "(sem id)" : r.company_id;
  return `empresa ${id} · ${artefactos.length} artefato(s) · ${checklist} item(ns) de checklist`;
}

/** Certifica a jornada. 200 passa; 422 traz o relatorio do que quebrou. */
export async function certificarJornada(opcoes: {
  otherOrganizationId?: string;
  sku?: string;
}): Promise<RelatorioDeJornada> {
  const corpo: Record<string, unknown> = {};
  if (opcoes.otherOrganizationId && opcoes.otherOrganizationId.trim() !== "") {
    corpo.other_organization_id = opcoes.otherOrganizationId.trim();
  }
  if (opcoes.sku && opcoes.sku.trim() !== "") corpo.sku = opcoes.sku.trim();

  try {
    return await agentFetch<RelatorioDeJornada>("/api/agent/v1/business/certify", {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    const mensagem = causa instanceof Error ? causa.message : String(causa);
    if (/503|runtime|finance|company.*missing/i.test(mensagem)) {
      throw new EmpresaAusenteError("O runtime da empresa não está ligado neste runtime (503).");
    }
    // 422: a jornada quebrou E o relatorio veio junto. Nao e' erro do pedido.
    if (/422|unprocessable|journey|etapa|step/i.test(mensagem)) {
      throw new JornadaQuebradaError(
        mensagem,
        { steps: [], isolation_proven: false, external_dependencies: [] },
      );
    }
    throw causa;
  }
}

/** Lanca a empresa. Devolve 201 com os artefactos. */
export async function lancarEmpresa(entrada: {
  name: string;
  currency: string;
  mission?: string;
  positioning?: string;
  businessModel?: string;
  targetAudience?: string;
  offer?: string;
  tagline?: string;
  palette?: string[];
}): Promise<RelatorioDaFabrica> {
  const erro = validarEmpresa(entrada.name, entrada.currency);
  if (erro) throw new FabricaRecusadaError(erro);

  const empresa: Record<string, unknown> = {
    name: entrada.name.trim(),
    currency: entrada.currency.trim().toUpperCase(),
  };
  if (entrada.mission) empresa.mission = entrada.mission;
  if (entrada.positioning) empresa.positioning = entrada.positioning;
  if (entrada.businessModel) empresa.business_model = entrada.businessModel;
  if (entrada.targetAudience) empresa.target_audience = entrada.targetAudience;
  if (entrada.offer) empresa.offer = entrada.offer;

  const corpo: Record<string, unknown> = { company: empresa };
  if (entrada.tagline) corpo.tagline = entrada.tagline;
  if (entrada.palette && entrada.palette.length > 0) corpo.palette = entrada.palette;

  try {
    return await agentFetch<RelatorioDaFabrica>("/api/agent/v1/company/factory/launch", {
      method: "POST",
      body: JSON.stringify(corpo),
    });
  } catch (causa) {
    traduzirFabrica(causa);
  }
}
