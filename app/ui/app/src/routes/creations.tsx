import { createFileRoute } from "@tanstack/react-router";
import { CreationsPage } from "@/components/CreationsPage";

export const Route = createFileRoute("/creations")({
  component: CreationsPage,
});
