import { createFileRoute } from "@tanstack/react-router";
import { SigningIdentitiesPage } from "@/components/SigningIdentitiesPage";

// FASE 28.5 do adendo V3: o Signing Identity Center no Mission Control.
// Le apenas metadados; o valor privado nunca atravessa esta rota.
export const Route = createFileRoute("/signing")({
  component: SigningIdentitiesPage,
});
