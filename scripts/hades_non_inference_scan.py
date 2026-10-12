#!/usr/bin/env python3
"""Scanner de ORÁCULOS DE ENUMERAÇÃO por assimetria de erro (E-191).

O QUE ELE PROCURA
-----------------
A forma do defeito B-15 (E-189/E-190): um método recebe `(organização, id)`, busca o recurso e devolve **erros
diferentes** para (a) "não existe" e (b) "existe, mas é de outro tenant". A diferença — de status ou de mensagem —
deixa um tenant descobrir quais IDs existem sem acessar dado nenhum.

A DETECÇÃO, EM DUAS ETAPAS (reescrita depois de MEDIR que a primeira versão era cega)
------------------------------------------------------------------------------------
A primeira versão usava um único regex e a calibração mostrou os dois lados do problema: ela apontou **6 falsos
positivos** (o padrão de "id" casava com a própria `organizationID`) e, depois de corrigir isso, passou a **não
encontrar nada** — inclusive a forma que eu tinha acabado de corrigir à mão. Um instrumento que erra para os dois
lados não serve para julgar código.

A detecção agora é um pequeno autômato de leitura:

1. **portão 1** — existe um `if err != nil { ... return ... err }` que devolve o erro da busca **cru**, sem normalizar;
2. **portão 2** — depois dele, uma comparação de organização cujo ramo devolve **outra** coisa que não aquele mesmo
   `err` (erro diferente ou mensagem própria).

Os dois juntos são a forma do oráculo. **Não é prova**: pode não vazar nada (as duas negações podem coincidir no
transporte). Por isso a saída é lista de suspeitos para verificação, e cada um precisa de um teste que compare as duas
recusas — como faz o `server/company_scope_non_inference_test.go`.

CALIBRAÇÃO
----------
`scripts/hades_non_inference_scan_test.py` alimenta o scanner com a forma **pré-correção** (positivo conhecido) e com a
**pós-correção** (negativo conhecido). A calibração roda no CI: um scanner que não acha o que já foi encontrado à mão
está cego, e um que acusa a versão corrigida está quebrado.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ALVOS = ("internal/agent", "server")

ASSINATURA = re.compile(r"func\s+\([^)]*\)\s+(?P<nome>\w+)\((?P<params>[^)]*)\)")
PARAM_ORG = re.compile(r"(?P<nome>\w*(?:organization|org)\w*)\s+string", re.I)
PARAM_ID = re.compile(r"(?P<nome>\w*id)\s+string", re.I)
PARAM_NAO_ORG = re.compile(r"(?i)organization|org")

GETTER = re.compile(r"=\s*[\w.]+\.(?:Get|Load|Find|Fetch|Lookup)\(")
IF_ERR = re.compile(r"if\s+err\s*!=\s*nil\s*\{")
RETURN_ERR = re.compile(r"return\b[^\n]*\berr\b")
COMPARA_ORG = re.compile(r"(organizationID|OrganizationID|consentScope\()", re.I)


def _bloco(texto: str, inicio: int) -> tuple[str, int]:
    """Devolve o corpo do bloco `{...}` que começa em `inicio` e a posição seguinte ao fechamento."""
    profundidade = 0
    for posicao in range(inicio, len(texto)):
        if texto[posicao] == "{":
            profundidade += 1
        elif texto[posicao] == "}":
            profundidade -= 1
            if profundidade == 0:
                return texto[inicio + 1 : posicao], posicao + 1
    return texto[inicio + 1 :], len(texto)


def corpo_do_metodo(texto: str, inicio: int) -> str:
    chave = texto.find("{", inicio)
    if chave < 0:
        return ""
    corpo, _ = _bloco(texto, chave)
    return corpo


def nomes_de_string(params: str) -> list[str]:
    """Nomes de parâmetros declarados como `string`, tratando DECLARAÇÃO AGRUPADA.

    Go permite `func f(id, organizationID string)`: dois nomes, um tipo. A primeira versão deste scanner exigia
    `nome string` colado e por isso era CEGA a assinaturas agrupadas — foi a calibração (E-191) que mostrou isso, ao
    não encontrar o positivo conhecido. Este é o conserto.
    """
    nomes: list[str] = []
    acumulado: list[str] = []
    for bruto in params.split(","):
        partes = bruto.strip().split()
        if not partes:
            continue
        if partes[-1] == "string":
            acumulado.extend(partes[:-1])
            nomes.extend(acumulado)
            acumulado = []
        elif len(partes) > 1:
            # Outro tipo explícito: descarta os nomes acumulados que pertenciam a ele.
            acumulado = []
        else:
            acumulado.append(partes[0])
    return nomes


def detectar(nome: str, params: str, corpo: str) -> dict | None:
    """Aplica os dois portões. Devolve o detalhe do suspeito, ou None."""
    nomes = nomes_de_string(params)
    organizacao = next((n for n in nomes if PARAM_NAO_ORG.search(n)), None)
    identificador = next(
        (n for n in nomes if n.lower().endswith("id") and not PARAM_NAO_ORG.search(n) and n != organizacao),
        None,
    )
    if not organizacao or not identificador:
        return None
    if not COMPARA_ORG.search(corpo):
        return None

    # Portão 1: o erro da busca é repassado CRU.
    portao1 = False
    for getter in GETTER.finditer(corpo):
        restante = corpo[getter.end() :]
        iferr = IF_ERR.search(restante)
        if not iferr:
            continue
        bloco, _ = _bloco(restante, iferr.end() - 1)
        if RETURN_ERR.search(bloco):
            portao1 = True
            break
    if not portao1:
        return None

    # Portão 2: o ramo que compara organização devolve algo diferente do erro cru. ATENÇÃO ao formato real: a
    # comparação é a PRÓPRIA condição do `if` (`if company.OrganizationID != organizationID { return ... }`), não um
    # `if err != nil` que vem depois — foi assim que a versão anterior ficou cega ao positivo conhecido.
    for comparacao in COMPARA_ORG.finditer(corpo):
        chave = corpo.find("{", comparacao.end())
        if chave < 0 or chave - comparacao.end() > 200:
            continue
        bloco, _ = _bloco(corpo, chave)
        if "return" in bloco and "err" not in bloco:
            return {
                "motivo": "ramo de organização devolve erro/mensagem própria",
                "ramo": " ".join(bloco.split())[:160],
            }
    return None


def metodos_suspeitos(texto: str, arquivo: str) -> list[dict]:
    suspeitos: list[dict] = []
    for assinatura in ASSINATURA.finditer(texto):
        corpo = corpo_do_metodo(texto, assinatura.end())
        if not corpo:
            continue
        detalhe = detectar(assinatura.group("nome"), assinatura.group("params"), corpo)
        if detalhe is None:
            continue
        suspeitos.append(
            {
                "arquivo": arquivo,
                "linha": texto[: assinatura.start()].count("\n") + 1,
                "metodo": assinatura.group("nome"),
                "parametros": " ".join(assinatura.group("params").split()),
                **detalhe,
            }
        )
    return suspeitos


def varrer() -> list[dict]:
    achados: list[dict] = []
    for alvo in ALVOS:
        for caminho in sorted((RAIZ / alvo).rglob("*.go")):
            if caminho.name.endswith("_test.go"):
                continue
            try:
                texto = caminho.read_text(encoding="utf-8")
            except (UnicodeDecodeError, OSError):
                continue
            achados.extend(metodos_suspeitos(texto, str(caminho.relative_to(RAIZ)).replace("\\", "/")))
    return achados


def main() -> int:
    parser = argparse.ArgumentParser(description="Aponta candidatos a oráculo de enumeração por assimetria de erro")
    parser.add_argument("--json", action="store_true", help="saída estruturada")
    args = parser.parse_args()

    achados = varrer()
    if args.json:
        print(json.dumps(achados, ensure_ascii=False, indent=2))
        return 0

    print("# Candidatos a verificação (heurístico de código, NÃO veredito)")
    print()
    print(f"Encontrados: {len(achados)}")
    print()
    if achados:
        print("| arquivo | linha | método | motivo |")
        print("| --- | --- | --- | --- |")
        for achado in achados:
            print(f"| {achado['arquivo']} | {achado['linha']} | `{achado['metodo']}` | {achado['motivo']} |")
        print()
    print(
        "Cada candidato precisa de um teste que compare as duas recusas (não existe × é de outro tenant) antes de ser\n"
        "considerado vazamento OU considerado seguro. Apontar não é provar."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
