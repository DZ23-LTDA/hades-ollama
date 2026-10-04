import { describe, it, expect } from "vitest";
import { generateStudioHTML } from "./studioHtml";
import type { BuilderProject, VisualComponent } from "@/lib/agenticClient";

function project(components: VisualComponent[], name = "Meu site"): BuilderProject {
  return {
    id: "b1",
    organization_id: "local",
    name,
    kind: "website",
    entry: "index.html",
    version: 1,
    status: "draft",
    root: "/tmp",
    created_at: "",
    updated_at: "",
    components,
  };
}

describe("generateStudioHTML", () => {
  it("produces a complete standalone HTML document with the project title", () => {
    const html = generateStudioHTML(project([], "Loja da Ana"));
    expect(html).toContain("<!doctype html>");
    expect(html).toContain('<html lang="pt-BR">');
    expect(html).toContain("<title>Loja da Ana</title>");
    expect(html).toContain("</html>");
  });

  it("renders each known component type", () => {
    const html = generateStudioHTML(
      project([
        { id: "1", type: "heading", props: { text: "Bem-vindo" } },
        { id: "2", type: "paragraph", props: { text: "Um parágrafo." } },
        { id: "3", type: "button", props: { label: "Comprar" } },
        { id: "4", type: "card", props: { title: "Plano", body: "Detalhes" } },
        { id: "5", type: "metric", props: { label: "Vendas", value: "42" } },
        { id: "6", type: "navbar", props: { brand: "Ana", links: "Início, Sobre" } },
      ]),
    );
    expect(html).toContain("<h2 class=\"c-heading\">Bem-vindo</h2>");
    expect(html).toContain("Um parágrafo.");
    expect(html).toContain("Comprar");
    expect(html).toContain("Plano");
    expect(html).toContain("Vendas");
    expect(html).toContain("42");
    expect(html).toContain("Ana");
    expect(html).toContain("<span>Início</span>");
    expect(html).toContain("<span>Sobre</span>");
  });

  it("renders the extended component types (hero, input, link, list, divider)", () => {
    const html = generateStudioHTML(
      project([
        { id: "h", type: "hero", props: { title: "Grande", subtitle: "Sub", cta: "Ir" } },
        { id: "f", type: "input", props: { label: "E-mail", placeholder: "voce@ex.com" } },
        { id: "l", type: "link", props: { text: "Abrir", href: "https://ex.com" } },
        { id: "u", type: "list", props: { items: "Um, Dois" } },
        { id: "d", type: "divider" },
      ]),
    );
    expect(html).toContain('<section class="c-hero">');
    expect(html).toContain("Grande");
    expect(html).toContain('placeholder="voce@ex.com"');
    expect(html).toContain('href="https://ex.com"');
    expect(html).toContain("<li>Um</li>");
    expect(html).toContain('<hr class="c-divider" />');
  });

  it("blocks dangerous link hrefs (javascript:)", () => {
    const html = generateStudioHTML(
      project([{ id: "l", type: "link", props: { text: "x", href: "javascript:alert(1)" } }]),
    );
    expect(html).not.toContain("javascript:alert(1)");
    expect(html).toContain('href="#"');
  });

  it("escapes user content so a component cannot inject markup (XSS-safe)", () => {
    const html = generateStudioHTML(
      project([{ id: "x", type: "heading", props: { text: "<script>alert(1)</script>" } }]),
    );
    expect(html).not.toContain("<script>alert(1)</script>");
    expect(html).toContain("&lt;script&gt;");
  });

  it("blocks non-http image sources (no javascript: URLs)", () => {
    const html = generateStudioHTML(
      project([{ id: "i", type: "image", props: { src: "javascript:alert(1)", alt: "x" } }]),
    );
    expect(html).not.toContain("javascript:alert(1)");
    const ok = generateStudioHTML(
      project([{ id: "i", type: "image", props: { src: "https://ex.com/a.png", alt: "x" } }]),
    );
    expect(ok).toContain('src="https://ex.com/a.png"');
  });

  it("shows a friendly placeholder when there are no components", () => {
    const html = generateStudioHTML(project([]));
    expect(html).toContain("Adicione componentes no Studio");
  });

  it("uses the prop fallback when a value is missing", () => {
    const html = generateStudioHTML(project([{ id: "1", type: "button" }]));
    expect(html).toContain("Botão");
  });
});
