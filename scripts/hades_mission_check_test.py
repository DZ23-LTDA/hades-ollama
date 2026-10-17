#!/usr/bin/env python3
"""Testa o verificador de escrituração da missão (E-180).

Um verificador que **sempre passa** não vale nada: ele dá confiança sem lastro. Este teste injeta defeitos em
fixtures e exige que o verificador os **encontre** — evidência fantasma, ID duplicado, buraco na numeração,
bloqueio órfão, `execution` ausente e veredito otimista com pendência. Segue a convenção do repositório
(`classify_ci_surfaces_test.py` roda ao lado do script que testa).
"""

from __future__ import annotations

import json
import re
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_mission_check as verificador  # noqa: E402

RAIZ = Path(__file__).resolve().parent.parent
ESTADO_REAL = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
INDICE_REAL = RAIZ / "docs/mission/HADES_EVIDENCE_INDEX.md"


class EscrituracaoTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp())
        self.estado = self.tmp / "HADES_MASTER_STATE.json"
        self.indice = self.tmp / "HADES_EVIDENCE_INDEX.md"
        shutil.copyfile(ESTADO_REAL, self.estado)
        # Índice pequeno e controlado: 3 evidências coerentes com os estágios do fixture.
        self.indice.write_text(
            "# Índice de teste\n\n"
            "| evidence_id | tipo |\n| --- | --- |\n"
            "| E-001 | TEST_RESULT |\n| E-002 | TEST_RESULT |\n| E-003 | TEST_RESULT |\n",
            encoding="utf-8",
        )
        estado = json.loads(self.estado.read_text(encoding="utf-8"))
        estado["stages"] = [
            {"id": 1, "title": "um", "execution": "IN_PROGRESS", "evidence": ["E-001"], "blocker": "B-01", "state": "IN_PROGRESS"},
            {"id": 2, "title": "dois", "execution": "NOT_STARTED", "evidence": ["E-002", "E-003"], "state": "IN_PROGRESS"},
        ]
        estado["blockers"] = [{"id": "B-01", "tipo": "EXTERNO", "descricao": "externo"}]
        estado["release_state"] = "NOT_RELEASE_READY"
        self.escrever_estado(estado)

    def tearDown(self) -> None:
        shutil.rmtree(self.tmp, ignore_errors=True)

    def escrever_estado(self, estado: dict) -> None:
        self.estado.write_text(json.dumps(estado, ensure_ascii=False, indent=2), encoding="utf-8")

    def ler_estado(self) -> dict:
        return json.loads(self.estado.read_text(encoding="utf-8"))

    def problemas(self) -> list[str]:
        problemas, _, _ = verificador.verificar(self.estado, self.indice, conferir_gate=False)
        return problemas

    def test_fixture_saudavel_passa(self) -> None:
        self.assertEqual(self.problemas(), [], "o fixture sem defeito precisa passar")

    def test_evidencia_sem_vinculo_e_encontrada(self) -> None:
        """Linha que nenhum estágio cita e que não declara a ausência vira problema: órfã silenciosa não passa."""
        estado = self.ler_estado()
        estado["stages"][1]["evidence"] = ["E-002"]
        self.escrever_estado(estado)
        problemas = self.problemas()
        self.assertTrue(
            any("SEM vínculo de estágio" in p and "E-003" in p for p in problemas),
            f"esperava problema de vínculo para E-003, veio: {problemas}",
        )

    def test_evidencia_sem_vinculo_declarado_passa(self) -> None:
        """Declarar a ausência do vínculo (linha legada) é registro honesto e NÃO pode ser acusado."""
        estado = self.ler_estado()
        estado["stages"][1]["evidence"] = ["E-002"]
        self.escrever_estado(estado)
        self.indice.write_text(
            self.indice.read_text(encoding="utf-8").replace(
                "| E-003 | TEST_RESULT |",
                "| E-003 | TEST_RESULT | VINCULO_NAO_DECLARADO |",
            ),
            encoding="utf-8",
        )
        problemas = self.problemas()
        self.assertFalse(
            any("SEM vínculo de estágio" in p for p in problemas),
            f"declarar a ausência do vínculo não pode ser acusado: {problemas}",
        )

    def test_documento_com_mojibake_e_encontrado(self) -> None:
        """Texto duplamente codificado torna o documento ilegivel: precisa ser acusado."""
        alvo = self.tmp / "docs" / "mission"
        alvo.mkdir(parents=True, exist_ok=True)
        (alvo / "COM_DEFEITO.md").write_text("objetivo: integrar com seguranÃ§a e sem [a-circ-duplo]nimo", encoding="utf-8")
        problemas, resumo = verificador.verificar_mojibake(self.tmp)
        self.assertTrue(
            any("TEXTO CORROMPIDO" in p and "COM_DEFEITO.md" in p for p in problemas),
            f"esperava acusar o mojibake, veio: {problemas}",
        )
        self.assertGreaterEqual(resumo["documentos_varridos"], 1)

    def test_documento_correto_nao_e_acusado(self) -> None:
        """Português acentuado LEGÍTIMO não pode ser confundido com corrupção."""
        alvo = self.tmp / "docs" / "mission"
        alvo.mkdir(parents=True, exist_ok=True)
        (alvo / "CORRETO.md").write_text("objetivo: integrar com segurança e sem ânimo — pronto", encoding="utf-8")
        problemas, _ = verificador.verificar_mojibake(self.tmp)
        self.assertEqual(problemas, [], f"texto correto não pode ser acusado: {problemas}")

    def test_evidencia_fantasma_e_encontrada(self) -> None:
        estado = self.ler_estado()
        estado["stages"][0]["evidence"] = ["E-999"]
        self.escrever_estado(estado)
        problemas = self.problemas()
        self.assertTrue(
            any("NÃO existe no índice" in p and "E-999" in p for p in problemas),
            f"evidência fantasma precisa ser acusada: {problemas}",
        )

    def test_id_duplicado_e_encontrado(self) -> None:
        self.indice.write_text(
            "| E-001 | a |\n| E-001 | b |\n",
            encoding="utf-8",
        )
        self.assertTrue(any("DUPLICADOS" in p for p in self.problemas()), self.problemas())

    def test_buraco_na_numeracao_e_encontrado(self) -> None:
        self.indice.write_text("| E-001 | a |\n| E-003 | c |\n", encoding="utf-8")
        self.assertTrue(any("buracos" in p for p in self.problemas()), self.problemas())

    def test_bloqueio_orfao_e_encontrado(self) -> None:
        estado = self.ler_estado()
        estado["stages"][0]["blocker"] = "B-77"
        self.escrever_estado(estado)
        self.assertTrue(any("B-77" in p for p in self.problemas()), self.problemas())

    def test_execution_ausente_e_encontrado(self) -> None:
        estado = self.ler_estado()
        del estado["stages"][1]["execution"]
        self.escrever_estado(estado)
        self.assertTrue(any("SEM o campo `execution`" in p for p in self.problemas()), self.problemas())

    def test_execution_invalido_e_encontrado(self) -> None:
        estado = self.ler_estado()
        estado["stages"][1]["execution"] = "QUASE_PRONTO"
        self.escrever_estado(estado)
        self.assertTrue(any("fora do permitido" in p for p in self.problemas()), self.problemas())

    def test_veredito_otimista_com_pendencia_e_encontrado(self) -> None:
        estado = self.ler_estado()
        estado["release_state"] = "COMPLETED_VERIFIED"
        self.escrever_estado(estado)
        problemas = self.problemas()
        self.assertTrue(any("COMPLETED_VERIFIED com estágios" in p for p in problemas), problemas)
        self.assertTrue(any("bloqueio externo em aberto" in p for p in problemas), problemas)

    def test_indice_real_do_repositorio_e_consistente(self) -> None:
        # O fixture é pequeno, mas o índice REAL precisa estar consistente: é o que o CI vai conferir.
        problemas, _, resumo = verificador.verificar(ESTADO_REAL, INDICE_REAL, conferir_gate=False)
        self.assertEqual(problemas, [], f"escrituração real inconsistente: {problemas}")
        self.assertGreaterEqual(resumo["evidencias"], 100)


