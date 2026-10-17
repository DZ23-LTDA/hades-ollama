#!/usr/bin/env python3
"""LIGACAO docs -> artefactos (`E-1054`): *um resultado sem ponteiro existe e nao se encontra.*

POR QUE ISTO EXISTE.

No `E-1053` medi que os **tres** documentos canonicos do ESTAGIO 21 descreviam as medicões e
**nenhum** mencionava `docs/parity/HADES_T2_MATRIX.md` nem `HADES_T2_REPETICAO.md`. Liguei-os --
**com texto**. E o texto nao vigia nada: se alguem renomear um artefacto, **nada acusa**.

*Ligar com texto nao e' ligar: e' descrever a ligacao.* Este checker transforma a ligacao em
mecanismo -- o mesmo padrao da catraca do encaixe (`E-1044`) e da catraca do opt-in (`E-1036`).

O QUE ELE MEDE:

  A. cada **artefacto declarado existe** em disco (com o tamanho, para um ficheiro vazio nao passar);
  B. cada **documento declarado cita** os artefactos que lhe competem;
  C. e nada mais -- *nao julga a qualidade do que esta' escrito, so' que o ponteiro esta' la'*.

A FONTE DA VERDADE e' o mapa `LIGACOES`, aqui em baixo. *Uma lista declarada e' auditavel; uma
convencao tacita nao e'.*

Uso:
  python -X utf8 scripts/hades_stage_artifacts_check.py            # relatorio completo
  python -X utf8 scripts/hades_stage_artifacts_check.py --check    # silencioso, exit 1 se falhar

ASCII puro, sem dependencias.
"""
import io
import os
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# ---------------------------------------------------------------------------
# A LISTA DECLARADA. Doc -> artefactos que TEM de citar.
# *Se um artefacto mudar de nome, este mapa e' o unico sitio a corrigir -- e o
# checker acusa ate' la'.*
# ---------------------------------------------------------------------------
LIGACOES = {
    "docs/mission/HADES_STAGE_21_BENCHMARK.md": [
        "docs/parity/HADES_T2_MATRIX.md",
        "docs/parity/HADES_T2_REPETICAO.md",
    ],
    "docs/mission/HADES_BENCHMARK_METHOD.md": [
        "docs/parity/HADES_T2_MATRIX.md",
        "docs/parity/HADES_T2_REPETICAO.md",
    ],
    "docs/mission/HADES_BENCHMARK_21.md": [
        "docs/parity/HADES_T2_MATRIX.md",
        "docs/parity/HADES_T2_REPETICAO.md",
    ],
}

# Tamanho minimo: um ficheiro vazio nao e' um artefacto.
MINIMO_BYTES = 400


def ler(caminho):
    try:
        return io.open(caminho, encoding="utf-8").read()
    except OSError:
        return None


def medir():
    achados = []
    linhas_relatorio = []
    for doc, artefactos in sorted(LIGACOES.items()):
        caminho_doc = os.path.join(RAIZ, doc)
        texto = ler(caminho_doc)
        if texto is None:
            achados.append("DOCUMENTO AUSENTE: %s" % doc)
            linhas_relatorio.append("  %-44s AUSENTE" % doc)
            continue
        for art in artefactos:
            caminho_art = os.path.join(RAIZ, art)
            tamanho = os.path.getsize(caminho_art) if os.path.exists(caminho_art) else -1
            if tamanho < 0:
                achados.append("ARTEFACTO AUSENTE: %s (citado por %s)" % (art, doc))
                estado = "ARTEFACTO AUSENTE"
            elif tamanho < MINIMO_BYTES:
                achados.append(
                    "ARTEFACTO SUSPEITO: %s tem %d bytes (< %d) -- *um ficheiro vazio nao e' um "
                    "artefacto*" % (art, tamanho, MINIMO_BYTES)
                )
                estado = "SUSPEITO (%d bytes)" % tamanho
            elif os.path.basename(art) in texto:
                estado = "OK (%d bytes)" % tamanho
            else:
                # o documento existe e o artefacto existe, mas nao ha' ponteiro
                achados.append(
                    "SEM PONTEIRO: %s NAO cita %s -- *o resultado existe e nao se encontra*"
                    % (doc, art)
                )
                estado = "SEM PONTEIRO"
            linhas_relatorio.append("  %-44s %-16s %s" % (doc, os.path.basename(art), estado))
    return achados, linhas_relatorio


def principal():
    achados, relatorio = medir()
    print("  LIGACAO docs -> artefactos (%d documentos declarados)" % len(LIGACOES))
    print("")
    for l in relatorio:
        print(l)
    print("")
    if achados:
        print("  *** %d LIGACOES EM FALTA ***" % len(achados))
        for a in achados:
            print("    - %s" % a)
        print("")
        print("  *Ligar com texto nao e' ligar: e' descrever a ligacao.*")
        print("  Corrija o documento -- ou, se o artefacto mudou de nome, corrija o mapa")
        print("  `LIGACOES` deste checker. **Nao apague a entrada.**")
        return 1
    print("  TODAS AS LIGACOES DECLARADAS EXISTEM: %d documentos, %d artefactos."
          % (len(LIGACOES), sum(len(v) for v in LIGACOES.values())))
    print("  *Nao julga a qualidade do que esta' escrito -- so' que o ponteiro esta' la'.*")
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--check":
        a, _ = medir()
        if a:
            for x in a:
                print("  FALHA: %s" % x)
        sys.exit(1 if a else 0)
    sys.exit(principal())
