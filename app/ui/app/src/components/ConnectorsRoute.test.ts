import { describe, expect, it } from "vitest";
import { ConnectorsPage } from "@/components/ConnectorsPage";
import { Route } from "@/routes/connectors";

describe("/connectors route", () => {
  it("renders the Conectores page, not a generated placeholder", () => {
    expect(Route.options.component).toBe(ConnectorsPage);
  });
});
