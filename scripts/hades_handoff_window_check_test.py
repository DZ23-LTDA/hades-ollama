#!/usr/bin/env python3
"""Fault injection da catraca da JANELA DE CITACAO (`E-1008`).

**Porque existe.** *A catraca mede `atras=7` nos tres documentos de retomada, e **passa**
(janela 12).* => *Isso e' o comportamento certo, mas nao **prova** que ela falharia se a janela
fosse excedida.* => *Este teste injecta as tres anomalias e exige que ela as **encontre**.*

As tres, e o defeito real de cada uma:

1. **Documento obsoleto** -- o que o `E-1008` existe para apanhar: *citar so' evidencia antiga
   e ficar 13+ atras da cabeca, **sem o `mission_check` acusar** (o limiar dele e' `E-960`,
   e um documento parado no `E-999` satisfaz).*
2. **ID citado sem entrada** -- a mesma regra do `E-1007`, agora nos documentos de retomada.
3. **Newline LITERAL** -- o defeito do `E-999`.

Uso:
    python scripts/hades_handoff_window_check_test.py
"""

import io
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(RAIZ / "scripts"))

import hades_handoff_window_check as c  # noqa: E402

OK = 0
FALHAS: list[str] = []


def T(nome: str, cond: bool) -> None:
    global OK
    if cond:
        OK += 1
    else:
        FALHAS.append(nome)


def main() -> int:
    if not c.INDICE.exists():
        print("INDICE AUSENTE: %s" % c.INDICE)
        return 1

    indice = io.open(c.INDICE, encoding="utf-8").read()
    registados = {
        int(c.CABECALHO.match(l).group(1).split("-")[1])
        for l in indice.split("\n")
        if c.CABECALHO.match(l)
    }
    cabeca = c.cabeca()
    T("a cabeca do indice foi medida", cabeca > 0)

    # (0) **Os 3 documentos reais passam.** *Se algum estiver obsoleto, este teste falha com o
    # defeito a' mostra, e nao com uma crenca.*
    for rel, _papel in c.DOCUMENTOS:
        m = c.medir(RAIZ / rel, registados, cabeca)
        T("%s existe e cita evidencia" % rel.split("/")[-1], not m["ausente"] and m["citados"] > 0)
        T("%s esta' dentro da janela" % rel.split("/")[-1], m["atras"] is not None and m["atras"] <= c.JANELA)
        T("%s nao cita ID sem entrada" % rel.split("/")[-1], len(m["sem_entrada"]) == 0)
        T("%s nao tem newline literal" % rel.split("/")[-1], len(m["literais"]) == 0)

    tmp = Path(tempfile.mkdtemp(prefix="hades-window-"))

    # (1) **DOCUMENTO OBSOLETO** -- o defeito do `E-1008`.
    velho = cabeca - c.JANELA - 1
    f = tmp / "obsoleto.md"
    f.write_text("Ver E-%d para o estado.\n" % velho, encoding="utf-8")
    m = c.medir(f, registados, cabeca)
    T("um documento %d atras da cabeca -> atras > JANELA" % (c.JANELA + 1), m["atras"] > c.JANELA)
    # E um documento DENTRO da janela nao dispara.
    f2 = tmp / "ok.md"
    f2.write_text("Ver E-%d para o estado.\n" % (cabeca - c.JANELA), encoding="utf-8")
    m2 = c.medir(f2, registados, cabeca)
    T("um documento exactamente na janela -> NAO dispara", m2["atras"] == c.JANELA)

    # (2) **ID CITADO SEM ENTRADA** -- construido em runtime (a licao do `E-1007`).
    id_fantasma = cabeca + 900
    f3 = tmp / "fantasma.md"
    f3.write_text("Estado: E-%d.\n" % id_fantasma, encoding="utf-8")
    m3 = c.medir(f3, registados, cabeca)
    T("um ID citado sem entrada -> detectado", id_fantasma in m3["sem_entrada"])

    # (3) **A FAIXA HISTORICA NAO E EXIGIDA.**
    f4 = tmp / "faixa.md"
    f4.write_text("A faixa E-571 a E-777 esta declarada ausente.\n", encoding="utf-8")
    m4 = c.medir(f4, registados, cabeca)
    T("a faixa E-571..E-777 nao e exigida", len(m4["sem_entrada"]) == 0)

    # (4) **NEWLINE LITERAL** -- o defeito do `E-999`.
    # *O caso do DANO REAL (o `E-999`), e nao um caso artificial:* o literal **antes de um
    # cabecalho** -- foi assim que ele **colou** duas entradas numa linha.
    # => *Procurar `<barra+n>` em qualquer posicao tornaria o detector sensivel a **mencoes****
    # (*as entradas `E-999`/`E-1007` **descrevem** o literal*) -- o `E-1003` outra vez.
    f5 = tmp / "literal.md"
    f5.write_text("Estado.\n" + ("\\" + "n") + "| E-%d | outra entrada |\n" % cabeca, encoding="utf-8")
    m5 = c.medir(f5, registados, cabeca)
    T("um newline LITERAL -> detectado", len(m5["literais"]) >= 1)

    # (5) **DOCUMENTO SEM EVIDENCIA** -- um documento de retomada que nao cita nada.
    f6 = tmp / "vazio.md"
    f6.write_text("Sem evidencias.\n", encoding="utf-8")
    m6 = c.medir(f6, registados, cabeca)
    T("um documento sem evidencia citada -> citados == 0", m6["citados"] == 0)

    # (6) **DOCUMENTO AUSENTE.**
    m7 = c.medir(tmp / "nao_existe.md", registados, cabeca)
    T("um documento ausente -> ausente=True", m7["ausente"] is True)

    print("JANELA DE CITACAO - fault injection: %d OK, %d FAIL" % (OK, len(FALHAS)))
    for f in FALHAS:
        print("  FALHOU: %s" % f)
    if FALHAS:
        return 1
    print("  o detector encontra as 5 anomalias, e nao acusa a faixa declarada.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
