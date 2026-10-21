import { createFileRoute } from "@tanstack/react-router";
import { CapabilitiesPage } from "@/components/CapabilitiesPage";

// FASE 28.8 do adendo V3: a area "Capacidades e Ferramentas" no Mission Control.
// A rota fica no menu unico (seccao Conteudo), nao num painel paralelo.
export const Route = createFileRoute("/capabilities")({
  component: CapabilitiesPage,
});
