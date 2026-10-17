#!/usr/bin/env python3
"""Catraca da JANELA DE CITACAO dos documentos de retomada (`E-1008`).

**Porque existe.** O `E-1007` mediu a **forma** do indice e declarou, em LIMITES:
*«a catraca mede a forma do INDICE, e nao de outros documentos -- os
`HADES_PENDING_WORK.md`, `HADES_NEXT_ACTION.md` e `HADES_HANDOFF.md` tem a mesma
vulnerabilidade e nao estao cobertos»*. => *Ao medir os tres, apareceu **outra** coisa:*

    HADES_PENDING_WORK.md   max=E-1000
    HADES_NEXT_ACTION.md    max=E-1000
    HADES_HANDOFF.md        max=E-1000     <- e o indice estava em E-1007

**Sete evidencias de atraso, e o `mission_check` nao acusa.** => *Porque o limiar dele e'
«citar **alguma** evidencia >= `E-960`», e `E-1000` **satisfaz**. => *Logo a catraca dele mede
**ha quanto tempo** o documento nao esta' vazio de evidencia recente -- e **nao** mede se
continua **actualizado**.*

**O que esta catraca verifica**, para cada documento de retomada declarado:

1. **A janela**: o maior ID citado tem de estar a menos de `JANELA` evidencias da cabeca do
   indice. => *`JANELA = 12`: um documento que pare 13 evidencias atras fica **obsoleto** sem
   ninguem notar.*
2. **Nenhum ID citado sem entrada**, fora da faixa historica declarada (`E-571..E-777`). => *a
   mesma regra do `E-1007`, agora aplicada aos documentos de **retomada**.*
3. **Nenhum newline LITERAL** (`\\` + `n`) -- *o defeito do `E-999`.*
4. **O documento existe e nao esta' vazio.** => *um documento de retomada com 0 IDs citados
   nao da' para retomar nada.*

**Porque `JANELA = 12`, e nao 1.** => *exigir a cabeca exacta tornaria a catraca **ruidosa**:
qualquer evidencia nova registada depois do checkpoint faria o documento parecer obsoleto ate'
alguem o reescrever.* => *12 e' o compromisso entre «actualizado» e «nao bloquear cada commit»
=> e o valor fica **declarado aqui**, e nao escondido.*

Uso:
    python scripts/hades_handoff_window_check.py            # mede e reporta
    python scripts/hades_handoff_window_check.py --check    # exit 1 se houver defeito
"""

import argparse
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"

# **Os documentos de retomada**, com o papel de cada um. *Uma lista declarada, e nao um glob:*
# um documento novo entra aqui **de proposito**, e nao por acidente.
DOCUMENTOS = [
    ("docs/mission/HADES_PENDING_WORK.md", "o que falta fazer"),
    ("docs/mission/HADES_NEXT_ACTION.md", "a proxima accao unica"),
    ("docs/mission/HADES_HANDOFF.md", "o estado e as instrucoes de retomada"),
]

# **A janela declarada**: quantas evidencias um documento pode ficar atras da cabeca.
JANELA = 12

# A faixa historica declarada: 207 IDs ausentes (`EVIDENCE_LOST_HISTORICAL`).
FAIXA_AUSENTE = (571, 777)

CABECALHO = re.compile(r"^\|\s*(E-\d+)\s*\|")
CITACAO = re.compile(r"\bE-(\d{3,})\b")


def cabeca() -> int:
    """O maior ID **registado** no indice."""
    texto = io.open(INDICE, encoding="utf-8").read()
    ids = [int(CABECALHO.match(l).group(1).split("-")[1]) for l in texto.split("\n") if CABECALHO.match(l)]
    if not ids:
        raise SystemExit("INDICE SEM ENTRADAS: %s" % INDICE)
    return max(ids)


