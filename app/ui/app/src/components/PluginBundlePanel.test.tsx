// Testes da instalação de pacote de plugin.
//
// Discriminantes: 403 «desligado» NÃO é 403 «caminho recusado»; o painel NÃO reimplementa a fronteira do
// caminho; o sha256 tem de ser um digest; o corpo leva manifest + bundle_path + signature.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { PluginBundlePanel } from "@/components/PluginBundlePanel";
import {
  descreverDigest,
  descreverInstalacao,
  descreverRecusaDoPacote,
  PacotesDesligadosError,
  CaminhoRecusadoError,
  validarAssinatura,
  validarCaminho,
  validarManifesto,
} from "@/lib/pluginBundleClient";

const mockAgentFetch = vi.mocked(agentFetch);
const SHA = "a".repeat(64);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<PluginBundlePanel />);
  });
  return renderer;
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join(" ");
  if (nodo === null || typeof nodo !== "object") return "";
  return juntarTexto((nodo as { children?: unknown }).children);
}

function texto(renderer: ReturnType<typeof create>, id: string): string {
  const out: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) out.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(renderer.toJSON());
  return out.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((n) => n.props["data-testid"] === id);
}

async function escrever(renderer: ReturnType<typeof create>, id: string, valor: string) {
  await act(async () => {
    porTestId(renderer, `plugin-bundle-${id}`)[0].props.onChange({ target: { value: valor } });
  });
}

async function clicar(renderer: ReturnType<typeof create>) {
  await act(async () => {
    porTestId(renderer, "plugin-bundle-instalar")[0].props.onClick();
  });
}

/** Preenche tudo o que e' obrigatorio. */
async function preencher(renderer: ReturnType<typeof create>) {
  await escrever(renderer, "id", "meu-plugin");
  await escrever(renderer, "versao", "1.0.0");
  await escrever(renderer, "caminho", "/srv/bundles/meu-plugin-1.0.0.zip");
  await escrever(renderer, "sha256", SHA);
  await escrever(renderer, "assinatura", "sig-abc");
  await escrever(renderer, "chave", "key-1");
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes do pacote", () => {
  it("manifesto sem id e' recusado com o nome do campo", () => {
    expect(validarManifesto({ version: "1", kind: "tool" })).toContain("`id`");
  });

  it("manifesto sem version e' recusado", () => {
    expect(validarManifesto({ id: "x", kind: "tool" })).toContain("`version`");
  });

  it("manifesto sem kind e' recusado", () => {
    expect(validarManifesto({ id: "x", version: "1" })).toContain("`kind`");
  });

  it("caminho vazio e' recusado", () => {
    expect(validarCaminho("  ")).toContain("bundle_path");
  });

  it("sha256 que nao tem 64 hex e' recusado: nao e' um digest", () => {
    expect(validarAssinatura({ artifact: "x", sha256: "abc", signing_key_id: "k", signature: "s" })).toContain(
      "64 dígitos",
    );
  });

  it("assinatura sem signature e' recusada", () => {
    expect(validarAssinatura({ artifact: "x", sha256: SHA, signing_key_id: "k", signature: "" })).toContain(
      "`signature`",
    );
  });

  it("403 DESLIGADO e 403 CAMINHO RECUSADO sao descritos de forma DIFERENTE", () => {
    const desligado = descreverRecusaDoPacote(new PacotesDesligadosError("403: plugin bundle is not available"));
    expect(desligado).toContain("DESLIGADO");
    expect(desligado).toContain("OLLAMA_AGENT_PLUGIN_BUNDLE_DIR");
    expect(desligado).toContain("não é um erro do pedido");

    const recusado = descreverRecusaDoPacote(new CaminhoRecusadoError("403: bundle must live inside"));
    expect(recusado).toContain("recusou o caminho");
    expect(recusado).toContain("symlink");
    // E nao se confundem.
    expect(recusado).not.toContain("OLLAMA_AGENT_PLUGIN_BUNDLE_DIR");
    expect(desligado).not.toContain("depois de resolvidos os symlinks");
  });

  it("o digest mostra o inicio e o fim, nao os 64 caracteres", () => {
    const d = descreverDigest({ artifact: "x", sha256: SHA, signing_key_id: "k", signature: "s" });
    expect(d).toContain("sha256");
    expect(d).toContain("aaaaaaaaaaaa");
    expect(d).toContain("aaaaaa");
    expect(d.length).toBeLessThan(40);
  });

  it("instalacao sem campos devolve so' o status, sem inventar id", () => {
    expect(descreverInstalacao({})).toContain("201");
    expect(descreverInstalacao({})).not.toContain("v");
  });
});

