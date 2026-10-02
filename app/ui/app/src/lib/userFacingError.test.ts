import { describe, expect, it } from "vitest";
import { humanizeApiError } from "./userFacingError";

describe("humanizeApiError", () => {
  it("transforma indisponibilidade em retry e preserva detalhe técnico separado", () => {
    const result = humanizeApiError(new TypeError("Failed to fetch"));
    expect(result.message).toContain("backend está offline");
    expect(result.action).toBe("retry");
    expect(result.technical).toBe("Failed to fetch");
  });

  it("orienta sessão expirada sem expor erro cru", () => {
    expect(humanizeApiError(new Error("request failed: 401")).action).toBe("sign_in");
  });

  it("explica conflito CAS com ação de atualização", () => {
    const result = humanizeApiError(new Error("expected_version conflict (409)"));
    expect(result.message).toContain("mudou em outro lugar");
    expect(result.action).toBe("resolve_conflict");
  });

  it("não promete integração externa sem configuração", () => {
    const result = humanizeApiError(new Error("BLOCKED_EXTERNAL: provider not configured"));
    expect(result.message).toContain("serviço externo");
    expect(result.action).toBe("configure");
  });
});
