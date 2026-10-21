// Testes da mídia (OCR e tom).
//
// Discriminantes: NÃO corre ao montar; o caminho é RELATIVO ao workspace (o painel não reimplementa
// `containedPath`); sem missão não há fronteira; 502 é a dependência a falhar e NÃO é erro do pedido;
// os dois devolvem 201 e criam um Artifact.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { MediaPanel } from "@/components/MediaPanel";
import {
  descreverFronteiraDoCaminho,
  descreverResultado,
  validarCaminhoDeEntrada,
  validarMissao,
  validarTom,
} from "@/lib/mediaClient";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<MediaPanel />);
  });
  return renderer;
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join("");
  if (nodo === null || typeof nodo !== "object") return "";
  return juntarTexto((nodo as { children?: unknown }).children);
}

function texto(renderer: ReturnType<typeof create>, id: string): string {
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(renderer.toJSON());
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

async function preencher(renderer: ReturnType<typeof create>, campos: Record<string, string>) {
  for (const [id, valor] of Object.entries(campos)) {
    const el = porTestId(renderer, `midia-${id}`);
    await act(async () => {
      el[0].props.onChange({ target: { value: valor } });
    });
  }
}

async function confirmar(renderer: ReturnType<typeof create>) {
  const cs = porTestId(renderer, "midia-confirmar");
  await act(async () => {
    cs[0].props.onChange({ target: { checked: true } });
  });
}

async function clicar(renderer: ReturnType<typeof create>, botao: string) {
  const bs = porTestId(renderer, `midia-${botao}`);
  await act(async () => {
    bs[0].props.onClick();
  });
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("funcoes da midia", () => {
  it("sem missao nao ha' fronteira, e o texto di-lo", () => {
    const t = validarMissao("");
    expect(t).toContain("mission_id");
    expect(t).toContain("workspace");
  });

  it("caminho absoluto e' recusado", () => {
    expect(validarCaminhoDeEntrada("/etc/passwd")).toContain("relativo ao workspace");
    expect(validarCaminhoDeEntrada("C:\\Windows\\x.png")).toContain("relativo ao workspace");
  });

  it("caminho com `..` e' recusado", () => {
    const t = validarCaminhoDeEntrada("../../etc/passwd");
    expect(t).toContain("`..`");
    expect(t).toContain("servidor");
  });

  it("um caminho relativo simples e' aceite", () => {
    expect(validarCaminhoDeEntrada("docs/foto.png")).toBeNull();
    expect(validarCaminhoDeEntrada("foto.png")).toBeNull();
  });

  it("caminho vazio e' recusado com o nome do campo", () => {
    expect(validarCaminhoDeEntrada("")).toContain("input_path");
  });

  it("a fronteira diz que o servidor resolve o caminho, e o painel NAO reimplementa", () => {
    const t = descreverFronteiraDoCaminho();
    expect(t).toContain("DENTRO do workspace");
    expect(t).toContain("não reimplementa");
    expect(t).toContain("segunda fronteira");
  });

  it("frequencia zero ou negativa e' recusada", () => {
    expect(validarTom(0, 500)).toContain("positiva");
    expect(validarTom(-100, 500)).toContain("positiva");
  });

  it("frequencia acima de 20 kHz nao e' audivel", () => {
    expect(validarTom(25000, 500)).toContain("não é audível");
  });

  it("duracao tem de ser inteira em milissegundos", () => {
    expect(validarTom(440, 1.5)).toContain("inteira em milissegundos");
    expect(validarTom(440, 500)).toBeNull();
  });

  it("o resultado diz o artificio, nao 'correu bem'", () => {
    const t = descreverResultado({ artifact: { id: "art-1", kind: "ocr_text" } } as never);
    expect(t).toContain("art-1");
    expect(t).toContain("ocr_text");
  });

  it("sem artefato, o resultado NAO finge sucesso", () => {
    expect(descreverResultado({} as never)).toContain("sem artefato");
  });
});

describe("MediaPanel", () => {
  it("NAO corre ao montar -- sao duas escritas", async () => {
    const r = await montar();

    expect(mockAgentFetch).not.toHaveBeenCalled();
    expect(texto(r, "midia-pronto")).toContain("criam um artefacto");
  });

  it("sem confirmacao, recusa correr", async () => {
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "docs/foto.png" });
    await clicar(r, "ocr");

    expect(texto(r, "midia-erro")).toContain("Confirme antes");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("sem missao, recusa antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { caminho: "docs/foto.png" });
    await confirmar(r);
    await clicar(r, "ocr");

    expect(texto(r, "midia-recusado")).toContain("mission_id");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("caminho com `..` e' recusado antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "../../etc/passwd" });
    await confirmar(r);
    await clicar(r, "ocr");

    expect(texto(r, "midia-recusado")).toContain("`..`");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("o OCR vai para /media/ocr e leva mission_id e input_path relativos", async () => {
    mockAgentFetch.mockResolvedValueOnce({ artifact: { id: "art-1", kind: "ocr_text" } } as never);
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "docs/foto.png", idioma: "pt" });
    await confirmar(r);
    await clicar(r, "ocr");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    const init = mockAgentFetch.mock.calls[0][1] as { method?: string; body?: string };
    expect(caminho).toBe("/api/agent/v1/media/ocr");
    expect(init.method).toBe("POST");
    const corpo = JSON.parse(init.body ?? "{}");
    expect(corpo.mission_id).toBe("m-1");
    expect(corpo.input_path).toBe("docs/foto.png");
    expect(corpo.language).toBe("pt");
  });

  it("o resultado diz 201 e o artefacto -- nao 'correu bem'", async () => {
    mockAgentFetch.mockResolvedValueOnce({ artifact: { id: "art-1", kind: "ocr_text" } } as never);
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "foto.png" });
    await confirmar(r);
    await clicar(r, "ocr");

    const t = texto(r, "midia-resultado");
    expect(t).toContain("201");
    expect(t).toContain("art-1");
  });

  it("502 e' a DEPENDENCIA a falhar, nao erro do pedido", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("502: motor de OCR falhou"));
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "foto.png" });
    await confirmar(r);
    await clicar(r, "ocr");

    const t = texto(r, "midia-indisponivel");
    expect(t).toContain("não respondeu");
    expect(t).toContain("502");
    // Nao pode cair em erro generico nem em recusa.
    expect(porTestId(r, "midia-erro")).toHaveLength(0);
    expect(porTestId(r, "midia-recusado")).toHaveLength(0);
  });

  it("o tom vai para /media/tone com frequencia e duracao", async () => {
    mockAgentFetch.mockResolvedValueOnce({ artifact: { id: "art-2", kind: "audio" } } as never);
    const r = await montar();
    await preencher(r, { missao: "m-1", frequencia: "440", duracao: "500" });
    await confirmar(r);
    await clicar(r, "tom");

    const caminho = mockAgentFetch.mock.calls[0][0] as string;
    expect(caminho).toBe("/api/agent/v1/media/tone");
    const corpo = JSON.parse((mockAgentFetch.mock.calls[0][1] as { body?: string }).body ?? "{}");
    expect(corpo.mission_id).toBe("m-1");
    expect(corpo.frequency).toBe(440);
    expect(corpo.duration_ms).toBe(500);
  });

  it("frequencia invalida e' recusada antes de chamar", async () => {
    const r = await montar();
    await preencher(r, { missao: "m-1", frequencia: "0", duracao: "500" });
    await confirmar(r);
    await clicar(r, "tom");

    expect(texto(r, "midia-recusado")).toContain("positiva");
    expect(mockAgentFetch).not.toHaveBeenCalled();
  });

  it("o 503 aparece como runtime de midia ausente", async () => {
    mockAgentFetch.mockRejectedValueOnce(new Error("runtime de midia nao esta' ligado"));
    const r = await montar();
    await preencher(r, { missao: "m-1", caminho: "foto.png" });
    await confirmar(r);
    await clicar(r, "ocr");

    expect(texto(r, "midia-ausente")).toContain("não está ligado");
  });

  it("a fronteira diz que o servidor resolve o caminho e que 201 nao e' 200", async () => {
    const r = await montar();

    const t = texto(r, "midia-fronteira");
    expect(t).toContain("DENTRO do workspace");
    expect(t).toContain("201");
  });
});
