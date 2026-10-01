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
    expect(SLASH_COMMANDS.map((command) => command.id)).toEqual(["goal", "plan", "test", "review"]);
    expect(filterSlashCommands("goa").map((command) => command.id)).toEqual(["goal"]);
    expect(filterSlashCommands("não-existe")).toEqual([]);
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
