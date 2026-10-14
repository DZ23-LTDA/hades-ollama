import { describe, expect, it, vi } from "vitest";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";

import {
  CompanionMascote,
  ONBOARDING_DO_COMPANION,
  type CompanionEstado,
} from "./CompanionMascote";

/**
 * E-538 (Estágio 46) — o teste da **acessibilidade** e do **movimento honesto**.
 *
 * *Estes testes não medem o desenho: medem as três regras que o complemento impõe —*
 *  1. **não animar a fingir trabalho** (o `companion-idle` não anima quando não há eventos);
 *  2. **avisar quem não vê** (o estado é anunciado, e com o `aria-live` correcto);
 *  3. **declarar a ausência** (a voz indisponível não se disfarça de botão que funciona).
 */

function html(props: Partial<Parameters<typeof CompanionMascote>[0]> = {}): string {
  return renderToStaticMarkup(
    createElement(CompanionMascote, { estado: "ausente" as CompanionEstado, razao: "", ...props }),
  );
}

describe("CompanionMascote — movimento derivado do estado", () => {
  it("não anima quando não há eventos (ausente)", () => {
    const saida = html({ estado: "ausente" });
    // ***A REGRA MAIS IMPORTANTE DO ESTÁGIO 46.*** *Um mascote a rodar sem nada a acontecer é a mentira mais fácil de
    // escrever — e `companion-idle` é a classe que NÃO anima.*
    expect(saida).toContain("companion-idle");
    expect(saida).not.toContain("companion-typing");
    expect(saida).not.toContain("companion-pulse");
  });

  it("anima a «trabalhar» — e só nesse estado afirma trabalho", () => {
    const saida = html({ estado: "a_trabalhar" });
    expect(saida).toContain("companion-typing");
    expect(saida).toContain("A trabalhar");
  });

  it("não conclui nem cancela com animação de trabalho", () => {
    for (const estado of ["concluido", "cancelado"] as CompanionEstado[]) {
      const saida = html({ estado });
      expect(saida).toContain("companion-idle");
      expect(saida).not.toContain("companion-typing");
    }
  });

  it("expõe o estado num atributo, para que o CSS e os testes o possam ler", () => {
    expect(html({ estado: "bloqueado" })).toContain('data-state="bloqueado"');
  });
});

describe("CompanionMascote — o anúncio para quem não vê", () => {
  it("anuncia com `assertive` SÓ quando a decisão é do humano", () => {
    // *A aprovação é o único estado em que a acção é DELE — e por isso é o único que interrompe a leitura.*
    expect(html({ estado: "a_aprovar" })).toContain('aria-live="assertive"');
    for (const estado of ["a_trabalhar", "ausente", "concluido"] as CompanionEstado[]) {
      const saida = html({ estado });
      expect(saida).toContain('aria-live="polite"');
      expect(saida).not.toContain('aria-live="assertive"');
    }
  });

  it("mostra a procedência do estado, sempre que ela existe", () => {
    const saida = html({ estado: "concluido", razao: "mission.completed" });
    // *Um estado sem procedência é indistinguível de um estado inventado — o E-536 devolve a razão para que ela possa
    // ser mostrada.*
    expect(saida).toContain("mission.completed");
    expect(saida).toContain("Procedência");
  });

  it("marca o mascote como decorativo, para não repetir o anúncio", () => {
    expect(html()).toContain('aria-hidden="true"');
  });
});

describe("CompanionMascote — a ausência declarada (nunca disfarçada)", () => {
  it("com a voz indisponível, o botão diz que não faz e está DESACTIVADO", () => {
    const saida = html({ podeFalar: false });
    // ***«Um botão que não faz nada» vs. «um botão que diz que ainda não faz».***
    expect(saida).toContain("Voz indisponível");
    expect(saida).toContain("disabled");
    expect(saida).toContain("Sem provider de fala ligado neste ambiente.");
  });

  it("com a voz disponível, o botão faz e chama quem o carregar", () => {
    const aoPedir = vi.fn();
    const saida = html({ podeFalar: true, onFalarPedido: aoPedir });
    expect(saida).toContain("Falar com o Companion");
    expect(saida).not.toContain("Voz indisponível");
    // *E o componente NÃO chama nada sozinho: a única chamada possível é a do humano.*
    expect(aoPedir).not.toHaveBeenCalled();
  });
});

describe("Onboarding do Companion", () => {
  it("declara o que está AUSENTE, e não só o que promete", () => {
    // *Um onboarding que só promete é publicidade. Este diz o que falta, para que a ausência seja um facto declarado.*
    const ausentes = ONBOARDING_DO_COMPANION.filter((i) => i.ausente);
    expect(ausentes.length).toBeGreaterThanOrEqual(2);
    expect(ausentes.map((i) => i.titulo).join(" ")).toMatch(/Voz/i);
    expect(ausentes.map((i) => i.titulo).join(" ")).toMatch(/Não respondo/i);
  });

  it("explica que o movimento reduzido é respeitado", () => {
    expect(ONBOARDING_DO_COMPANION.map((i) => i.corpo).join(" ")).toMatch(/movimento/i);
  });
});
