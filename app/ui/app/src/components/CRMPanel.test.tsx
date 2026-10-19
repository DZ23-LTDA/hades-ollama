// Testes do painel de CRM.
//
// O caso que mais importa é a PII: `ana.silva@exemplo.com` e `+351912345678` NÃO
// podem aparecer inteiros numa lista por omissão. Um teste que só verificasse
// "renderiza a tabela" passaria com os dados pessoais expostos.
import { act, create } from "react-test-renderer";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/agenticClient", () => ({ agentFetch: vi.fn() }));

import { agentFetch } from "@/lib/agenticClient";
import { CRMPanel } from "@/components/CRMPanel";

const mockAgentFetch = vi.mocked(agentFetch);

async function montar() {
  let renderer!: ReturnType<typeof create>;
  await act(async () => {
    renderer = create(<CRMPanel />);
  });
  return renderer;
}

async function agir(accao: () => unknown) {
  await act(async () => {
    await accao();
  });
}

function juntarTexto(nodo: unknown): string {
  if (typeof nodo === "string") return nodo;
  if (typeof nodo === "number") return String(nodo);
  if (Array.isArray(nodo)) return nodo.map(juntarTexto).join("");
  if (nodo === null || typeof nodo !== "object") return "";
  return juntarTexto((nodo as { children?: unknown }).children);
}

function texto(renderer: ReturnType<typeof create>, id: string): string {
  const arvore = renderer.toJSON();
  const encontrados: string[] = [];
  const visitar = (nodo: unknown) => {
    if (Array.isArray(nodo)) { nodo.forEach(visitar); return; }
    if (nodo === null || typeof nodo !== "object") return;
    const props = (nodo as { props?: Record<string, unknown> }).props;
    if (props && props["data-testid"] === id) encontrados.push(juntarTexto(nodo));
    visitar((nodo as { children?: unknown }).children);
  };
  visitar(arvore);
  return encontrados.join(" ");
}

function porTestId(renderer: ReturnType<typeof create>, id: string) {
  return renderer.root.findAll((nodo) => nodo.props["data-testid"] === id);
}

const CONTACTOS = [
  {
    id: "c1", organization_id: "o1", name: "Ana Silva",
    phone: "+351912345678", email: "ana.silva@exemplo.com",
    created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z",
  },
  {
    id: "c2", organization_id: "o1", name: "Bruno Costa",
    created_at: "2026-10-19T00:00:00Z", updated_at: "2026-10-19T00:00:00Z",
  },
];

const FUNIL = {
  organization_id: "o1", currency: "BRL",
  stages: [
    { stage: "qualificacao", count: 2, total_cents: 125000 },
    { stage: "proposta", count: 1, total_cents: 50000 },
  ],
  open_count: 3, open_value_cents: 175000, won_value_cents: 250000, lost_count: 1,
};

function responder() {
  mockAgentFetch
    .mockResolvedValueOnce({ contacts: CONTACTOS, count: CONTACTOS.length })
    .mockResolvedValueOnce(FUNIL);
}

afterEach(() => {
  mockAgentFetch.mockReset();
});