def medir(caminho: Path, registados: set, cabeca_atual: int) -> dict:
    if not caminho.exists():
        return {"ausente": True}
    texto = io.open(caminho, encoding="utf-8").read()
    linhas = texto.split("\n")

    citados = {int(m) for m in CITACAO.findall(texto)}
    # **O LIMITE INFERIOR, e ele faltava aqui.** *O `hades_index_shape_check.py` curou-o no
    # `E-1007`: um `E-1` num texto corrido nao e' uma citacao de evidencia, e o `E-000`
    # (sentinela/placeholder) vira `0` no `int()` -- logo entrava como «citado sem entrada»
    # em **qualquer** documento que o mencionasse.*
    # => *Este checker tinha a **mesma** logica e **nao** tinha a cura* => *e o `E-1074` expos
    # isso ao citar o `E-000` na prova dos falsos positivos, nos `NEXT_ACTION` e `HANDOFF`.*
    # => *Dois checkers, uma regra, e **um deles ficou para tras**: e' o `E-1004` na forma de
    # paridade -- «uma cura que se julga completa fecha a investigacao antes do fim».*
    _abaixo = min(registados) if registados else 1
    sem_entrada = sorted(
        c
        for c in citados
        if c not in registados
        and not (FAIXA_AUSENTE[0] <= c <= FAIXA_AUSENTE[1])
        and c >= _abaixo
    )
    literais = [
        (k + 1, linhas[k][:100])
        for k in range(len(linhas))
        if ("\\" + "n| E-") in linhas[k] or ("\\" + "n") in linhas[k][-4:]
    ]
    maximo = max(citados) if citados else None
    atras = (cabeca_atual - maximo) if maximo is not None else None

    return {
        "ausente": False,
        "linhas": len(linhas),
        "citados": len(citados),
        "max": maximo,
        "atras": atras,
        "sem_entrada": sem_entrada,
        "literais": literais,
    }


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--check", action="store_true", help="nao escreve: exit 1 se houver defeito")
    args = ap.parse_args()

    if not INDICE.exists():
        print("INDICE AUSENTE: %s" % INDICE)
        return 1

    indice = io.open(INDICE, encoding="utf-8").read()
    registados = {int(CABECALHO.match(l).group(1).split("-")[1]) for l in indice.split("\n") if CABECALHO.match(l)}
    cabeca_atual = cabeca()

    print("JANELA DE CITACAO DOS DOCUMENTOS DE RETOMADA (cabeca do indice: E-%d, janela: %d)" % (cabeca_atual, JANELA))
    problemas = []
    for rel, papel in DOCUMENTOS:
        m = medir(RAIZ / rel, registados, cabeca_atual)
        nome = rel.split("/")[-1]
        if m["ausente"]:
            print("  AUSENTE  %-26s (%s)" % (nome, papel))
            problemas.append("DOCUMENTO AUSENTE: %s (%s)" % (rel, papel))
            continue
        print(
            "  %-8s %-26s max=%s  atras=%-4s citados=%-3d linhas=%d"
            % (
                "OK" if m["atras"] is not None and m["atras"] <= JANELA else "OBSOLETO",
                nome,
                ("E-%d" % m["max"]) if m["max"] is not None else "(nenhum)",
                m["atras"] if m["atras"] is not None else "-",
                m["citados"],
                m["linhas"],
            )
        )
        if m["citados"] == 0:
            problemas.append("SEM EVIDENCIA CITADA: %s (%s) => *nao da' para retomar nada*" % (rel, papel))
        elif m["atras"] > JANELA:
            problemas.append(
                "DOCUMENTO OBSOLETO: %s => *cita ate' E-%d e a cabeca e' E-%d: **%d evidencias "
                "de atraso**, acima da janela de %d* => *o `mission_check` **nao** acusa, porque "
                "so' exige «alguma evidencia >= E-960»*" % (rel, m["max"], cabeca_atual, m["atras"], JANELA)
            )
        if m["sem_entrada"]:
            problemas.append(
                "IDs CITADOS SEM ENTRADA em %s: %s => *os documentos de retomada tambem tem de "
                "apontar para evidencia registada*"
                % (rel, ", ".join("E-%d" % i for i in m["sem_entrada"][:8]))
            )
        if m["literais"]:
            problemas.append(
                "NEWLINE LITERAL em %s: linha(s) %s => *o defeito do `E-999`*"
                % (rel, ", ".join(str(l) for l, _ in m["literais"][:5]))
            )

    if problemas:
        print("PROBLEMAS:")
        for p in problemas:
            print("  - %s" % p)
        return 1 if args.check else 0

    print("JANELA CONSISTENTE: os 3 documentos citam a cabeca dentro de %d evidencias," % JANELA)
    print("  nenhum ID citado sem entrada, e sem newline literal.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
