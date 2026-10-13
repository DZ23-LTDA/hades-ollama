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
