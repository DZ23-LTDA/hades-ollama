import { act, create, type ReactTestInstance } from "react-test-renderer";
import { beforeEach, describe, expect, it, vi } from "vitest";

import MissionCapabilityPicker from "./MissionCapabilityPicker";
import { grantableScopes, type ScopeOption } from "@/lib/missionCapabilities";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

const OPTIONS: ScopeOption[] = grantableScopes([
  { name: "read_file", risk: "low", scopes: ["workspace:read"] },
  {
    name: "write_file",
    risk: "medium",
    scopes: ["workspace:write"],
    requires_approval: true,
  },
  {
    name: "run_cli",
    risk: "high",
    scopes: ["terminal:allowlisted"],
    requires_approval: true,
  },
]);

function render(
  props: Partial<React.ComponentProps<typeof MissionCapabilityPicker>> = {},
) {
  const onToggle = vi.fn();
  let renderer!: ReturnType<typeof create>;
  act(() => {
    renderer = create(
      <MissionCapabilityPicker
        options={OPTIONS}
        selected={[]}
        onToggle={onToggle}
        {...props}
      />,
    );
  });
  return { renderer, onToggle };
}

describe("MissionCapabilityPicker", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  });

  it("lista os escopos do catálogo com risco e exigência de aprovação", () => {
    const { renderer } = render();
    const text = textContent(renderer.root);
    expect(text).toContain("Escopos concedíveis");
    expect(text).toContain("workspace:read");
    expect(text).toContain("workspace:write");
    expect(text).toContain("terminal:allowlisted");
    expect(text).toContain("risco alto");
    expect(text).toContain("exige aprovação");
    expect(text).toContain("Ferramentas: run_cli.");
  });

  it("mostra o baseline como fixo e desabilitado, e os elevados como caixas clicáveis", () => {
    const { renderer } = render();
    const inputs = renderer.root.findAllByType("input");
    expect(inputs).toHaveLength(3);
    const baseline = renderer.root.findByProps({
      "aria-label": "workspace:read",
    });
    expect(baseline.props.checked).toBe(true);
    expect(baseline.props.disabled).toBe(true);
    const elevated = renderer.root.findByProps({
      "aria-label": "workspace:write",
    });
    expect(elevated.props.checked).toBe(false);
    expect(elevated.props.disabled).toBe(false);
  });

  it("propaga o toggle do escopo elevado para o chamador", () => {
    const { renderer, onToggle } = render();
    act(() => {
      renderer.root
        .findByProps({ "aria-label": "terminal:allowlisted" })
        .props.onChange();
    });
    expect(onToggle).toHaveBeenCalledWith("terminal:allowlisted");
  });

  it("avisa em alerta os escopos elevados selecionados e cita os que exigem aprovação", () => {
    const { renderer } = render({ selected: ["workspace:write"] });
    const alert = renderer.root.findByProps({ role: "alert" });
    const text = textContent(alert);
    expect(text).toContain("workspace:write");
    expect(text).toContain("Exigem aprovação explícita");
    expect(
      renderer.root.findByProps({ "aria-label": "workspace:write" }).props
        .checked,
    ).toBe(true);
  });

  it("ignora escopo selecionado que o catálogo não declara (fail-closed)", () => {
    const { renderer } = render({ selected: ["root:everything"] });
    const text = textContent(renderer.root);
    expect(text).not.toContain("root:everything");
    expect(renderer.root.findAllByProps({ role: "alert" })).toHaveLength(0);
  });

  it("degrada com aviso quando o catálogo de ferramentas está indisponível", () => {
    const { renderer } = render({ catalogUnavailable: true });
    const text = textContent(renderer.root);
    expect(text).toContain("Catálogo de ferramentas indisponível");
    expect(text).not.toContain("Escopos concedíveis");
    expect(
      textContent(renderer.root.findByProps({ role: "status" })),
    ).toContain("somente");
    expect(renderer.root.findAllByProps({ role: "alert" })).toHaveLength(0);
  });

  it("avisa quando o runtime não publicou nenhuma ferramenta", () => {
    const { renderer } = render({ options: [] });
    expect(
      textContent(renderer.root.findByProps({ role: "status" })),
    ).toContain("não publicou nenhuma ferramenta");
    expect(renderer.root.findAllByProps({ role: "alert" })).toHaveLength(0);
  });

  it("desabilita as caixas elevadas quando a missão está em andamento", () => {
    const { renderer } = render({ disabled: true });
    expect(
      renderer.root.findByProps({ "aria-label": "workspace:write" }).props
        .disabled,
    ).toBe(true);
  });

  it("descreve cada escopo elevado para leitores de tela", () => {
    const { renderer } = render();
    for (const name of ["workspace:write", "terminal:allowlisted"]) {
      const input = renderer.root.findByProps({ "aria-label": name });
      expect(typeof input.props["aria-describedby"]).toBe("string");
      expect(input.props["aria-describedby"].length).toBeGreaterThan(0);
    }
  });
});
