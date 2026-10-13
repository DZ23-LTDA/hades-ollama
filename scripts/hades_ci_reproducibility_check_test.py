#!/usr/bin/env python3
"""Testes do guarda de CI reprodutível (E-309).

Um guarda sem teste NEGATIVO é uma promessa: se alguém inverter a condição, a suíte continua verde (lição de E-287).
Aqui o guarda é exercitado nas DUAS direções, com workflows de mentira em diretório temporário — e também contra os
workflows REAIS do repositório, que é o que o CI executa.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from hades_ci_reproducibility_check import analisar_workflow, auditar  # noqa: E402

WORKFLOW_FLUTUANTE = """\
name: teste
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
      - uses: golangci/golangci-lint-action@db9de0fc1a667e1a49d2291a1a042dff081d78f6 # v9
      - uses: actions/setup-node
        with:
          version: latest
      - uses: actions/setup-go@v7
"""

WORKFLOW_FIXO = """\
name: teste
jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4
      - uses: golangci/golangci-lint-action@db9de0fc1a667e1a49d2291a1a042dff081d78f6 # v9
        with:
          version: v2.13.0
"""


class GuardaDeCIReprodutivelTest(unittest.TestCase):
    def test_workflow_flutuante_e_acusado_nas_tres_regras(self) -> None:
        violacoes = analisar_workflow(Path("workflow.yaml"), WORKFLOW_FLUTUANTE)
        texto = "\n".join(violacoes)
        self.assertIn("versão flutuante 'latest'", texto, "a versão flutuante precisa ser acusada")
        self.assertIn("sem `with: version:`", texto, "a action de lint sem versão precisa ser acusada")
        self.assertIn("action externa sem referência", texto, "action sem @ precisa ser acusada")
        self.assertIn("TAG MÓVEL", texto, "action presa a tag móvel precisa ser acusada (a tag pode se mover)")

    def test_workflow_fixo_passa(self) -> None:
        violacoes = analisar_workflow(Path("workflow.yaml"), WORKFLOW_FIXO)
        self.assertEqual([], violacoes, f"workflow fixo não pode ser acusado: {violacoes}")

    def test_version_em_passo_SEGUINTE_nao_vale_para_o_anterior(self) -> None:
        # Trava o caso que mais importa: `with: version:` de OUTRO passo não pode absolver a action de lint.
        cruzado = """\
steps:
  - uses: golangci/golangci-lint-action@db9de0fc1a667e1a49d2291a1a042dff081d78f6 # v9
  - uses: actions/setup-node@11d5960a326750d5838078e36cf38b85af677262 # v4
    with:
      version: 22
"""
        violacoes = analisar_workflow(Path("workflow.yaml"), cruzado)
        self.assertTrue(
            any("sem `with: version:`" in v for v in violacoes),
            f"o version: do passo seguinte não pode absolver o anterior: {violacoes}",
        )

    def test_excecao_declarada_e_respeitada_e_esta_visivel(self) -> None:
        # A exceção existe para a ÚNICA ação fora de SHA; ela precisa estar declarada com motivo, não ignorada em
        # silêncio. Se a lista ficar vazia no futuro, este teste continua válido (nada a respeitar).
        from hades_ci_reproducibility_check import EXCECOES_DECLARADAS

        for acao, motivo in EXCECOES_DECLARADAS.items():
            self.assertTrue(motivo.strip(), f"toda exceção precisa de motivo escrito: {acao}")
        excecao = "steps:\n  - uses: Jimver/cuda-toolkit@v0.2.21\n"
        self.assertEqual([], analisar_workflow(Path("workflow.yaml"), excecao))

    def test_action_local_sem_arroba_nao_e_acusada(self) -> None:
        local = "steps:\n  - uses: ./.github/actions/meu-passo\n"
        self.assertEqual([], analisar_workflow(Path("workflow.yaml"), local))

    def test_workflows_REAIS_do_repositorio_passam(self) -> None:
        # Este é o teste que o CI executa: o repositório de verdade tem de estar reprodutível.
        violacoes = auditar()
        self.assertEqual([], violacoes, "o próprio repositório não pode ter gate flutuante:\n" + "\n".join(violacoes))

    def test_diretorio_inexistente_nao_quebra(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual([], auditar(Path(tmp) / "nao-existe"))


if __name__ == "__main__":
    unittest.main(verbosity=2)
