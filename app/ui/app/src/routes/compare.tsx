import { createFileRoute } from "@tanstack/react-router";
import { ModelComparisonPage } from "@/components/ModelComparisonPage";

export const Route = createFileRoute("/compare")({
  component: ModelComparisonPage,
});
