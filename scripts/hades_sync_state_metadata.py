#!/usr/bin/env python3
"""hades_sync_state_metadata.py — sincroniza §13 (medicao) no estado canonico, de forma CIRURGICA.

POR QUE ISTO EXISTE
-------------------
A secao 13 da missao exige que `docs/mission/HADES_MASTER_STATE.json` distinga **`measured_at`** de **commit
corrente**. Na pratica isso significa reescrever tres campos a cada fatia.

O jeito ERRADO e' `json.loads` + `json.dumps`: o estado tem ~1 MB e a re-serializacao reformata o ficheiro
inteiro (milhares de linhas de diff), muda a ordem/espacamento das chaves e **mistura uma alteracao semantica de
tres campos com uma reescrita do artefacto historico**. *Foi exactamente esse tipo de escrita descuidada que
produziu o incidente do BOM + dupla codificacao cp1252 no commit `6a391ef5c`.*

Este script faz **substituicao de texto ancorada no nome do campo**, com quatro travas:
  1. exige **exactamente uma** ocorrencia de cada campo (nem zero, nem duas);
  2. valida o JSON **antes** de gravar;
  3. confere que a contagem de chaves de topo **nao mudou**;
  4. relê do disco e confirma os tres valores.

Uso
---
    python scripts/hades_sync_state_metadata.py            # grava
    python scripts/hades_sync_state_metadata.py --check    # so' confere (nao grava); falha se divergir do HEAD

Depois de usar, regenere os derivados: `hades_release_gate.py`, `hades_baseline_report.py`, `hades_stage_map.py`.
"""

from __future__ import annotations

import argparse
import datetime
import json
import pathlib
import re
import subprocess
import sys

RAIZ = pathlib.Path(__file__).resolve().parent.parent
ESTADO = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
FUSO_BR = datetime.timezone(datetime.timedelta(hours=-3))


def head_real() -> str:
    return subprocess.run(["git", "rev-parse", "HEAD"], cwd=RAIZ, capture_output=True,
                          text=True, check=True).stdout.strip()


def agora_br() -> str:
    return datetime.datetime.now(FUSO_BR).replace(microsecond=0).isoformat()


def conferir(estado: dict, head: str) -> int:
    medido = estado.get("measured_commit")
    if medido != head:
        print("DIVERGENTE: measured_commit=%s | HEAD real=%s" % (str(medido)[:12], head[:12]))
        return 1
    print("CONSISTENTE: measured_commit == HEAD (%s)" % head[:12])
    print("  measured_at=%s" % estado.get("measured_at"))
    return 0


def gravar(head: str) -> int:
    texto = ESTADO.read_text(encoding="utf-8")
    antes = json.loads(texto)
    quando = agora_br()

    for campo, valor in (("measured_commit", head), ("measured_at", quando), ("updated_at", quando)):
        padrao = re.compile(r'("%s"\s*:\s*)"[^"]*"' % re.escape(campo))
        texto, quantas = padrao.subn(lambda m: m.group(1) + '"%s"' % valor, texto, count=1)
        print("  %-16s substituicoes=%d" % (campo, quantas))
        if quantas != 1:
            print("ABORTADO: esperava exactamente 1 ocorrencia de %r (escrita NAO feita)" % campo)
            return 1

    depois = json.loads(texto)
    if len(depois) != len(antes):
        print("ABORTADO: a contagem de chaves de topo mudou (%d -> %d)" % (len(antes), len(depois)))
        return 1

    ESTADO.write_text(texto, encoding="utf-8", newline="")
    conferido = json.loads(ESTADO.read_text(encoding="utf-8"))
    ok = (conferido.get("measured_commit") == head
          and conferido.get("measured_at") == quando
          and conferido.get("updated_at") == quando)
    print("gravado e relido do disco: %s (measured_at=%s)" % ("OK" if ok else "FALHOU", quando))
    return 0 if ok else 1


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Sincroniza a medicao (§13) do estado canonico, sem reformatar.")
    parser.add_argument("--check", action="store_true", help="apenas confere; nao grava")
    args = parser.parse_args(argv[1:])

    estado = json.loads(ESTADO.read_text(encoding="utf-8"))
    head = head_real()
    if args.check:
        return conferir(estado, head)
    print("medido_commit antes: %s" % str(estado.get("measured_commit"))[:12])
    print("HEAD real: %s" % head[:12])
    return gravar(head)


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
