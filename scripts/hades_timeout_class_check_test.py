#!/usr/bin/env python3
"""Testes da catraca da classe de orçamento (E-362).

A catraca só vale se ela reprova a classe certa **e** deixa passar o que não é da classe. Os testes cobrem as duas
direções — inclusive o caso que quase me fez estragar 30 testes: **valores de domínio e esperas não são orçamento**.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_timeout_class_check as catraca  # noqa: E402


class CatracaDeOrcamentoTest(unittest.TestCase):
    def _dir(self, **arquivos: str) -> Path:
        temp = Path(tempfile.mkdtemp())
        for nome, conteudo in arquivos.items():
            (temp / nome).write_text(conteudo, encoding="utf-8")
        return temp

    def test_orcamento_curto_esperando_recusa_e_violacao(self) -> None:
        raiz = self._dir(**{"caso_test.go": (
            "package p\n\nfunc TestX(t *testing.T) {\n"
            "\tshortCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)\n"
            "\tdefer cancel()\n"
            "\terr := Rotacionar(shortCtx)\n"
            "\tif err == nil || !strings.Contains(err.Error(), \"runtime is not quiescent\") {\n"
            "\t\tt.Fatalf(\"want quiescence refusal: %v\", err)\n\t}\n}\n")})
        violacoes = catraca.auditar(raiz)
        self.assertEqual(1, len(violacoes), violacoes)
        self.assertIn("ADV-38", violacoes[0])

    def test_orcamento_curto_testando_o_proprio_deadline_e_permitido(self) -> None:
        # A classe é PROIBIDA, não o número: se o teste afirma o deadline, o orçamento curto é o objeto do teste.
        raiz = self._dir(**{"deadline_test.go": (
            "package p\n\nfunc TestY(t *testing.T) {\n"
            "\tctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)\n"
            "\tdefer cancel()\n"
            "\tif err := Espera(ctx); !errors.Is(err, context.DeadlineExceeded) {\n"
            "\t\tt.Fatalf(\"want DeadlineExceeded: %v\", err)\n\t}\n}\n")})
        self.assertEqual([], catraca.auditar(raiz))

    def test_valores_de_dominio_e_esperas_nao_sao_orcamento(self) -> None:
        # O caso que quase me fez alargar 30 testes por engano: nada disto é contexto.
        raiz = self._dir(**{"dominio_test.go": (
            "package p\n\nfunc TestZ(t *testing.T) {\n"
            "\tqueue.leaseDuration = 120 * time.Millisecond\n"
            "\t_ = queue.reclaimExpired(ctx, time.Now().UTC().Add(40*time.Millisecond))\n"
            "\ttime.Sleep(80 * time.Millisecond)\n"
            "\t_ = time.Now().UTC().Add(300 * time.Millisecond)\n}\n")})
        self.assertEqual([], catraca.auditar(raiz))

    def test_orcamento_de_um_segundo_ou_mais_nao_e_da_classe(self) -> None:
        raiz = self._dir(**{"longo_test.go": (
            "package p\n\nfunc TestW(t *testing.T) {\n"
            "\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n"
            "\tdefer cancel()\n"
            "\t_, _ = ctx, cancel\n}\n")})
        self.assertEqual([], catraca.auditar(raiz))

    def test_repositorio_real_esta_limpo(self) -> None:
        raiz = Path(__file__).resolve().parent.parent / "internal"
        self.assertEqual([], catraca.auditar(raiz), "a classe do ADV-38 voltou; corrija a causa em vez de dispensar")


if __name__ == "__main__":
    unittest.main(verbosity=2)
