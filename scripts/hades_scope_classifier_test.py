#!/usr/bin/env python3
"""Testes do classificador de escopo (E-352).

O classificador decide se um uso de `sanitizeIDComponent` é CAMINHO (precisa de guarda de tenant) ou IDENTIFICADOR
(outro risco). Um classificador que erra a categoria faz a catraca proteger a coisa errada — então ele é testado nas
duas direções, com arquivos de fixture.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_scope_classifier as classificador  # noqa: E402


class ClassificadorDeEscopoTest(unittest.TestCase):
    def _dir(self, **arquivos: str) -> Path:
        temp = Path(tempfile.mkdtemp())
        for nome, conteudo in arquivos.items():
            (temp / nome).write_text(conteudo, encoding="utf-8")
        return temp

    def test_linha_de_comentario_e_ignorada(self) -> None:
        self.assertEqual("comentario", classificador.classificar_uso('// sanitizeIDComponent("") devolve fallback'))
        self.assertEqual("comentario", classificador.classificar_uso('\tvalor := "x" // usa sanitizeIDComponent(v)'))

    def test_definicao_da_funcao_nao_e_uso(self) -> None:
        # MEDIDO: a definição aparecia como "indefinido" antes desta regra.
        self.assertEqual("definicao", classificador.classificar_uso("func sanitizeIDComponent(value string) string {"))

    def test_caminho_e_reconhecido(self) -> None:
        self.assertEqual("caminho",
                         classificador.classificar_uso('return filepath.Join(s.root, sanitizeIDComponent(org)+".json")'))
        self.assertEqual("caminho",
                         classificador.classificar_uso('withFileLock(filepath.Join(s.root, sanitizeIDComponent(org)+".lock"), f)'))

    def test_identificador_e_reconhecido(self) -> None:
        self.assertEqual("identificador",
                         classificador.classificar_uso('ID: "prod_" + sanitizeIDComponent(plan.OrganizationID) + "-x",'))
        self.assertEqual("identificador",
                         classificador.classificar_uso('br.ID = "brk_" + sanitizeIDComponent(fmt.Sprintf("%d", br.Weekday))'))

    def test_indefinido_nao_passa_em_silencio(self) -> None:
        # A direção que importa: o que não se sabe classificar TEM de ser marcado como indefinido.
        self.assertEqual("indefinido", classificador.classificar_uso('x := sanitizeIDComponent(v)'))

    def test_contagem_por_categoria_em_fixture(self) -> None:
        raiz = self._dir(**{
            "caminho.go": 'package p\n\nfunc f(org string) string {\n\treturn filepath.Join("/r", sanitizeIDComponent(org)+".json")\n}\n',
            "identificador.go": 'package p\n\nfunc g(org string) string {\n\treturn "prod_" + sanitizeIDComponent(org)\n}\n',
            "comentario.go": 'package p\n\n// sanitizeIDComponent(org) aqui é só comentário\nvar x = 1\n',
            "ignorado_test.go": 'package p\n\nfunc TestX(t *testing.T) { _ = sanitizeIDComponent("a") }\n',
        })
        usos = classificador.classificar_usos(raiz)
        self.assertIn("caminho.go", usos)
        self.assertEqual(1, len(usos["caminho.go"]["caminho"]))
        self.assertIn("identificador.go", usos)
        self.assertEqual(1, len(usos["identificador.go"]["identificador"]))
        self.assertEqual(1, len(usos["comentario.go"]["comentario"]))
        # Arquivo de TESTE é ignorado: a varredura é sobre código de produção.
        self.assertNotIn("ignorado_test.go", usos)

    def test_comentario_acima_nao_muda_a_classe_do_uso(self) -> None:
        """ADV-56/1 (3ª auditoria independente, medido): o `join` para trás entrava na linha de COMENTÁRIO acima e o
        uso real virava `comentario` ✗. Caso real do repositório: `scheduling_crossprocess.go:46`.

        ***Instrumento cujo veredito depende do comentário vizinho não mede a construção: mede a vizinhança.***
        """
        # O comentário tem de estar IMEDIATAMENTE acima do uso (é assim o caso real do repositório).
        with_comentario = (
            "package p\n\n"
            "func f(org string) string {\n"
            "\t// withFileLock tem a mesma assinatura nos dois sistemas\n"
            '\treturn filepath.Join("/r", sanitizeIDComponent(org)+".lock")\n'
            "}\n"
        )
        sem_comentario = (
            "package p\n\n"
            "func f(org string) string {\n"
            '\treturn filepath.Join("/r", sanitizeIDComponent(org)+".lock")\n'
            "}\n"
        )
        com = self._dir(**{"com_comentario.go": with_comentario})
        sem = self._dir(**{"sem_comentario.go": sem_comentario})
        # A CLASSE PRECISA SER A MESMA: o comentário acima não é parte do statement.
        self.assertEqual(
            1, len(classificador.classificar_usos(com)["com_comentario.go"]["caminho"]),
            "um comentário IMEDIATAMENTE acima do uso não pode virar 'comentario' (era o defeito do ADV-56/1): %s"
            % (classificador.classificar_usos(com)["com_comentario.go"],),
        )
        self.assertEqual(
            1, len(classificador.classificar_usos(sem)["sem_comentario.go"]["caminho"]),
            classificador.classificar_usos(sem)["sem_comentario.go"],
        )
        # E o mesmo uso, visto POR LINHA, também é caminho: a correção não podia depender do modo de varredura.
        self.assertEqual("caminho", classificador.classificar_uso('\treturn filepath.Join("/r", sanitizeIDComponent(org)+".lock")'))

    def test_contagens_do_repositorio_estao_pinadas(self) -> None:
        """ADV-56/5: nenhum teste pinava as contagens, então o recurso do E-355 podia ser revertido com a suíte VERDE.

        MEDIDO: reverter `classificar_statement` para classificação por linha deixava **8 testes OK** ✗. Uma métrica que
        nenhum teste pina deriva em silêncio — e foi o que aconteceu: a documentação dizia 22 e 23, o instrumento dizia
        23, e a verdade (depois de corrigir o join) era **24**. Subiu para **25** no E-1366, quando
