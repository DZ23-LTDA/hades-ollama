// Instalação de pacote de plugin -- `POST /plugins/bundle` (201 CREATED).
//
// A FRONTEIRA E' RESOLVIDA PELO SERVIDOR, E TEM TRES PASSOS
//
//     root, _      := filepath.Abs(allowedDir)      // OLLAMA_AGENT_PLUGIN_BUNDLE_DIR
//     candidate, _ := filepath.Abs(requestedPath)
//     candidate, _  = filepath.EvalSymlinks(candidate)   // ANTES de comparar
//     root, _       = filepath.EvalSymlinks(root)
//     relative, _   := filepath.Rel(root, candidate)
//     if relative == ".." || HasPrefix(relative, ".."+sep) { 403 }
//
//   *Um caminho que passa por um symlink nao e' o caminho que se ve'. Sem o EvalSymlinks, uma pasta dentro
//   do diretorio permitido que aponta para fora passaria na comparacao de texto e entregaria o ficheiro de
//   fora. O `Abs` normaliza; o `EvalSymlinks` desmente.*
//
// O PAINEL NAO REIMPLEMENTA ESTA FRONTEIRA
//
//   *Uma segunda verdade sobre o que esta' dentro seria uma segunda fronteira, e a mais fraca e' a que
//   fica. O painel mostra o `bundle_path` como o utilizador o escreveu e deixa o servidor decidir.*
//
// 403 NAO E' UMA COISA SO'
//
// Sem a variavel de ambiente, o subsistema esta' DESLIGADO -- nao e' um erro do utilizador. Fora do
// diretorio, e' um pedido recusado. A mesma resposta HTTP, duas causas, e o painel tem de as distinguir
// pelo texto do erro, declarando quando nao consegue.
import { agentFetch } from "@/lib/agenticClient";

export interface ManifestoDePlugin {
  id?: string;
  version?: string;
  name?: string;
  kind?: string;
  description?: string;
  scopes?: string[];
  dependencies?: string[];
  min_engine?: string;
  license?: string;
  homepage?: string;
  signing_key_id?: string;
  content_sha256?: string;
  signature?: string;
  [k: string]: unknown;
}

export interface ArtefactoAssinado {
  artifact: string;
  sha256: string;
  signing_key_id: string;
  signature: string;
}

export interface ResultadoDeInstalacao {
  [k: string]: unknown;
}

/** O subsistema de pacotes esta' desligado neste runtime (403 sem env var). */
export class PacotesDesligadosError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "PacotesDesligadosError";
  }
}

/** O caminho foi recusado: fora do diretorio permitido, ou symlink a apontar para fora. */
export class CaminhoRecusadoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "CaminhoRecusadoError";
  }
}

/** Assinatura ausente ou invalida (403). */
export class AssinaturaRecusadaError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "AssinaturaRecusadaError";
  }
}

/** Manifesto invalido, capacidade desconhecida (400) ou org ausente (403). */
export class PacoteRecusadoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "PacoteRecusadoError";
  }
}

/** Ja' existe uma instalacao com esta versao (409). */
export class VersaoEmConflitoError extends Error {
  constructor(m: string) {
    super(m);
    this.name = "VersaoEmConflitoError";
  }
}

function traduzir(causa: unknown): never {
  const m = causa instanceof Error ? causa.message : String(causa);

  if (/409|conflict|version/i.test(m)) throw new VersaoEmConflitoError(m);
  if (/signature|assinatura|signing[_ ]?key/i.test(m)) throw new AssinaturaRecusadaError(m);
  // "not available" = subsistema desligado -- NAO e' o mesmo 403 que "fora do diretorio".
  if (/not available|indispon[ií]vel|n[aã]o est[aá] dispon/i.test(m)) throw new PacotesDesligadosError(m);
  if (/inside the allowed|allowed directory|fora do diret[oó]rio|407|bundl/i.test(m)) {
    throw new CaminhoRecusadoError(m);
  }
  if (/400|manifest|capabilit|organization/i.test(m)) throw new PacoteRecusadoError(m);
  if (/403|forbidden/i.test(m)) throw new PacoteRecusadoError(m);
  throw causa;
}

