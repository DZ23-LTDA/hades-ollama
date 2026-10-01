import { createFileRoute } from "@tanstack/react-router";
import { ProvidersPage } from "@/components/ProvidersPage";

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
});
