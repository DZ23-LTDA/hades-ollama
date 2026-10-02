import { expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

type Violation = {
  id: string;
  impact?: string | null;
  help: string;
  nodes: Array<{ target: string[] }>;
};

// Keep the assertion explicit and reusable in Playwright output. The cast keeps
// the project independent from Jest-specific matcher packages.
expect.extend({
  toHaveNoViolations(actual: Violation[]) {
    const pass = Array.isArray(actual) && actual.length === 0;
    const details = pass
      ? ""
      : actual
          .map(
            (violation) =>
              `${violation.id} (${violation.impact ?? "unknown"}): ${violation.help} [${violation.nodes
                .map((node) => node.target.join(" "))
                .join(", ")}]`,
          )
          .join("\n");
    return {
      pass,
      message: () =>
        pass
          ? "expected the page to have accessibility violations"
          : `Expected no axe violations, received:\n${details}`,
    };
  },
});

const routes = ["/", "/connect", "/endpoint", "/library", "/settings", "/agentic"];
const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 390, height: 844 },
] as const;

for (const viewport of viewports) {
  test.describe(`axe WCAG gate — ${viewport.name}`, () => {
    test(`all critical routes have no accessibility violations`, async ({ browser }) => {
      for (const route of routes) {
        const context = await browser.newContext({ viewport });
        const page = await context.newPage();
        const consoleErrors: string[] = [];
        const httpErrors: string[] = [];
        page.on("console", (message) => {
          if (message.type() !== "error") return;
          const text = message.text();
          // The shell-smoke E2E runs without a backend, so /api calls proxy-fail.
          // That offline behavior is asserted by shell.spec; ignore it here so the
          // a11y gate stays deterministic and tests accessibility, not backend
          // connectivity (otherwise it flakes on proxy 500/connection errors).
          if (text.includes("Failed to load resource")) return;
          consoleErrors.push(text);
        });
        page.on("response", (response) => {
          const url = response.url();
          if (response.status() >= 400 && !url.includes("favicon") && !url.includes("/api/")) {
            httpErrors.push(`${response.status()} ${response.request().method()} ${url}`);
          }
        });

        await page.goto(route, { waitUntil: "networkidle" });
        const results = await new AxeBuilder({ page }).analyze();
        const axeExpect = expect(results.violations) as unknown as {
          toHaveNoViolations: () => void;
        };
        axeExpect.toHaveNoViolations();
        expect(consoleErrors, `${viewport.name} ${route} console errors`).toEqual([]);
        expect(httpErrors, `${viewport.name} ${route} HTTP errors`).toEqual([]);
        await context.close();
      }
    });
  });
}

test("mobile shell is operable by keyboard without a trap", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  const menuButton = page.locator('button[aria-label*="sidebar" i]');
  await expect(menuButton).toHaveCount(1);
  if ((await menuButton.getAttribute("aria-label")) === "Hide sidebar") {
    await menuButton.focus();
    await page.keyboard.press("Enter");
    await expect(menuButton).toHaveAttribute("aria-label", "Show sidebar");
  }
  await menuButton.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("button", { name: "Fechar menu" })).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(page.getByRole("button", { name: "Fechar menu" })).toBeHidden();
  await expect(menuButton).toBeFocused();
});
