import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { PREFERENCIAS_PADRAO, aplicarAccao } from "@/lib/companionPreferencias";

/**
 * `E-1086` — o teste da **ligação** das preferências ao `CompanionLigado`.
 *
 * ***PORQUE ESTE TESTE TEM DUAS PARTES, E PORQUE A SEGUNDA É DIFERENTE.***
 *
 *  1. **`aplicarAccao`** — a função pura da UI. *Testa-se sem DOM, e discrimina as três acções.*
 *  2. **O CONTRATO ESTRUTURAL** — *o `CompanionLigado` tem de passar `preferencias` ao
 *     `CompanionMascote`.*
 *
 * ***PORQUE A SEGUNDA NÃO É UM RENDER.*** O `vitest.config.ts` corre em **`environment: "node"`**, e o
 * projecto **não tem `@testing-library/react` nem `jsdom`** ⇒ *não há `useEffect` em `renderToStaticMarkup`,
 * logo um teste de render ficaria sempre no «a carregar» e **não provaria a passagem da prop**.*
 * ⇒ *a alternativa honesta é ler o fonte e exigir a passagem: é um contrato estrutural, e falha se
 * alguém a remover.*
 *
 * ***E É O ÚNICO TESTE DESTE FICHEIRO QUE APANHA A LACUNA REAL.*** *A lacuna do `E-1086` não estava no
 * `aplicarAccao` (que não existia): estava na prop que **não era passada**. Uma função pura
 * correcta com a prop esquecida é exactamente o defeito que se está a fechar.*
 */
describe("aplicarAccao — as três acções do item 4", () => {
  it("alternarVisivel inverte só o `visivel`", () => {
    const r = aplicarAccao(PREFERENCIAS_PADRAO, "alternarVisivel");
    expect(r).toEqual({ ...PREFERENCIAS_PADRAO, visivel: false });
    expect(aplicarAccao(r, "alternarVisivel")).toEqual(PREFERENCIAS_PADRAO);
  });

  it("alternarSilenciado inverte só o `silenciado`", () => {
    const r = aplicarAccao(PREFERENCIAS_PADRAO, "alternarSilenciado");
    expect(r).toEqual({ ...PREFERENCIAS_PADRAO, silenciado: true });
  });

  it("restaurar repõe o default, mesmo a partir de um estado todo mudado", () => {
    const mudado = { visivel: false, silenciado: true, movimentoReduzido: true };
    expect(aplicarAccao(mudado, "restaurar")).toEqual(PREFERENCIAS_PADRAO);
  });

  it("nao muta o objecto recebido -- devolve um novo", () => {
    const antes = { ...PREFERENCIAS_PADRAO };
    aplicarAccao(antes, "alternarVisivel");
    expect(antes).toEqual(PREFERENCIAS_PADRAO);
  });

  it("as tres accoes nao se confundem entre si", () => {
    const a = aplicarAccao(PREFERENCIAS_PADRAO, "alternarVisivel");
    const b = aplicarAccao(PREFERENCIAS_PADRAO, "alternarSilenciado");
    const c = aplicarAccao(PREFERENCIAS_PADRAO, "restaurar");
    expect(a).not.toEqual(b);
    expect(c).toEqual(PREFERENCIAS_PADRAO);
  });
});

describe("o CONTRATO ESTRUTURAL — o CompanionLigado passa as preferências (E-1086)", () => {
  const fonte = readFileSync(resolve(__dirname, "CompanionLigado.tsx"), "utf8");

  it("importa as preferências do módulo certo", () => {
    expect(fonte).toContain('from "@/lib/companionPreferencias"');
    expect(fonte).toContain("carregarPreferencias");
  });

  it("lê as preferências no mount (o `useEffect` de leitura)", () => {
    expect(fonte).toMatch(/useEffect\(\(\) => \{[^}]*carregarPreferencias\(\)/s);
  });

  it("passa `preferencias` ao CompanionMascote -- a prop que estava em falta", () => {
    // ***O CORAÇÃO DO E-1086.*** *Se esta linha desaparecer, o componente volta a receber o default
    // e a preferência do utilizador deixa de chegar — e este teste falha.*
    expect(fonte).toMatch(/<CompanionMascote[\s\S]*?preferencias=\{preferencias\}/);
  });

  it("passa também `sistemaPedeMenosMovimento`", () => {
    expect(fonte).toMatch(/<CompanionMascote[\s\S]*?sistemaPedeMenosMovimento=\{sistemaReduzido\}/);
  });

  it("oferece as três acções -- ocultar, silenciar e restaurar", () => {
    expect(fonte).toContain('agir("alternarVisivel")');
    expect(fonte).toContain('agir("alternarSilenciado")');
    expect(fonte).toContain('agir("restaurar")');
  });

  it("declara quando a escolha NÃO persistiu", () => {
    // *O `guardarPreferencias` devolve `false`; o componente tem de o mostrar, e não engolir.*
    expect(fonte).toContain("naoPersistiu");
    expect(fonte).toContain("data-companion-nao-persistiu");
  });
});