class RelatorioTest(unittest.TestCase):
    """Numeração do relatório e evidências que ele cita (E-181)."""

    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp())
        self.relatorio = self.tmp / "HADES_STAGE_REPORT.md"
        self.ids = {"E-001", "E-002", "E-003"}

    def tearDown(self) -> None:
        shutil.rmtree(self.tmp, ignore_errors=True)

    def escrever(self, texto: str) -> None:
        self.relatorio.write_text(texto, encoding="utf-8")

    def problemas(self, ids: set[str] | None = None) -> list[str]:
        problemas, _ = verificador.verificar_relatorio(self.relatorio, ids if ids is not None else self.ids)
        return problemas

    def test_numeracao_contigua_passa(self) -> None:
        self.escrever("1. **primeiro (E-001).** texto\n2. **segundo (E-002).** texto\n")
        self.assertEqual(self.problemas(), [])

    def test_item_duplicado_e_encontrado(self) -> None:
        self.escrever("1. **um (E-001).**\n2. **dois (E-002).**\n2. **outro dois (E-003).**\n")
        problemas = self.problemas()
        self.assertTrue(any("DUPLICADOS" in p for p in problemas), problemas)

    def test_buraco_no_relatorio_e_encontrado(self) -> None:
        self.escrever("1. **um (E-001).**\n3. **tres (E-002).**\n")
        self.assertTrue(any("buracos" in p for p in self.problemas()), self.problemas())

    def test_evidencia_fantasma_em_texto_normal_e_encontrada(self) -> None:
        self.escrever("1. **um (E-777).**\n")
        self.assertTrue(any("E-777" in p for p in self.problemas()), self.problemas())

    def test_identificador_entre_crases_e_mencao_e_nao_falso_positivo(self) -> None:
        # A regra declarada: ID entre crases é MENÇÃO (exemplo/identificador), não citação.
        crase = chr(96)
        self.escrever(f"1. **um (E-001) e o verificador pega {crase}E-999{crase} como exemplo.**\n")
        self.assertEqual(self.problemas(), [], "menção entre crases não pode virar problema")

    def test_relatorio_sem_itens_e_acusado(self) -> None:
        self.escrever("# relatório sem itens numerados\n")
        self.assertTrue(any("sem nenhum item numerado" in p for p in self.problemas()), self.problemas())


