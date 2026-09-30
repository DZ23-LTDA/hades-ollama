import { createFileRoute } from "@tanstack/react-router";
import { AutomationsPage } from "@/components/AutomationsPage";

export const Route = createFileRoute("/scheduled")({
  component: AutomationsPage,
});
