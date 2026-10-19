#!/usr/bin/env python3
"""HADES -- prova de que a catraca de package Android e' DISCRIMINANTE (E-1293).

O caso central e' o defeito REAL: manifest com `.MainActivity` e `namespace com.dz23ltd.hades`, mas
os ficheiros Kotlin em `com.lmprado.dz23agentic`. O Android procura uma classe que nao existe e o app
nao abre -- e o build passa.

    *Um erro que so' o dispositivo encontra fica a espera do utilizador.*
"""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"


def carregar_checker():
    caminho = SCRIPTS / "hades_android_package_check.py"
    spec = importlib.util.spec_from_file_location("hades_android_package_check", caminho)
    modulo = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(modulo)
    return modulo


checker = carregar_checker()


def kotlin(relativo: str, package: str) -> tuple[str, str]:
    return (relativo, package)


class DiscriminacaoTest(unittest.TestCase):
    def test_estado_corrigido_passa(self) -> None:
        problemas = checker.avaliar_ficheiros(
            [kotlin("com/dz23ltd/hades/MainActivity.kt", "com.dz23ltd.hades")],
            namespace="com.dz23ltd.hades",
            relativos=["MainActivity"],
            resolvidos={"com.dz23ltd.hades.MainActivity"},
        )
        self.assertEqual([], problemas, problemas)

    def test_o_defeito_real_falha(self) -> None:
        """O caso exacto do E-1293: namespace novo, package antigo."""
        problemas = checker.avaliar_ficheiros(
            [kotlin("com/lmprado/dz23agentic/MainActivity.kt", "com.lmprado.dz23agentic")],
            namespace="com.dz23ltd.hades",
            relativos=["MainActivity"],
            resolvidos={"com.lmprado.dz23agentic.MainActivity"},
        )
        self.assertTrue(problemas, "o checker PASSOU o defeito real -- nao e' discriminante")
        # Tem de apanhar as DUAS coisas: package != caminho E classe inexistente.
        self.assertTrue(
            any("NAMESPACE SEM CLASSE" in p or "PACKAGE != CAMINHO" in p for p in problemas),
            problemas,
        )
        self.assertTrue(
            any("CLASSE INEXISTENTE" in p for p in problemas),
            "esperado 'CLASSE INEXISTENTE' (resolucao do manifest), obtido: %r" % (problemas,),
        )

    def test_package_diferente_do_caminho_falha(self) -> None:
        problemas = checker.avaliar_ficheiros(
            [kotlin("com/dz23ltd/hades/MainActivity.kt", "com.outra.coisa")],
            namespace="com.dz23ltd.hades",
            relativos=[],
            resolvidos={"com.outra.coisa.MainActivity"},
        )
        self.assertTrue(any("PACKAGE != CAMINHO" in p for p in problemas), problemas)

    def test_namespace_sem_classe_falha(self) -> None:
        problemas = checker.avaliar_ficheiros(
            [kotlin("com/outro/MainActivity.kt", "com.outro")],
            namespace="com.dz23ltd.hades",
            relativos=[],
            resolvidos={"com.outro.MainActivity"},
        )
        self.assertTrue(any("NAMESPACE SEM CLASSE" in p for p in problemas), problemas)

    def test_main_application_tambem_tem_de_resolver(self) -> None:
        """O manifest referencia `.MainApplication` alem de `.MainActivity`."""
        problemas = checker.avaliar_ficheiros(
            [kotlin("com/dz23ltd/hades/MainActivity.kt", "com.dz23ltd.hades")],
            namespace="com.dz23ltd.hades",
            relativos=["MainActivity", "MainApplication"],
            resolvidos={"com.dz23ltd.hades.MainActivity"},
        )
        self.assertTrue(
            any("MainApplication" in p and "CLASSE INEXISTENTE" in p for p in problemas), problemas
        )

    def test_sem_kotlin_nao_aprova(self) -> None:
        self.assertTrue(checker.avaliar_ficheiros([], "com.dz23ltd.hades", [], set()))