/** O manifesto tem de ter id, version e kind. Nao validamos o schema do servidor: validamos o que falta. */
export function validarManifesto(m: ManifestoDePlugin | undefined): string | null {
  if (!m) return "falta o manifesto do plugin";
  if (!(m.id ?? "").trim()) return "o manifesto não declara `id`";
  if (!(m.version ?? "").trim()) return "o manifesto não declara `version`";
  if (!(m.kind ?? "").trim()) return "o manifesto não declara `kind`";
  return null;
}

/** A assinatura tem de ser um artefacto com sha256 e signature. */
export function validarAssinatura(a: ArtefactoAssinado | undefined): string | null {
  if (!a) return "falta a assinatura do pacote";
  if (!(a.sha256 ?? "").trim()) return "a assinatura não declara `sha256`";
  if (!(a.signature ?? "").trim()) return "a assinatura não declara `signature`";
  if (!/^[0-9a-f]{64}$/i.test(a.sha256.trim())) {
    return "o `sha256` não tem 64 dígitos hexadecimais: não é um digest, é um lapso";
  }
  return null;
}

/** O caminho e' do servidor; aqui so' se recusa o obvio. */
export function validarCaminho(caminho: string): string | null {
  const c = (caminho ?? "").trim();
  if (c === "") return "falta o `bundle_path`";
  return null;
}

/** O 403, dito pela causa certa -- sem confundir desligado com recusado. */
export function descreverRecusaDoPacote(causa: unknown): string {
  if (causa instanceof PacotesDesligadosError) {
    return (
      "O subsistema de pacotes está DESLIGADO neste runtime: falta OLLAMA_AGENT_PLUGIN_BUNDLE_DIR. " +
      "Isto não é um erro do pedido — é uma capacidade ausente."
    );
  }
  if (causa instanceof CaminhoRecusadoError) {
    return (
      "O servidor recusou o caminho: o pacote tem de viver dentro do diretório permitido, depois de " +
      "resolvidos os symlinks. Um caminho que passa por um symlink não é o caminho que se vê."
    );
  }
  if (causa instanceof AssinaturaRecusadaError) {
    return "O servidor recusou a assinatura: o pacote não está assinado por uma chave autorizada.";
  }
  if (causa instanceof VersaoEmConflitoError) {
    return "Já existe uma instalação com esta versão: desinstale ou suba a versão — não se instala por cima.";
  }
  if (causa instanceof PacoteRecusadoError) {
    return "O servidor recusou o manifesto: `id`, `version`, `kind` e as capacidades declaradas têm de ser válidos.";
  }
  return causa instanceof Error ? causa.message : "Não foi possível instalar o pacote.";
}

/** O que a instalacao devolveu, sem inventar campo. */
export function descreverInstalacao(r: ResultadoDeInstalacao | null): string {
  if (!r) return "o servidor respondeu sem instalação";
  const partes: string[] = ["201 CREATED"];
  const reg = (r as { plugin?: { id?: string; version?: string }; id?: string }).plugin;
  const id = reg?.id ?? (r as { id?: string }).id;
  const versao = reg?.version ?? (r as { version?: string }).version;
  if (id) partes.push(String(id));
  if (versao) partes.push("v" + String(versao));
  return partes.join(" · ");
}

/** O `sha256` que o SERVIDOR verificou, mostrado como o cliente o enviou. */
export function descreverDigest(a: ArtefactoAssinado | undefined): string {
  const s = (a?.sha256 ?? "").trim();
  if (!s) return "sem digest";
  return "sha256 " + s.slice(0, 12) + "…" + s.slice(-6);
}

export async function instalarPacote(
  manifesto: ManifestoDePlugin,
  caminho: string,
  assinatura: ArtefactoAssinado,
): Promise<ResultadoDeInstalacao> {
  const erroManifesto = validarManifesto(manifesto);
  if (erroManifesto) throw new PacoteRecusadoError(erroManifesto);
  const erroCaminho = validarCaminho(caminho);
  if (erroCaminho) throw new CaminhoRecusadoError(erroCaminho);
  const erroAssinatura = validarAssinatura(assinatura);
  if (erroAssinatura) throw new AssinaturaRecusadaError(erroAssinatura);

  try {
    return await agentFetch<ResultadoDeInstalacao>("/api/agent/v1/plugins/bundle", {
      method: "POST",
      body: JSON.stringify({
        manifest: manifesto,
        bundle_path: caminho.trim(),
        signature: assinatura,
      }),
    });
  } catch (causa) {
    traduzir(causa);
  }
}
