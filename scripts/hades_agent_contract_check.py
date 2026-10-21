#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Gate de CONTRATO entre o Agent Center (painel) e o servidor.

O PROBLEMA QUE ESTE GATE FECHA
==============================

O `hades_rotas_orfas.py` prova que toda rota tem consumidor. Nao prova que o consumidor e o produtor falam a
mesma lingua:

    *Um painel provado contra um mock e um backend provado contra `httptest` nao provam que um fala com o
    outro.*

Um caminho pode existir e a chave que o painel le' pode nao existir na resposta. O TypeScript nao apanha:
`relatorio.inference_available` e' `boolean | undefined`, e `undefined` renderiza como nada. O painel mostra
um campo vazio e ninguem sabe se e' "falso" ou "nao enviado".

    *Um estado que mente e' pior do que nenhum.*

O QUE ELE MEDE
==============

1. Toda chave JSON que o `agentCenterClient.ts` LE tem de ser EMITIDA em algum sitio do servidor
   (`server/agent_center_routes.go`, `server/agent_eval_runner.go`) ou de um struct de `internal/agent/`
   com tag `json:"..."`.

2. Toda funcao exportada do client tem de ter caminho `/api/agent/v1/...` e esse caminho tem de existir no
   router -- a metade que o `rotas_orfas` ja' cobre, repetida aqui para o gate ser auto-suficiente.

O QUE ELE NAO MEDE
==================

Nao corre o servidor. Nao valida tipos nem nulabilidade. Prova que os NOMES batem; a forma exacta (e o valor
real) continua a precisar de um pedido HTTP a serio. Nao e' um substituto do E2E -- e' o piso.

Uso: python scripts/hades_agent_contract_check.py [--check]
"""

import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

CLIENT = os.path.join(RAIZ, "app", "ui", "app", "src", "lib", "agentCenterClient.ts")
FONTES_SERVIDOR = [
    os.path.join(RAIZ, "server", "agent_center_routes.go"),
    os.path.join(RAIZ, "server", "agent_eval_runner.go"),
]
ROTAS = os.path.join(RAIZ, "server", "agent_routes.go")
DIR_AGENT = os.path.join(RAIZ, "internal", "agent")

# Chaves que o client le da resposta. Extraidas por regex de `interface` e de acessos `x.chave`.
# A lista e' derivada, nao escrita a mao -- ver `chaves_do_client`.
IGNORAR = {
    # Nao sao chaves de resposta: sao opcoes de RequestInit, tipos e palavras do JS.
    "method", "body", "headers", "POST", "GET", "json", "string", "number", "boolean",
    "undefined", "length", "trim", "map", "filter", "join", "slice", "replace", "toString",
}


def ler(caminho):
    return io.open(caminho, encoding="utf-8").read()


def chaves_do_client(texto):
    """Chaves que o painel LE da resposta.

    Duas fontes: campos declarados nas `interface` (o contrato que o autor escreveu) e acessos literais
    `x["chave"]`. Campos de `interface` sao a fonte principal -- sao o que o painel espera receber.
    """
    chaves = set()
    # Campos dentro de blocos `interface Nome { ... }`.
    for bloco in re.finditer(r"interface\s+\w+\s*\{([^}]*)\}", texto, re.S):
        for linha in bloco.group(1).split("\n"):
            m = re.match(r"\s*(\w+)\??\s*:", linha)
            if m:
                chaves.add(m.group(1))
    # Acessos literais.
    for m in re.finditer(r'\[\s*"([a-z][a-z0-9_]*)"\s*\]', texto):
        chaves.add(m.group(1))
    return sorted(c for c in chaves if c not in IGNORAR)


def chaves_do_servidor(fontes, dir_agent):
    """Chaves que o servidor EMITE: literais `gin.H{...}` e tags `json:"..."` dos structs."""
    emitidas = set()
    for caminho in fontes:
        if not os.path.exists(caminho):
            continue
        texto = ler(caminho)
        for m in re.finditer(r'"([a-z][a-z0-9_]*)"\s*:', texto):
            emitidas.add(m.group(1))
        for m in re.finditer(r'json:"([a-z][a-z0-9_]*)[,"]', texto):
            emitidas.add(m.group(1))
    if os.path.isdir(dir_agent):
        for nome in os.listdir(dir_agent):
            if not nome.endswith(".go") or nome.endswith("_test.go"):
                continue
            texto_go = ler(os.path.join(dir_agent, nome))
            for m in re.finditer(r'json:"([a-z][a-z0-9_]*)[,"]', texto_go):
                emitidas.add(m.group(1))
            # Mapas literais `gin.H`/`map[string]any` tambem EMITEM chaves -- sem isto o gate acusava de
            # orfa uma chave que existe (foi o caso de `can_go_to` e `terminal`).
            for m in re.finditer(r'"([a-z][a-z0-9_]*)"\s*:', texto_go):
                emitidas.add(m.group(1))
    return emitidas


def rotas_do_router(texto):
    """Caminhos registados no router do agent API."""
    return set(re.findall(r'group\.(?:GET|POST|PUT|DELETE|PATCH)\("([^"]+)"', texto))


def caminhos_do_client(texto):
    """Caminhos que o client chama, sem o prefixo /api/agent/v1."""
    achados = re.findall(r'"(/api/agent/v1/[^"]+)"', texto)
    return sorted({c[len("/api/agent/v1"):] for c in achados})


def main():
    problemas = []

    if not os.path.exists(CLIENT):
        print("  AUSENTE: %s" % CLIENT)
        return 1
    texto_client = ler(CLIENT)

    chaves_client = chaves_do_client(texto_client)
    chaves_servidor = chaves_do_servidor(FONTES_SERVIDOR, DIR_AGENT)

    faltando = [c for c in chaves_client if c not in chaves_servidor]

    print("  CONTRATO PAINEL <-> SERVIDOR (Agent Center)")
    print("  chaves lidas pelo painel : %d" % len(chaves_client))
    print("  chaves emitidas pelo servidor: %d" % len(chaves_servidor))
    print("")

    for c in faltando:
        problemas.append(
            "CHAVE ORFA: o painel le' `%s` e o servidor nao a emite -- o painel mostra vazio e nao "
            "distingue 'falso' de 'nao enviado'" % c
        )

    # Rotas: as que o client chama tem de existir no router.
    caminhos = caminhos_do_client(texto_client)
    registadas = rotas_do_router(ler(ROTAS)) if os.path.exists(ROTAS) else set()
    print("  caminhos chamados pelo painel: %d   rotas /agents registadas: %d" % (len(caminhos), len(registadas)))
    for c in caminhos:
        # Comparacao por substring: o router pode ter subgrupos.
        alvo = c.rsplit("/", 1)[-1]
        if not any(c == r or c.endswith(r) or r.endswith(alvo) for r in registadas):
            problemas.append("ROTA AUSENTE: o painel chama `%s` e nenhuma rota registada corresponde" % c)
    print("")

    if problemas:
        print("  PROBLEMAS:")
        for p in problemas:
            print("    - %s" % p)
        print("")
        print("  VERMELHO: %d problema(s) de contrato." % len(problemas))
        return 1

    print("  VERDE: toda chave que o painel le' e' emitida pelo servidor; todo caminho existe no router.")
    print("  (prova os NOMES -- nao o valor. O E2E num navegador continua a ser outro gate.)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
