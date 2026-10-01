import { createFileRoute } from "@tanstack/react-router";
import { EndpointPage } from "@/components/EndpointPage";

export const Route = createFileRoute("/endpoint")({
  component: EndpointPage,
});
