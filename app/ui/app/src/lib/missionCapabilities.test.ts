import { describe, it, expect } from "vitest";
import {
  BASELINE_CAPABILITY,
  buildMissionCapabilities,
  describeScope,
  elevatedWarning,
  filterGrantable,
  grantableScopes,
  isGrantableScope,
  type ScopeOption,
  type ToolScopeSource,
} from "./missionCapabilities";

const CATALOG: ToolScopeSource[] = [
  {
    name: "read_file",
    risk: "low",
    scopes: ["workspace:read"],
    requires_approval: false,
  },
  {
    name: "write_file",
    risk: "medium",
    scopes: ["workspace:write"],
    requires_approval: true,
  },
  {
    name: "run_cli",
    risk: "high",
    scopes: ["terminal:allowlisted", "harness.cli"],
    requires_approval: true,
  },
  {
    name: "drive_desktop",
    risk: "critical",
    scopes: ["desktop:input", "workspace:read"],
  },
];

describe("grantableScopes", () => {
  it("agrega escopos de todas as tools, sem duplicatas e com o baseline primeiro", () => {
    const options = grantableScopes(CATALOG);
    expect(options.map((option) => option.name)).toEqual([
      "workspace:read",
      "desktop:input",
      "harness.cli",
      "terminal:allowlisted",
      "workspace:write",
    ]);
  });

  it("usa o maior risco entre as tools que declaram o mesmo escopo", () => {
    const options = grantableScopes(CATALOG);
    const baseline = options.find((option) => option.name === "workspace:read");
    expect(baseline?.risk).toBe("critical");
    expect(baseline?.tools).toEqual(["drive_desktop", "read_file"]);
    expect(baseline?.elevated).toBe(false);
  });

  it("marca requires_approval quando qualquer tool exigente declara o escopo", () => {
    const options = grantableScopes(CATALOG);
    expect(
      options.find((option) => option.name === "workspace:write")
        ?.requiresApproval,
    ).toBe(true);
    expect(
      options.find((option) => option.name === "desktop:input")
        ?.requiresApproval,
    ).toBe(false);
  });

  it("marca como elevado todo escopo diferente do baseline", () => {
    const options = grantableScopes(CATALOG);
    const elevated = options
      .filter((option) => option.elevated)
      .map((option) => option.name);
    expect(elevated).toEqual([
      "desktop:input",
      "harness.cli",
      "terminal:allowlisted",
      "workspace:write",
    ]);
  });

  it("descarta escopos em branco e catálogo ausente ou vazio", () => {
    expect(grantableScopes([{ name: "x", scopes: ["  ", ""] }])).toEqual([]);
    expect(grantableScopes([])).toEqual([]);
    expect(grantableScopes(null)).toEqual([]);
    expect(grantableScopes(undefined)).toEqual([]);
  });

  it("normaliza risco desconhecido para 'unknown'", () => {
    const [option] = grantableScopes([
      { name: "t", scopes: ["sandbox:execute"], risk: "banana" },
    ]);
    expect(option.risk).toBe("unknown");
  });
});

describe("isGrantableScope / filterGrantable", () => {
  const options = grantableScopes(CATALOG);

  it("só aceita escopo declarado pelo catálogo", () => {
    expect(isGrantableScope("workspace:write", options)).toBe(true);
    expect(isGrantableScope("root:everything", options)).toBe(false);
    expect(isGrantableScope("", options)).toBe(false);
    expect(isGrantableScope("   ", options)).toBe(false);
  });

  it("remove do payload qualquer escopo que o catálogo não declara", () => {
    expect(
      filterGrantable(["workspace:write", "root:everything", ""], options),
    ).toEqual(["workspace:write"]);
    expect(filterGrantable(["root:everything"], options)).toEqual([]);
  });
});

describe("buildMissionCapabilities", () => {
  it("sempre inclui o baseline de leitura", () => {
    expect(buildMissionCapabilities([])).toEqual([BASELINE_CAPABILITY]);
    expect(buildMissionCapabilities(null)).toEqual([BASELINE_CAPABILITY]);
    expect(buildMissionCapabilities(undefined)).toEqual([BASELINE_CAPABILITY]);
  });

  it("deduplica, ignora vazios e mantém ordem estável com baseline primeiro", () => {
    expect(
      buildMissionCapabilities([
        "workspace:write",
        BASELINE_CAPABILITY,
        "  ",
        "a:b",
        "a:b",
      ]),
    ).toEqual([BASELINE_CAPABILITY, "a:b", "workspace:write"]);
  });

  it("preserva a ordem alfabética após o baseline independente da ordem de entrada", () => {
    const first = buildMissionCapabilities(["z:z", "a:a"]);
    const second = buildMissionCapabilities(["a:a", "z:z"]);
    expect(first).toEqual(second);
  });
});

describe("describeScope / elevatedWarning", () => {
  const options = grantableScopes(CATALOG);
  const byName = (name: string): ScopeOption => {
    const found = options.find((option) => option.name === name);
    if (!found) throw new Error(`escopo ausente no catálogo de teste: ${name}`);
    return found;
  };

  it("explica o efeito de escopos conhecidos em pt-BR", () => {
    expect(describeScope(byName("workspace:write"))).toContain("altera");
    expect(describeScope(byName("harness.cli"))).toContain("CLIs externas");
    expect(describeScope(byName(BASELINE_CAPABILITY))).toContain("Lê arquivos");
  });

  it("cai em explicação por risco quando o escopo é desconhecido da tabela", () => {
    const unknown: ScopeOption = {
      name: "future:scope",
      elevated: true,
      risk: "critical",
      requiresApproval: true,
      tools: [],
    };
    expect(describeScope(unknown)).toContain("risco crítico");
    const low: ScopeOption = {
      name: "future:read",
      elevated: true,
      risk: "low",
      requiresApproval: false,
      tools: [],
    };
    expect(describeScope(low)).toContain("declarado pelo runtime");
    const baseline: ScopeOption = {
      name: "other:read",
      elevated: false,
      risk: "low",
      requiresApproval: false,
      tools: [],
    };
    expect(describeScope(baseline)).toContain("leitura do projeto");
  });

  it("não avisa nada quando só o baseline está selecionado", () => {
    expect(elevatedWarning([byName(BASELINE_CAPABILITY)])).toBe("");
  });

  it("avisa os escopos elevados e cita os que exigem aprovação", () => {
    const warning = elevatedWarning([
      byName("workspace:write"),
      byName("desktop:input"),
    ]);
    expect(warning).toContain("workspace:write");
    expect(warning).toContain("desktop:input");
    expect(warning).toContain("Exigem aprovação explícita: workspace:write");
    expect(warning.split("Exigem aprovação explícita: ")[1]).toBe(
      "workspace:write.",
    );
  });
});
