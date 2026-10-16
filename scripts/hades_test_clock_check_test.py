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
        negócio, corrigido com `instanteDeTeste(t)`; se for legítimo, entra em `checker.TRIAGADOS` com a razão
        MEDIDA.

        E-909(g) / finding 6: **esta lista era própria (`conhecidos`) e divergia de `checker.TRIAGADOS`** — o teste
        tinha **3** sítios e o checker **2**. *Duas allowlists com o mesmo propósito e conteúdos diferentes: um
        sítio no teste e não no checker faz o `--check` **reprovar** e o teste **aprovar** — discordam sobre o
        mesmo repositório.* Agora deriva da **única** fonte.
        """
        raiz = Path("internal")
        if not raiz.is_dir():
            self.skipTest("sem internal/ — rode da raiz do repositório")
        altos = [(a, checker.assinatura(t)) for a, _l, c, t in checker.varrer(raiz) if c == "ALTO"]
        novos = [x for x in altos if x not in checker.TRIAGADOS]
        self.assertEqual(
            [], novos,
            "sítio(s) NOVO(s) de ALTO risco: inspecione cada um. Se for instante de NEGÓCIO, use "
            "instanteDeTeste(t) (E-421); se for vencimento/timestamp/escopo, acrescente a `checker.TRIAGADOS` "
            "com a razão MEDIDA — nunca só para o teste passar: %s" % novos,
        )

    def test_triagem_nao_tem_dispensa_morta(self) -> None:
        """E-909(c) / finding 4: **toda dispensa tem de corresponder a um achado real.**

        *Uma dispensa que já não casa com nada é pior do que não existir*: ela fica na lista a **esconder** um
        ALTO futuro que caia naquela posição — e ninguém percebe, porque a lista "já previa" aquele sítio.

        O caso medido: `scheduling_waitlist_test.go:187` era *«a linha do COMENTARIO»*. Quando
        `sem_comentarios()` passou a remover comentários, o sítio **deixou de ser varrido** — e a dispensa ficou
        sem objeto. *Foi por isso que ela saiu, e este teste é o que garante que não volta.*
        """
        raiz = Path("internal")
        if not raiz.is_dir():
            self.skipTest("sem internal/ — rode da raiz do repositório")
        reais = {(a, checker.assinatura(t)) for a, _l, c, t in checker.varrer(raiz) if c == "ALTO"}
        mortas = [chave for chave in checker.TRIAGADOS if chave not in reais]
        self.assertEqual([], mortas, "dispensa(s) sem achado correspondente (escondem ALTO futuro): %s" % mortas)

    # --- DIREÇÃO 4: OS QUATRO DEFEITOS DO INSTRUMENTO (E-909) --------------

    def test_um_uso_seguro_nao_esconde_um_de_negocio_na_mesma_linha(self) -> None:
        """E-909(a) / finding 2: a classificação era **da linha**, não **do uso**.

        Medido antes da correção: `OBSERVACAO` continha `"At:"`, e **`NextRunAt:` contém `At:`** ⇒
        `NextRunAt: now.Add(2*time.Hour)` saía como *timestamp* apesar de `NextRunAt` estar em `NEGOCIO`.

        Aqui os dois usos estão na MESMA linha: um observação (seguro) e um de negócio (ALTO). *O sítio tem de
        ser ALTO — o rótulo seguro não pode cobrir o inseguro.*

        Nota de construção: o snippet **precisa** de `agora := time.Now()` — *o checker rastreia a origem da
        variável; sem isso nenhuma das duas formas casa, e o teste mediria outra coisa.*
        """
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\tagora := time.Now().UTC()\n"
            '\t_ = S{CreatedAt: agora.Add(time.Minute), StartsAt: agora.Add(2 * time.Hour)}\n'
            "}\n"
        )
        classes = [a[2] for a in achados]
        self.assertIn("ALTO", classes, "um uso de negocio na mesma linha de um seguro tem de aparecer")

    def test_nao_para_no_primeiro_add_da_variavel(self) -> None:
        """E-909(b) / finding 3: o `break` fazia ver **só o primeiro** `.Add(` da janela.

        *Uma variável com dois deslocamentos — um seguro e um de negócio — escondia o de negócio.* Aqui o
        primeiro `.Add(` é seguro (negativo) e o segundo é de negócio; os **dois** têm de ser vistos.
        """
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\tagora := time.Now().UTC()\n"
            "\tvencido := agora.Add(-time.Minute)\n"
            "\talvo := agora.Add(2 * time.Hour)\n"
            "\t_, _ = vencido, alvo\n"
            "}\n"
        )
        linhas = sorted(a[1] for a in achados)
        self.assertIn(4, linhas, "o uso seguro tambem e' triagem")
        self.assertIn(5, linhas, "o segundo uso NAO pode ser escondido pelo primeiro (era o `break`)")
        classes = {a[1]: a[2] for a in achados}
        self.assertEqual("ALTO", classes.get(5), "o segundo uso e' de negocio: `alvo` e' escolha")

    def test_comentario_nao_e_acusado(self) -> None:
        """E-909(d) / finding 5: a varredura acusava **texto**.

        Um dos dois sítios triados era *«a linha do COMENTARIO do E-421»* — o instrumento apontava uma **frase
        sobre o defeito**, não o defeito. *Aqui o comentário descreve um `StartsAt` deslocado e **não** pode
        produzir achado.*
        """
        achados = self._varrer(
            "package a\nfunc t() {\n"
            "\t// _ = S{StartsAt: time.Now().UTC().Add(2 * time.Hour)} — exemplo no comentario\n"
            "\t_ = 1\n"
            "}\n"
        )
        self.assertEqual([], achados, "comentario nao e' codigo: nao pode ser acusado")

    def test_assinatura_ignora_reindentacao_mas_nao_mudanca(self) -> None:
        """E-909(c) / finding 4: a chave da triagem era `arquivo:linha`.

        O próprio comentário da lista admitia *«Pinar NÚMERO DE LINHA é brittleness conhecida»* — e o E-432 já
        tinha medido o custo: um cabeçalho novo moveu o sítio de **28** para **36**, e a catraca acusou o
        **movimento**, não a **mudança**.

        A assinatura tem de ser **estável** à indentação e **sensível** à expressão.
        """
        a = "\t_ = S{StartsAt: agora.Add(time.Hour)}"
        b = "        _ = S{StartsAt: agora.Add(time.Hour)}"          # mesma expressao, outra indentacao
        c = "\t_ = S{StartsAt: agora.Add(2 * time.Hour)}"            # expressao DIFERENTE
        self.assertEqual(checker.assinatura(a), checker.assinatura(b), "reindentar nao pode reabrir o achado")
        self.assertNotEqual(checker.assinatura(a), checker.assinatura(c), "mudar a expressao TEM de reabrir")


if __name__ == "__main__":
    unittest.main(verbosity=2)
