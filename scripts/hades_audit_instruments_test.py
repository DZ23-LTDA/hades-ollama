"""Testes do runner dos instrumentos de auditoria (E-256).

Provam os dois sentidos: a linha de base declarada bate com o REPOSITORIO real, e uma regressao (candidato novo no
oraculo ou rota acima do teto) FALHA. Sem o segundo sentido, o runner seria decorativo.
"""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import hades_audit_instruments as runner  # noqa: E402


class ParserDoInstrumentoTest(unittest.TestCase):
    def test_le_identidades_da_saida_real_do_scanner(self) -> None:
        saida = (
            "candidatos (erro de RECURSO devolvido ANTES da comparacao de tenant): 1\n\n"
            "  internal\\agent\\mcp.go :: register (comparacao de tenant na linha 62)\n"
            "      L25  return errors.New(\"MCP argument is invalid or too long\")\n"
        )
        self.assertEqual(["internal/agent/mcp.go::register"], runner.candidatos_oraculo(saida))

    def test_le_a_contagem_de_residuais(self) -> None:
        saida = "rotas com parametro de caminho: 181\nhandlers SEM escopo apos resolver o padrao *ForRequest: 35\n"
        self.assertEqual(35, runner.residuais_rota(saida))

    def test_saida_sem_contagem_e_erro(self) -> None:
        with self.assertRaises(ValueError):
            runner.residuais_rota("saida inesperada")


class LinhaDeBaseTest(unittest.TestCase):
    def test_baseline_declarada_bate_com_o_repositorio(self) -> None:
        # O teste mais importante: a linha de base nao pode ser uma promessa; ela e' conferida contra o codigo.
        candidatos, residuais = runner.coletar()
        self.assertEqual(
            [],
            runner.avaliar(candidatos, residuais),
            "a linha de base declarada precisa refletir o repositorio: %s / residuais=%d" % (candidatos, residuais),
        )

    def test_candidato_novo_no_oraculo_falha(self) -> None:
        problemas = runner.avaliar(list(runner.BASELINE_ORACULO) + ["internal/agent/novo.go::qualquer"], runner.TETO_ROTAS)
        self.assertTrue(any("ORACULO NOVO" in p for p in problemas), problemas)

    def test_oraculo_que_desaparece_nao_falha(self) -> None:
        # Melhoria (defeito corrigido) NAO pode virar falha de CI.
        self.assertEqual([], runner.avaliar(list(runner.BASELINE_ORACULO)[:-1], runner.TETO_ROTAS))

    def test_rota_acima_do_teto_falha(self) -> None:
        problemas = runner.avaliar(list(runner.BASELINE_ORACULO), runner.TETO_ROTAS + 1)
        self.assertTrue(any("acima do teto" in p for p in problemas), problemas)

    def test_rota_igual_ao_teto_nao_falha(self) -> None:
        self.assertEqual([], runner.avaliar(list(runner.BASELINE_ORACULO), runner.TETO_ROTAS))


if __name__ == "__main__":
    unittest.main(verbosity=2)
