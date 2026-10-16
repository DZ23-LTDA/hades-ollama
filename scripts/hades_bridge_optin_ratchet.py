#!/usr/bin/env python3
"""CATRACA SOBRE O CODIGO: todo o chamador de `validarUrl`/`guardarRota` passa o opt-in.

POR QUE ISTO EXISTE (`E-1036`).

O defeito do opt-in de rede privada apareceu em **tres locais sucessivos**:

  * `conectarAbaAtiva` (curado no `E-1033`)
  * `tabs.onUpdated`    (curado no `E-1033`)
  * `tratarAccao`       (curado no `E-1035`, e so' descoberto porque o E2E mediu)

Cada cura foi **pontual**: resolvia o sintoma que se tinha medido. *Uma cura pontual nao
fecha a classe do defeito* -- e a quarta ocorrencia seria igual, silenciosa, e so'
apareceria noutro E2E, noutra ronda.

Esta catraca nao mede o **resultado**; mede o **codigo**. Se alguem acrescentar um chamador
novo de `validarUrl` ou `guardarRota` sem o opt-in, isto fica VERMELHO -- antes de o defeito
chegar a correr.

O QUE ISTO **NAO** FAZ: nao prova que o opt-in esta' correto, nem que o valor lido e' o
certo. Prova que a **decisao foi tomada explicitamente** em todos os sitios.

ASCII puro.
"""
import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def definir_raiz(nova: str) -> None:
    """Aponta a catraca a outra raiz. Usado pelo TESTE, que trabalha numa COPIA.

    *Um teste que muta o codigo de producao e' um risco*: se for interrompido a meio, deixa
    o ficheiro com a injecao dentro. Com `--raiz`, o teste injeta numa copia descartavel e o
    repositorio nunca e' tocado.
    """
    global RAIZ, BRIDGE
    RAIZ = os.path.abspath(nova)
    BRIDGE = os.path.join(RAIZ, "apps", "hades-browser-bridge")


BRIDGE = os.path.join(RAIZ, "apps", "hades-browser-bridge")

# Onde procurar. Ficheiros de teste nao contam: eles chamam a funcao para a exercitar,
# nao para decidir uma rota de producao.
ALVOS = ["src", "popup"]

# As duas funcoes que validam uma URL contra a politica de rede privada.
FUNCOES = ("validarUrl", "guardarRota")

# A definicao em `protocol.js` e' a fonte: ela PROPAGA a opcao, nao a decide.
# `guardarRota` chama `validarUrl` internamente e repassa `opcoes.permitirPrivado`.
ISENTOS = {
    os.path.join("src", "protocol.js"),
}

# Ficheiros que sao ponto de entrada do navegador: o service worker e' quem le' o storage.
# Qualquer OUTRO ficheiro que chame a funcao tem de trazer a opcao -- e se nao conseguir
# ler o storage, tem de a receber de quem le'.

PADRAO = re.compile(r"\b(" + "|".join(FUNCOES) + r")\s*\(")
OPTIN = re.compile(r"permitirPrivado")
# Um comentario que mencione a funcao nao e' um chamador. *Uma string e' codigo, um
# comentario nao e'* -- e a distincao ja' me custou quatro falsos positivos.
COMENTARIO_LINHA = re.compile(r"//.*$")
COMENTARIO_BLOCO = re.compile(r"/\*.*?\*/", re.S)


def sem_comentarios(texto: str) -> str:
    """Remove comentarios PRESERVANDO as linhas, para o numero da linha continuar real."""
    texto = COMENTARIO_BLOCO.sub(lambda m: "\n" * m.group(0).count("\n"), texto)
    linhas = []
    for linha in texto.split("\n"):
        linhas.append(COMENTARIO_LINHA.sub("", linha))
    return "\n".join(linhas)


def ficheiros():
    for pasta in ALVOS:
        base = os.path.join(BRIDGE, pasta)
        if not os.path.isdir(base):
            continue
        for raiz, _, nomes in os.walk(base):
            for nome in sorted(nomes):
                if not nome.endswith(".js"):
                    continue
                rel = os.path.relpath(os.path.join(raiz, nome), BRIDGE).replace("\\", "/")
                yield rel, os.path.join(raiz, nome)


def analisar():
    """Devolve (chamadores, sem_optin)."""
    chamadores, sem_optin = [], []
    for rel, caminho in ficheiros():
        if rel in {i.replace("\\", "/") for i in ISENTOS}:
            continue
        with io.open(caminho, encoding="utf-8") as f:
            cru = f.read()
        texto = sem_comentarios(cru)
        for i, linha in enumerate(texto.split("\n"), start=1):
            m = PADRAO.search(linha)
            if not m:
                continue
            # Uma chamada pode trazer o opt-in na propria linha ou nas seguintes
            # (multilinha). Olhamos a linha e as DUAS seguintes -- o suficiente para o
            # estilo deste repositorio, e curto o bastante para nao apanhar um else distante.
            contexto = "\n".join(texto.split("\n")[i - 1:i + 2])
            tem = bool(OPTIN.search(contexto))
            chamadores.append((rel, i, m.group(1), tem))
            if not tem:
                sem_optin.append((rel, i, m.group(1), linha.strip()))
    return chamadores, sem_optin


def principal() -> int:
    chamadores, sem_optin = analisar()
    print("  CHAMADORES de %s: %d" % ("/".join(FUNCOES), len(chamadores)))
    for rel, linha, fn, tem in chamadores:
        print("    %-22s :%-4d %-12s opt-in=%s" % (rel, linha, fn, "SIM" if tem else "NAO"))
    if sem_optin:
        print("")
        print("  NAO VERIFICA -- chamadores SEM o opt-in:")
        for rel, linha, fn, txt in sem_optin:
            print("    %s:%d  %s(...)  ->  %s" % (rel, linha, fn, txt[:70]))
        print("")
        print("  *Um chamador novo sem `permitirPrivado` repete o defeito do E-1033/E-1035:*")
        print("  a mesma origem concede e depois recusa, e a incoerencia so' aparece num E2E.")
        print("  Passe `{ permitirPrivado: await lerOptIn() }` ou receba o valor de quem le'.")
        return 1
    print("")
    print("  TODOS os chamadores passam o opt-in explicitamente.")
    print("  *Um chamador novo sem `permitirPrivado` fica VERMELHO aqui, antes de correr.*")
    return 0


if __name__ == "__main__":
    # `--raiz CAMINHO` permite ao teste trabalhar numa copia descartavel.
    if len(sys.argv) >= 3 and sys.argv[1] == "--raiz":
        definir_raiz(sys.argv[2])
    sys.exit(principal())