describe("PluginBundlePanel", () => {
  it("nao instala ao montar e mostra a fronteira", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "plugin-bundle-pronto")).toContain("Nada instalado");
    expect(texto(r, "plugin-bundle-fronteira")).toContain("EvalSymlinks");
  });

  it("tudo vazio e' recusado antes de chamar, dizendo os dois campos", async () => {
    const r = await montar();
    await clicar(r);

    // O codigo para no PRIMEIRO problema -- e' o `id`. A `version` so' aparece depois de o `id` existir.
    expect(texto(r, "plugin-bundle-recusado")).toContain("`id`");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("vai para /plugins/bundle com manifest, bundle_path e signature", async () => {
    mockAgentFetch.mockResolvedValueOnce({ plugin: { id: "meu-plugin", version: "1.0.0" } } as never);
    const r = await montar();
    await preencher(r);
    await clicar(r);

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/plugins/bundle");
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.manifest.id).toBe("meu-plugin");
    expect(corpo.manifest.version).toBe("1.0.0");
    expect(corpo.bundle_path).toBe("/srv/bundles/meu-plugin-1.0.0.zip");
    expect(corpo.signature.sha256).toBe(SHA);
    expect(corpo.signature.signature).toBe("sig-abc");
  });

  it("201 mostra a instalacao e o digest", async () => {
    mockAgentFetch.mockResolvedValueOnce({ plugin: { id: "meu-plugin", version: "1.0.0" } } as never);
    const r = await montar();
    await preencher(r);
    await clicar(r);

    const t = texto(r, "plugin-bundle-instalado");
    expect(t).toContain("201");
    expect(t).toContain("meu-plugin");
    expect(t).toContain("1.0.0");
    expect(texto(r, "plugin-bundle-digest")).toContain("sha256");
  });

  it("403 'not available' aparece como DESLIGADO, nao como recusa do pedido", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("403: plugin bundle is not available"));
    const r = await montar();
    await preencher(r);
    await clicar(r);

    const t = texto(r, "plugin-bundle-desligado");
    expect(t).toContain("DESLIGADO");
    expect(t).toContain("OLLAMA_AGENT_PLUGIN_BUNDLE_DIR");
    // E NAO aparece no estado de recusa.
    expect(porTestId(r, "plugin-bundle-recusado")).toHaveLength(0);
  });

  it("403 'inside the allowed directory' aparece como caminho recusado", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("403: bundle must live inside the allowed directory"));
    const r = await montar();
    await preencher(r);
    await clicar(r);

    const t = texto(r, "plugin-bundle-recusado");
    expect(t).toContain("recusou o caminho");
    expect(t).toContain("symlink");
    expect(porTestId(r, "plugin-bundle-desligado")).toHaveLength(0);
  });

  it("409 aparece como conflito de versao, nao como recusa generica", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("409: plugin version conflict"));
    const r = await montar();
    await preencher(r);
    await clicar(r);

    expect(texto(r, "plugin-bundle-conflito")).toContain("Já existe uma instalação");
    expect(porTestId(r, "plugin-bundle-recusado")).toHaveLength(0);
  });

  it("assinatura invalida e' recusada antes de chamar", async () => {
    const r = await montar();
    await escrever(r, "id", "x");
    await escrever(r, "versao", "1");
    await escrever(r, "caminho", "/a/b.zip");
    await escrever(r, "sha256", "curto");
    await escrever(r, "assinatura", "sig");   // sem isto, a recusa e' a `signature` ausente
    await clicar(r);

    expect(texto(r, "plugin-bundle-recusado")).toContain("64 dígitos");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });
});
