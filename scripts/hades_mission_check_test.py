#!/usr/bin/env python3
"""Testa o verificador de escrituração da missão (E-180).

Um verificador que **sempre passa** não vale nada: ele dá confiança sem lastro. Este teste injeta defeitos em
fixtures e exige que o verificador os **encontre** — evidência fantasma, ID duplicado, buraco na numeração,
bloqueio órfão, `execution` ausente e veredito otimista com pendência. Segue a convenção do repositório
(`classify_ci_surfaces_test.py` roda ao lado do script que testa).
"""

from __future__ import annotations

import json
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


if __name__ == "__main__":
    unittest.main(verbosity=2)
