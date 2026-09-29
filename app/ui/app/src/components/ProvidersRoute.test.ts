import { describe, expect, it } from "vitest";
import { ProvidersPage } from "@/components/ProvidersPage";
import { Route } from "@/routes/providers";

describe("/providers route", () => {
  it("renders the Provedores de IA page", () => {
    expect(Route.options.component).toBe(ProvidersPage);
  });
});
