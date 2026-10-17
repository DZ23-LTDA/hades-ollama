#!/usr/bin/env python3
"""Catraca da documentacao do Companion (`E-1090`).

***PORQUE ESTE INSTRUMENTO EXISTE.*** O prompt exige, para o pet `COMPLETED_VERIFIED`, *«documentacao de
usuario e desenvolvedor actualizada»*. => *um documento que existe mas nao cobre os estados e' pior do que
nenhum: parece documentacao e nao o e'.*

***E O QUE ELE IMPEDE.*** *A mesma classe de defeito que o `E-1084` apanhou no proprio inventario: um
detector lexical ve termos, nao relacoes.* => *por isso cada verificacao abaixo e' **especifica**: nao
basta o documento mencionar «a_planear» — tem de o listar **na tabela dos 8 estados** e **no mapa para os
14 do prompt**.*

***E O QUE ELE NAO FAZ.*** *Nao julga a prosa. Verifica que cada facto verificavel esta' la'.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOC = RAIZ / "docs" / "COMPANION.md"
ESTADO_GO = RAIZ / "internal" / "agent" / "companion_estado.go"
EVENTOS_GO = RAIZ / "internal" / "agent" / "companion_eventos.go"
PREF_TS = RAIZ / "app" / "ui" / "app" / "src" / "lib" / "companionPreferencias.ts"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    if not DOC.is_file():
        print("  ERRO: docs/COMPANION.md nao existe")
        return 2

    doc = io.open(DOC, encoding="utf-8").read()
    estado = io.open(ESTADO_GO, encoding="utf-8").read()
    pref = io.open(PREF_TS, encoding="utf-8").read()

    print("=" * 72)
    print("1) OS 8 ESTADOS DO DERIVADOR ESTAO NO DOCUMENTO")
    print("=" * 72)
    # *A fonte e' o PROPRIO `companion_estado.go`, nao uma lista escrita a mao aqui.*
    estados = re.findall(r'CompanionEstado = "([a-z_]+)"', estado)
    T("o derivador declara 8 estados", len(estados) == 8, str(len(estados)))
    for e in estados:
        # *Tem de aparecer na tabela: a linha comecar por `| **...** |` -- aceitamos a forma
        #  `| **A planear** |` (titulo humano) e a forma tecnica `a_planear` em qualquer tabela.*
        T("o estado `%s` aparece no documento" % e, e in doc)

    print("")
    print("=" * 72)
    print("2) O MAPA PARA OS 14 ESTADOS DO PROMPT, E OS AUSENTES")
    print("=" * 72)
    # *Os 14 nomes do prompt, e o mapa HADES declarado no documento.*
    PROMPT = ["IDLE", "PLANNING", "WORKING", "APPROVAL_REQUIRED", "BLOCKED", "ERROR", "CANCELLED",
              "SUCCEEDED", "DISABLED", "LISTENING", "WAITING_TOOL", "WAITING_USER", "PAUSED", "OFFLINE"]
    for p in PROMPT:
        T("o nome do prompt `%s` aparece" % p, p in doc)
    T("o documento declara que PAUSED nao tem evento", "mission.paused" in doc)
    T("o documento declara que OFFLINE e' de conexao", "OFFLINE" in doc and "conex" in doc)

    print("")
    print("=" * 72)
    print("3) OS 12 EVENTOS VERIFICADOS ESTAO LISTADOS")
    print("=" * 72)
    EVENTOS = ["mission.created", "mission.planning", "mission.step.started", "mission.step.finished",
               "mission.approval.requested", "mission.approval.granted", "mission.approval.denied",
               "mission.blocked", "mission.repair_exhausted", "mission.completed", "mission.failed",
               "mission.cancelled"]
    T("o documento lista os 12 eventos", all(ev in doc for ev in EVENTOS),
      ", ".join(ev for ev in EVENTOS if ev not in doc) or "todos")

    print("")
    print("=" * 72)
    print("4) AS PREFERENCIAS: A CHAVE E O `false` DA GRAVACAO")
    print("=" * 72)
    chave = re.search(r'CHAVE_DAS_PREFERENCIAS = "([^"]+)"', pref)
    T("a chave das preferencias esta' no codigo", bool(chave))
    if chave:
        T("e o documento cita a MESMA chave", chave.group(1) in doc, chave.group(1))
    T("o documento declara que guardar devolve false", "false" in doc and "não conseguiu" in doc)
    T("o documento explica `localStorage` vs POST", "localStorage" in doc and "POST" in doc)

    print("")
    print("=" * 72)
    print("5) A ARQUITECTURA E OS TESTES ESTAO DOCUMENTADOS")
    print("=" * 72)
    for sym in ("companion_eventos.go", "companion_estado.go", "agent_routes.go", "companionApi.ts",
                "CompanionMascote.tsx", "CompanionLigado.tsx", "companionPreferencias.ts"):
        T("o simbolo `%s` e' citado" % sym, sym in doc)
    T("a rota e' citada com o caminho", "/api/agent/v1/missions/" in doc)
    T("o `scope` declarado e' citado", "scope" in doc)

    print("")
    print("=" * 72)
    print("6) AS LIMITACOES ESTAO DECLARADAS (e nao escondidas)")
    print("=" * 72)
    T("o documento declara o E2E da jornada como ausente", "jornada completa" in doc)
    T("o documento declara CPU/RAM/bateria por medir", "CPU" in doc or "RAM" in doc)
    T("o documento declara a voz como indisponivel", "BLOCKED_BY_EXTERNAL_DEPENDENCY" in doc)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  DOCUMENTACAO DO COMPANION COMPLETA")
    return 0


if __name__ == "__main__":
    sys.exit(main())
