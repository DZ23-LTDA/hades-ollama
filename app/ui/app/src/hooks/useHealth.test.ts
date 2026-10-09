import { describe, expect, it } from "vitest";

import { healthRefetchInterval, HEALTH_RECHECK_INTERVAL_MS } from "./useHealth";

describe("healthRefetchInterval", () => {
  it("re-checks a down backend on a sane interval, not a tight loop", () => {
    expect(healthRefetchInterval(false)).toBe(HEALTH_RECHECK_INTERVAL_MS);
    // Guard against regressing to the old 10ms busy-poll.
    expect(HEALTH_RECHECK_INTERVAL_MS).toBeGreaterThanOrEqual(1000);
  });

  it("stops polling once the backend is healthy or still unknown", () => {
    expect(healthRefetchInterval(true)).toBe(false);
    expect(healthRefetchInterval(undefined)).toBe(false);
  });
});
