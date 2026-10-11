import { act, create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CompanyCampaignSchedulingPanel } from "./CompanyCampaignSchedulingPanel";
import type { AgentCompany } from "@/lib/agenticClient";

// Substituição explícita das funções usadas pelo painel: o repositório já usa este padrão, e o
// `importOriginal` com spread não funciona nesta configuração (__spreadValues).
vi.mock("@/lib/agenticClient", () => ({
  getCompanyCampaignReport: vi.fn(),
  getMarketplaceChannels: vi.fn(),
  recordCompanyCampaignMetrics: vi.fn(),
  scheduleCompanyCampaign: vi.fn(),
  syncMarketplace: vi.fn(),
}));

import {
  getCompanyCampaignReport,
  getMarketplaceChannels,
  recordCompanyCampaignMetrics,
  scheduleCompanyCampaign,
  syncMarketplace,
} from "@/lib/agenticClient";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function buttonsWithText(root: ReactTestRenderer, text: string): ReactTestInstance[] {
  return root.root
    .findAll((node) => node.type === "button")
    .filter((node) => textContent(node).includes(text));
}

const declaredReport = {
  company_id: "cmp_1",
  campaigns: [],
  total: {
    campaign_id: "cmp_1", status: "", mode: "sandbox", impressions: 400, clicks: 40, conversions: 8,
    spend_cents: 700, click_rate: "10.00%", conversion_rate: "20.00%", publicado_externamente: false,
    publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
  },
  sandbox: true,
  publicacao_externa: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
  motivo: "publicar campanha exige conta autorizada e scopes válidos da plataforma",
};

const catalog = {
  channels: [
    { id: "mercado_livre", name: "Mercado Livre", credential_env: "HADES_MARKETPLACE_MERCADO_LIVRE_TOKEN", state: "CONFIG_REQUIRED", capabilities: ["estoque"], live_tested: false, unblock_procedure: "definir a variável HADES_MARKETPLACE_MERCADO_LIVRE_TOKEN e executar um teste de conexão real" },
  ],
};

function companyWith(campaigns: AgentCompany["campaigns"]): AgentCompany {
  return { id: "cmp_1", name: "Loja", status: "active", campaigns } as AgentCompany;
}

describe("CompanyCampaignSchedulingPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    vi.mocked(getCompanyCampaignReport).mockResolvedValue(declaredReport as never);
    vi.mocked(getMarketplaceChannels).mockResolvedValue(catalog as never);
  });

  it("declara que a publicação externa está bloqueada, com o motivo, e não inventa selo de campanha no ar", async () => {
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<CompanyCampaignSchedulingPanel company={companyWith([])} onCompanyChange={() => {}} />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("BLOCKED_BY_EXTERNAL_DEPENDENCY");
    expect(rendered).toContain("publicar campanha exige conta autorizada");
    expect(rendered).not.toContain("no ar");
  });

  it("mostra o estado REAL do canal em vez de selo verde, e o procedimento de desbloqueio", async () => {
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<CompanyCampaignSchedulingPanel company={companyWith([])} onCompanyChange={() => {}} />);
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("CONFIG_REQUIRED");
    expect(rendered).toContain("Testado ao vivo: não");
    expect(rendered).toContain("HADES_MARKETPLACE_MERCADO_LIVRE_TOKEN");
  });

  it("oferece Agendar apenas para campanha aprovada, e envia as horas ao servidor", async () => {
    const onChange = vi.fn();
    vi.mocked(scheduleCompanyCampaign).mockResolvedValue(companyWith([]) as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(
        <CompanyCampaignSchedulingPanel
          company={companyWith([
            { id: "cmp_rascunho", name: "Rascunho", channel: "social", objective: "leads", status: "draft", daily_budget_cents: 0, approval_required: true, approved: false, conversions: 0, spend_cents: 0 },
            { id: "cmp_aprovada", name: "Aprovada", channel: "social", objective: "leads", status: "approved", daily_budget_cents: 0, approval_required: true, approved: true, conversions: 0, spend_cents: 0 },
          ])}
          onCompanyChange={onChange}
        />,
      );
    });
    // Sem aprovação não há controle de agendamento: seria um botão que sempre falha.
    const agendar = buttonsWithText(root, "Agendar");
    expect(agendar).toHaveLength(1);
    await act(async () => {
      agendar[0].props.onClick();
    });
    expect(scheduleCompanyCampaign).toHaveBeenCalledWith("cmp_1", "cmp_aprovada", 2);
    expect(onChange).toHaveBeenCalled();
  });

  it("oferece Registrar métricas apenas depois que a campanha executou, enviando números", async () => {
    vi.mocked(recordCompanyCampaignMetrics).mockResolvedValue({} as never);
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(
        <CompanyCampaignSchedulingPanel
          company={companyWith([
            { id: "cmp_aprovada", name: "Só aprovada", channel: "social", objective: "leads", status: "approved", daily_budget_cents: 0, approval_required: true, approved: true, conversions: 0, spend_cents: 0 },
            { id: "cmp_agendada", name: "Agendada", channel: "social", objective: "leads", status: "scheduled", daily_budget_cents: 0, approval_required: true, approved: true, conversions: 0, spend_cents: 0, scheduled_at: "2026-10-14T12:00:00Z" },
          ])}
          onCompanyChange={() => {}}
        />,
      );
    });
    const registrar = buttonsWithText(root, "Registrar métricas");
    // Medir campanha que não executou seria inventar resultado: só a agendada oferece.
    expect(registrar).toHaveLength(1);
    await act(async () => {
      registrar[0].props.onClick();
    });
    expect(recordCompanyCampaignMetrics).toHaveBeenCalledWith("cmp_1", "cmp_agendada", {
      impressions: 100, clicks: 5, conversions: 0, spend_cents: 0,
    });
  });

  it("mostra a falha do servidor como falha, sem marcar sucesso", async () => {
    vi.mocked(scheduleCompanyCampaign).mockRejectedValue(new Error("campanha precisa de aprovação antes do agendamento"));
    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(
        <CompanyCampaignSchedulingPanel
          company={companyWith([
            { id: "cmp_aprovada", name: "Aprovada", channel: "social", objective: "leads", status: "approved", daily_budget_cents: 0, approval_required: true, approved: true, conversions: 0, spend_cents: 0 },
          ])}
          onCompanyChange={() => {}}
        />,
      );
    });
    await act(async () => {
      buttonsWithText(root, "Agendar")[0].props.onClick();
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("precisa de aprovação antes do agendamento");
    expect(rendered).not.toContain("Campanha agendada no sandbox.");
  });

  it("monta itens com versões crescentes e envia EXATAMENTE o que o operador montou", async () => {
    vi.mocked(syncMarketplace).mockResolvedValue({
      channel_id: "mercado_livre", direction: "push", sandbox: true, homologacao_live: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
      motivo: "sem credencial", applied: 2, skipped: 0, conflicts: 0, operations_used: 2, operation_limit: 50,
      outcomes: [
        { entity: "estoque", sku: "SKU-1", version: 1, result: "aplicado" },
        { entity: "estoque", sku: "SKU-1", version: 2, result: "aplicado" },
      ],
    } as never);
    const withProduct = { ...companyWith([]), products: [{ id: "prd_1", sku: "SKU-1", name: "Produto" }] } as AgentCompany;

    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<CompanyCampaignSchedulingPanel company={withProduct} onCompanyChange={() => {}} />);
    });
    // Sem item montado o botão fica DESABILITADO: sincronizar sem item é recusado pelo servidor.
    expect(buttonsWithText(root, "Sincronizar")[0].props.disabled).toBe(true);

    await act(async () => {
      buttonsWithText(root, "Adicionar item")[0].props.onClick();
    });
    await act(async () => {
      buttonsWithText(root, "Adicionar item")[0].props.onClick();
    });
    // A VERSÃO incrementa: o servidor é idempotente por versão e reenviar a mesma seria ignorado.
    const rendered = textContent(root.root);
    expect(rendered).toContain("versão 1");
    expect(rendered).toContain("versão 2");
    expect(buttonsWithText(root, "Sincronizar")[0].props.disabled).not.toBe(true);

    await act(async () => {
      buttonsWithText(root, "Sincronizar")[0].props.onClick();
    });
    expect(syncMarketplace).toHaveBeenCalledWith("cmp_1", {
      channel_id: "mercado_livre",
      direction: "push",
      items: [
        { entity: "estoque", sku: "SKU-1", version: 1, quantity: 1 },
        { entity: "estoque", sku: "SKU-1", version: 2, quantity: 1 },
      ],
    });
    // Aplicados saem da fila; o relatório mostra os números reais.
    expect(textContent(root.root)).toContain("2 aplicados");
    expect(textContent(root.root)).not.toContain("versão 1");
  });

  it("mantém conflito e ignorado VISÍVEIS, sem dar a impressão de sincronização completa", async () => {
    vi.mocked(syncMarketplace).mockResolvedValue({
      channel_id: "amazon", direction: "push", sandbox: true, homologacao_live: "BLOCKED_BY_EXTERNAL_DEPENDENCY",
      motivo: "sem credencial", applied: 0, skipped: 1, conflicts: 1, operations_used: 0, operation_limit: 50,
      outcomes: [
        { entity: "estoque", sku: "SKU-1", version: 1, result: "ignorado", detail: "versão já aplicada" },
        { entity: "estoque", sku: "SKU-1", version: 2, result: "conflito", detail: "remoto=10 local=5 desejado=5: divergência não é sobrescrita" },
      ],
    } as never);
    const withProduct = { ...companyWith([]), products: [{ id: "prd_1", sku: "SKU-1", name: "Produto" }] } as AgentCompany;

    let root!: ReactTestRenderer;
    await act(async () => {
      root = create(<CompanyCampaignSchedulingPanel company={withProduct} onCompanyChange={() => {}} />);
    });
    await act(async () => {
      buttonsWithText(root, "Adicionar item")[0].props.onClick();
    });
    await act(async () => {
      buttonsWithText(root, "Sincronizar")[0].props.onClick();
    });
    const rendered = textContent(root.root);
    expect(rendered).toContain("conflito: SKU-1 v2");
    expect(rendered).toContain("divergência não é sobrescrita");
    expect(rendered).toContain("ignorado: SKU-1 v1");
    // Nada foi aplicado, então o item CONTINUA na fila — esconder isso seria mentir.
    expect(rendered).toContain("versão 1");
  });
});
