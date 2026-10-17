#!/usr/bin/env python3
"""hades_fix_mojibake.py — cura a dupla codificacao REAL em ficheiros de texto.

POR QUE ISTO EXISTE (E-951):
  Cinco vezes nesta janela (E-944, E-945, E-946, E-947, E-949, E-950) o proprio
  ato de DOCUMENTAR um defeito de codificacao reintroduziu o defeito: eu escrevi
  as sequencias de mojibake dentro do documento canonico para as explicar.

  O `hades_encoding_check.py` detecta-as e diz a regra exacta:
    "descreva o artefato, nao o transcreva".

  Este script aplica essa regra de forma mecanica: em vez de eu escrever a
  sequencia (e me enganar outra vez), ele substitui cada sequencia por um ROTULO
  ASCII que a NOMEIA.  O sentido fica; a corrupcao sai.

USO:
  python -X utf8 scripts/hades_fix_mojibake.py --check            # so' mede
  python -X utf8 scripts/hades_fix_mojibake.py --apply <ficheiro> # cura um
  python -X utf8 scripts/hades_fix_mojibake.py --apply-all        # cura os docs

NAO toca em `hades_encoding_check.py` nem na fixture dele (as 4 dispensas
declaradas pelo checker) — o detector PRECISA de conter os padroes.
"""
import argparse
import io
import os
import sys

# As sequencias de dupla codificacao (UTF-8 lido como Latin-1/Windows-1252).
# O valor e' o ROTULO ASCII que as nomeia — nunca a propria sequencia.
MOJIBAKE = {
    "\u00c3\u00a7": "[c-cedilha-duplo]",
    "\u00c3\u00a3": "[a-til-duplo]",
    "\u00c3\u00a1": "[a-agudo-duplo]",
    "\u00c3\u00a9": "[e-agudo-duplo]",
    "\u00c3\u00ad": "[i-agudo-duplo]",
    "\u00c3\u00b3": "[o-agudo-duplo]",
    "\u00c3\u00ba": "[u-agudo-duplo]",
    "\u00c3\u00b5": "[o-til-duplo]",
    "\u00c3\u00a2": "[a-circ-duplo]",
    "\u00c3\u00aa": "[e-circ-duplo]",
    "\u00c3\u00b4": "[o-circ-duplo]",
    "\u00c3\u00a0": "[a-crase-duplo]",
    "\u00c3\u0080": "[A-crase-duplo]",
    "\u00c3\u0087": "[C-cedilha-duplo]",
    "\u00c3\u0083": "[A-til-duplo]",
    "\u00c3\u0081": "[A-agudo-duplo]",
    "\u00c3\u0089": "[E-agudo-duplo]",
    "\u00c3\u0093": "[O-agudo-duplo]",
    "\u00c3\u0095": "[O-til-duplo]",
    "\u00c3\u0082": "[A-circ-duplo]",
    "\u00c3\u008a": "[E-circ-duplo]",
    "\u00c3\u008d": "[I-agudo-duplo]",
    "\u00c3\u009a": "[U-agudo-duplo]",
    "\u00c3\u0080": "[A-crase-duplo]",
    "\u00c3\u0089": "[E-agudo-duplo]",
    "\u00c3\u00bc": "[u-diaerese-duplo]",
    "\u00c3\u00a4": "[a-diaerese-duplo]",
    "\u00c3\u00b6": "[o-diaerese-duplo]",
    "\u00c3\u00a8": "[e-crase-duplo]",
    "\u00c3\u00ac": "[i-crase-duplo]",
    "\u00c3\u00b2": "[o-crase-duplo]",
    "\u00c3\u00b9": "[u-crase-duplo]",
    "\u00c2\u00ab": "[angulo-esq-duplo]",
    "\u00c2\u00bb": "[angulo-dir-duplo]",
    "\u00c2\u00b0": "[grau-duplo]",
    "\u00c2\u00ba": "[ordinal-masc-duplo]",
    "\u00c2\u00aa": "[ordinal-fem-duplo]",
    "\u00c2\u00a0": "[espaco-inquebravel-duplo]",
    "\u00e2\u0080\u0094": "[travessao-duplo]",
    "\u00e2\u0080\u0093": "[meia-risca-duplo]",
    "\u00e2\u0080\u009c": "[aspas-esq-duplo]",
    "\u00e2\u0080\u009d": "[aspas-dir-duplo]",
    "\u00e2\u0080\u0098": "[plica-esq-duplo]",
    "\u00e2\u0080\u0099": "[plica-dir-duplo]",
    "\u00e2\u0080\u00a6": "[reticencias-duplo]",
    "\u00e2\u0082\u00ac": "[euro-duplo]",
    "\u00e2\u0086\u0092": "[seta-duplo]",
    "\u00e2\u009c\u0093": "[visto-duplo]",
    "\u00e2\u009d\u008c": "[cruz-duplo]",
    "\u00e2\u009a\u00a0": "[aviso-duplo]",
    "\u00e2\u0098\u0085": "[estrela-duplo]",
    "\u00e2\u009c\u0085": "[visto-branco-duplo]",
    "\u00e2\u0080\u008b": "[espaco-zero-duplo]",
    "\u00cb\u0086": "[seta-baixo-duplo]",
    "\u00c3\u0097": "[x-duplo]",
}