describe("CRMPanel", () => {
  it("PII MASCARADA por omissão: email e telefone completos NÃO aparecem", async () => {
    responder();

    const renderer = await montar();

    const tabela = texto(renderer, "crm-contactos");
    // Os valores completos NAO podem estar na lista.
    expect(tabela).not.toContain("ana.silva@exemplo.com");
    expect(tabela).not.toContain("+351912345678");
    // As versoes mascaradas estao.
    expect(tabela).toContain("a•••@exemplo.com");
    expect(tabela).toContain("+351••••••678");
    // E o painel diz que mascarou.
    expect(texto(renderer, "crm-aviso-pii")).toContain("mascarados");
  });

  it("com 'Mostrar dados pessoais' ligado, mostra os valores completos", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ contacts: CONTACTOS, count: 2 })
      .mockResolvedValueOnce(FUNIL)
      .mockResolvedValueOnce({ contacts: CONTACTOS, count: 2 })
      .mockResolvedValueOnce(FUNIL);

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "crm-mostrar-pii")[0].props.onChange({ target: { checked: true } }),
    );
    await agir(() => porTestId(renderer, "crm-actualizar")[0].props.onClick());

    const tabela = texto(renderer, "crm-contactos");
    expect(tabela).toContain("ana.silva@exemplo.com");
    expect(tabela).toContain("+351912345678");
    // Sem aviso quando esta' a mostrar.
    expect(porTestId(renderer, "crm-aviso-pii")).toHaveLength(0);
  });

  it("mostra o funil com os valores em reais, não em centavos", async () => {
    responder();

    const renderer = await montar();

    const funil = texto(renderer, "crm-funil");
    expect(funil).toContain("R$ 1.750,00");
    expect(funil).toContain("R$ 2.500,00");
    expect(funil).not.toContain("175000");
    expect(funil).toContain("3 oportunidade(s) aberta(s)");
  });

  it("mostra as etapas do funil com etiqueta legível", async () => {
    responder();

    const renderer = await montar();

    const etapas = texto(renderer, "crm-etapas");
    expect(etapas).toContain("qualificacao: 2");
    expect(etapas).toContain("R$ 1.250,00");
    expect(etapas).toContain("proposta: 1");
  });

  it("o 503 aparece como CRM AUSENTE, não como lista vazia", async () => {
    const erro = new Error("CRM não está disponível no runtime");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    const semCrm = texto(renderer, "crm-sem-crm");
    expect(semCrm).toContain("não está ligado");
    expect(semCrm).toContain("não é «não há contactos»");
    expect(porTestId(renderer, "crm-erro")).toHaveLength(0);
  });

  it("CRM ligado com zero contactos diz isso, e é diferente de ausente", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ contacts: [], count: 0 })
      .mockResolvedValueOnce({ ...FUNIL, open_count: 0, open_value_cents: 0, won_value_cents: 0, lost_count: 0, stages: [] });

    const renderer = await montar();

    expect(texto(renderer, "crm-sem-contactos")).toContain("não há contactos");
    expect(porTestId(renderer, "crm-sem-crm")).toHaveLength(0);
  });

  it("contacto sem telefone nem email mostra travessão, não string vazia", async () => {
    responder();

    const renderer = await montar();

    const linha = texto(renderer, "crm-contacto-c2");
    expect(linha).toContain("Bruno Costa");
    expect(linha).toContain("—");
  });

  it("a procura leva o filtro na query", async () => {
    mockAgentFetch
      .mockResolvedValueOnce({ contacts: [], count: 0 })
      .mockResolvedValueOnce(FUNIL)
      .mockResolvedValueOnce({ contacts: [], count: 0 })
      .mockResolvedValueOnce(FUNIL);

    const renderer = await montar();
    await agir(() =>
      porTestId(renderer, "crm-filtro")[0].props.onChange({ target: { value: "ana" } }),
    );
    await agir(() => porTestId(renderer, "crm-procurar")[0].props.onClick());

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string);
    expect(urls).toContain("/api/agent/v1/crm/contacts?q=ana");
  });

  it("o estado de consentimento diz que é lido do registro, não do contacto", async () => {
    responder();

    const renderer = await montar();

    const consentimento = texto(renderer, "crm-consentimento");
    expect(consentimento).toContain("registro de consentimento");
    expect(consentimento).toContain("não do contacto");
  });

  it("um erro que não é 503 continua a ser erro", async () => {
    const erro = new Error("organizacao invalida");
    mockAgentFetch.mockRejectedValueOnce(erro).mockRejectedValueOnce(erro);

    const renderer = await montar();

    expect(texto(renderer, "crm-erro")).toContain("organizacao invalida");
    expect(porTestId(renderer, "crm-sem-crm")).toHaveLength(0);
  });

  it("as duas chamadas usam o prefixo explícito e são GET", async () => {
    responder();

    await montar();

    const urls = mockAgentFetch.mock.calls.map((c) => c[0] as string).sort();
    expect(urls).toEqual([
      "/api/agent/v1/crm/contacts",
      "/api/agent/v1/crm/pipeline",
    ]);
    for (const chamada of mockAgentFetch.mock.calls) {
      const init = chamada[1] as RequestInit | undefined;
      expect(init?.method ?? "GET").toBe("GET");
    }
  });
});
