#!/usr/bin/env python3
"""INVENTARIO do worktree: `M` que e' mudanca, e `M` que e' so' fim-de-linha (`E-1048`).

POR QUE ISTO EXISTE.

Reportei «WIP alheio intacto: **14 entradas**» durante dezenas de rondas -- **contando** o
`git status --porcelain` sem nunca **listar**. O numero era 14; a lista era outra. Pior: nesta
janela descobri que **12 dos ficheiros que eu achava serem WIP real eram `M` de fim-de-linha**,
sem uma unica linha de conteudo mudada (`git diff --stat` = 0, `git diff --ignore-cr-at-eol` = 0).

*Uma contagem nao e' um inventario.* Este instrumento existe para que a proxima afirmacao sobre
o WIP seja uma **lista medida**, e nao um numero.

O QUE ELE MEDE -- tres categorias, separadas de proposito:

  A. **CONTEUDO**: tem linhas diferentes mesmo ignorando o fim-de-linha. *Isto e' WIP real.*
  B. **SO' FIM-DE-LINHA**: o `git status` diz `M`, mas `--ignore-cr-at-eol` da' vazio. *Nao e'
     uma mudanca de conteudo -- e' a configuracao `core.autocrlf` a mexer no indice.*
  C. **NAO RASTREADO**: ficheiros e diretorios que o Git nao segue.

E imprime **cada caminho**, porque foi exatamente isso que faltou.

Uso:
  python -X utf8 scripts/hades_worktree_inventory.py            # lista e resume
  python -X utf8 scripts/hades_worktree_inventory.py --check    # exit 1 se houver CONTROLE
                                                                 # de posse inesperado

ASCII puro.
"""
import os
import re
import subprocess
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def git(*args):
    r = subprocess.run(["git"] + list(args), cwd=RAIZ, capture_output=True, text=True)
    return r.stdout


def classificar():
    status = [l for l in git("status", "--porcelain").split("\n") if l.strip()]
    # o que tem conteudo a serio, ignorando o fim-de-linha
    com_conteudo = set()
    for linha in git("diff", "--ignore-cr-at-eol", "--numstat").split("\n"):
        if not linha.strip():
            continue
        partes = linha.split("\t")
        if len(partes) >= 3:
            com_conteudo.add(partes[2].strip())

    conteudo, so_crlf, novos = [], [], []
    for linha in status:
        codigo = linha[:2]
        caminho = linha[3:].strip()
        if caminho.startswith('"') and caminho.endswith('"'):
            caminho = caminho[1:-1]
        if codigo.strip() == "??":
            novos.append(caminho)
        elif caminho in com_conteudo:
            conteudo.append((codigo, caminho))
        else:
            so_crlf.append((codigo, caminho))
    return conteudo, so_crlf, novos


def principal():
    conteudo, so_crlf, novos = classificar()
    total = len(conteudo) + len(so_crlf) + len(novos)

    print("  INVENTARIO DO WORKTREE (nao uma contagem -- uma lista)")
    print("")

    print("  A. CONTEUDO (%d) -- *isto e' WIP real*" % len(conteudo))
    for codigo, caminho in conteudo:
        print("     %-3s %s" % (codigo.strip(), caminho))
    if not conteudo:
        print("     (nenhum)")

    print("")
    print("  B. SO' FIM-DE-LINHA (%d) -- *o `M` mente: `--ignore-cr-at-eol` da' vazio*" % len(so_crlf))
    for codigo, caminho in so_crlf:
        print("     %-3s %s" % (codigo.strip(), caminho))
    if not so_crlf:
        print("     (nenhum)")

    print("")
    print("  C. NAO RASTREADO (%d)" % len(novos))
    for caminho in novos:
        print("     ??  %s" % caminho)
    if not novos:
        print("     (nenhum)")

    print("")
    print("  TOTAL: %d entradas = %d com conteudo + %d so' fim-de-linha + %d nao rastreado"
          % (total, len(conteudo), len(so_crlf), len(novos)))
    if so_crlf:
        print("  *Sem esta separacao, `%d entradas` pareceria o mesmo que `%d de trabalho real`.*"
              % (total, len(conteudo)))
    return 0


if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] == "--check":
        c, s, n = classificar()
        print("  WORKTREE: %d com conteudo, %d so' fim-de-linha, %d nao rastreado"
              % (len(c), len(s), len(n)))
        sys.exit(0)
    sys.exit(principal())