class BloqueiosTest(unittest.TestCase):
    """Paridade entre o documento de bloqueios e o estado, nos DOIS sentidos (E-181)."""

    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp())
        self.bloqueios = self.tmp / "HADES_BLOCKERS.md"

    def tearDown(self) -> None:
        shutil.rmtree(self.tmp, ignore_errors=True)

    def escrever(self, texto: str) -> None:
        self.bloqueios.write_text(texto, encoding="utf-8")

    def problemas(self, ids: set[str]) -> list[str]:
        problemas, _ = verificador.verificar_bloqueios(self.bloqueios, ids)
        return problemas

    def test_paridade_perfeita_passa(self) -> None:
        self.escrever("| B-01 | EXTERNO |\n| B-02 | RESOLVIDO |\n")
        self.assertEqual(self.problemas({"B-01", "B-02"}), [])

    def test_bloqueio_no_estado_ausente_do_documento_e_encontrado(self) -> None:
        self.escrever("| B-01 | EXTERNO |\n")
        self.assertTrue(any("B-02" in p and "AUSENTES" in p for p in self.problemas({"B-01", "B-02"})))

    def test_bloqueio_no_documento_ausente_do_estado_e_encontrado(self) -> None:
        self.escrever("| B-01 | EXTERNO |\n| B-77 | FANTASMA |\n")
        self.assertTrue(any("B-77" in p and "INEXISTENTES" in p for p in self.problemas({"B-01"})))

    def test_documentos_reais_do_repositorio_sao_consistentes(self) -> None:
        estado = json.loads(ESTADO_REAL.read_text(encoding="utf-8"))
        ids = {b.get("id") for b in estado.get("blockers", [])}
        relatorio, _ = verificador.verificar_relatorio(
            RAIZ / "docs/mission/HADES_STAGE_REPORT.md",
            set(re.findall(r"^\|\s*(E-\d+)\s*\|", INDICE_REAL.read_text(encoding="utf-8"), re.M)),
        )
        bloqueios, _ = verificador.verificar_bloqueios(RAIZ / "docs/mission/HADES_BLOCKERS.md", ids)
        self.assertEqual(relatorio, [], f"relatório real inconsistente: {relatorio}")
        self.assertEqual(bloqueios, [], f"bloqueios reais inconsistentes: {bloqueios}")




