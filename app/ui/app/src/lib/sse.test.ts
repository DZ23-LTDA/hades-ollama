import { describe, expect, it } from "vitest";

import { MAX_SSE_RETRIES, shouldStopSseReconnect } from "./sse";

describe("shouldStopSseReconnect", () => {
  it("keeps reconnecting below the cap", () => {
    expect(shouldStopSseReconnect(0)).toBe(false);
    expect(shouldStopSseReconnect(MAX_SSE_RETRIES - 1)).toBe(false);
  });

  it("stops once the cap is reached, to avoid a reconnect storm", () => {
    expect(shouldStopSseReconnect(MAX_SSE_RETRIES)).toBe(true);
    expect(shouldStopSseReconnect(MAX_SSE_RETRIES + 3)).toBe(true);
  });

  it("uses a sane finite cap", () => {
    expect(MAX_SSE_RETRIES).toBeGreaterThanOrEqual(1);
    expect(MAX_SSE_RETRIES).toBeLessThanOrEqual(20);
  });
});
