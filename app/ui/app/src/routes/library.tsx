import { createFileRoute } from "@tanstack/react-router";
import { LibraryPage } from "@/components/LibraryPage";

export const Route = createFileRoute("/library")({
  component: LibraryPage,
});
