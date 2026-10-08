import { describe, expect, it } from "vitest";

import { DEFAULT_FETCH_TIMEOUT_MS, timeoutSignal } from "./fetchTimeout";

describe("timeoutSignal", () => {
  it("returns an abort signal that is not aborted yet", () => {
    const signal = timeoutSignal(10_000);
    expect(signal).toBeInstanceOf(AbortSignal);
    expect(signal?.aborted).toBe(false);
  });

  it("uses a sane, generous default", () => {
    expect(DEFAULT_FETCH_TIMEOUT_MS).toBeGreaterThanOrEqual(5000);
  });

  it("aborts after the timeout elapses", async () => {
    const signal = timeoutSignal(5);
    await new Promise((resolve) => setTimeout(resolve, 40));
    expect(signal?.aborted).toBe(true);
  });
});
