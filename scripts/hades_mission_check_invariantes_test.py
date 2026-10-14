#!/usr/bin/env python3
"""Testa as invariantes de auditoria acrescentadas em E-444 e E-446 (cruzamento e prova podre).

***Um instrumento sem teste proprio e' a lacuna que ele mesmo denuncia.*** Os dois verificadores abaixo foram escritos
para pegar registro vencido - e ficaram **sem** teste ate' agora, exatamente como as `2 defesas sem prova` que eu vinha
criticando. Este arquivo fecha isso.

Cada invariante e' testada nas **DUAS** direcoes: *dispara quando deve* e **passa quando deve**. *Um verificador que so'
dispara e' tao ruim quanto um que nunca dispara - foi o que o controle de E-434 mostrou ("senao o teste passaria com a
validacao recusando tudo").*
"""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_mission_check as verificador  # noqa: E402

RAIZ = Path(__file__).resolve().parent.parent


def estado_com(proximos_ataques: list, achados: list, citados: str = "") -> dict:
    return {
        "v22": {
            "auditoria_adversarial": {
                "proximos_ataques": proximos_ataques,
                "achados": achados + ([{"id": "ADV-CITADO", "estado": citados}] if citados else []),
            }
        }
    }


class CruzamentoDeAuditoriaTest(unittest.TestCase):
    """E-444: um id NAO pode estar em `proximos_ataques` E em `achados` com estado encerrado."""

    def test_ataque_que_duplica_achado_encerrado_e_encontrado(self) -> None:
        # O caso REAL medido: ADV-60 estava nos ataques E em achados como "CORRIGIDO COM LIMITE DECLARADO".
        problemas, resumo = verificador.verificar_cruzamento_de_auditoria(
            estado_com(
                [{"id": "ADV-60 (P0)", "descricao": "…"}],
                [{"id": "ADV-60", "estado": "CORRIGIDO COM LIMITE DECLARADO"}],
            )
        )
        self.assertTrue(problemas, "ataque duplicando achado encerrado TEM de ser acusado")
        self.assertIn("AO MESMO TEMPO", problemas[0])
        self.assertEqual(1, resumo["ataques_duplicando_achado_encerrado"])

    def test_ataque_com_achado_aberto_passa(self) -> None:
        # CONTROLE: se o achado correspondente NAO esta' encerrado, o ataque e' legitimo e nao ha' o que acusar.
        problemas, resumo = verificador.verificar_cruzamento_de_auditoria(
            estado_com(
                [{"id": "ADV-99", "descricao": "…"}],
                [{"id": "ADV-99", "estado": "ABERTO: ainda nao corrigido"}],
            )
        )
        self.assertEqual([], problemas, "ataque com achado ABERTO nao pode ser acusado")
        self.assertEqual(0, resumo["ataques_duplicando_achado_encerrado"])

    def test_id_com_sufixo_e_normalizado(self) -> None:
        # *Foi a FALTA desta normalizacao que produziu o registro falso de E-443*: o id era "ADV-60 (P0)" e a comparacao
        # usava "ADV-60" exato, entao o item escapou da conferencia.
        problemas, _ = verificador.verificar_cruzamento_de_auditoria(
            estado_com([{"id": "ADV-60 (P0)"}], [{"id": "ADV-60", "estado": "FECHADO"}])
        )
        self.assertTrue(problemas, 'o id "ADV-60 (P0)" tem de casar com o achado "ADV-60"')


class ProvaPodreTest(unittest.TestCase):
    """E-446: todo teste CITADO no estado tem de existir como `func Test…`."""

    def test_teste_citado_inexistente_e_encontrado(self) -> None:
        problemas, resumo = verificador.verificar_testes_citados(
            RAIZ, estado_com([], [], citados="CORRIGIDO: TestQueNaoExisteEmLugarNenhum cobre isso")
        )
        self.assertTrue(problemas, "teste citado que nao existe TEM de ser acusado")
        self.assertIn("PROVA PODRE", problemas[0])
        self.assertEqual(1, resumo["testes_citados_inexistentes"])

    def test_teste_citado_que_existe_passa(self) -> None:
        # CONTROLE: um teste que EXISTE de verdade nao pode ser acusado. *Usei um nome real do repositorio - se ele for
        # renomeado, este teste falha, e isso e' desejado: ele mede a existencia, que e' o que a invariante promete.*
        problemas, resumo = verificador.verificar_testes_citados(
            RAIZ, estado_com([], [], citados="CORRIGIDO: TestValidatePlanGraphRecusaGrafoInvalido cobre isso")
        )
        self.assertEqual([], problemas, "teste que existe nao pode ser acusado")
        self.assertEqual(0, resumo["testes_citados_inexistentes"])

    def test_estado_sem_citacoes_passa(self) -> None:
        # E o caso vazio: sem citacao nao ha' o que conferir - e o resumo diz 0, nao "ok".
        problemas, resumo = verificador.verificar_testes_citados(RAIZ, estado_com([], []))
        self.assertEqual([], problemas)
        self.assertEqual(0, resumo["testes_citados"])


if __name__ == "__main__":
    unittest.main()
