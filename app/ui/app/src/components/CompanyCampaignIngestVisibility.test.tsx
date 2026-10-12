import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CompanyCampaignSchedulingPanel } from "./CompanyCampaignSchedulingPanel";
import type { AgentCompany } from "@/lib/agenticClient";

vi.mock("@/lib/agenticClient", () => ({
  getCompanyCampaignReport: vi.fn(),
  getMarketplaceChannels: vi.fn(),
  recordCompanyCampaignMetrics: vi.fn(),
  scheduleCompanyCampaign: vi.fn(),
  syncMarketplace: vi.fn(),
}));

import { getCompanyCampaignReport, getMarketplaceChannels } from "@/lib/agenticClient";

function textContent(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : textContent(child))).join("");
}

const catalog = { channels: [] };
function company(): AgentCompany {
  return { id: "cmp_1", name: "Loja", status: "active", campaigns: [] } as AgentCompany;
}

async function renderCom(ingestoes?: string[]): Promise<ReactTestRenderer> {
  vi.mocked(getCompanyCampaignReport).mockResolvedValue({
    company_id: "cmp_1", campaigns: [],
    total: {
      campaign_id: "cmp_1", status: "", mode: "sandbox", impressions: 1510, clicks: 40, conversions: 4,
      spend_cents: 0, click_rate: "2.65%", conversion_rate: "10.00%", publicado_externamente: false,
      publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
    },
    sandbox: true, publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
    motivo: "publicar campanha exige conta autorizada", ingestoes,
  } as never);
  vi.mocked(getMarketplaceChannels).mockResolvedValue(catalog as never);
  let root!: ReactTestRenderer;
  await act(async () => {
    root = create(<CompanyCampaignSchedulingPanel company={company()} onCompanyChange={() => {}} />);
  });
  return root;
}

// E-268: o operador precisa VER que uma entrega automatica foi contada - e que reentrega nao soma. Sem isso, o
// numero do painel nao e' auditavel por quem opera.
describe("CompanyCampaignSchedulingPanel — visibilidade da ingestão", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  });

  it("mostra as entregas aceitas e declara que reentrega não soma", async () => {
    const root = await renderCom(["instagram|evt-1", "facebook|evt-1"]);
    const texto = textContent(root.root);
    expect(texto).toContain("Entregas aceitas (idempotência): 2");
    expect(texto).toContain("não soma");
    expect(texto).toContain("instagram");
    expect(texto).toContain("facebook");
    // A limitação externa continua declarada na mesma tela.
    expect(texto).toContain("BLOCKED_BY_EXTERNAL_DEPENDENCY");
  });

  it("sem entrega automática, declara que os números vêm de registro manual", async () => {
    const root = await renderCom(undefined);
    const texto = textContent(root.root);
    expect(texto).toContain("Nenhuma entrega automática registrada");
    expect(texto).toContain("registro manual");
    expect(texto).not.toContain("Entregas aceitas");
  });
});
