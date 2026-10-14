import {
  act,
  create,
  type ReactTestInstance,
  type ReactTestRenderer,
} from "react-test-renderer";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { AgentMission } from "@/lib/agenticClient";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  createMission: vi.fn(),
  listMissions: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    children,
    className,
  }: {
    to: string;
    children: React.ReactNode;
    className?: string;
  }) => (
    <a href={to} className={className}>
      {children}
    </a>
  ),
  useNavigate: () => mocks.navigate,
}));

vi.mock("@/components/layout/layout", () => ({
  SidebarLayout: ({ children }: { children: React.ReactNode }) => (
    <main>{children}</main>
  ),
}));

vi.mock("@/components/AppSidebar", () => ({
  AppSidebar: () => <nav />,
}));

vi.mock("@/lib/agenticClient", async (importOriginal) =>
  Object.assign(
    {},
    await importOriginal<typeof import("@/lib/agenticClient")>(),
    {
      createMission: mocks.createMission,
      listMissions: mocks.listMissions,
    },
  ),
);

import { NewTaskPage } from "./NewTaskPage";
import { ProductWorkspacePage } from "./ProductWorkspacePage";

function textContent(node: ReactTestInstance): string {
  return node.children
    .map((child) => (typeof child === "string" ? child : textContent(child)))
    .join("");
}

function findButton(root: ReactTestInstance, label: string) {
  return root.findAll(
    (node) => node.type === "button" && textContent(node).includes(label),
  )[0];
}

function mission(overrides: Partial<AgentMission> = {}): AgentMission {
  return {
    id: "mission-1",
    version: 1,
    objective: "Revisar o README",
    state: "PLANNED",
    plan: [],
    artifacts: [],
    created_at: "2026-09-29T12:00:00Z",
    updated_at: "2026-09-29T12:00:00Z",
    ...overrides,
  };
}

async function flush() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}

let renderer: ReactTestRenderer | undefined;

beforeEach(() => {
  vi.stubGlobal("IS_REACT_ACT_ENVIRONMENT", true);
  mocks.navigate.mockResolvedValue(undefined);
});

afterEach(async () => {
  if (renderer) await act(async () => renderer!.unmount());
  renderer = undefined;
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

describe("Nova tarefa", () => {
  async function renderNewTask() {
    await act(async () => {
      renderer = create(<NewTaskPage />);
    });
    return renderer!.root;
  }

  async function typeObjective(root: ReactTestInstance, value: string) {
    await act(async () => {
      root
        .findByProps({ id: "new-task-objective" })
        .props.onChange({ target: { value } });
    });
  }

  async function submit(root: ReactTestInstance) {
    await act(async () => {
      root.findByType("form").props.onSubmit({ preventDefault: vi.fn() });
    });
  }

  it("blocks an empty objective without calling the runtime", async () => {
    const root = await renderNewTask();
    await typeObjective(root, "   ");
    await submit(root);

    expect(mocks.createMission).not.toHaveBeenCalled();
    const textarea = root.findByProps({ id: "new-task-objective" });
    expect(textarea.props["aria-invalid"]).toBe(true);
    expect(textContent(root)).toContain("Descreva o objetivo da tarefa");
  });

  it("shows the pending state and navigates to Tarefas with the created mission id", async () => {
    let resolveCreate!: (value: AgentMission) => void;
    mocks.createMission.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        }),
    );
    const root = await renderNewTask();
    await typeObjective(root, "  Revisar o README  ");
    await submit(root);

    expect(mocks.createMission).toHaveBeenCalledExactlyOnceWith({
      objective: "Revisar o README",
      capabilities: ["workspace:read"],
      auto_run: false,
    });
    const busyButton = root.findByProps({ type: "submit" });
    expect(busyButton.props.disabled).toBe(true);
    expect(busyButton.props["aria-busy"]).toBe(true);
    expect(textContent(busyButton)).toBe("Criando tarefa…");
    expect(mocks.navigate).not.toHaveBeenCalled();

    // A second submit while pending must not create a duplicate mission.
    await submit(root);
    expect(mocks.createMission).toHaveBeenCalledOnce();

    await act(async () => {
      resolveCreate(mission({ id: "mission-42" }));
    });
    expect(mocks.navigate).toHaveBeenCalledExactlyOnceWith({
      to: "/tasks",
      search: { created: "mission-42" },
    });
  });

  it("keeps the objective and announces the runtime error without navigating", async () => {
    mocks.createMission.mockRejectedValue(new Error("objective is too long"));
    const root = await renderNewTask();
    await typeObjective(root, "Revisar o README");
    await submit(root);

    const alert = root.findByProps({ role: "alert" });
    expect(alert.props["aria-live"]).toBe("assertive");
    expect(textContent(alert)).toContain("A tarefa não foi criada.");
    expect(textContent(alert)).toContain("objective is too long");
    expect(root.findByProps({ id: "new-task-objective" }).props.value).toBe(
      "Revisar o README",
    );
    expect(textContent(root.findByProps({ type: "submit" }))).toBe(
      "Tentar novamente",
    );
    expect(mocks.navigate).not.toHaveBeenCalled();
  });
});

