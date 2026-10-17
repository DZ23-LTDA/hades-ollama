#!/usr/bin/env python3
"""hades_ci_gate_check.py — o gate do `gofmt` nao pode voltar a ser VERDE POR CONSTRUCAO (E-665).

POR QUE ISTO EXISTE (medido, e custou um CI vermelho)
-----------------------------------------------------
O passo do CI era:

    test -z "$(gofmt -l internal/agent internal/multillm server/agent_routes.go)"

Com um arquivo que **nao parseia**, o `gofmt` escreve o erro em **STDERR**, sai com codigo != 0 e deixa **STDOUT
VAZIO** ⇒ `test -z ""` **PASSA**. *Foi assim que um `internal/agent/adaptive_router.go` com uma `}` faltando
atravessou o CI verde* — o commit `fc7b71ccd` so' reprovou **depois**, no `golangci-lint`.

A correcao (conferir o **codigo de saida**) foi feita em `class-a-plus-integrity.yaml`. Este checker e' a
**catraca**: ele le' os workflows versionados e **recusa** a forma que nao confere o codigo de saida.

***Um gate que so' olha para `stdout` e' um gate que aprova arquivo ilegivel.***

Uso
---
    python scripts/hades_ci_gate_check.py                  # mede .github/workflows/*.yaml
    python scripts/hades_ci_gate_check.py <arquivo.yaml>   # mede um workflow especifico
"""

from __future__ import annotations

import pathlib
import re
import sys

RAIZ = pathlib.Path(__file__).resolve().parent.parent
WORKFLOWS = RAIZ / ".github" / "workflows"

# Formas CEGAS: testam `stdout` de um `gofmt` em substituicao de comando, sem olhar o codigo de saida.
FORMAS_CEGAS = [
    (re.compile(r"test\s+-z\s+\"?\$\(\s*gofmt"), "`test -z \"$(gofmt ...)\"` (stdout vazio PASSA)"),
    (re.compile(r"\[\s*-z\s+\"?\$\(\s*gofmt"), "`[ -z \"$(gofmt ...)\" ]` (stdout vazio PASSA)"),
    (re.compile(r"test\s+-n\s+\"?\$\(\s*gofmt"), "`test -n \"$(gofmt ...)\"` (mede quantidade, nao sucesso)"),
]

# Formas que REALMENTE conferem: o codigo de saida do gofmt e' observado.
#
# NOTA MEDIDA (E-665): uma PIPELINE (`gofmt ... | tee ... && test ! -s ...`) **nao** confere por si so' — sem
# `set -o pipefail` o status da pipeline e' o do ULTIMO comando (`tee`). Por isso `set -o pipefail` e' aceito como
# forma de conferir, e a pipeline SEM ele e' recusada. *As formas cegas sao verificadas PRIMEIRO: uma delas nunca
# passa so' porque o bloco tambem tem `pipefail`* (com substituicao de comando, o `set -e` nao aborta).
FORMAS_QUE_CONFEREM = [
    re.compile(r"if\s+!\s+\w+\s*=\s*\"?\$\(\s*gofmt", re.M),   # if ! saida="$(gofmt ...)"; then
    re.compile(r"gofmt[^\n]*\|\|\s*exit", re.M),                 # gofmt ... || exit 1
    re.compile(r"if\s+!\s+gofmt", re.M),                         # if ! gofmt ...; then
    re.compile(r"set\s+-o\s+pipefail", re.M),                    # set -o pipefail (torna a pipeline honesta)
]


def blocos_run(texto: str) -> list[str]:
    """Extrai o corpo de cada `run:` (bloco `|` ou linha unica) de um workflow YAML, sem depender de PyYAML."""
    linhas = texto.splitlines()
    blocos: list[str] = []
    indice = 0
    while indice < len(linhas):
        linha = linhas[indice]
        bloco = re.match(r"^(\s*)run:\s*\|\s*$", linha)
        if bloco:
            recuo = len(bloco.group(1))
            corpo: list[str] = []
            indice += 1
            while indice < len(linhas):
                seguinte = linhas[indice]
                if seguinte.strip() == "":
                    corpo.append(seguinte)
                    indice += 1
                    continue
                if len(seguinte) - len(seguinte.lstrip()) <= recuo:
                    break
                corpo.append(seguinte)
                indice += 1
            blocos.append("\n".join(corpo))
            continue
        inline = re.match(r"^\s*run:\s*(.+)$", linha)
        if inline:
            blocos.append(inline.group(1))
        indice += 1
    return blocos


def medir(texto: str) -> tuple[int, list[str]]:
    """Devolve (quantidade de passos com gofmt, violacoes)."""
    passos = 0
    violacoes: list[str] = []
    for bloco in blocos_run(texto):
        if "gofmt" not in bloco:
            continue
        passos += 1
        for forma, descricao in FORMAS_CEGAS:
            if forma.search(bloco):
                violacoes.append("passo com gofmt usa %s" % descricao)
                break
        else:
            if not any(forma.search(bloco) for forma in FORMAS_QUE_CONFEREM):
                violacoes.append(
                    "passo com gofmt nao confere o codigo de saida (aceito: `if ! x=\"$(gofmt ...)\"`, "
                    "`gofmt ... || exit`, `if ! gofmt ...`) — *stdout vazio nao e' prova de sucesso*")
    return passos, violacoes


def alvos(argv: list[str]) -> list[pathlib.Path]:
    if len(argv) > 1:
        return [pathlib.Path(argv[1])]
    return sorted(WORKFLOWS.glob("*.yaml")) + sorted(WORKFLOWS.glob("*.yml"))


def main(argv: list[str]) -> int:
    arquivos = alvos(argv)
    if not arquivos:
        print("NADA MEDIDO: nenhum workflow encontrado em %s — isto NAO e aprovacao" % WORKFLOWS)
        return 1
    total_passos = 0
    total_violacoes: list[str] = []
    for caminho in arquivos:
        if not caminho.is_file():
            print("AUSENTE: %s" % caminho)
            return 1
        passos, violacoes = medir(caminho.read_text(encoding="utf-8"))
        total_passos += passos
        total_violacoes.extend("%s: %s" % (caminho.name, v) for v in violacoes)
    print("GATE DO GOFMT: %d passo(s) com gofmt em %d arquivo(s)" % (total_passos, len(arquivos)))
    if total_passos == 0:
        print("NADA MEDIDO: nenhum passo usa gofmt — *o gate pode ter sido removido ou renomeado; "
              "ausencia de medicao nao e' aprovacao*")
        return 1
    for violacao in total_violacoes:
        print("VIOLACAO: %s" % violacao)
    if not total_violacoes:
        print("  (todos os passos conferem o codigo de saida do gofmt)")
    return 1 if total_violacoes else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
