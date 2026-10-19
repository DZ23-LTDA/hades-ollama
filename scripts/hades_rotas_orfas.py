# -*- coding: utf-8 -*-
"""GATE: rotas de negocio que o backend regista e o frontend do HADES nao opera.

NASCE DE DOIS DEFEITOS SEGUIDOS. O E-1316 e o E-1318 foram o mesmo padrao:
codigo escrito, testado, e nunca ligado ao produto. Achei-os um de cada vez,
por sorte, ao trabalhar na fatia. Este instrumento procura o padrao em TODAS
as rotas de uma vez.

NAO E' UM GATE DE "ESTA TUDO LIGADO". Ha' rotas que nao devem estar no UI:
webhooks (quem chama e' o canal externo), metricas (o Prometheus raspa), SAML
(o IdP redirecciona), MCP (o cliente MCP chama), pareamento (o celular
completa). Essas sao orfandade CORRECTA e estao declaradas em CONSUMIDOR_EXTERNO.

O gate mede o resto: rota de negocio com backend pronto e sem superficie no
Mission Control.

    *Uma rota que existe e nao e' consumida e' uma promessa do backend, nao uma
    funcionalidade do produto.*

O TETO NAO E' UMA APROVACAO. E' o numero de hoje, congelado para que a proxima
ronda nao o aumente sem que alguem escreva porque. Baixar o teto e' progresso;
subir exige justificacao no proprio commit.
"""
import io
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Numero medido em 2026-10-19 no commit 77e58c183. Ver E-1319.
TETO_ORFAS = 93

# Quem chama nao e' o frontend do HADES -- a ausencia no UI e' correcta.
CONSUMIDOR_EXTERNO = (
    ("/omnichannel/webhooks/", "o canal externo chama o HADES"),
    ("/telegram/", "o Telegram chama"),
    ("/whatsapp/", "o WhatsApp chama"),
    ("/metrics/", "o Prometheus raspa"),
    ("/auth/saml/", "o IdP redirecciona"),
    ("/auth/dev/token", "so' existe fora de producao"),
    ("/mcp/rpc", "o cliente MCP chama"),
    ("/devices/pair/complete", "o celular completa o pareamento"),
    ("/egress/", "diagnostico de rede do operador"),
    ("/commands", "superficie de CLI"),
)


def rotas_do_backend():
    caminho = os.path.join(RAIZ, "server", "agent_routes.go")
    texto = io.open(caminho, encoding="utf-8", errors="replace").read()
    return [(m.group(1), m.group(2)) for m in
            re.compile(r'group\.(GET|POST|PUT|DELETE|PATCH)\(\s*"(/[^"]*)"').finditer(texto)]


def fonte_do_frontend():
    partes = []
    for sub in ("src", "e2e"):
        raiz = os.path.join(RAIZ, "app", "ui", "app", sub)
        if not os.path.isdir(raiz):
            continue
        for dirpath, dirnames, filenames in os.walk(raiz):
            dirnames[:] = [d for d in dirnames if d not in ("node_modules", "dist")]
            for nome in filenames:
                if nome.endswith((".ts", ".tsx", ".mjs", ".js")):
                    try:
                        partes.append(io.open(os.path.join(dirpath, nome),
                                              encoding="utf-8", errors="replace").read())
                    except OSError:
                        pass
    return "\n".join(partes)


def medir():
    """Devolve (total, orfas, externas, lacunas, detalhe_por_dominio)."""
    rotas = rotas_do_backend()
    fonte = fonte_do_frontend()
    orfas = []
    for metodo, caminho in rotas:
        prefixo = caminho.split(":")[0].rstrip("/") or caminho
        if prefixo not in fonte:
            orfas.append((metodo, caminho))
    externas = [r for r in orfas if any(r[1].startswith(p) for p, _ in CONSUMIDOR_EXTERNO)]
    lacunas = [r for r in orfas if r not in externas]
    dominios = {}
    for _, caminho in lacunas:
        dominios.setdefault(caminho.strip("/").split("/")[0], []).append(caminho)
    return len(rotas), len(orfas), len(externas), len(lacunas), dominios


def main():
    total, orfas, externas, lacunas, dominios = medir()
    print("rotas no backend      : %d" % total)
    print("orfas no frontend     : %d" % orfas)
    print("  consumidor externo  : %d  (orfandade correcta, declarada)" % externas)
    print("  lacuna de produto   : %d  (teto %d)" % (lacunas, TETO_ORFAS))
    print("")
    print("=== por dominio ===")
    for chave in sorted(dominios, key=lambda k: -len(dominios[k])):
        print("  %-15s %2d  %s" % (chave, len(dominios[chave]),
                                   " ".join(sorted(dominios[chave]))[:110]))
    print("")
    if lacunas > TETO_ORFAS:
        print("VERMELHO: %d lacunas, acima do teto %d." % (lacunas, TETO_ORFAS))
        print("Uma rota de negocio nova sem superficie no Mission Control nao e' progresso:")
        print("e' a mesma promessa do backend que os dois defeitos anteriores.")
        print("Suba o teto no commit COM a razao escrita, ou ligue a rota ao UI.")
        return 1
    if lacunas < TETO_ORFAS:
        print("VERDE, e o numero DESCEU (%d < %d). Baixe o TETO_ORFAS no mesmo commit"
              % (lacunas, TETO_ORFAS))
        print("para que o progresso nao volte a ser desperdicio na proxima ronda.")
        return 0
    print("VERDE: %d, exactamente no teto declarado." % lacunas)
    return 0


if __name__ == "__main__":
    sys.exit(main())
