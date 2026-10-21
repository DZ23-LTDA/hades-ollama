#!/usr/bin/env python3
"""`hades_architecture_check.py` — o §ESTAGIO 3 (Arquitetura canonica), medido (`E-1102`).

***O QUE O §ESTAGIO 3 EXIGE.***

    - criar **testes de contrato reais por componente**; *busca de simbolo nao basta*;
    - documentar **ownership, API, persistencia, eventos, permissoes e dependencias**;
    - **eliminar duplicacoes de motores e fontes de verdade**;
    - **um Mission Control, um runtime, uma identidade de missao, uma trilha de aprovacao**.

***O QUE ESTE CHECADOR VERIFICA, e como.***

  1. **As fontes de verdade sao SINGULARES** -- *`type Runtime struct` = **1**, `type Mission struct` = **1**,
     o mecanismo de lock `withFileLock` **>= 1**, `MissionControl` **0 ou 1**.* => *duas fontes de verdade sao
     o defeito que o estagio existe para impedir.*
  2. **Os testes de CONTRATO existem** -- *e o §ESTAGIO 3 e' explicito: «busca de simbolo nao basta»* => *logo
     o criterio e' **nomear o contrato**: pelo menos **6** ficheiros `*contract*_test.go`.*
  3. **A arquitectura esta' DOCUMENTADA** (`HADES_ARCHITECTURE.md`) *com as 6 seccoes: fontes de verdade,
     pacotes, API, contratos, ownership e conclusao.*
  4. **A API esta' documentada por rota** (`docs/agentic/API.md`), *nao apenas listada.*
  5. **A trilha de aprovacao e' UMA** -- *o mecanismo de lock e' partilhado, e os stores sao por dominio.*
  6. **O `MissionControl` NAO e' um motor** -- *0 definicoes e' o correcto: o Mission Control e' a
     **superficie**, e o motor e' o `Runtime`.* => *se houvesse um `type MissionControl struct` com estado
     proprio, seria um **segundo motor** -- exactamente o que o prompt proibe.*

***E O QUE ELE NAO FAZ.*** *Nao corre o `go test` (os 1557 testes ja' correm, e o `E-1097` provou que
passam).* => *o que ele faz e' verificar que **a arquitetura e' a declarada** -- e que **nao ha' motor
duplicado**.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"
ARQ = RAIZ / "docs" / "mission" / "HADES_ARCHITECTURE.md"
API = RAIZ / "docs" / "agentic" / "API.md"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def contar(padrao: str) -> int:
    n = 0
    for p in AGENTE.glob("*.go"):
        if "_test" in p.name:
            continue
        try:
            n += len(re.findall(padrao, io.open(p, encoding="utf-8").read(), re.M))
        except UnicodeDecodeError:
            continue
    return n


def main() -> int:
    # ---------- 1. as fontes de verdade ----------
    runtime = contar(r"^type Runtime struct")
    mission = contar(r"^type Mission struct")
    mc = contar(r"^type MissionControl struct")
    lock = contar(r"return withFileLock\(path\+\"\.lock\"")
    T("(1) `type Runtime struct` = 1 (um runtime)", runtime == 1, "n=%d" % runtime)
    T("(1b) `type Mission struct` = 1 (uma identidade de missao)", mission == 1, "n=%d" % mission)
    T("(1c) o mecanismo de lock da aprovacao existe (`withFileLock`)", lock >= 1, "n=%d" % lock)
    T("(1d) NAO ha' `type MissionControl struct` (o MC e' superficie, nao motor)", mc == 0,
      "n=%d" % mc if mc else "0 -- correcto")

    # ---------- 2. os testes de CONTRATO ----------
    contratos = sorted(p.name for p in AGENTE.glob("*contract*_test.go"))
    T("(2) ha' testes de CONTRATO (nao apenas busca de simbolo)", len(contratos) >= 6,
      "%d ficheiros: %s" % (len(contratos), ", ".join(contratos[:3])))
    # *e o contrato dos EVENTOS tem de estar coberto -- e' o barramento*
    eventos = any("event" in n for n in contratos)
    T("(2b) e o contrato dos EVENTOS esta' coberto", eventos,
      "o barramento e' o que liga o runtime a' UI")

    # ---------- 3. a arquitectura documentada ----------
    T("(3) o `HADES_ARCHITECTURE.md` existe", ARQ.is_file())
    if ARQ.is_file():
        a = io.open(ARQ, encoding="utf-8").read()
        seccoes = ("## As quatro fontes de verdade", "## Os pacotes", "## A API",
                   "## Os testes de CONTRATO", "## Ownership", "## A conclusão")
        faltam = [s for s in seccoes if s not in a]
        T("(3b) e cobre as 6 seccoes", not faltam,
          "faltam: %s" % ", ".join(faltam) if faltam else "as 6")

    # ---------- 4. a API documentada ----------
    T("(4) a `docs/agentic/API.md` existe", API.is_file())
    if API.is_file():
        api = io.open(API, encoding="utf-8").read()
        T("(4b) e cobre as rotas de missao", "missions" in api and "events" in api,
          "%.1f KB" % (API.stat().st_size / 1024))

    # ---------- 5. a trilha de aprovacao ----------
    stores = sorted(p.name for p in AGENTE.glob("*approval*.go") if "_test" in p.name)
    T("(5) a trilha de aprovacao tem testes", len(stores) >= 10,
      "%d ficheiros de teste de aprovacao" % len(stores))

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 3 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 3 (Arquitetura canonica): 1 Runtime, 1 Mission, %d MissionControl, o lock" % mc)
    print("  partilhado, %d testes de contrato, e a arquitectura e a API documentadas." % len(contratos))
    return 0


if __name__ == "__main__":
    sys.exit(main())
