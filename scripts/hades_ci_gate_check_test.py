#!/usr/bin/env python3
"""Teste-do-teste do hades_ci_gate_check.py (E-665).

Um checker que nunca dispara nao e' checker. Cada fixture aqui prova uma direcao — *e a primeira e' a que custou
um CI vermelho de verdade:* a forma `test -z "$(gofmt ...)"`, que **aprova arquivo ilegivel**.
"""

import importlib.util
import pathlib
import unittest

RAIZ = pathlib.Path(__file__).resolve().parent.parent


def carregar():
    spec = importlib.util.spec_from_file_location("ci_gate", RAIZ / "scripts" / "hades_ci_gate_check.py")
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


FORMA_CEGA = """\
name: demo
jobs:
  integrity:
    steps:
      - name: Verify Go formatting
        run: |
          test -z "$(gofmt -l internal/agent internal/multillm server/agent_routes.go)"
"""

FORMA_CORRETA = """\
name: demo
jobs:
  integrity:
    steps:
      - name: Verify Go formatting
        run: |
          if ! saida="$(gofmt -l internal/agent internal/multillm server/agent_routes.go 2>&1)"; then
            echo "gofmt FALHOU"; echo "$saida"; exit 1
          fi
          echo "gofmt OK"
"""

FORMA_DE_PIPELINE_CEGA = """\
name: demo
jobs:
  gates:
    steps:
      - name: gofmt
        run: gofmt -l internal/agent server/agent_routes.go | tee /tmp/gofmt.txt && test ! -s /tmp/gofmt.txt
"""

FORMA_DE_PIPELINE_COM_PIPEFAIL = """\
name: demo
jobs:
  gates:
    steps:
      - name: gofmt
        run: |
          set -o pipefail
          gofmt -l internal/agent server/agent_routes.go | tee /tmp/gofmt.txt
          test ! -s /tmp/gofmt.txt
"""

SEM_GOFMT = """\
name: demo
jobs:
  integrity:
    steps:
      - name: Nada
        run: echo ola
"""


class CatracaDoGateDeGofmtTest(unittest.TestCase):
    def setUp(self):
        self.checker = carregar()

    def test_forma_cega_e_acusada(self):
        """A CONTRA-PROVA do defeito real: `test -z "$(gofmt ...)"` TEM de ser acusada."""
        passos, violacoes = self.checker.medir(FORMA_CEGA)
        self.assertEqual(passos, 1)
        self.assertEqual(len(violacoes), 1, "o checker PRECISA acusar a forma que aprova arquivo ilegivel")
        self.assertIn("stdout vazio PASSA", violacoes[0])

    def test_forma_correta_passa(self):
        passos, violacoes = self.checker.medir(FORMA_CORRETA)
        self.assertEqual(passos, 1)
        self.assertEqual(violacoes, [])

    def test_pipeline_sem_pipefail_e_acusada(self):
        """MEDIDO: sem `set -o pipefail`, o status de uma pipeline e' o do ULTIMO comando (`tee`) — se o gofmt
        falhar, o `test ! -s` ve um ficheiro vazio e PASSA. *Foi esta a segunda instancia real encontrada no
        repositorio (`dz23-agentic-quality.yaml`).*"""
        passos, violacoes = self.checker.medir(FORMA_DE_PIPELINE_CEGA)
        self.assertEqual(passos, 1)
        self.assertEqual(len(violacoes), 1, "a pipeline sem `pipefail` tem de ser acusada")
        self.assertIn("codigo de saida", violacoes[0])

    def test_pipeline_com_pipefail_passa(self):
        """Com `set -o pipefail` a pipeline e' honesta: a falha do gofmt propaga."""
        passos, violacoes = self.checker.medir(FORMA_DE_PIPELINE_COM_PIPEFAIL)
        self.assertEqual(passos, 1)
        self.assertEqual(violacoes, [])

    def test_workflow_sem_gofmt_e_declarado_como_nao_medido(self):
        """Ausencia de medicao nao e' aprovacao: zero passos e' um estado DECLARADO, nao um `ok`."""
        passos, violacoes = self.checker.medir(SEM_GOFMT)
        self.assertEqual(passos, 0)
        self.assertEqual(violacoes, [])

    def test_workflows_reais_passam(self):
        """A catraca no repositorio REAL: nenhum passo com gofmt na forma cega."""
        alvos = sorted((RAIZ / ".github/workflows").glob("*.yaml")) + sorted((RAIZ / ".github/workflows").glob("*.yml"))
        self.assertTrue(alvos, "tem de haver workflows versionados para medir")
        total_passos = 0
        for caminho in alvos:
            passos, violacoes = self.checker.medir(caminho.read_text(encoding="utf-8"))
            total_passos += passos
            self.assertEqual(violacoes, [], "violacao de gate em %s: %s" % (caminho.name, violacoes))
        self.assertGreater(total_passos, 0, "nenhum passo com gofmt foi medido nos workflows reais")


if __name__ == "__main__":
    unittest.main(verbosity=2)