class CruzamentoDeAuditoriaTest(unittest.TestCase):
    """E-913 / finding 14 do bot: `PARCIALMENTE FECHADO` NAO e' estado terminal.

    O defeito que este teste discrimina: a lista `encerrado` incluia `PARCIALMENTE FECHADO`, e um achado
    cujo proprio texto diz que a integracao de producao continua aberta era tratado como **encerrado**.
    => *a lista de ataques aponta-lo e' **legitimo** (ha' trabalho por fazer), e o cruzamento acusava de
    «duas verdades sobre o mesmo id» um caso em que so' ha' **uma**.*

    **Com o defeito (a palavra na lista), o primeiro teste falha** -- e o segundo continua a passar, o que
    mostra que a cura nao desligou o cruzamento: ele continua a apanhar o que e' terminal.
    """

    def _estado(self, estado_do_achado: str) -> dict:
        return {
            "v22": {
                "auditoria_adversarial": {
                    "proximos_ataques": [{"id": "ADV-65", "descricao": "integracao de producao pendente"}],
                    "achados": [{"id": "ADV-65", "estado": estado_do_achado}],
                }
            }
        }

    def test_parcialmente_fechado_nao_e_encerrado(self) -> None:
        """`PARCIALMENTE FECHADO: falta a integracao` NAO pode ser acusado de duplicar o ataque."""
        problemas, resumo = verificador.verificar_cruzamento_de_auditoria(
            self._estado("PARCIALMENTE FECHADO: falta a integracao de producao")
        )
        self.assertEqual([], problemas, "um achado PARCIAL nao encerra: a lista de ataques e' legitima")
        self.assertEqual(0, resumo["ataques_duplicando_achado_encerrado"])

    def test_corrigido_continua_a_ser_encerrado(self) -> None:
        """A contraprova: um estado TERMINAL continua a acusar -- a cura nao desligou o cruzamento."""
        problemas, resumo = verificador.verificar_cruzamento_de_auditoria(
            self._estado("CORRIGIDO: TestX verde")
        )
        self.assertEqual(1, len(problemas), "um achado CORRIGIDO na lista de ataques e' duas verdades")
        self.assertEqual(1, resumo["ataques_duplicando_achado_encerrado"])

    def test_aberto_com_a_palavra_corrigido_nao_acusa(self) -> None:
        """O `E-447`: `ABERTO: ainda nao corrigido` contem `CORRIGIDO`, e o casamento e' por PREFIXO."""
        problemas, _ = verificador.verificar_cruzamento_de_auditoria(
            self._estado("ABERTO: ainda nao corrigido")
        )
        self.assertEqual([], problemas, "o prefixo e' ABERTO, e o resto e' explicacao")


if __name__ == "__main__":
    unittest.main(verbosity=2)
