import { createFileRoute } from "@tanstack/react-router";

import { LiveBuildPage } from "@/components/LiveBuildPage";

export const Route = createFileRoute("/build")({
  component: LiveBuildPage,
});
