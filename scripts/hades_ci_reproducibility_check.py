#!/usr/bin/env python3
"""HADES — CI REPRODUTÍVEL (E-306/E-309).

POR QUE ESTE INSTRUMENTO EXISTE
-------------------------------
MEDIDO em E-306: `main` estava **verde** em `22307bf41` e, **20 minutos depois**, o mesmo gate
(`Go agentic and server gates`) **reprovou sem que ninguém tocasse no código** — porque o passo

    - uses: golangci/golangci-lint-action@<sha> # v9

**não fixava versão**, e o padrão da action é `latest`. O lint baixou a v2.14.0, cujo dicionário de `misspell` passou
a marcar **27 palavras portuguesas** (`dependencias`, `proposito`, `MOMENTO`…) em arquivos de um repositório cuja
língua é pt-BR.

***Um gate que se move sozinho não mede o código: mede o dia.***

Este instrumento transforma aquela lição em GUARDA: ele reprova o repositório quando um gate de CI pode mudar de
versão sem revisão. Ele NÃO desliga linter nenhum — apenas exige que a versão seja uma DECISÃO.

REGRAS
------
  A. `version: latest` (ou variante) em qualquer workflow ⇒ violação;
  B. passo usando `golangci/golangci-lint-action` sem `with: version:` ⇒ violação (o defeito medido em E-306);
  C. `uses:` de action EXTERNA precisa de **SHA de commit** (40 hex). Tag (`@v4`) é MÓVEL: o mantenedor pode
     movê-la sem aviso — ***mesma classe do `version: latest`***. Sem referência também é violação.
     **Exceção**: as declaradas em `EXCECOES_DECLARADAS`, com motivo escrito — *exceção visível, não silenciosa*.

LIMITE DECLARADO: este instrumento audita a **declaração** nos workflows. Ele não baixa nada nem prova que a versão
fixada existe — provar isso exige rede e o registry, e fica fora do escopo de um checker hermérico.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

WORKFLOW_DIR = Path(".github/workflows")
# Actions cuja versão da FERRAMENTA é escolhida por `with: version:` — as que já reprovaram por flutuação.
ACTIONS_COM_VERSAO_EXIGIDA = ("golangci/golangci-lint-action",)
PADRAO_USES = re.compile(r"^\s*-?\s*uses:\s*(\S+)")
PADRAO_VERSION = re.compile(r"^\s*version:\s*['\"]?([^'\"\s#]+)")
PADRAO_WITH = re.compile(r"^\s*with:\s*$")
PADRAO_SHA = re.compile(r"@[0-9a-f]{40}$")

# EXCEÇÕES DECLARADAS (ação -> motivo). Toda exceção aqui está VISÍVEL e datada; o objetivo é zerar esta lista.
EXCECOES_DECLARADAS = {
    "Jimver/cuda-toolkit@v0.2.21": (
        "E-309: única ação fora de SHA no repositório (143 de 144 usam SHA). Fixar exige o SHA do commit da tag, "
        "que não se inventa — declarada em vez de silenciada. Desbloqueio: trocar pelo SHA da v0.2.21."
    ),
}


def workflows(diretorio: Path = WORKFLOW_DIR) -> list[Path]:
    """Devolve os arquivos de workflow, em ordem estável."""
    if not diretorio.is_dir():
        return []
    return sorted([*diretorio.glob("*.yaml"), *diretorio.glob("*.yml")])


def analisar_workflow(caminho: Path, texto: str | None = None) -> list[str]:
    """Devolve a lista de violações do arquivo (vazia = conforme)."""
    conteudo = texto if texto is not None else caminho.read_text(encoding="utf-8")
    linhas = conteudo.split("\n")
    violacoes: list[str] = []

    for numero, linha in enumerate(linhas, 1):
        # Regra A: versão flutuante declarada.
        achado = PADRAO_VERSION.match(linha)
        if achado and achado.group(1).lower() in {"latest", "master", "main", "head"}:
            violacoes.append(f"{caminho}:{numero}: versão flutuante '{achado.group(1)}' — fixe uma versão (E-306)")

        # Regra C: action externa sem referência.
        uses = PADRAO_USES.match(linha)
        if uses:
            referencia = uses.group(1)
            if not referencia.startswith("./") and referencia not in EXCECOES_DECLARADAS:
                if "@" not in referencia:
                    violacoes.append(f"{caminho}:{numero}: action externa sem referência: {referencia}")
                elif not PADRAO_SHA.search(referencia):
                    violacoes.append(
                        f"{caminho}:{numero}: action externa presa a TAG MÓVEL ({referencia}) — use o SHA do commit"
                    )

            # Regra B: action que exige `with: version:` e não declara nenhuma.
            for action in ACTIONS_COM_VERSAO_EXIGIDA:
                if referencia.startswith(action):
                    if not _tem_version_no_bloco(linhas, numero):
                        violacoes.append(
                            f"{caminho}:{numero}: {action} sem `with: version:` — o padrão é `latest` (E-306)"
                        )
    return violacoes


def _tem_version_no_bloco(linhas: list[str], linha_do_uses: int) -> bool:
    """Diz se o passo que começa em `linha_do_uses` (1-based) declara `version:` antes do próximo passo."""
    indentacao_do_uses = len(linhas[linha_do_uses - 1]) - len(linhas[linha_do_uses - 1].lstrip())
    em_with = False
    for linha in linhas[linha_do_uses:]:
        if not linha.strip() or linha.lstrip().startswith("#"):
            continue
        indentacao = len(linha) - len(linha.lstrip())
        if indentacao <= indentacao_do_uses and PADRAO_USES.match(linha):
            return False  # começou outro passo
        if PADRAO_WITH.match(linha):
            em_with = True
            continue
        if em_with:
            if indentacao <= indentacao_do_uses:
                return False  # saiu do bloco `with:`
            if PADRAO_VERSION.match(linha):
                return True
    return False


def auditar(diretorio: Path = WORKFLOW_DIR) -> list[str]:
    """Audita todos os workflows e devolve as violações encontradas."""
    violacoes: list[str] = []
    for caminho in workflows(diretorio):
        violacoes.extend(analisar_workflow(caminho))
    return violacoes


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Audita reprodutibilidade dos gates de CI (E-306/E-309).")
    parser.add_argument("--check", action="store_true", help="falha (exit 1) se houver violação")
    parser.add_argument("--workflows", default=str(WORKFLOW_DIR), help="diretório dos workflows")
    args = parser.parse_args(argv)

    violacoes = auditar(Path(args.workflows))
    if violacoes:
        for violacao in violacoes:
            print(f"VIOLACAO: {violacao}")
        print(f"\n{len(violacoes)} violação(ões): o gate pode mudar sem revisão.")
        return 1
    print("CI REPRODUTÍVEL: nenhum gate flutuante — toda versão é uma decisão declarada.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
