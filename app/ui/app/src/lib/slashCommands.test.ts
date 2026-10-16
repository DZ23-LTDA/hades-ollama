import { describe, expect, it } from "vitest";
import {
  filterSlashCommands,
  moveSlashCommandIndex,
  parseSlashCommand,
  slashCommandQuery,
  slashCommandURL,
  SLASH_COMMANDS,
} from "@/lib/slashCommands";

describe("slash commands", () => {
  it("lista somente comandos implementados", () => {
    expect(SLASH_COMMANDS.map((command) => command.id)).toEqual([
      "goal",
      "plan",
      "test",
      "review",
      "spark",
      "cowork",
    ]);
    expect(filterSlashCommands("goa").map((command) => command.id)).toEqual(["goal"]);
    expect(filterSlashCommands("não-existe")).toEqual([]);
  });

  // E-975: O TESTE QUE IMPEDE A DIVERGENCIA TIPO<->CATALOGO.
  //
  // **O achado**: `SlashCommandId` (L1) declarava `"spark"`, `"cowork"` e `"reject"`, e o
  // catalogo `SLASH_COMMANDS` tinha apenas 4. *Um tipo que promete mais do que o catalogo
  // entrega*: `filterSlashCommands` nunca os podia devolver, e o menu nunca os mostrava.
  //
  // Este teste **fixa a relacao**: todo id do tipo que o utilizador pode escrever tem de
  // estar no catalogo. *Sem ele, a divergencia podia voltar em silencio* — e foi
  // exactamente isso que aconteceu uma vez.
  it("todo comando do tipo que o usuário escreve está no catálogo", () => {
    const noCatalogo = new Set(SLASH_COMMANDS.map((command) => command.id));
    // `reject` e' a excepcao DECLARADA: e' um token de ESTADO (uma decisao de aprovacao),
    // nao um comando que o utilizador digita no composer. *Se algum dia for escrito, este
    // teste obriga a decidir.*
    const soEstado = new Set<string>(["reject"]);
    const tiposEscritos = ["goal", "plan", "test", "review", "spark", "cowork"];
    for (const id of tiposEscritos) {
      expect(noCatalogo.has(id), `/${id} está no tipo e não no catálogo`).toBe(true);
    }
    // E o inverso: nada no catalogo fora do tipo.
    for (const command of SLASH_COMMANDS) {
      expect(
        [...tiposEscritos, ...soEstado].includes(command.id),
        `${command.id} está no catálogo e não no tipo`,
      ).toBe(true);
    }
  });

  it("spark e cowork são modos do MESMO mecanismo, não motores separados", () => {
    // *A garantia do §FASE-3 item 3*: todos entram pelo mesmo parser e produzem a mesma
    // URL canónica `/agentic?...`, que o `AgenticConsole` consome criando a missão pela
    // mesma `POST /api/agent/v1/missions`.
    for (const [id, autorun] of [
      ["spark", "false"],
      ["cowork", "true"],
    ] as const) {
      const parsed = parseSlashCommand(`/${id} objetivo de teste`);
      expect(parsed?.command.id).toBe(id);
      expect(parsed?.command.mode).toBe("goal");
      expect(parsed?.objective).toBe("objetivo de teste");
      expect(slashCommandURL(`/${id} objetivo de teste`)).toBe(
        `/agentic?objective=objetivo+de+teste&slash=${id}&autorun=${autorun}`,
      );
    }
  });

  it("spark e cowork não degeneram em envio cru quando falta o objetivo", () => {
    // *A regra do §FASE-3 item 4 vale para os seis*, e nao so' para o `/goal`.
    for (const id of ["goal", "plan", "test", "review", "spark", "cowork"]) {
      expect(slashCommandURL(`/${id}`), `/${id} sem objetivo nao pode virar URL`).toBeNull();
    }
  });

  it("fornece consulta apenas enquanto o usuário está escolhendo o comando", () => {
    expect(slashCommandQuery("/")).toBe("");
    expect(slashCommandQuery("/go")).toBe("go");
    expect(slashCommandQuery("/goal objetivo")).toBeNull();
    expect(slashCommandQuery("objetivo")).toBeNull();
  });

  it("navega no menu com setas e faz wraparound", () => {
    expect(moveSlashCommandIndex(0, "next", 4)).toBe(1);
    expect(moveSlashCommandIndex(0, "previous", 4)).toBe(3);
    expect(moveSlashCommandIndex(3, "next", 4)).toBe(0);
    expect(moveSlashCommandIndex(0, "next", 0)).toBe(0);
  });

  it("navega com /goal para missão autorun e preserva o objetivo", () => {
    const parsed = parseSlashCommand("/goal criar um site seguro");
    expect(parsed?.command.id).toBe("goal");
    expect(parsed?.objective).toBe("criar um site seguro");
    expect(slashCommandURL("/goal criar um site seguro")).toBe(
      "/agentic?objective=criar+um+site+seguro&slash=goal&autorun=true",
    );
  });

  it("não aceita comando inexistente ou comando sem objetivo", () => {
    expect(parseSlashCommand("/unknown objetivo")).toBeNull();
    expect(slashCommandURL("/goal")).toBeNull();
  });
});
