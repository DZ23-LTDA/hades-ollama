import { describe, expect, it } from "vitest";

import { Route as RootRoute } from "@/routes/__root";
import { RouteErrorFallback } from "@/components/RouteErrorFallback";

describe("root route error boundary", () => {
  it("registers the friendly route-level error fallback", () => {
    // Without this, a failing route falls back to TanStack's bare default error
    // component instead of our recoverable RouteErrorFallback.
    expect(RootRoute.options.errorComponent).toBe(RouteErrorFallback);
  });
});