class RepositorioRealTest(unittest.TestCase):
    def test_o_repositorio_real_e_coerente(self) -> None:
        self.assertEqual([], checker.avaliar())

    def test_o_namespace_real_e_o_canonico(self) -> None:
        self.assertEqual("com.dz23ltd.hades", checker.ler_namespace(checker.BUILD_GRADLE))

    def test_os_kt_estao_no_caminho_canonico(self) -> None:
        for nome in ("MainActivity.kt", "MainApplication.kt"):
            caminho = checker.JAVA_RAIZ / "com" / "dz23ltd" / "hades" / nome
            self.assertTrue(caminho.is_file(), "falta %s" % caminho)
            self.assertEqual("com.dz23ltd.hades", checker.ler_package(caminho))

    def test_o_package_historico_nao_esta_em_codigo(self) -> None:
        self.assertEqual([], checker.verificar_package_historico())

    def test_o_instrumento_exclui_se_a_si_mesmo(self) -> None:
        """A constante `PACKAGE_HISTORICO` e as fixtures dos testes TEM de conter o texto.

        A primeira versao devolveu 10 problemas, todos no proprio ficheiro e no seu teste.

            *Um scanner que nao se exclui mede a si mesmo e chama-lhe defeito alheio.*
        """
        self.assertIn(
            "scripts/hades_android_package_check.py", checker.FICHEIROS_DO_INSTRUMENTO
        )
        self.assertIn(
            "scripts/hades_android_package_check_test.py", checker.FICHEIROS_DO_INSTRUMENTO
        )
        # E a exclusao e' por nome exacto, nao por prefixo.
        self.assertNotIn(
            "scripts/hades_android_package_check_algo.py", checker.FICHEIROS_DO_INSTRUMENTO
        )


class ComentarioTest(unittest.TestCase):
    """Um comentario que documenta o defeito NAO e' o defeito.

    A primeira versao do checker acusou os comentarios do workflow e do `.ps1` que explicam a
    correccao. Recusa-los obrigaria a apagar a explicacao para o gate ficar verde.

        *Um gate que exige apagar a explicacao do defeito para passar ensina a nao documentar.*
    """

    def test_comentario_hash_e_aceite(self) -> None:
        self.assertTrue(
            checker.linha_e_comentario(
                "# o namespace era com.lmprado.dz23agentic e passou a com.dz23ltd.hades", "x.py"
            )
        )

    def test_comentario_de_barra_e_aceite(self) -> None:
        self.assertTrue(
            checker.linha_e_comentario(
                "// antes: com.lmprado.dz23agentic", "x.kt"
            )
        )

    def test_codigo_real_nao_e_comentario(self) -> None:
        self.assertFalse(
            checker.linha_e_comentario("namespace 'com.lmprado.dz23agentic'", "build.gradle")
        )

    def test_codigo_kotlin_nao_e_comentario(self) -> None:
        self.assertFalse(checker.linha_e_comentario("package com.lmprado.dz23agentic", "MainActivity.kt"))

    def test_url_com_duas_barras_nao_e_comentario(self) -> None:
        """`https://` tem `//` mas nao e' comentario."""
        self.assertFalse(
            checker.linha_e_comentario(
                "ver https://exemplo/com.lmprado.dz23agentic agora", "x.kt"
            )
        )

    def test_comentario_de_fim_de_linha_e_aceite(self) -> None:
        self.assertTrue(
            checker.linha_e_comentario(
                "valor = 1  # era com.lmprado.dz23agentic", "x.py"
            )
        )

    def test_o_manifest_usa_nomes_relativos(self) -> None:
        """Se o manifest passar a nome absoluto, a resolucao muda e o instrumento tem de saber."""
        relativos = checker.nomes_relativos_do_manifest(checker.MANIFEST)
        self.assertIn("MainActivity", relativos)
        self.assertIn("MainApplication", relativos)


if __name__ == "__main__":
    unittest.main(verbosity=2)