describe("Tarefas", () => {
  async function renderTasks(createdMissionId?: string) {
    await act(async () => {
      renderer = create(
        <ProductWorkspacePage
          kind="tasks"
          createdMissionId={createdMissionId}
        />,
      );
    });
    return renderer!.root;
  }

  it("shows loading, then the empty state that opens Nova tarefa", async () => {
    let resolveList!: (value: { missions: AgentMission[] }) => void;
    mocks.listMissions.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveList = resolve;
        }),
    );
    const root = await renderTasks();

    const loading = root.findByProps({ "aria-busy": "true" });
    expect(loading.props.role).toBe("status");

    await act(async () => resolveList({ missions: [] }));
    expect(textContent(root)).toContain("Nenhuma tarefa criada ainda.");

    const emptyAction = root
      .findAll(
        (node) =>
          node.type === "button" && textContent(node).includes("Nova tarefa"),
      )
      .at(-1)!;
    await act(async () => emptyAction.props.onClick());
    expect(mocks.navigate).toHaveBeenCalledWith({ to: "/tasks/new" });
  });

  it("shows the load error with a retry that queries the runtime again", async () => {
    mocks.listMissions
      .mockRejectedValueOnce(new Error("runtime unavailable"))
      .mockResolvedValueOnce({ missions: [mission()] });
    const root = await renderTasks();
    await flush();

    const alert = root.findByProps({ role: "alert" });
    expect(textContent(alert)).toContain("runtime unavailable");

    await act(async () => findButton(root, "Tentar novamente").props.onClick());
    await flush();
    expect(mocks.listMissions).toHaveBeenCalledTimes(2);
    expect(textContent(root)).toContain("Revisar o README");
  });

  it("confirms the created mission only when the persisted listing returns it", async () => {
    mocks.listMissions.mockResolvedValue({
      missions: [
        mission({ id: "other", objective: "Outra" }),
        mission({
          id: "mission-42",
          objective: "Revisar o README",
          state: "PLANNED",
        }),
      ],
    });
    const root = await renderTasks("mission-42");
    await flush();

    const banner = root.findByProps({
      role: "status",
      "aria-live": "polite",
      tabIndex: -1,
    });
    expect(textContent(banner)).toContain("Tarefa criada");
    expect(textContent(banner)).toContain("Revisar o README · estado PLANNED");
    const current = root.findAll(
      (node) => node.type === "li" && node.props["aria-current"] === "true",
    );
    expect(current).toHaveLength(1);
    expect(textContent(current[0])).toContain("mission-42");
  });

  it("reports a created id that is missing from the listing instead of assuming success", async () => {
    mocks.listMissions.mockResolvedValue({ missions: [] });
    const root = await renderTasks("mission-404");
    await flush();

    expect(textContent(root)).not.toContain("Tarefa criada");
    const warning = root.findByProps({ role: "alert", tabIndex: -1 });
    expect(textContent(warning)).toContain(
      "A tarefa criada não apareceu na listagem",
    );
    expect(textContent(warning)).toContain("mission-404");
  });
});
