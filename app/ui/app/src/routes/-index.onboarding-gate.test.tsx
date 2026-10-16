import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as api from "@/api";
import { Settings } from "@/gotypes";
import { CURRENT_ONBOARDING_VERSION } from "@/lib/onboarding";
import { Route } from "@/routes/index";

// P0-4: "Decidir o gate de onboarding quando GET /settings falha".
// O critério de pronto pedia "decisão documentada + teste que fixe o
// comportamento escolhido (hoje há apenas o comportamento implícito)".
// A decisão esta' documentada em routes/index.tsx:17-20 (o shell local-first
// tem de continuar utilizavel quando a API de preferencias opcional esta'
// offline). Este ficheiro fixa o comportamento, nos tres casos.

const beforeLoad = Route.options.beforeLoad as (args: {
  context: { queryClient: QueryClient };
}) => Promise<unknown>;

function freshClient(): QueryClient {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity } },
  });
}

/** Corre o beforeLoad e devolve o destino do redirect (ou um marcador). */
async function destination(client: QueryClient): Promise<string> {
  try {
    await beforeLoad({ context: { queryClient: client } });
    return "SEM_REDIRECT";
  } catch (thrown) {
    // O TanStack Router lanca um objecto Redirect; a forma exacta nao se
    // adivinha - procura-se em todos os sitios plausiveis e, se nenhum servir,
    // devolvem-se as chaves para o proprio teste dizer onde esta'.
    const alvo = thrown as {
      to?: unknown;
      href?: unknown;
      options?: { to?: unknown; href?: unknown };
    };
    for (const candidato of [
      alvo?.to,
      alvo?.options?.to,
      alvo?.href,
      alvo?.options?.href,
    ]) {
      if (typeof candidato === "string") return candidato;
    }
    const chaves = thrown && typeof thrown === "object" ? Object.keys(thrown) : [];
    return `SEM_DESTINO chaves=[${chaves.join(",")}] options=[${
      alvo?.options && typeof alvo.options === "object"
        ? Object.keys(alvo.options).join(",")
        : "-"
    }]`;
  }
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("gate de onboarding em routes/index.tsx (P0-4)", () => {
  it("com GET /settings offline manda para o chat, NAO para o onboarding", async () => {
    vi.spyOn(api, "getSettings").mockRejectedValue(new Error("backend offline"));
    const to = await destination(freshClient());
    expect(to).not.toBe("/onboarding");
    expect(to).toBe("/c/$chatId");
  });

  it("com OnboardingVersion antiga manda para o onboarding", async () => {
    const settings = new Settings({ OnboardingVersion: 0 });
    vi.spyOn(api, "getSettings").mockResolvedValue({ settings });
    const to = await destination(freshClient());
    expect(to).toBe("/onboarding");
  });

  it("com OnboardingVersion actual manda para o chat", async () => {
    const settings = new Settings({
      OnboardingVersion: CURRENT_ONBOARDING_VERSION,
    });
    vi.spyOn(api, "getSettings").mockResolvedValue({ settings });
    const to = await destination(freshClient());
    expect(to).not.toBe("/onboarding");
    expect(to).toBe("/c/$chatId");
  });
});
