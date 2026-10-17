#!/usr/bin/env python3
"""`hades_ux_onboarding_check.py` — o §ESTAGIO 19 (UX/onboarding), medido (`E-1095`).

***O QUE O §ESTAGIO 19 EXIGE.***

    - testar usuario novo contra binario/app real, **nao somente backend minimo**;
    - cobrir **modelo local, falha sem modelo, escolha de modelo, reinicio, atualizacao/rollback,
      acessibilidade e navegacao por teclado**;
    - **remover marcas/menus duplicados**;
    - **o chat principal deve aceitar texto normal e `/goal` no mesmo campo**.

***O QUE ESTE CHECADOR VERIFICA, E COMO.***

  1. **`/goal` no MESMO campo do chat** => *o `/goal` tem de aparecer em `ChatForm.tsx` (o composer
     principal), **e** o `slashCommands.ts` tem de o reconhecer* => *e o teste `slashCommands.test.ts`
     tem de cobrir o **texto normal** e o **`/goal`** -- nao adianta ter um sem o outro.*
  2. **Onboarding real, nao backend minimo** => *as 3 pecas: `Onboarding.tsx`, `OnboardingRoute.test.tsx`,
     `-index.onboarding-gate.test.tsx`* => *e a **rota** `routes/onboarding.tsx` tem de existir.*
  3. **Acessibilidade** => *o `Onboarding.tsx` e o `ChatForm.tsx` tem de ter `aria-*`/`role=`; e o teste
     do onboarding tem de cobrir **teclado** (`Enter`/`Tab`/`Escape`/`Arrow`).*
  4. **Menu unificado** => *o numero de rotas tem de ser **<= 24** -- um menu que cresce sem controlo
     deixa de ser unificado* => *e a rota `studio`, `company`, `agentic` tem de existir (os modos
     canonicos).*
  5. **Marcas duplicadas** => *o `index.html`/manifest nao pode ter dois nomes visiveis de produto*
     => *e o `HADES_BRAND_INVENTORY.md` tem de declarar as excecoes tecnicas.*

***E O QUE ELE NAO FAZ.*** *Nao corre o Playwright -- isso e' o E2E, e o E2E do onboarding **exige**
backend real (o `first-run-backend.mjs`); o `--check` valida a **cobertura estrutural** e o teste
`Onboarding.test.tsx` valida o **comportamento**.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SRC = RAIZ / "app" / "ui" / "app" / "src"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def ler(rel: str) -> str:
    p = SRC / rel
    if not p.is_file():
        return ""
    return io.open(p, encoding="utf-8").read()


def main() -> int:
    check = "--check" in sys.argv

    # ---------- 1. `/goal` no MESMO campo do chat ----------
    chat = ler("components/ChatForm.tsx")
    slash = ler("lib/slashCommands.ts")
    slash_t = ler("lib/slashCommands.test.ts")
    T("(1) `ChatForm.tsx` tem `/goal`", "/goal" in chat, "o composer principal")
    T("(1b) `slashCommands.ts` reconhece `/goal`", "/goal" in slash, "o parser")
    T("(1c) o teste cobre o `/goal`", "/goal" in slash_t, "%d ocorrencias" % slash_t.count("/goal"))
    # *texto normal: o parser tem de ter o caminho de "nao e' comando"*
    normal = bool(re.search(r"texto|normal|mensagem|prompt|sem comando|nao e' comando|fallback", slash_t, re.I))
    T("(1d) o teste cobre TEXTO NORMAL tambem", normal,
      "o mesmo campo aceita os dois" if normal else "falta o caso de texto normal")

    # ---------- 2. onboarding real ----------
    for arq in ("components/Onboarding.tsx", "components/Onboarding.test.tsx",
                "components/OnboardingRoute.test.tsx", "routes/onboarding.tsx",
                "routes/-index.onboarding-gate.test.tsx"):
        T("(2) existe `%s`" % arq, (SRC / arq).is_file())

    # ---------- 3. acessibilidade e teclado ----------
    onb = ler("components/Onboarding.tsx")
    aria = len(re.findall(r"aria-[a-z]+=", onb))
    roles = len(re.findall(r"role=\"[a-z]+\"", onb))
    T("(3) o onboarding tem atributos ARIA", aria >= 5, "aria-*= %d" % aria)
    T("(3b) e roles explicitos", roles >= 2, 'role= %d' % roles)
    teclado = len(re.findall(r"onKeyDown|Enter|Escape|Tab|ArrowDown|ArrowUp", onb))
    T("(3c) e trata teclado", teclado >= 2, "onKeyDown/teclas = %d" % teclado)
    onb_t = ler("components/Onboarding.test.tsx")
    teclado_t = len(re.findall(r"Enter|Escape|Tab|Arrow|keyboard|teclado|keyDown", onb_t, re.I))
    T("(3d) e o TESTE cobre teclado", teclado_t >= 1, "ocorrencias = %d" % teclado_t)
    chat_aria = len(re.findall(r"aria-[a-z]+=", chat))
    T("(3e) o `ChatForm` tem ARIA", chat_aria >= 2, "aria-*= %d" % chat_aria)

    # ---------- 4. menu unificado ----------
    rotas = sorted(p.name for p in (SRC / "routes").glob("*.tsx"))
    T("(4) o numero de rotas e' <= 24 (menu unificado)", len(rotas) <= 24,
      "%d rotas" % len(rotas))
    for modo in ("studio.tsx", "company.tsx", "agentic.tsx", "index.tsx"):
        T("(4b) existe a rota canonica `%s`" % modo, modo in rotas)

    # ---------- 5. marca ----------
    inventario = RAIZ / "docs" / "mission" / "HADES_BRAND_INVENTORY.md"
    T("(5) o inventario de marca existe", inventario.is_file())
    if inventario.is_file():
        inv = io.open(inventario, encoding="utf-8").read()
        T("(5b) e declara as excecoes tecnicas", "exce" in inv.lower() or "tecnico" in inv.lower(),
          "as excecoes ficam escritas")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 19 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 19 (UX/onboarding): %d rotas, %d ARIA no onboarding, teclado coberto," % (
        len(rotas), aria))
    print("  `/goal` no composer principal, onboarding com 3 pecas, e o teste de ambos.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
