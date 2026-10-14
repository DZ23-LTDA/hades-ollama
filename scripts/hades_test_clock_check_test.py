#!/usr/bin/env python3
"""E-422 — Testes do checker do relógio de parede.

**Um verificador só vale se reprova a CLASSE CERTA e deixa passar o que NÃO é dela.** Os testes cobrem as duas
direções, e as duas foram escolhidas por medição, não por imaginação:

  * o **falso negativo** que deixou o CI vermelho (E-421): a forma em **duas linhas**
    (`agora := time.Now()` … `agora.Add(90min)`) que a varredura do E-418 **não via** ✗;
  * os **falsos positivos** que eu medi um a um ao rodar o checker pela primeira vez: os três sítios marcados ALTO
    eram *vencimento*, *org vazia recusada pelo portão de escopo* e *a linha do meu próprio comentário*.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_test_clock_check as checker  # noqa: E402


class CheckerDoRelogioTest(unittest.TestCase):
    def _varrer(self, conteudo: str) -> list[tuple[str, int, str, str]]:
        with tempfile.TemporaryDirectory() as tmp:
            raiz = Path(tmp) / "internal"
            raiz.mkdir(parents=True)
            (raiz / "exemplo_test.go").write_text(conteudo, encoding="utf-8")
            return checker.varrer(Path(tmp) / "internal")

    # --- DIREÇÃO 1: REPROVA A CLASSE CERTA ---------------------------------

    def test_pega_a_forma_de_uma_linha(self) -> None:
        achados = self._varrer(
            "package a\nfunc t() {\n"
            '\t_ = BookRequest{StartsAt: time.Now().UTC().Add(90 * time.Minute)}\n'
            "}\n"
        )
        self.assertEqual(1, len(achados), "a forma direta tem de ser vista")
        self.assertEqual("ALTO", achados[0][2])

    def test_pega_a_forma_de_duas_linhas_o_ponto_cego_do_E_418(self) -> None:
        """★ ESTE É O TESTE QUE IMPORTA: foi por aqui que o E-421 deixou o CI vermelho."""
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\tagora := time.Now().UTC()\n"
            "\tmarcado := agora.Add(90 * time.Minute)\n"
            '\t_ = BookRequest{StartsAt: marcado}\n'
            "}\n"
        )
        self.assertEqual(1, len(achados), "a forma em duas linhas NÃO pode escapar (era o ponto cego do E-418)")
        self.assertEqual("ALTO", achados[0][2])
        self.assertEqual(4, achados[0][1], "o achado aponta a linha do .Add, não a do Now()")

    def test_pega_variavel_com_nome_de_negocio_a_distancia(self) -> None:
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\talvo := time.Now()\n"
            "\t// comentário no meio\n"
            "\t// outro comentário\n"
            "\t_ = Schedule{NextRunAt: alvo.Add(2 * time.Hour)}\n"
            "}\n"
        )
        self.assertEqual(1, len(achados))

    # --- DIREÇÃO 2: DEIXA PASSAR O QUE NÃO É DA CLASSE ---------------------

    def test_nao_marca_vencimento(self) -> None:
        for trecho in (
            "\t_ = outbox.ClaimDue(time.Now().UTC().Add(time.Second))",
            "\tschedule.NextRunAt = time.Now().UTC().Add(-time.Second)",
        ):
            with self.subTest(trecho=trecho):
                achados = self._varrer("package a\nfunc t() {\n%s\n}\n" % trecho)
                classes = [a[2] for a in achados]
                self.assertNotIn("ALTO", classes, "vencimento não depende do calendário: %s" % trecho)

    def test_nao_marca_timestamp(self) -> None:
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\t_ = Contact{CreatedAt: time.Now().UTC().Add(-2 * time.Hour)}\n"
            "}\n"
        )
        self.assertEqual([], [a for a in achados if a[2] == "ALTO"])

    def test_nao_marca_instante_escolhido(self) -> None:
        """O helper `instanteDeTeste` — a correção — não pode ser acusado por este checker."""
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\tbase := time.Now().UTC().Add(24 * time.Hour)\n"
            "\talvo := time.Date(base.Year(), base.Month(), base.Day(), 10, 0, 0, 0, time.UTC)\n"
            "}\n"
        )
        self.assertEqual([], [a for a in achados if a[2] == "ALTO"])

    # --- DIREÇÃO 3: O REPOSITÓRIO REAL ------------------------------------

    def test_repo_real_nao_tem_alto_risco_sem_justificativa(self) -> None:
        """*A catraca do mundo real.* Se um sítio ALTO aparecer, ele tem de ser inspecionado — e se for instante de
        negócio, corrigido com `instanteDeTeste(t)`. Os três sítios ALTO conhecidos são **triagem já resolvida**:
        vencimento (`ClaimDue`), recusa por escopo (org vazia) e a linha de um **comentário**."""
        raiz = Path("internal")
        if not raiz.is_dir():
            self.skipTest("sem internal/ — rode da raiz do repositório")
        altos = [(a, l) for a, l, c, _ in checker.varrer(raiz) if c == "ALTO"]
        conhecidos = {
            ("internal/agent/push_outbox_test.go", 274),          # ClaimDue: vencimento
            # E-432: a linha era 28 e virou 36 quando o cabeçalho deste teste mudou (o `agora.Add(time.Hour)` da
            # ORGANIZAÇÃO VAZIA é triagem resolvida: o portão de ESCOPO dispara antes do expediente).
            # *Pinar NÚMERO DE LINHA é brittleness conhecida* - um cabeçalho novo move o sítio.
            ("internal/agent/declared_categories_measured_test.go", 36),  # org vazia: recusado por escopo
            ("internal/agent/scheduling_waitlist_test.go", 187),  # a linha do COMENTÁRIO do E-421
        }
        novos = [x for x in altos if x not in conhecidos]
        self.assertEqual(
            [], novos,
            "sítio(s) NOVO(s) de ALTO risco: inspecione cada um. Se for instante de NEGÓCIO, use "
            "instanteDeTeste(t) (E-421); se for vencimento/timestamp/escopo, acrescente a `conhecidos` com a razão "
            "MEDIDA — nunca só para o teste passar: %s" % novos,
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
