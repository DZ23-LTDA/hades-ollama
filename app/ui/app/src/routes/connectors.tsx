import { createFileRoute } from "@tanstack/react-router";
import { ConnectorsPage } from "@/components/ConnectorsPage";

export const Route = createFileRoute("/connectors")({
  component: ConnectorsPage,
});
