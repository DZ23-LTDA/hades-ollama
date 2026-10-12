"""Testes da checagem de documentos obrigatórios da seção 5 (E-186).

Cada regra tem caso de FALHA e caso de SUCESSO — uma checagem que só sabe acusar não prova que sabe aprovar.
"""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

import hades_mission_check as verificador  # noqa: E402

RAIZ = Path(__file__).resolve().parent.parent


class DocumentosObrigatoriosTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp())

    def tearDown(self) -> None:
        import shutil

        shutil.rmtree(self.tmp, ignore_errors=True)

    def test_raiz_vazia_acusa_todos_os_documentos_obrigatorios(self) -> None:
        problemas, resumo = verificador.verificar_documentos_obrigatorios(self.tmp)
        self.assertTrue(problemas, "raiz sem os documentos precisa acusar")
        total = sum(len(c) for c in verificador.DOCUMENTOS_OBRIGATORIOS_POR_ESTAGIO.values())
        self.assertEqual(resumo["documentos_obrigatorios_conferidos"], total)
        # Cada ausência precisa dizer ESTÁGIO e CAMINHO: mensagem que não orienta não serve para corrigir.
        self.assertTrue(any("estagio 2" in p and "HADES_BASELINE_REPORT.md" in p for p in problemas))
        self.assertTrue(any("estagio 22" in p and "HADES_ENGINEERING_GATE.md" in p for p in problemas))

    def test_fixture_com_os_documentos_passa(self) -> None:
        for caminhos in verificador.DOCUMENTOS_OBRIGATORIOS_POR_ESTAGIO.values():
            for relativo in caminhos:
                destino = self.tmp / relativo
                destino.parent.mkdir(parents=True, exist_ok=True)
                destino.write_text("conteúdo de teste", encoding="utf-8")
        problemas, resumo = verificador.verificar_documentos_obrigatorios(self.tmp)
        self.assertEqual(problemas, [], "com todos os documentos presentes não pode haver problema")
        self.assertGreater(resumo["documentos_obrigatorios_conferidos"], 0)

    def test_documento_presente_mas_como_diretorio_e_recusado(self) -> None:
        # Um DIRETÓRIO com o nome do documento não é evidência: o artefato precisa ser arquivo.
        alvo = self.tmp / "docs/mission/HADES_BASELINE_REPORT.md"
        alvo.mkdir(parents=True)
        problemas, _ = verificador.verificar_documentos_obrigatorios(self.tmp)
        self.assertTrue(any("HADES_BASELINE_REPORT.md" in p for p in problemas))

    def test_repositorio_real_tem_os_documentos_obrigatorios(self) -> None:
        problemas, resumo = verificador.verificar_documentos_obrigatorios(RAIZ)
        self.assertEqual(problemas, [], f"o repositório real precisa cumprir a seção 5: {problemas}")
        self.assertEqual(
            resumo["documentos_obrigatorios_conferidos"],
            sum(len(c) for c in verificador.DOCUMENTOS_OBRIGATORIOS_POR_ESTAGIO.values()),
        )

    def test_estagio_sem_evidencia_e_problema(self) -> None:
        import json
        import shutil

        tmp = Path(tempfile.mkdtemp())
        try:
            estado = tmp / "estado.json"
            indice = tmp / "indice.md"
            shutil.copyfile(RAIZ / "docs/mission/HADES_MASTER_STATE.json", estado)
            indice.write_text("# Índice\n\n| evidence_id | tipo |\n| --- | --- |\n| E-001 | TEST_RESULT |\n", encoding="utf-8")
            dados = json.loads(estado.read_text(encoding="utf-8"))
            dados["stages"] = [
                {"id": 1, "execution": "IN_PROGRESS", "evidence": ["E-001"], "state": "IN_PROGRESS"},
                {"id": 2, "execution": "IN_PROGRESS", "evidence": [], "state": "IN_PROGRESS"},
            ]
            dados["blockers"] = []
            estado.write_text(json.dumps(dados, ensure_ascii=False), encoding="utf-8")
            problemas, _, _ = verificador.verificar(estado, indice, conferir_gate=False)
            self.assertTrue(
                any("nao declara NENHUMA evidencia" in p for p in problemas),
                f"estágio sem evidência precisa ser problema: {problemas}",
            )
        finally:
            shutil.rmtree(tmp, ignore_errors=True)

    def test_verificar_com_raiz_confere_os_documentos(self) -> None:
        import json
        import shutil

        tmp = Path(tempfile.mkdtemp())
        try:
            estado = tmp / "estado.json"
            indice = tmp / "indice.md"
            shutil.copyfile(RAIZ / "docs/mission/HADES_MASTER_STATE.json", estado)
            indice.write_text("# Índice\n\n| evidence_id | tipo |\n| --- | --- |\n| E-001 | TEST_RESULT |\n", encoding="utf-8")
            dados = json.loads(estado.read_text(encoding="utf-8"))
            dados["stages"] = [{"id": 1, "execution": "IN_PROGRESS", "evidence": ["E-001"], "state": "IN_PROGRESS"}]
            dados["blockers"] = []
            estado.write_text(json.dumps(dados, ensure_ascii=False), encoding="utf-8")
            problemas, _, _ = verificador.verificar(estado, indice, conferir_gate=False, raiz=tmp)
            self.assertTrue(
                any("OBRIGATORIA" in p for p in problemas),
                f"com `raiz` sem documentos, a checagem precisa acusar: {problemas}",
            )
            # E sem `raiz`, a checagem de documentos não roda: é o que mantém os fixtures antigos válidos.
            problemas_sem_raiz, _, _ = verificador.verificar(estado, indice, conferir_gate=False)
            self.assertFalse(any("OBRIGATORIA" in p for p in problemas_sem_raiz))
        finally:
            shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
