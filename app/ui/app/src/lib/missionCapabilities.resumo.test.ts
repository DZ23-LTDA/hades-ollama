import { describe, expect, it } from "vitest";

import { resumoPermissoes, type ScopeOption } from "./missionCapabilities";

function opcao(
  name: string,
  elevated: boolean,
  risk: string,
  requiresApproval = false,
): ScopeOption {
  return { name, elevated, risk, requiresApproval, tools: [] };
}

// I0c: a barra de permissoes tem de dizer o nivel SEM obrigar a abrir nada.
// Antes, tudo vivia num <details> fechado e o nivel de acesso nao aparecia.
describe("resumoPermissoes", () => {
  const catalogo = [
    opcao("workspace:read", false, "low"),
    opcao("workspace:write", true, "medium"),
    opcao("shell:exec", true, "high", true),
  ];

  it("NEGATIVO: sem escopos elevados diz 'so leitura' e nao lista nada", () => {
    const resumo = resumoPermissoes(catalogo, ["workspace:read"]);
    expect(resumo.rotulo).toBe("somenteLeitura");
    expect(resumo.elevados).toEqual([]);
    expect(resumo.base).toBe("workspace:read");
  });

  it("POSITIVO: com workspace:write mostra o escopo e o maior risco", () => {
    const resumo = resumoPermissoes(catalogo, ["workspace:read", "workspace:write"]);
    expect(resumo.rotulo).toBe("comElevados");
    expect(resumo.elevados).toEqual(["workspace:write"]);
    expect(resumo.maiorRisco).toBe("medium");
    expect(resumo.exigeAprovacao).toBe(false);
  });

  it("sinaliza aprovacao humana quando a ferramenta exige", () => {
    const resumo = resumoPermissoes(catalogo, ["workspace:read", "shell:exec"]);
    expect(resumo.exigeAprovacao).toBe(true);
    expect(resumo.maiorRisco).toBe("high");
  });

  it("NEGATIVO: escopo pedido que nao existe no catalogo nao e' inventado", () => {
    const resumo = resumoPermissoes(catalogo, ["workspace:read", "root:everything"]);
    expect(resumo.elevados).toEqual([]);
    expect(resumo.rotulo).toBe("somenteLeitura");
  });

  it("ordena os elevados para a barra nao mudar de ordem entre renders", () => {
    const resumo = resumoPermissoes(catalogo, ["shell:exec", "workspace:write"]);
    expect(resumo.elevados).toEqual(["shell:exec", "workspace:write"]);
  });
});
