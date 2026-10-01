import { createFileRoute } from "@tanstack/react-router";
import { StudioCanvasPage } from "@/components/StudioCanvasPage";

export const Route = createFileRoute("/studio")({
  component: StudioCanvasPage,
});
