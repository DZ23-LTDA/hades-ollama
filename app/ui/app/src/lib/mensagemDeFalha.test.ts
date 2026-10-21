import { describe, expect, it } from "vitest";

import { mensagemDaFalhaCom, motivoDoCorpoDaApi } from "./mensagemDeFalha";

/**
 * O discriminante do E-1416.
 *
 * O servidor responde 503 com `error`, `reason`, `path` e `note`. O painel mostrava o `error` -- o
 * codigo -- e deitava fora o `reason`. Este teste falha se alguem voltar a preferir o codigo.
 */
describe("mensagemDaFalhaCom", () => {
  const alternativa = "nao foi possivel detectar o projecto";

  it("prefere o reason ao codigo do corpo", () => {
    const falha = Object.assign(new Error("agent_api_unavailable"), {
      corpo: {
        error: "agent_api_unavailable",
        reason: "o runtime do agente nao foi inicializado",
        note: "o servidor esta' a servir o Ollama core",
      },
    });
    expect(mensagemDaFalhaCom(falha, alternativa)).toBe(
      "o runtime do agente nao foi inicializado",
    );
  });

  it("nao mostra o codigo do corpo quando so' ha' error", () => {
    const falha = Object.assign(new Error("falha de rede"), {
      corpo: { error: "agent_api_unavailable" },
    });
    const msg = mensagemDaFalhaCom(falha, alternativa);
    expect(msg).not.toBe("agent_api_unavailable");
    expect(msg).toBe("falha de rede");
  });

  it("cai para o note quando nao ha' reason", () => {
    const falha = Object.assign(new Error("falhou"), {
      corpo: { note: "o Agent Center nao foi montado" },
    });
    expect(mensagemDaFalhaCom(falha, alternativa)).toBe("o Agent Center nao foi montado");
  });

  it("usa a mensagem da excepcao quando nao ha' corpo", () => {
    expect(mensagemDaFalhaCom(new Error("sem rede"), alternativa)).toBe("sem rede");
  });

  it("usa a alternativa quando nao ha' nada", () => {
    expect(mensagemDaFalhaCom({}, alternativa)).toBe(alternativa);
    expect(mensagemDaFalhaCom(null, alternativa)).toBe(alternativa);
    expect(mensagemDaFalhaCom(new Error("   "), alternativa)).toBe(alternativa);
  });

  it("ignora reason vazio ou que nao seja texto", () => {
    expect(motivoDoCorpoDaApi({ corpo: { reason: "   " } })).toBe("");
    expect(motivoDoCorpoDaApi({ corpo: { reason: 42 } })).toBe("");
    expect(motivoDoCorpoDaApi({ corpo: "texto" })).toBe("");
    expect(motivoDoCorpoDaApi(undefined)).toBe("");
  });
});
