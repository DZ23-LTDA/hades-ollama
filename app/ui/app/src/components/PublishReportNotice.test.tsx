import { create, type ReactTestInstance, type ReactTestRenderer } from "react-test-renderer";
import { act } from "react-test-renderer";
import { describe, expect, it, vi } from "vitest";
import { PublishReportNotice } from "./PublishReportNotice";
import type { BuilderPublishReport } from "@/lib/agenticClient";

function texto(node: ReactTestInstance): string {
  return node.children.map((child) => (typeof child === "string" ? child : texto(child))).join("");
}
function render(report?: BuilderPublishReport | null): ReactTestRenderer {
  let root!: ReactTestRenderer;
  act(() => {
    root = create(<PublishReportNotice report={report} />);
  });
  return root;
}

// E-282 — A UI EXIBE O RELATÓRIO DE PUBLICAÇÃO. O limite de E-281 era "o relatório está no payload, mas a
// interface ainda não o exibe".
//
// A propriedade central: NUNCA afirmar sucesso quando o relatório diz que a aplicação não é executável — é
// exatamente o "botão que aparenta funcionar" que o master proíbe.
describe("PublishReportNotice", () => {
  it("avisa e nomeia o que falta quando a aplicação não é executável", () => {
    vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
    const root = render({
      files_copied: 2,
      backend_included: false,
      auth_included: false,
      database_included: false,
      runnable: false,
      missing: ["backend", "auth", "banco"],
      note: "arquivos copiados; aplicação NÃO executável",
    });
    const mostrado = texto(root.root);
    expect(mostrado).toContain("NÃO executável");
    expect(mostrado).toContain("backend");
    expect(mostrado).toContain("auth");
    expect(mostrado).toContain("banco");
    // Nenhuma promessa de sucesso: o relatório diz o contrário.
    expect(mostrado.toLowerCase()).not.toContain("com sucesso");
    // Aviso ANUNCIADO: aviso só visual não chega a quem usa leitor de tela.
    expect(root.root.findAll((node) => node.props?.role === "status").length).toBeGreaterThan(0);
  });

  it("informa sem alarme falso quando há backend", () => {
    const root = render({
      files_copied: 1,
      backend_included: true,
      auth_included: true,
      database_included: true,
      runnable: true,
      note: "backend presente; execução real ainda depende de rodar e testar o artefato",
    });
    const mostrado = texto(root.root);
    expect(mostrado).toContain("backend detectado");
    // Mesmo com backend, a nota honesta continua: rodar e testar ainda é o próximo passo.
    expect(mostrado).toContain("rodar e testar");
    expect(mostrado).not.toContain("NÃO executável");
  });

  it("não renderiza nada sem relatório (não inventa status)", () => {
    const semRelatorio = render(undefined);
    expect(semRelatorio.toJSON()).toBeNull();
    const nulo = render(null);
    expect(nulo.toJSON()).toBeNull();
  });
});
