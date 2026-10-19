#!/usr/bin/env python3
"""HADES -- prova de que a catraca de sintaxe PowerShell e' DISCRIMINANTE (E-1292).

O caso central e' o defeito REAL que esta catraca existe para impedir: uma string delimitada por `'`
com um apostrofo interno nao escapado, exactamente como estava na linha 787 do
`scripts/hades_local_gates.ps1`.

    *Um teste que nao falharia com o defeito nao prova nada sobre o defeito.*
"""

from __future__ import annotations

import importlib.util
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"

# E-1300: `powershell.exe` NAO EXISTE EM LINUX/macOS.
#
# Este instrumento mede sintaxe de `.ps1` com o `PSParser::Tokenize` do PROPRIO PowerShell. Sem o
# interpretador nao ha' medida. No CI Ubuntu o teste falhava com
# `FileNotFoundError: [Errno 2] No such file or directory: 'powershell.exe'` -- **8 erros** -- e foi
# isso que deixou o job `Preserve Class A+ surfaces` vermelho em todos os commits.
#
# O `skipUnless` abaixo NAO e' um skip oportunista: e' a declaracao de que o objecto medido
# (sintaxe de PowerShell) exige o interpretador de PowerShell. O que impede o skip de esconder um
# defeito e' a outra metade da correccao: o `hades_ps1_syntax_check.py` devolve **exit 3**
# (`BLOCKED_BY_EXTERNAL_DEPENDENCY`) quando nao pode medir, e o workflow trata esse codigo como
# bloqueio DECLARADO -- nunca como aprovacao.
#
#     *Uma ausencia de medida nao e' uma medida: declara-se, nao se converte em aprovacao nem em
#     defeito.*
HA_POWERSHELL = shutil.which("powershell.exe") is not None
MOTIVO_BLOQUEIO = (
    "powershell.exe ausente neste SO: o instrumento mede sintaxe de .ps1 com o "
    "PSParser::Tokenize do proprio PowerShell, e sem o interpretador nao ha' medida. "
    "O checker devolve BLOCKED (exit 3), nao aprovacao. No Windows os 11 testes correm."
)


def carregar_checker():
    caminho = SCRIPTS / "hades_ps1_syntax_check.py"
    spec = importlib.util.spec_from_file_location("hades_ps1_syntax_check", caminho)
    modulo = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(modulo)
    return modulo


checker = carregar_checker()


def avaliar_texto(texto: str) -> list[str]:
    """Escreve o texto num `.ps1` temporario, mede com o PowerShell real, avalia."""
    with tempfile.TemporaryDirectory() as pasta:
        ficheiro = Path(pasta) / "teste.ps1"
        ficheiro.write_text(texto, encoding="utf-8")
        medicoes = checker.avaliar_com_powershell([ficheiro])
        return checker.avaliar(medicoes)


@unittest.skipUnless(HA_POWERSHELL, MOTIVO_BLOQUEIO)
class DiscriminacaoTest(unittest.TestCase):
    def test_script_valido_passa(self) -> None:
        self.assertEqual([], avaliar_texto("Write-Host 'ok'\n$x = 1\n"))

    def test_o_defeito_real_falha(self) -> None:
        """O caso exacto da linha 787: `e'` dentro de string com `'`."""
        texto = "Write-Host '--- 44. SNAPSHOT: o estado e' o HEAD antes de gerar'\nWrite-Host 'ok'\n"
        problemas = avaliar_texto(texto)
        self.assertTrue(problemas, "o checker PASSOU o defeito real -- nao e' discriminante")
        self.assertTrue(
            any("SINTAXE" in p for p in problemas),
            "esperado um problema de SINTAXE, obtido: %r" % (problemas,),
        )

    def test_o_idioma_do_powershell_passa(self) -> None:
        """A correccao aplicada: `e''` (apostrofo duplicado) e' valido e NAO pode falhar."""
        texto = "Write-Host '--- 44. SNAPSHOT: o estado e'' o HEAD antes de gerar'\n"
        self.assertEqual([], avaliar_texto(texto))

    def test_chaveta_desbalanceada_falha(self) -> None:
        self.assertTrue(avaliar_texto("if ($true) {\n  Write-Host 'x'\n"))

    def test_parenteses_desbalanceados_falham(self) -> None:
        self.assertTrue(avaliar_texto("Write-Host ('a'\n"))

    def test_here_string_valida_passa(self) -> None:
        """Here-strings tem regras proprias; o parser real sabe-as e o checker delega nele."""
        texto = "$t = @'\nlinha com e' apostrofo\n'@\nWrite-Host $t\n"
        self.assertEqual([], avaliar_texto(texto))

    def test_comentario_de_bloco_valido_passa(self) -> None:
        texto = "<#\ncomentario com e' apostrofo\n#>\nWrite-Host 'ok'\n"
        self.assertEqual([], avaliar_texto(texto))


@unittest.skipUnless(HA_POWERSHELL, MOTIVO_BLOQUEIO)
class MedicaoTest(unittest.TestCase):
    def test_sem_ficheiros_nao_aprova(self) -> None:
        """Ausencia de medida nao e' aprovacao."""
        self.assertTrue(checker.avaliar({}))

    def test_excecao_do_avaliador_falha(self) -> None:
        """Se o avaliador nao consegue medir, o resultado e' problema -- nao aprovacao tacita."""
        problemas = checker.avaliar({"/x/a.ps1": ("EXCECAO", "sem saida")})
        self.assertTrue(problemas)
        self.assertTrue(any("MEDICAO" in p for p in problemas))


@unittest.skipUnless(HA_POWERSHELL, MOTIVO_BLOQUEIO)
class RepositorioRealTest(unittest.TestCase):
    def test_encontra_os_ps1_versionados(self) -> None:
        ficheiros = checker.ficheiros_ps1()
        self.assertGreater(len(ficheiros), 0)
        nomes = {f.name for f in ficheiros}
        self.assertIn("hades_local_gates.ps1", nomes)

    def test_o_gate_local_real_parseia(self) -> None:
        """Regressao directa do E-1291: o ficheiro que estava partido tem de continuar valido."""
        alvo = SCRIPTS / "hades_local_gates.ps1"
        self.assertTrue(alvo.is_file())
        medicoes = checker.avaliar_com_powershell([alvo])
        self.assertEqual([], checker.avaliar(medicoes), medicoes)


if __name__ == "__main__":
    unittest.main(verbosity=2)
