#!/usr/bin/env python3
"""Testes do guarda de descoberta honesta (E-322).

O guarda é exercitado nas DUAS direções com fixtures em diretório temporário — e também contra o repositório REAL, que
é o que o CI executa. ***Um guarda só vale se souber reprovar; e uma proibição só vale se doer no CI.***
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from hades_discoverability_check import (  # noqa: E402
    auditar,
    auditar_idiomas,
    auditar_promessas,
    validar_jsonld,
    validar_sitemap,
)


class DescobertaHonestaTest(unittest.TestCase):
    def _raiz(self, readme: str = "Guia em português (pt-BR). English guide below.\n", **arquivos: str) -> Path:
        # Chaves sao CAMINHOS relativos; "__" vira "/" para expressar subdiretorio. MEDIDO: usar o sufixo do
        # arquivo como chave ("sitemap__xml") criava sitemap/xml e o teste acusava o PROPRIO fixture.
        temp = Path(tempfile.mkdtemp())
        (temp / "README.md").write_text(readme, encoding="utf-8")
        for nome, conteudo in arquivos.items():
            alvo = temp / nome.replace("__", "/")
            alvo.parent.mkdir(parents=True, exist_ok=True)
            alvo.write_text(conteudo, encoding="utf-8")
        return temp

    # ---- direções NEGATIVAS: cada promessa proibida precisa doer ----

    def test_promessa_de_ranking_e_acusada(self) -> None:
        for frase in [
            "Este projeto garante o primeiro lugar nas buscas.",
            "Seremos recomendados por IAs em qualquer consulta.",
            "A plataforma vai aparecer em primeiro no Google.",
            "O Hades é melhor que todos os concorrentes.",
            "Produto 100% pronto para uso.",
        ]:
            raiz = self._raiz(readme="Guia em português (pt-BR). English guide.\n" + frase + "\n")
            violacoes = auditar_promessas(raiz)
            self.assertTrue(violacoes, f"promessa não acusada: {frase!r}")

    def test_readme_honesto_passa(self) -> None:
        raiz = self._raiz(readme=(
            "Guia em português (pt-BR).\nEnglish guide: see README.en.md.\n"
            "Estado: NOT_RELEASE_READY. Este documento não promete posição em buscadores nem recomendação por IAs.\n"))
        self.assertEqual([], auditar(raiz))

    def test_ausencia_de_entrada_em_ingles_e_acusada(self) -> None:
        raiz = self._raiz(readme="Guia em português (pt-BR) apenas.\n")
        violacoes = auditar_idiomas(raiz)
        self.assertTrue(any("inglês" in v for v in violacoes), violacoes)

    def test_sitemap_invalido_e_acusado(self) -> None:
        raiz = self._raiz(**{"sitemap.xml": "<urlset><url><loc>nao-absoluta</loc></url></urlset>"})
        violacoes = validar_sitemap(raiz / "sitemap.xml")
        self.assertTrue(any("não absoluta" in v for v in violacoes), violacoes)
        raiz2 = self._raiz(**{"sitemap.xml": "<urlset><url><loc>"})
        self.assertTrue(any("inválido" in v for v in validar_sitemap(raiz2 / "sitemap.xml")))

    def test_sitemap_ausente_nao_e_violacao(self) -> None:
        # Sem site hospedado, a ausência é honesta: não se inventa sitemap de um domínio que não existe.
        self.assertEqual([], validar_sitemap(Path(tempfile.mkdtemp()) / "sitemap.xml"))

    def test_jsonld_invalido_ou_sem_contexto_e_acusado(self) -> None:
        raiz = self._raiz(readme='<script type="application/ld+json">{"nome": "sem contexto"}</script>\n')
        violacoes = validar_jsonld(raiz / "README.md")
        self.assertTrue(any("sem @context" in v for v in violacoes), violacoes)
        raiz2 = self._raiz(readme='<script type="application/ld+json">{quebrado}</script>\n')
        self.assertTrue(any("inválido" in v for v in validar_jsonld(raiz2 / "README.md")))

    def test_jsonld_valido_passa(self) -> None:
        raiz = self._raiz(readme=(
            "Guia em português (pt-BR). English.\n"
            '<script type="application/ld+json">{"@context":"https://schema.org","@type":"SoftwareApplication","name":"Hades"}</script>\n'))
        self.assertEqual([], auditar(raiz))

    # ---- o repositório REAL: é este o teste que o CI executa ----

    def test_repositorio_real_passa(self) -> None:
        violacoes = auditar()
        self.assertEqual([], violacoes, "descoberta inconsistente no repositório:\n" + "\n".join(violacoes))


if __name__ == "__main__":
    unittest.main(verbosity=2)