# `agent_persona_ledger.go` passou a montar o caminho por escopo -- com `requireOrganization` na mesma
# funcao, que e' o que o `hades_tenant_scope_check.py` exige. As duas catracas concordam: um uso NOVO em
# caminho, e a guarda de tenant presente. *Um numero que sobe sem motivo e' narrativa; um que sobe com a
# guarda ao lado e' medicao.*
        """
        resumo = classificador.resumo_da_classificacao()
        esperado = {"caminho": 25, "identificador": 5, "comentario": 2, "indefinido": 1}
        obtido = {chave: resumo[chave] for chave in esperado}
        self.assertEqual(
            esperado, obtido,
            "as contagens de escopo mudaram. Se a mudança é legítima, ATUALIZE este teste e a documentação com a "
            "medição nova — não deixe a narrativa divergir do instrumento: %s" % (obtido,),
        )

    def test_indefinido_sem_declaracao_reprova(self) -> None:
        raiz = self._dir(**{"solta.go": 'package p\n\nfunc h(v string) string {\n\treturn sanitizeIDComponent(v)\n}\n'})
        pendentes = classificador.indefinidos_nao_declarados(raiz)
        self.assertTrue(pendentes, "uso indefinido sem declaração precisa aparecer como pendência")
        self.assertTrue(pendentes[0].startswith("solta.go:"), pendentes)

    def test_repositorio_real_nao_tem_indefinido_nao_declarado(self) -> None:
        pendentes = classificador.indefinidos_nao_declarados()
        self.assertEqual([], pendentes, "todo indefinido precisa de declaração com motivo:\n" + "\n".join(pendentes))
        resumo = classificador.resumo_da_classificacao()
        self.assertGreater(resumo["caminho"], 0, "a varredura precisa encontrar os usos de caminho")
        self.assertGreater(resumo["identificador"], 0, "a varredura precisa encontrar os usos de identificador")


if __name__ == "__main__":
    unittest.main(verbosity=2)
