#!/usr/bin/env python3
"""Confere a ESCRITURAÇÃO da missão: nenhuma afirmação sem lastro (E-180).

Por que este script existe
--------------------------
A seção 3 das diretrizes exige que toda capacidade promovida tenha **evidência anexada ao índice**. Isso só vale
se alguém conferir: um estágio pode citar `E-999` que nunca existiu, dois estágios podem disputar o mesmo ID, um
bloqueio pode ser referenciado sem estar declarado — e nada disso apareceria como falha. Enquanto o índice de
evidências for mantido à mão (e ele é, por ser prosa), o mínimo é **verificar a estrutura dele por máquina**.

O que é conferido
-----------------
1. `HADES_EVIDENCE_INDEX.md`: linhas `| E-NNN |` com IDs **únicos** e **sem buracos** a partir de E-001.
2. Toda evidência citada por um estágio (`state.stages[].evidence`) **existe** no índice.
3. Todo `blocker` citado por um estágio (`state.stages[].blocker`) **existe** na lista de bloqueios.
4. `execution` de cada estágio está no conjunto permitido pela seção 7, e nenhum estágio ficou sem o campo.
5. O veredito global é permitido e coerente: `COMPLETED_VERIFIED` exige que **nenhum** requisito da matriz esteja
   pendente — o script falha se alguém declarar release completo com item em aberto.
6. O gate gerado continua consistente (`scripts/hades_release_gate.py --check`).

Uso
---
    python scripts/hades_mission_check.py
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ESTADO_PADRAO = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
INDICE_PADRAO = RAIZ / "docs/mission/HADES_EVIDENCE_INDEX.md"

EXECUCOES_PERMITIDAS = {
    "NOT_STARTED",
    "IN_PROGRESS",
    "TESTING",
    "FAILED",
    "BLOCKED_BY_EXTERNAL_DEPENDENCY",
    "COMPLETED_VERIFIED",
}
VEREDITOS_PERMITIDOS = {"NOT_RELEASE_READY", "RELEASE_CANDIDATE", "COMPLETED_VERIFIED"}


def verificar(estado_path: Path, indice_path: Path, conferir_gate: bool = True) -> tuple[list[str], list[str], dict]:
    """Devolve (problemas, avisos, resumo). Aceita caminhos para ser TESTÁVEL com fixtures."""
    problemas: list[str] = []
    avisos: list[str] = []

    estado = json.loads(estado_path.read_text(encoding="utf-8"))
    indice = indice_path.read_text(encoding="utf-8")

    # 1. IDs do índice.
    ids = re.findall(r"^\|\s*(E-\d+)\s*\|", indice, re.M)
    if not ids:
        problemas.append("índice de evidências sem nenhuma linha `| E-NNN |`")
    duplicados = sorted({i for i in ids if ids.count(i) > 1})
    if duplicados:
        problemas.append(f"IDs de evidência DUPLICADOS no índice: {', '.join(duplicados)}")
    numeros = sorted(int(i.split("-")[1]) for i in ids)
    if numeros:
        esperado = list(range(1, max(numeros) + 1))
        faltando = sorted(set(esperado) - set(numeros))
        if faltando:
            problemas.append(
                "buracos na numeração do índice (IDs ausentes): "
                + ", ".join(f"E-{n:03d}" for n in faltando[:12])
                + ("..." if len(faltando) > 12 else "")
            )
    conjunto = set(ids)

    # 2 e 3. Referências dos estágios.
    ids_de_bloqueio = {b.get("id") for b in estado.get("blockers", [])}
    for estagio in estado["stages"]:
        ident = estagio.get("id")
        for evidencia in estagio.get("evidence") or []:
            if evidencia not in conjunto:
                problemas.append(f"estágio {ident} cita evidência que NÃO existe no índice: {evidencia}")
        bloqueio = estagio.get("blocker")
        if bloqueio and bloqueio not in ids_de_bloqueio:
            problemas.append(f"estágio {ident} cita bloqueio que NÃO está declarado: {bloqueio}")

    # 4. Campos de execução/estado.
    for estagio in estado["stages"]:
        ident = estagio.get("id")
        execucao = estagio.get("execution")
        if not execucao:
            problemas.append(f"estágio {ident} SEM o campo `execution` exigido pela seção 7")
        elif execucao not in EXECUCOES_PERMITIDAS:
            problemas.append(f"estágio {ident} tem `execution` fora do permitido: {execucao}")
        if not estagio.get("state"):
            avisos.append(f"estágio {ident} sem `state` legível")

    # 5. Veredito coerente com o estado.
    veredito = estado.get("release_state")
    if veredito not in VEREDITOS_PERMITIDOS:
        problemas.append(f"`release_state` fora do permitido: {veredito}")
    if veredito == "COMPLETED_VERIFIED":
        pendentes = [
            e.get("id")
            for e in estado["stages"]
            if e.get("execution") not in ("COMPLETED_VERIFIED",)
        ]
        if pendentes:
            problemas.append(
                "veredito COMPLETED_VERIFIED com estágios fora de COMPLETED_VERIFIED: "
                + ", ".join(str(p) for p in pendentes[:12])
            )
        externos = [b for b in estado.get("blockers", []) if "EXTERNO" in (b.get("tipo") or "").upper()]
        if externos:
            problemas.append(
                "veredito COMPLETED_VERIFIED com bloqueio externo em aberto: "
                + ", ".join(str(b.get("id")) for b in externos)
            )

    # 6. Gate gerado consistente.
    if conferir_gate:
        resultado = subprocess.run(
            [sys.executable, str(RAIZ / "scripts/hades_release_gate.py"), "--check"],
            capture_output=True,
            text=True,
        )
        if resultado.returncode != 0:
            problemas.append("gate gerado DIVERGENTE do estado: " + (resultado.stdout or "").strip())

    resumo = {
        "evidencias": len(ids),
        "ultimo_id": max(numeros) if numeros else 0,
        "estagios": len(estado["stages"]),
        "bloqueios": len(ids_de_bloqueio),
        "veredito": veredito,
    }
    return problemas, avisos, resumo


def main() -> int:
    parser = argparse.ArgumentParser(description="Confere a escrituração da missão.")
    parser.add_argument("--state", type=Path, default=ESTADO_PADRAO)
    parser.add_argument("--index", type=Path, default=INDICE_PADRAO)
    parser.add_argument("--no-gate-check", action="store_true", help="não confere o gate gerado")
    args = parser.parse_args()

    problemas, avisos, resumo = verificar(args.state, args.index, conferir_gate=not args.no_gate_check)

    print(f"índice: {resumo['evidencias']} evidências (E-001..E-{resumo['ultimo_id']:03d})")
    print(f"estágios: {resumo['estagios']} · bloqueios declarados: {resumo['bloqueios']}")
    print(f"veredito: {resumo['veredito']}")
    if avisos:
        print("avisos:")
        for aviso in avisos:
            print("  -", aviso)
    if problemas:
        print("PROBLEMAS:")
        for problema in problemas:
            print("  -", problema)
        return 1
    print("ESCRITURAÇÃO CONSISTENTE: nenhuma evidência fantasma, nenhum bloqueio órfão, veredito coerente.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
