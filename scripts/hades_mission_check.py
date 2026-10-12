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
RELATORIO_PADRAO = RAIZ / "docs/mission/HADES_STAGE_REPORT.md"
BLOQUEIOS_PADRAO = RAIZ / "docs/mission/HADES_BLOCKERS.md"

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


def verificar_relatorio(relatorio_path: Path, conjunto_ids: set[str]) -> tuple[list[str], dict]:
    """Confere a NUMERAÇÃO do relatório de estágios e as evidências que ele cita (E-181).

    O relatório é prosa, mas a numeração é ÍNDICE: item duplicado ou faltando torna impossível citar uma rodada
    sem ambiguidade. E uma evidência citada no relatório que não existe no índice é afirmação sem lastro.
    """
    problemas: list[str] = []
    texto = relatorio_path.read_text(encoding="utf-8")
    linhas = texto.splitlines()

    numeros: list[int] = []
    for linha in linhas:
        casamento = re.match(r"^(\d+)\. \*\*", linha)
        if casamento:
            numeros.append(int(casamento.group(1)))

    if not numeros:
        problemas.append("relatório de estágios sem nenhum item numerado (`N. **`)")
        return problemas, {"itens": 0, "ultimo": 0}

    duplicados = sorted({n for n in numeros if numeros.count(n) > 1})
    if duplicados:
        problemas.append(
            "itens DUPLICADOS no relatório de estágios: "
            + ", ".join(str(n) for n in duplicados)
            + " (numeração ambígua: duas rodadas com o mesmo índice)"
        )
    buracos = sorted(set(range(1, max(numeros) + 1)) - set(numeros))
    if buracos:
        problemas.append(
            "buracos na numeração do relatório (itens ausentes): "
            + ", ".join(str(n) for n in buracos[:12])
            + ("..." if len(buracos) > 12 else "")
        )

    # ID entre crases e MENC AO (exemplo/identificador), nao CITACAO: no proprio relatorio isso acontece quando
    # um item explica o que o verificador pega e cita "E-999" como exemplo. Sem esta regra, a explicacao vira
    # falso positivo - foi o que a primeira execucao mostrou.
    CRASE = chr(96)
    sem_crase = re.sub(CRASE + r"[^" + CRASE + r"]*" + CRASE, " ", texto)
    citadas = sorted(set(re.findall(r"\bE-\d+\b", sem_crase)))
    fantasmas = [e for e in citadas if e not in conjunto_ids]
    if fantasmas:
        problemas.append(
            "relatório cita evidências que NÃO existem no índice: " + ", ".join(fantasmas[:12])
        )
    return problemas, {"itens": len(numeros), "ultimo": max(numeros), "citadas": len(citadas)}


def verificar_bloqueios(bloqueios_path: Path, ids_do_estado: set[str]) -> tuple[list[str], dict]:
    """Confere o documento de bloqueios contra o estado (E-181), nos DOIS sentidos.

    Bloqueio declarado e ausente do documento é impedimento que ninguém lê; linha no documento sem bloqueio no
    estado é fantasma. Os dois casos são divergência silenciosa.
    """
    problemas: list[str] = []
    texto = bloqueios_path.read_text(encoding="utf-8")
    no_documento = set(re.findall(r"\|\s*(B-\d+)\s*\|", texto))

    so_no_estado = sorted(ids_do_estado - no_documento)
    if so_no_estado:
        problemas.append(
            "bloqueios declarados no ESTADO e AUSENTES de HADES_BLOCKERS.md: " + ", ".join(so_no_estado)
        )
    so_no_documento = sorted(no_documento - ids_do_estado)
    if so_no_documento:
        problemas.append(
            "bloqueios listados em HADES_BLOCKERS.md e INEXISTENTES no estado: " + ", ".join(so_no_documento)
        )
    return problemas, {"no_documento": len(no_documento), "no_estado": len(ids_do_estado)}


def main() -> int:
    parser = argparse.ArgumentParser(description="Confere a escrituração da missão.")
    parser.add_argument("--state", type=Path, default=ESTADO_PADRAO)
    parser.add_argument("--index", type=Path, default=INDICE_PADRAO)
    parser.add_argument("--report", type=Path, default=RELATORIO_PADRAO)
    parser.add_argument("--blockers", type=Path, default=BLOQUEIOS_PADRAO)
    parser.add_argument("--no-gate-check", action="store_true", help="não confere o gate gerado")
    args = parser.parse_args()

    problemas, avisos, resumo = verificar(args.state, args.index, conferir_gate=not args.no_gate_check)

    # 7. Relatório de estágios (numeração + evidências citadas).
    estado = json.loads(args.state.read_text(encoding="utf-8"))
    indice = args.index.read_text(encoding="utf-8")
    conjunto_ids = set(re.findall(r"^\|\s*(E-\d+)\s*\|", indice, re.M))
    problemas_relatorio, resumo_relatorio = verificar_relatorio(args.report, conjunto_ids)
    problemas += problemas_relatorio

    # 8. Documento de bloqueios ⇄ estado (nos dois sentidos).
    ids_do_estado = {b.get("id") for b in estado.get("blockers", [])}
    problemas_bloqueios, resumo_bloqueios = verificar_bloqueios(args.blockers, ids_do_estado)
    problemas += problemas_bloqueios

    print(f"índice: {resumo['evidencias']} evidências (E-001..E-{resumo['ultimo_id']:03d})")
    print(f"estágios: {resumo['estagios']} · bloqueios declarados: {resumo['bloqueios']}")
    print(f"relatório: {resumo_relatorio['itens']} itens (até {resumo_relatorio['ultimo']}) · "
          f"{resumo_relatorio.get('citadas', 0)} evidências citadas")
    print(f"bloqueios no documento: {resumo_bloqueios['no_documento']}")
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