# Ficheiros onde as sequencias SAO o proposito (o detector e a sua fixture).
DISPENSADOS = {
    "scripts/hades_encoding_check.py",
    "scripts/hades_encoding_check_test.py",
    "scripts/hades_fix_mojibake.py",
    # E-951: o `--apply-all` correu sobre TODO o repo e alterou 2320 sequencias
    # em `mlxrunner/tokenizer/testdata/llama3.2/encoder.json` (test data do
    # tokenizer!) e as allowlists de dois checkers.  O dano foi revertido, mas a
    # causa era a amplitude.  Agora `--apply-all` so' varre `DOCS` e `RAIZ_DOCS`.
    "scripts/hades_mission_check.py",
    "scripts/hades_mission_check_test.py",
    "scripts/hades_test_clock_check.py",
    "scripts/hades_test_clock_check_test.py",
    "scripts/hades_tenant_scope_check.py",
    "scripts/hades_non_inference_scan.py",
    "scripts/hades_stage_map.py",
    "scripts/hades_release_gate.py",
    "scripts/hades_ci_reproducibility_check.py",
}

# `--apply-all` restringe-se a estes prefixos.  Fora deles, `--apply <ficheiro>`
# explicito continua a funcionar — mas nunca por varrimento automatico.
PREFIXOS_SEGUROS = ("docs/", "docs\\")

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def medir(texto):
    """Devolve [(sequencia, rotulo, contagem)] das sequencias presentes."""
    out = []
    for seq, rot in sorted(MOJIBAKE.items(), key=lambda kv: -len(kv[0])):
        n = texto.count(seq)
        if n:
            out.append((seq, rot, n))
    return out


def curar(caminho, aplicar):
    """Mede/cura um ficheiro.  Devolve (ocorrencias, linhas_alteradas)."""
    rel = os.path.relpath(caminho, RAIZ).replace("\\", "/")
    if rel in DISPENSADOS:
        return None
    with io.open(caminho, encoding="utf-8", errors="replace") as fh:
        linhas = fh.read().split("\n")
    total = 0
    mudadas = 0
    for i, linha in enumerate(linhas):
        nova = linha
        for seq, rot in sorted(MOJIBAKE.items(), key=lambda kv: -len(kv[0])):
            if seq in nova:
                total += nova.count(seq)
                nova = nova.replace(seq, rot)
        if nova != linha:
            linhas[i] = nova
            mudadas += 1
    if aplicar and mudadas:
        with io.open(caminho, "w", encoding="utf-8", newline="") as fh:
            fh.write("\n".join(linhas))
    return total, mudadas


def alvos():
    """Ficheiros de texto do repo, mas SO' sob PREFIXOS_SEGUROS.

    E-951: a primeira versao varria o repo inteiro e alterou o test data do
    tokenizer e as allowlists de dois checkers.  A varredura automatica passa a
    ser restrita a documentacao, onde a regra "descreva, nao transcreva" se
    aplica; qualquer outro ficheiro exige `--apply <caminho>` explicito.
    """
    ignorar = (".git", "_tmp", "node_modules", "dist", "goroot1269", "gopath",
               "gocache", "gomodcache", "gotmp", "webtemp", ".cache")
    for base, dirs, ficheiros in os.walk(RAIZ):
        dirs[:] = [d for d in dirs if d not in ignorar]
        for nome in ficheiros:
            caminho = os.path.join(base, nome)
            rel = os.path.relpath(caminho, RAIZ).replace("\\", "/")
            if not rel.startswith("docs/"):
                continue
            if os.path.splitext(nome)[1].lower() in (
                    ".md", ".json", ".csv", ".yaml", ".yml", ".ts", ".tsx",
                    ".js", ".mjs", ".go", ".py", ".ps1", ".sh", ".txt", ".iss",
                    ".html", ".css", ".svg", ".toml", ".cfg", ".ini", ".env"):
                yield caminho


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--check", action="store_true", help="so' mede (exit 1 se houver)")
    ap.add_argument("--apply", metavar="FICHEIRO", help="cura um ficheiro")
    ap.add_argument("--apply-all", action="store_true", help="cura todos os alvos")
    args = ap.parse_args()

    if args.apply:
        res = curar(os.path.abspath(args.apply), True)
        if res is None:
            print("DISPENSADO (o detector precisa de conter os padroes)")
            return 0
        print("CURADO: %d sequencia(s) em %d linha(s)" % res)
        return 0

    total = 0
    ficheiros = 0
    for caminho in (alvos() if args.apply_all else [os.path.join(RAIZ, "docs", "mission", "HADES_EVIDENCE_INDEX.md")]):
        res = curar(caminho, args.apply_all)
        if res and res[0]:
            total += res[0]
            ficheiros += 1
            rel = os.path.relpath(caminho, RAIZ).replace("\\", "/")
            print("  %s: %d sequencia(s) em %d linha(s)" % (rel, res[0], res[1]))
    if args.apply_all and total:
        print("CURADOS: %d sequencia(s) em %d ficheiro(s)" % (total, ficheiros))
        return 0
    if total:
        print("MOJIBAKE: %d sequencia(s) em %d ficheiro(s)" % (total, ficheiros))
        return 1
    print("LIMPO: nenhuma sequencia de dupla codificacao")
    return 0


if __name__ == "__main__":
    sys.exit(main())
