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
            {"id": 2, "title": "dois", "execution": "NOT_STARTED", "evidence": ["E-002"], "state": "IN_PROGRESS"},
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


if __name__ == "__main__":
    unittest.main(verbosity=2)
