#!/usr/bin/env python3
"""Teste-do-teste do hades_encoding_check.py (E-410; BOM coberto no E-658).

Um checker que nunca dispara não é checker. Este teste monta uma ÁRVORE FALSA em diretório temporário e
exige que o checker **acuse** o arquivo corrompido injetado — e que **não** acuse o arquivo limpo.

E prova também a metade que costuma ficar de fora: a **dispensa obsoleta**. *Dispensa que não casa com nada
é dívida disfarçada de exceção* — foi o defeito que corrigi no classificador de escopo (E-383), e aqui ele
tem de ser acusado pelo mesmo motivo.

E-658 — A SEGUNDA FORMA: o BOM. O checker cobria só dupla codificação, e um BOM UTF-8 passava em silêncio
(`read_text(encoding="utf-8")` não lança sobre BOM: o `\ufeff` vira o primeiro caractere). Foi assim que o
`HADES_MASTER_STATE.json` com BOM chegou ao CI e derrubou o `hades_release_gate.py`. As três provas abaixo
fecham a classe: **acusa o BOM injetado**, **não acusa árvore sem BOM** e **a árvore real não tem BOM**.
"""

import importlib.util
import shutil
import tempfile
import unittest
from pathlib import Path

# O checker precisa conter os padrões para procurá-los; por isso ele é uma dispensa legítima. A fixture abaixo
# também precisa conter os padrões LITERAIS, como BYTES no arquivo — *construí-la em tempo de execução
# (`.encode().decode()`) deixava a dispensa obsoleta, e foi o próprio checker que avisou.*
CORROMPIDO = "objetivo: integrar com seguranÃ§a e sem Ã¢nimo\n"
LIMPO = "objetivo: integrar com segurança e sem ânimo\n"


def carregar():
    spec = importlib.util.spec_from_file_location("enc", "scripts/hades_encoding_check.py")
    modulo = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(modulo)
    return modulo


class CheckerDeCodificacaoTest(unittest.TestCase):
    def setUp(self):
        self.raiz = Path(tempfile.mkdtemp())
        self.addCleanup(shutil.rmtree, self.raiz, ignore_errors=True)
        self.checker = carregar()

    def _escrever(self, nome, conteudo):
        alvo = self.raiz / nome
        alvo.parent.mkdir(parents=True, exist_ok=True)
        alvo.write_text(conteudo, encoding="utf-8")
        return alvo

    def _escrever_bytes(self, nome, dados):
        alvo = self.raiz / nome
        alvo.parent.mkdir(parents=True, exist_ok=True)
        alvo.write_bytes(dados)
        return alvo

    def test_arvore_limpa_passa(self):
        self._escrever("limpo.md", LIMPO)
        sujos, ilegiveis, obsoletas, com_bom = self.checker.medir(self.raiz)
        self.assertEqual(sujos, [])
        self.assertEqual(ilegiveis, [])
        self.assertEqual(com_bom, [])
        # As dispensas apontam para arquivos do REPOSITÓRIO, não da árvore temporária: portanto ficam obsoletas
        # AQUI de propósito. O teste verifica o comportamento na árvore real abaixo.
        self.assertIn("scripts/hades_mission_check.py", obsoletas)

    def test_arquivo_corrompido_e_acusado(self):
        self._escrever("ruim.md", CORROMPIDO)
        sujos, _, _, _ = self.checker.medir(self.raiz)
        self.assertEqual(sujos, ["ruim.md"], "o checker PRECISA acusar o arquivo corrompido injetado")

    def test_subpasta_tambem_e_varrida(self):
        self._escrever("a/b/ruim.md", CORROMPIDO)
        sujos, _, _, _ = self.checker.medir(self.raiz)
        self.assertEqual(sujos, ["a/b/ruim.md"], "a varredura precisa ser recursiva")

    def test_nao_decodifica_como_utf8_e_acusado(self):
        alvo = self.raiz / "binario.md"
        alvo.write_bytes(b"\xff\xfe\x00 ruim")
        _, ilegiveis, _, _ = self.checker.medir(self.raiz)
        self.assertEqual(len(ilegiveis), 1)

    def test_dispensa_obsoleta_e_acusada(self):
        # 'limpo.md' não contém o padrão, então a dispensa declarada para ele fica obsoleta.
        self._escrever("limpo.md", LIMPO)
        guardado = dict(self.checker.DISPENSAS)
        self.checker.DISPENSAS["limpo.md"] = "dispensa para prova"
        try:
            _, _, obsoletas, _ = self.checker.medir(self.raiz)
        finally:
            self.checker.DISPENSAS.clear()
            self.checker.DISPENSAS.update(guardado)
        self.assertIn("limpo.md", obsoletas, "dispensa que nao casa com nada precisa ser acusada")

    def test_dispensa_casada_nao_e_acusada(self):
        self._escrever("ruim.md", CORROMPIDO)
        guardado = dict(self.checker.DISPENSAS)
        self.checker.DISPENSAS["ruim.md"] = "dispensa declarada para a prova"
        try:
            sujos, _, obsoletas, _ = self.checker.medir(self.raiz)
        finally:
            self.checker.DISPENSAS.clear()
            self.checker.DISPENSAS.update(guardado)
        self.assertEqual(sujos, [])
        # MEDIDO: numa árvore TEMPORÁRIA as dispensas do repositório ficam obsoletas de propósito — por isso a
        # asserção é sobre a dispensa DECLARADA aqui, não sobre a lista inteira. *A primeira versão deste teste
        # exigia lista vazia e falhava por um motivo que não era o que ele queria provar.*
        self.assertNotIn("ruim.md", obsoletas)

    # --- E-658: a forma que o checker NÃO via ---------------------------------

    def test_bom_utf8_e_acusado_mesmo_com_conteudo_limpo(self):
        """A CONTRA-PROVA do ponto cego: texto perfeito + BOM tinha de ser acusado, e antes passava."""
        dados = b"\xef\xbb\xbf" + LIMPO.encode("utf-8")
        self._escrever_bytes("estado.json", dados)
        sujos, ilegiveis, _, com_bom = self.checker.medir(self.raiz)
        self.assertEqual(sujos, [], "o conteudo e' limpo: nao ha dupla codificacao a acusar")
        self.assertEqual(ilegiveis, [], "um BOM decodifica como UTF-8 sem erro — era exatamente o ponto cego")
        self.assertEqual(com_bom, ["estado.json"], "o checker PRECISA acusar o BOM injetado")

    def test_json_com_bom_quebraria_o_leitor_estrito(self):
        """Prova o DANO, não só a deteção: `json.loads` sobre UTF-8 estrito estoura com BOM."""
        import json
        dados = b"\xef\xbb\xbf" + b'{"a": 1}'
        with self.assertRaises(json.JSONDecodeError):
            json.loads(dados.decode("utf-8"))

    def test_repositorio_real_passa_e_todas_as_dispensas_sao_usadas(self):
        """A árvore REAL: sem violações, sem BOM e nenhuma dispensa obsoleta."""
        sujos, ilegiveis, obsoletas, com_bom = self.checker.medir(Path("."))
        self.assertEqual(sujos, [], "ha dupla codificacao REAL no repositorio")
        self.assertEqual(ilegiveis, [])
        self.assertEqual(com_bom, [], "ha BOM UTF-8 REAL no repositorio")
        self.assertEqual(obsoletas, [], "ha dispensa que nao casa com arquivo nenhum")


if __name__ == "__main__":
    unittest.main(verbosity=2)
