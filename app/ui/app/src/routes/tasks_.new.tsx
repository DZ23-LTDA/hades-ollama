import { createFileRoute } from "@tanstack/react-router";
import { NewTaskPage } from "@/components/NewTaskPage";

export const Route = createFileRoute("/tasks_/new")({
  component: NewTaskPage,
});
