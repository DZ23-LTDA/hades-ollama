#!/usr/bin/env python3
"""Fault injection da catraca da FORMA do indice (`E-1007`).

**Porque existe.** Uma catraca sem teste e' uma **crenca**. => *E, neste caso, o teste
justificou-se **tres vezes**: ele apanhou **tres defeitos da propria catraca**, cada um
na primeira execucao em que a regra foi usada.*

Os defeitos que este teste apanhou, e que valem como licoes:

1. **O loop do newline literal so' varria as linhas com cabecalho.** => *Quando o `\\n`
   literal entra **antes** de um cabecalho, a linha passa a comecar por barra + n em vez
   de `|`, logo **nao esta'** naquela lista => e o detector nao a via.*
2. **O `CITACAO` era `\\d{3,4}`** => *e `E-99999` **nao casava** => a catraca deixava
   passar um ID citado sem entrada.*
3. **O filtro tinha limite superior** (`abaixo <= c <= acima`) => *um ID citado **acima
   da cabeca** era silenciosamente ignorado* => *e e' **exactamente** o caso de um
   requisito activo a apontar para evidencia que nao existe.*

**E o teste e' o inverso do defeito**: cada asserção **injecta** uma das quatro anomalias
conhecidas e exige que a catraca a **encontre**. => *Se a catraca deixar de as ver, este
teste falha, e o CI nao passa.*

Uso:
    python scripts/hades_index_shape_check_test.py
"""

import io
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(RAIZ / "scripts"))

import hades_index_shape_check as c  # noqa: E402

INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"

OK = 0
FALHAS: list[str] = []


def T(nome: str, cond: bool) -> None:
    global OK
    if cond:
        OK += 1
    else:
        FALHAS.append(nome)


def main() -> int:
    if not INDICE.exists():
        print("INDICE AUSENTE: %s" % INDICE)
        return 1
    texto = io.open(INDICE, encoding="utf-8").read()
    base = c.medir(texto)

    # (0) **O indice real tem de estar limpo.** *Se nao estiver, este teste falha com o
    # defeito a' mostra, e nao com uma crenca.*
    T("o indice real tem 0 linhas orfas", len(base["orfas"]) == 0)
    T("o indice real tem 0 IDs duplicados", len(base["duplicados"]) == 0)
    T("o indice real tem 0 newlines literais", len(base["literais"]) == 0)
    T("o indice real tem 0 IDs citados sem entrada", len(base["sem_entrada"]) == 0)

    linhas = texto.split("\n")
    # *O regex do proprio modulo, e nao um `startswith`:* um prefixo de texto nao e' um
    # cabecalho (o `E-1003`). *O `startswith("| E-")` apanhava linhas cujo CONTEUDO tem uma
    # celula `| E-NNN | ...` -- ex.: a coluna de estado -- e o `int()` estourava.*
    base_entradas = [i for i in texto.split("\n") if c.CABECALHO.match(i)]
    k = next(i for i, l in enumerate(linhas) if l.startswith("| E-1006 |"))

    # (1) **A ENTRADA PARTIDA** -- o defeito do `E-1003`, e o que motivou esta catraca.
    q = list(linhas)
    q[k] = q[k][:200]
    q.insert(k + 1, "**VINCULO_NAO_DECLARADO: partido de proposito**")
    m = c.medir("\n".join(q))
    T("uma entrada partida em 2 linhas -> orfa detectada", len(m["orfas"]) >= 1)

    # (2) **O NEWLINE LITERAL** -- o defeito do `E-999`.
    m = c.medir(texto.replace("| E-560 | ", ("\\" + "n") + "| E-560 | ", 1))
    T("um newline LITERAL -> detectado", len(m["literais"]) >= 1)

    # (3) **ID CITADO SEM ENTRADA** -- inclusive **acima da cabeca** (o 3o defeito).
    # *O ID e' construido em runtime, e nao escrito por extenso:* o `hades_mission_check.py`
    # varre o **codigo** a' procura de IDs citados, e um literal seria **indistinguivel de
    # uma citacao real** -- foi assim que ele acusou este proprio teste.*
    # *O ID e' o GRUPO do regex, e nao um `split`:* a linha tem mais hifens depois do ID.
    id_acima = max(int(c.CABECALHO.match(i).group(1).split("-")[1]) for i in base_entradas) + 900
    m = c.medir(texto + "\nVer E-%d para detalhe.\n" % id_acima)
    T("um ID citado sem entrada ACIMA da cabeca -> detectado", id_acima in m["sem_entrada"])

    # (4) **ID DUPLICADO.**
    q = list(linhas)
    q.insert(k + 1, q[k])
    m = c.medir("\n".join(q))
    T("um ID duplicado -> detectado", "E-1006" in m["duplicados"])

    # (5) **A FAIXA HISTORICA NAO E EXIGIDA.** *Citar `E-571..E-777` nao pode acusar: a
    # ausencia esta' **declarada** (`EVIDENCE_LOST_HISTORICAL`).*
    m = c.medir(texto + "\nA faixa E-571 a E-777 esta declarada ausente.\n")
    T("a faixa E-571..E-777 nao e exigida", 571 not in m["sem_entrada"] and 777 not in m["sem_entrada"])

    # (6) **E NAO IMPEDE BURACO NOVO.** *Um ID **fora** da faixa, citado e sem entrada,
    # tem de acusar -- e' o §FASE-7 item 6.*
    id_fora = 1900  # *construido, e nao literal*: ver a nota do item (3).
    m = c.medir(texto + "\nVer tambem E-%d.\n" % id_fora)
    T("um ID fora da faixa citado sem entrada -> detectado", id_fora in m["sem_entrada"])

    # (7) **A ISENCAO DOS 2 FALSOS POSITIVOS NAO E' UMA PORTA ABERTA.**
    # *O `E-1074` isentou dois numeros que nao sao IDs de evidencia: o ano de um CVE e a cabeca de teste deste
    # detector.* => *mas uma isencao por **valor** pode
    # virar isencao por **classe** se ninguem a testar.*
    # => *Este teste prova que ela e' **minima**: um ID **vizinho** (outro ano) e um ID **abaixo** (outra cabeca) **continuam a acusar**.*
    for vizinho, rotulo in ((2027, "um ano vizinho"), (99998, "uma cabeca vizinha")):
        m = c.medir(texto + "\nVer E-%d.\n" % vizinho)
        T("%s (E-%d) citado sem entrada -> detectado, a isencao e' minima" % (rotulo, vizinho),
          vizinho in m["sem_entrada"])

    print("FORMA DO INDICE - fault injection: %d OK, %d FAIL" % (OK, len(FALHAS)))
    for f in FALHAS:
        print("  FALHOU: %s" % f)
    if FALHAS:
        return 1
    print("  o detector encontra as 4 anomalias conhecidas, e nao acusa a faixa declarada.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
