import { createFileRoute } from "@tanstack/react-router";
import { ProductWorkspacePage } from "@/components/ProductWorkspacePage";

type TasksSearch = { created?: string };

export const Route = createFileRoute("/tasks")({
  validateSearch: (search: Record<string, unknown>): TasksSearch => {
    const created =
      typeof search.created === "string" ? search.created.trim() : "";
    return created ? { created } : {};
  },
  component: TasksRoute,
});

function TasksRoute() {
  const { created } = Route.useSearch();
  return <ProductWorkspacePage kind="tasks" createdMissionId={created} />;
}
