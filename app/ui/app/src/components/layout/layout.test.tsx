import { act, create } from "react-test-renderer";
import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SidebarLayout } from "./layout";

vi.mock("@tanstack/react-router", () => ({
	Link: ({ children, ...props }: React.PropsWithChildren<Record<string, unknown>>) => (
		<a {...props}>{children}</a>
	),
}));

describe("SidebarLayout", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

	it("keeps the macOS title offset in step with the open sidebar", () => {
    vi.stubGlobal("window", { OLLAMA_PLATFORM: "darwin" });

    const html = renderToStaticMarkup(
      <SidebarLayout title="Connect your apps" sidebar={<nav />}>
        <div />
      </SidebarLayout>,
    );

	expect(html).toContain("pl-6");
    expect(html).toContain("transition-[padding-left]");
		expect(html).toContain("duration-300");
	});

	it("opens as a mobile drawer and closes from backdrop and Escape", () => {
		const listeners = new Map<string, EventListener>();
		vi.stubGlobal("window", {
			innerWidth: 390,
			OLLAMA_PLATFORM: "darwin",
			addEventListener: (type: string, listener: EventListener) => listeners.set(type, listener),
			removeEventListener: (type: string) => listeners.delete(type),
		});

		let renderer!: ReturnType<typeof create>;
		act(() => {
			renderer = create(
				<SidebarLayout sidebar={<nav aria-label="Navegação" />}>
					<section />
				</SidebarLayout>,
			);
		});

		const showButton = renderer.root.findByProps({ "aria-label": "Show sidebar" });
		act(() => showButton.props.onClick());
		expect(renderer.root.findByProps({ "aria-label": "Hide sidebar" })).toBeDefined();
		expect(renderer.root.findByProps({ "aria-label": "Fechar menu" })).toBeDefined();

		act(() => renderer.root.findByProps({ "aria-label": "Fechar menu" }).props.onClick());
		expect(renderer.root.findByProps({ "aria-label": "Show sidebar" })).toBeDefined();

		act(() => renderer.root.findByProps({ "aria-label": "Show sidebar" }).props.onClick());
		act(() => (listeners.get("keydown") as (event: KeyboardEvent) => void)({ key: "Escape" } as KeyboardEvent));
		expect(renderer.root.findByProps({ "aria-label": "Show sidebar" })).toBeDefined();
	});
});
