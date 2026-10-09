import { afterEach, describe, expect, it, vi } from "vitest";

import { API_BASE } from "@/lib/config";
import { enableCloudModelsOptIn } from "./cloudSettings";

function response(ok: boolean, status: number, body: unknown): Response {
  return {
    ok,
    status,
    json: vi.fn().mockResolvedValue(body),
  } as unknown as Response;
}

describe("enableCloudModelsOptIn", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends the backend contract and verifies the persisted opt-in", async () => {
    const fetcher = vi.fn().mockResolvedValue(response(true, 200, { disabled: false }));

    await expect(enableCloudModelsOptIn(fetcher)).resolves.toBeUndefined();
    expect(fetcher).toHaveBeenCalledWith(`${API_BASE}/api/v1/cloud`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled: true }),
    });
  });

  it("rejects non-2xx responses instead of silently continuing", async () => {
    const fetcher = vi.fn().mockResolvedValue(response(false, 500, { error: "store unavailable" }));
    await expect(enableCloudModelsOptIn(fetcher)).rejects.toThrow("HTTP 500");
  });

  it("rejects a success response that does not confirm cloud enabled", async () => {
    const fetcher = vi.fn().mockResolvedValue(response(true, 200, { disabled: true }));
    await expect(enableCloudModelsOptIn(fetcher)).rejects.toThrow("não confirmou a ativação");
  });
});
