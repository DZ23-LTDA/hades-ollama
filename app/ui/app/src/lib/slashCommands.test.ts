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

  // ***E-1095 (§ESTAGIO 19): O MESMO CAMPO ACEITA TEXTO NORMAL E `/goal`.***
  //
  // *O critério do prompt: «o chat principal deve aceitar texto normal e `/goal` no mesmo campo» e
  // «preservar texto longo, espaços, caminhos Windows e caracteres pt-BR sem o limite artificial
  // actual do parser de argumentos».* => *estes 5 testes **discriminam**: sem eles, o parser podia
  // tratar TODO o texto como comando (`parseSlashCommand` devolver algo em vez de `null`) e a suite
  // continuava verde.*
  describe("texto normal e `/goal` no mesmo campo", () => {
    it("texto normal NÃO é comando -- o parser devolve null", () => {
      expect(parseSlashCommand("olá, como estão as coisas?")).toBeNull();
      expect(parseSlashCommand("explique o erro do parser")).toBeNull();
      expect(parseSlashCommand("")).toBeNull();
      // *e uma barra que nao e' comando conhecido tambem nao*
      expect(parseSlashCommand("/etc/hosts")).toBeNull();
      expect(parseSlashCommand("/tmp/arquivo.txt")).toBeNull();
    });

    it("caminho Windows com barras invertidas é preservado", () => {
      const alvo = "C:\\Users\\DZ23\\Documents\\projeto";
      const parsed = parseSlashCommand("/goal revisar " + alvo);
      expect(parsed?.command.id).toBe("goal");
      expect(parsed?.objective).toBe("revisar " + alvo);
      expect(parsed?.objective).toContain("\\Users\\DZ23\\");
      // *e a URL codifica-o em vez de o perder*
      expect(slashCommandURL("/goal revisar " + alvo)).toContain("C%3A%5CUsers%5CDZ23");
    });

    it("caracteres pt-BR são preservados no objective", () => {
      const parsed = parseSlashCommand("/goal criar missão com acentuação e cedilha: ação, coração");
      expect(parsed?.objective).toBe("criar missão com acentuação e cedilha: ação, coração");
    });

    it("espaços internos e texto longo são preservados", () => {
      const longo = "a".repeat(300) + "  com  espaços   duplos";
      const parsed = parseSlashCommand("/goal " + longo);
      expect(parsed?.objective).toBe(longo);
      expect(parsed?.objective.length).toBe(300 + "  com  espaços   duplos".length);
    });

    it("`/goal` sem objective devolve comando com objective vazio (e a URL recusa)", () => {
      const parsed = parseSlashCommand("/goal");
      expect(parsed?.command.id).toBe("goal");
      expect(parsed?.objective).toBe("");
      // *o criterio: objetivo vazio pede complemento e NUNCA envia `/goal` cru ao modelo*
      expect(slashCommandURL("/goal")).toBeNull();
    });
  });
});
