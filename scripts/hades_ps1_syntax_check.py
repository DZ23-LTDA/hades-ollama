#!/usr/bin/env python3
"""HADES -- catraca de SINTAXE dos scripts PowerShell (E-1292).

O DEFEITO QUE ESTE INSTRUMENTO IMPEDE

O `scripts/hades_local_gates.ps1` -- a bateria local de gates, documentada em 11 ficheiros -- tinha, na
linha 787:

    '--- 44. SNAPSHOT: o `measured_commit` do estado e' o HEAD antes de gerar os derivados'

Um apostrofo interno (`e'`) dentro de string delimitada por `'` fecha a string. O PowerShell **analisa
o script inteiro antes de correr uma linha**, por isso o parse abortado significa que **nenhum gate
local corria**. O script esteve partido sem ninguem notar porque **o CI nao o executava** e porque o
unico checker que olhava para `.ps1` (`hades_ascii_ps1_check.py`) verifica **ASCII, nao sintaxe**.

    *Um gate que nao arranca nao e' um gate: e' um ficheiro com nome de gate.*

    *Um checker que mede a propriedade errada nao e' meio checker: e' um selo de aprovacao.*

PORQUE ESTE CHECKER NAO USA REGEX

Contar apostrofos por linha parece suficiente e nao e'. Here-strings (`@'...'@`), strings com
apostrofos duplicados (`''`), comentarios de bloco e `@\"...\"@` mudam a contagem. A unica autoridade
sobre a sintaxe do PowerShell e' o proprio PowerShell: o checker delega nele.

    *Um instrumento que reimplementa o parser mede a sua propria ideia de sintaxe.*
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"

# O avaliador PowerShell: tokeniza cada ficheiro e devolve o numero de erros. Uma so' invocacao para
# todos, porque arrancar o powershell custa ~1 s por chamada.
#
# A LISTA DE FICHEIROS VEM DE UM FICHEIRO, E NAO DE `$args`
#
# A primeira versao passava os caminhos como argumentos de `-Command`. O PowerShell tratou o primeiro
# caminho como parte do proprio comando e o avaliador devolveu "O arquivo ... nao foi encontrado" para
# os 14 ficheiros -- um falso vermelho total. A lista vai por um JSON temporario cujo caminho chega por
# variavel de ambiente.
#
#     *Um instrumento que nao consegue entregar os alvos ao medidor reporta erro sobre si mesmo, e nao
#     sobre os alvos.*
AVALIADOR = r"""
$ErrorActionPreference = 'Stop'
$lista = Get-Content -Raw -LiteralPath $env:HADES_PS1_LISTA | ConvertFrom-Json
$resultado = @()
foreach ($f in $lista) {
    try {
        $erros = $null
        $null = [System.Management.Automation.PSParser]::Tokenize(
            [System.IO.File]::ReadAllText($f), [ref]$erros)
        $n = @($erros).Count
        if ($n -eq 0) {
            $resultado += "$f|OK|"
        } else {
            $primeiro = $erros[0]
            $msg = ($primeiro.Message -replace '\|', '/') -replace "`r?`n", ' '
            $linha = $primeiro.Token.StartLine
            $resultado += "$f|ERRO|${n} erro(s); primeiro na linha ${linha}: $msg"
        }
    } catch {
        $msg = ($_.Exception.Message -replace '\|', '/') -replace "`r?`n", ' '
        $resultado += "$f|EXCECAO|$msg"
    }
}
$resultado | ConvertTo-Json -Compress
"""


def ficheiros_ps1() -> list[Path]:
    """Todos os `.ps1` versionados em `scripts/`, ordenados.

    Ordenados porque a saida de um gate tem de ser estavel: um gate que muda de ordem entre
    execucoes nao pode ser comparado com o de ontem.
    """
    if not SCRIPTS.is_dir():
        return []
    return sorted(SCRIPTS.glob("*.ps1"))


def avaliar_com_powershell(ficheiros: list[Path]) -> dict[str, tuple[str, str]]:
    """Devolve `{caminho: (estado, detalhe)}` conforme o PowerShell reporta."""
    if not ficheiros:
        return {}
    with tempfile.TemporaryDirectory() as pasta:
        lista = Path(pasta) / "alvos.json"
        lista.write_text(
            json.dumps([str(f) for f in ficheiros], ensure_ascii=False), encoding="utf-8"
        )
        ambiente = dict(os.environ)
        ambiente["HADES_PS1_LISTA"] = str(lista)
        conclusao = subprocess.run(
            ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", AVALIADOR],
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=180,
            env=ambiente,
        )
    saida = (conclusao.stdout or "").strip()
    if not saida:
        # Sem saida nao ha' veredicto. Nao converter ausencia de medida em aprovacao.
        return {
            str(f): (
                "EXCECAO",
                "o avaliador PowerShell nao devolveu saida: " + (conclusao.stderr or "")[:200],
            )
            for f in ficheiros
        }
    try:
        bruto = json.loads(saida)
    except json.JSONDecodeError:
        return {
            str(f): ("EXCECAO", "saida do avaliador nao e' JSON: " + saida[:200])
            for f in ficheiros
        }
    if isinstance(bruto, str):
        bruto = [bruto]
    resultado: dict[str, tuple[str, str]] = {}
    for item in bruto:
        partes = str(item).split("|", 2)
        if len(partes) != 3:
            continue
        resultado[partes[0]] = (partes[1], partes[2])
    return resultado


def avaliar(medicoes: dict[str, tuple[str, str]]) -> list[str]:
    """Problemas a partir das medicoes. Lista vazia = conforme."""
    problemas: list[str] = []
    if not medicoes:
        return ["nenhum ficheiro .ps1 medido: o instrumento nao consegue aprovar o que nao le'"]
    for caminho in sorted(medicoes):
        estado, detalhe = medicoes[caminho]
        nome = Path(caminho).name
        if estado == "OK":
            continue
        if estado == "ERRO":
            problemas.append(
                "SINTAXE: %s -> %s. O PowerShell analisa o script inteiro antes de correr: "
                "sintaxe invalida significa que NENHUMA linha do script executa" % (nome, detalhe)
            )
        else:
            problemas.append("MEDICAO: %s -> %s" % (nome, detalhe))
    return problemas


def main() -> int:
    ficheiros = ficheiros_ps1()
    if not ficheiros:
        print("PS1 SINTAXE: nenhum ficheiro .ps1 em scripts/")
        return 1

    medicoes = avaliar_com_powershell(ficheiros)
    problemas = avaliar(medicoes)

    if problemas:
        print("PS1 SINTAXE: %d PROBLEMA(S) em %d ficheiro(s)" % (len(problemas), len(ficheiros)))
        for problema in problemas:
            print("  - %s" % problema)
        return 1

    print(
        "PS1 SINTAXE: os %d scripts .ps1 parseiam. Nenhum tem sintaxe invalida -- e' isto que impede "
        "um gate local partido de passar por saudavel." % len(ficheiros)
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
