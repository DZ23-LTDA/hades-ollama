#!/usr/bin/env python3
"""HADES -- prova de que a catraca da assinatura Android e' DISCRIMINANTE (E-1290).

O PRINCIPIO

Um instrumento que so' devolve "conforme" nao e' um instrumento: e' um adorno. Estes testes fornecem ao
checker um `build.gradle` COM o defeito e exigem que ele FALHE, e um SEM o defeito e exigem que ele
PASSE.

    *Um teste que nao falharia com o defeito nao prova nada sobre o defeito.*
"""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"


def carregar_checker():
    caminho = SCRIPTS / "hades_android_signing_check.py"
    spec = importlib.util.spec_from_file_location("hades_android_signing_check", caminho)
    modulo = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(modulo)
    return modulo


checker = carregar_checker()


def gradle_corrigido() -> str:
    """O build.gradle como esta' no repositorio real."""
    return (RAIZ / "apps" / "mobile-agentic" / "android" / "app" / "build.gradle").read_text(
        encoding="utf-8"
    )


def gradle_com_o_defeito() -> str:
    """O template original do React Native: release assina com a debug key."""
    return (
        "android {\n"
        "    signingConfigs {\n"
        "        debug {\n"
        "            storeFile file('debug.keystore')\n"
        "            storePassword 'android'\n"
        "            keyAlias 'androiddebugkey'\n"
        "            keyPassword 'android'\n"
        "        }\n"
        "    }\n"
        "    buildTypes {\n"
        "        debug {\n"
        "            signingConfig signingConfigs.debug\n"
        "        }\n"
        "        release {\n"
        "            // Caution! In production, you need to generate your own keystore file.\n"
        "            signingConfig signingConfigs.debug\n"
        "        }\n"
        "    }\n"
        "}\n"
    )


class DiscriminacaoTest(unittest.TestCase):
    def test_repositorio_real_passa(self) -> None:
        problemas = checker.avaliar(gradle_corrigido())
        self.assertEqual([], problemas, problemas)

    def test_gradle_com_o_defeito_falha_com_p0(self) -> None:
        """O caso central: o template original TEM de falhar, e falhar como P0."""
        problemas = checker.avaliar(gradle_com_o_defeito())
        self.assertTrue(problemas, "o checker PASSOU o gradle com o defeito -- nao e' discriminante")
        self.assertTrue(
            any("P0" in p and "signingConfigs.debug" in p for p in problemas),
            "esperado um P0 a nomear signingConfigs.debug, obtido: %r" % (problemas,),
        )

    def test_release_sem_a_credencial_condicional_falha(self) -> None:
        """Tirar `hadesCredencialDeReleaseCompleta` do bloco release tem de falhar."""
        texto = gradle_corrigido().replace(
            "signingConfig hadesCredencialDeReleaseCompleta ? signingConfigs.release : null",
            "signingConfig null",
        )
        problemas = checker.avaliar(texto)
        self.assertTrue(
            any("hadesCredencialDeReleaseCompleta" in p for p in problemas), problemas
        )

    def test_falta_de_uma_variavel_de_credencial_falha(self) -> None:
        texto = gradle_corrigido().replace("HADES_ANDROID_KEY_PASSWORD", "OUTRA_COISA")
        problemas = checker.avaliar(texto)
        self.assertTrue(
            any("HADES_ANDROID_KEY_PASSWORD" in p for p in problemas), problemas
        )

    def test_falta_do_guard_em_bundle_release_falha(self) -> None:
        texto = gradle_corrigido().replace("'bundleRelease'", "'outraTarefa'")
        problemas = checker.avaliar(texto)
        self.assertTrue(any("bundleRelease" in p for p in problemas), problemas)

    def test_falta_de_gradle_exception_falha(self) -> None:
        """Um aviso nao aborta o build; so' uma excepcao aborta."""
        texto = gradle_corrigido().replace("GradleException", "println")
        problemas = checker.avaliar(texto)
        self.assertTrue(any("GradleException" in p for p in problemas), problemas)

    def test_gradle_vazio_falha(self) -> None:
        self.assertTrue(checker.avaliar(""))

    def test_bloco_release_ausente_falha(self) -> None:
        problemas = checker.avaliar("android {\n    buildTypes {\n    }\n}\n")
        self.assertTrue(problemas)

    def test_debug_pode_usar_a_debug_key(self) -> None:
        """A correccao NAO era remover a configuracao de debug -- era separar release de debug.

        Se o bloco `debug` desaparecer, o checker avisa: remover a debug key do build de debug
        quebraria o desenvolvimento sem resolver a distribuicao.
        """
        texto = gradle_corrigido()
        # Confirmar que o bloco `buildTypes.debug` real usa a debug key e mesmo assim passa.
        bloco_debug = checker.extrair_build_type(texto, "debug")
        self.assertIn("signingConfigs.debug", bloco_debug)
        self.assertEqual([], checker.avaliar(texto))

    def test_o_dois_blocos_release_nao_se_confundem(self) -> None:
        """Regressao do bug que este instrumento teve na primeira versao.

        O `build.gradle` tem DOIS blocos `release`: um dentro de `signingConfigs` (a definicao da
        credencial) e outro dentro de `buildTypes` (o que assina). Ler o primeiro daria veredicto
        sobre o bloco errado.
        """
        texto = gradle_corrigido()
        # O bloco `release` de signingConfigs NAO contem `signingConfig` (e' uma definicao).
        # O de buildTypes contem. O extractor tem de ir ao segundo.
        bloco_do_build_type = checker.extrair_build_type(texto, "release")
        self.assertIn("hadesCredencialDeReleaseCompleta", bloco_do_build_type)
        self.assertIn("signingConfig", bloco_do_build_type)

    def test_extrator_de_bloco_nao_para_na_chaveta_errada(self) -> None:
        """O extractor conta chavetas. Um bloco aninhado nao pode termina-lo antes do fim."""
        texto = (
            "release {\n"
            "    algumaCoisa {\n"
            "        interna { }\n"
            "    }\n"
            "    marcaFinal = true\n"
            "}\n"
        )
        bloco = checker.extrair_bloco(texto, "release")
        self.assertIn("marcaFinal", bloco)
        self.assertIn("interna", bloco)


class RepositorioRealTest(unittest.TestCase):
    def test_o_ficheiro_real_existe(self) -> None:
        self.assertTrue(
            (RAIZ / "apps" / "mobile-agentic" / "android" / "app" / "build.gradle").is_file()
        )

    def test_as_quatro_variaveis_estao_no_checker_e_no_gradle(self) -> None:
        """Uma divergencia silenciosa entre checker e gradle deixaria uma variavel sem guarda."""
        texto = gradle_corrigido()
        self.assertEqual(4, len(checker.VARIAVEIS_DE_CREDENCIAL))
        for variavel in checker.VARIAVEIS_DE_CREDENCIAL:
            self.assertIn(variavel, texto, "variavel %s ausente do gradle" % variavel)


if __name__ == "__main__":
    unittest.main(verbosity=2)
