import { describe, expect, it } from "vitest";

import { userRetryDelay } from "./useUser";

describe("userRetryDelay", () => {
  it("does not retry immediately on the first failure", () => {
    // Guard against the old 0ms first retry (500 * attemptIndex with 0-based index).
    expect(userRetryDelay(0)).toBeGreaterThan(0);
    expect(userRetryDelay(0)).toBe(500);
  });

  it("backs off and caps at 2000ms", () => {
    expect(userRetryDelay(1)).toBe(1000);
    expect(userRetryDelay(3)).toBe(2000);
    expect(userRetryDelay(10)).toBe(2000);
  });
});
