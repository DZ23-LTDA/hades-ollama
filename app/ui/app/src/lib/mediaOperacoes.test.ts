import { describe, expect, it } from "vitest";
import {
  OPERACOES_DE_MIDIA,
  motivoDeIndisponibilidade,
  operacaoPermitida,
  operacoesDisponiveis,
  resumoDoEstudio,
} from "./mediaOperacoes";

/**
 * O catalogo do Multimodal Studio (Estagio 45). *Estes testes sao o CONTRATO com o servidor:
 * se `internal/agent/media_tool.go` mudar a allowlist, um deles tem de falhar.*
 */

describe("allowlist espelhada", () => {
  it("tem exactamente as 6 operacoes que o servidor aceita", () => {
    expect(OPERACOES_DE_MIDIA.map((d) => d.operacao).sort()).toEqual([
      "audio.transcribe",
      "image.analyze",
      "image.generate",
      "speech.generate",
      "tone.generate",
      "video.generate",
    ]);
  });

  it("so^ `tone.generate` dispensa provider (e' a prova local do caminho)", () => {
    const semProvider = OPERACOES_DE_MIDIA.filter((d) => !d.exigeProvider);

    expect(semProvider).toHaveLength(1);
    expect(semProvider[0]?.operacao).toBe("tone.generate");
  });

  it("transporta os limites de bytes medidos (100 MB no STT, 25 MB na visao)", () => {
    const stt = OPERACOES_DE_MIDIA.find(
      (d) => d.operacao === "audio.transcribe",
    );
    const visao = OPERACOES_DE_MIDIA.find(
      (d) => d.operacao === "image.analyze",
    );

    expect(stt?.limiteBytes).toBe(100 << 20);
    expect(visao?.limiteBytes).toBe(25 << 20);
  });

  it("nao aceita operacao fora da allowlist", () => {
    expect(operacaoPermitida("video.generate")).toBe(true);
    // *O servidor responde «media operation %q is not allowlisted»: nada inventado passa.*
    expect(operacaoPermitida("video.upscale")).toBe(false);
    expect(operacaoPermitida("")).toBe(false);
    expect(operacaoPermitida("MEDIA.PROCESS")).toBe(false);
  });
});

describe("disponibilidade honesta", () => {
  it("sem provider oferece 1 de 6; com provider oferece as 6", () => {
    expect(operacoesDisponiveis(false)).toHaveLength(1);
    expect(operacoesDisponiveis(true)).toHaveLength(6);
  });

  it("devolve o motivo LITERAL do servidor quando falta provider", () => {
    // *A frase e' a do servidor, para o ecra^ e o log dizerem o mesmo.*
    expect(motivoDeIndisponibilidade("image.generate", false)).toBe(
      "media provider is not configured",
    );
    expect(motivoDeIndisponibilidade("speech.generate", false)).toBe(
      "media provider is not configured",
    );
  });

  it("nao inventa impedimento para o que e' local nem para o que esta' configurado", () => {
    expect(motivoDeIndisponibilidade("tone.generate", false)).toBeNull();
    expect(motivoDeIndisponibilidade("image.generate", true)).toBeNull();
  });

  it("explica a recusa de uma operacao fora da allowlist", () => {
    expect(motivoDeIndisponibilidade("video.upscale", true)).toContain(
      "allowlist",
    );
  });
});

describe("resumo do estudio", () => {
  it("sem provider diz quantas restam e PORQUE^ - nunca vazio", () => {
    const resumo = resumoDoEstudio(false);

    expect(resumo.trim().length).toBeGreaterThan(0);
    expect(resumo).toContain("1 de 6");
    expect(resumo).toContain("provider");
  });

  it("com provider declara as 6 disponiveis", () => {
    const resumo = resumoDoEstudio(true);

    expect(resumo).toContain("6 operacao");
    // *Nao deve continuar a falar de ausencia quando o provider existe.*
    expect(resumo).not.toContain("Sem provider");
  });
});
