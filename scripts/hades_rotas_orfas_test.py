# -*- coding: utf-8 -*-
"""Teste do instrumento de rotas orfas.

O que este teste trava, e porque cada caso falharia com o defeito:

  - o medidor tem de encontrar rotas no backend real. Se a regex partir, tudo
    passa a zero e o gate aprova um repo vazio. Um medidor que devolve 0 e'
    o resultado mais perigoso, porque parece sucesso.
  - /metrics/prometheus e /mcp/rpc TEM de cair em consumidor externo. Se
    cairem em lacuna, o gate pede UI para o Prometheus.
  - /scheduling/book NAO pode cair em consumidor externo. Se cair, o gate
    deixa de ver uma lacuna real.
  - o teto tem de estar no ficheiro. Um gate sem numero e' um comentario.
"""
import io
import os
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(RAIZ, "scripts"))

import hades_rotas_orfas as g  # noqa: E402


def test_le_as_rotas_do_backend():
    rotas = g.rotas_do_backend()
    assert len(rotas) > 300, "o medidor leu %d rotas -- a regex partiu?" % len(rotas)
    caminhos = {c for _, c in rotas}
    # Tres rotas que TEM de existir; se alguma sumir, o ficheiro mudou de forma
    # e o medidor precisa de ser revisto antes de voltar a aprovar.
    for esperada in ("/build/status/:id", "/capabilities", "/solutions/compare"):
        assert esperada in caminhos, "rota %s nao encontrada -- medidor desalinhado" % esperada


def test_metricas_e_mcp_sao_consumidor_externo():
    total, orfas, externas, lacunas, dominios = g.medir()
    assert externas > 0, "nenhuma rota caiu em consumidor externo -- a lista nao esta' a ser aplicada"
    assert "metrics" not in dominios, "/metrics caiu em lacuna de produto, devia ser consumidor externo"
    assert "mcp" not in dominios, "/mcp caiu em lacuna de produto, devia ser consumidor externo"


def test_dominio_de_negocio_e_lacuna():
    total, orfas, externas, lacunas, dominios = g.medir()
    assert lacunas > 0, "zero lacunas -- o medidor parou de ver o que devia ver"
    # scheduling tem o maior bloco; se desaparecer da lista, algo mudou.
    assert "scheduling" in dominios, "o dominio scheduling saiu da lista de lacunas"
    assert len(dominios["scheduling"]) >= 10


def test_o_teto_esta_declarado_e_e_inteiro():
    assert isinstance(g.TETO_ORFAS, int)
    assert g.TETO_ORFAS > 0, "um teto zero faria o gate passar em qualquer repo"
    # Uma linha de comentario no mesmo ficheiro tem de dizer de onde vem o numero.
    fonte = io.open(os.path.join(RAIZ, "scripts", "hades_rotas_orfas.py"), encoding="utf-8").read()
    assert "E-1319" in fonte, "o teto tem de apontar para a evidencia que o mediu"


def test_toda_a_entrada_de_consumidor_externo_tem_razao():
    for prefixo, razao in g.CONSUMIDOR_EXTERNO:
        assert prefixo.startswith("/"), "prefixo invalido: %r" % prefixo
        assert len(razao.strip()) > 5, "prefixo %r sem razao escrita" % prefixo


def test_o_gate_falha_acima_do_teto():
    # Baixa o teto temporariamente e confirma que o gate recusa.
    original = g.TETO_ORFAS
    try:
        g.TETO_ORFAS = 0
        assert g.main() == 1, "com teto 0 e lacunas > 0 o gate tinha de falhar"
    finally:
        g.TETO_ORFAS = original
