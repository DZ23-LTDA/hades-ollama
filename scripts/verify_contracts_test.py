#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""Teste do verify-contracts: o gate TEM de comparar o caminho NORMALIZADO.

O DEFEITO (E-1344, provado com um probe real):

    `agentFetch("/plugins")`  ->  raw  = "/plugins"
                                  path = "/api/agent/v1/plugins"

Com `routeMatches(call.raw, ...)`, o gate comparava "/plugins" com
"/api/agent/v1/plugins" e escrevia:

    [FAIL] GET /api/agent/v1/plugins -> NO MATCH

-- a rota que o log diz nao encontrar esta' no prefixo do proprio log.

Passou despercebido porque todos os clientes usam o caminho completo, onde
raw == path. O defeito so' aparecia com o caminho curto -- e nesse dia o gate
acusava o PRODUTO pelo que o INSTRUMENTO fazia.

    *Um gate que compara o texto cru com o caminho normalizado nao compara dois
    caminhos: compara duas maneiras de escrever o mesmo caminho, e chama-lhes
    nomes diferentes.*
"""
import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ALVO = os.path.join(RAIZ, "scripts", "verify-contracts.mjs")

falhas = []


def verificar(condicao, mensagem):
    if not condicao:
        falhas.append(mensagem)
        print("  [FAIL] " + mensagem)
    else:
        print("  [ok]   " + mensagem)


def main():
    if not os.path.exists(ALVO):
        print("FALHOU: verify-contracts.mjs nao existe")
        return 1

    texto = io.open(ALVO, encoding="utf-8").read()

    print("=== verify-contracts: o gate compara o caminho normalizado? ===")

    # 1. A funcao `matches` tem de usar `call.path`.
    verificar(
        "routeMatches(call.path, route.path)" in texto,
        "matches() usa call.path (o caminho normalizado, com prefixo)",
    )

    # 2. NAO pode voltar a usar `call.raw` -- e' o defeito.
    verificar(
        "routeMatches(call.raw, route.path)" not in texto,
        "matches() NAO usa call.raw (o texto cru) -- defeito E-1344",
    )

    # 3. `raw` continua a ser recolhido (outros consumidores podem usa-lo), mas nao para
    #    decidir o match.
    verificar(
        "raw," in texto or " raw " in texto,
        "o campo raw continua a existir (nao foi removido, so' deixou de decidir)",
    )

    # 4. O comentario do defeito tem de sobreviver a proxima limpeza.
    verificar(
        "E-1344" in texto,
        "o comentario cita o E-1344 (a proxima limpeza nao pode apagar a licao)",
    )

    # 5. A deduplicacao continua a usar `path` (o id estavel).
    verificar(
        re.search(r"new Map\(frontendCalls\.map\(c => \[`\$\{c\.method\} \$\{c\.path\}", texto)
        is not None,
        "a deduplicacao usa path (id estavel), nao raw",
    )

    print()
    if falhas:
        print("FALHOU: %d verificacao(oes) do gate de contratos." % len(falhas))
        return 1
    print("PASSOU: %d verificacoes do gate de contratos." % 5)
    return 0


if __name__ == "__main__":
    sys.exit(main())
