"""Testes do mapa estado × prompt master (E-187).

O mapa existe porque "estágio 24" tinha DOIS significados. Estes testes garantem que a tradução é mecanicamente
completa e que a dívida de governança é REPORTADA sem fingir que não existe.
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
import hades_stage_map as mapa  # noqa: E402

RAIZ = Path(__file__).resolve().parent.parent


class MapaDeEstagiosTest(unittest.TestCase):
    def test_o_mapa_cobre_exatamente_um_a_trinta_e_oito(self) -> None:
        ids = sorted(linha[0] for linha in mapa.MAPA)
        self.assertEqual(ids, list(range(1, 39)), "o mapa precisa ter exatamente uma linha por estágio do estado")
        self.assertEqual(sorted(mapa.TITULOS_MESTRES), list(range(1, 39)), "os títulos do master cobrem 1..38")

    def test_todo_estagio_do_estado_real_tem_linha_no_mapa(self) -> None:
        estado = json.loads((RAIZ / "docs/mission/HADES_MASTER_STATE.json").read_text(encoding="utf-8"))
        ids_estado = sorted(e["id"] for e in estado["stages"])
        ids_mapa = sorted(linha[0] for linha in mapa.MAPA)
        self.assertEqual(ids_estado, ids_mapa, "todo estágio do estado precisa de linha no mapa, e vice-versa")

    def test_estagio_sem_linha_no_mapa_e_problema(self) -> None:
        estado = {"stages": [{"id": 1}, {"id": 99}], "blockers": []}
        problemas, _, _ = verificador.verificar_mapa_de_estagios(estado)
        self.assertTrue(
            any("estágio 99 do estado NÃO tem linha no mapa" in p for p in problemas),
            f"estágio sem linha precisa ser problema: {problemas}",
        )

    def test_estado_completo_nao_gera_problema_de_mapa(self) -> None:
        estado = json.loads((RAIZ / "docs/mission/HADES_MASTER_STATE.json").read_text(encoding="utf-8"))
        problemas, avisos, resumo = verificador.verificar_mapa_de_estagios(estado)
        self.assertEqual(problemas, [], f"o mapa do repositório real precisa ser mecanicamente completo: {problemas}")
        # A dívida de governança é REPORTADA, não escondida nem fatal.
        self.assertIn("estagios_sem_cobertura_no_master", resumo)
        self.assertIn("estagios_com_escopo_divergente", resumo)
        self.assertTrue(avisos, "com escopo divergente declarado precisa haver aviso")
        # Depois da reconciliação por evidência (E-188) NÃO existe mais estágio do master sem cobertura; o aviso que
        # precisa aparecer é o da cobertura dupla (voz e telefonia divididas em dois estágios do estado). A asserção
        # antiga exigia o aviso de "sem cobertura" - ela ficou obsoleta justamente porque a dívida caiu a zero.
        self.assertTrue(
            any("MAIS DE UM estágio do estado" in a for a in avisos),
            f"a cobertura dupla declarada precisa aparecer: {avisos}",
        )
        self.assertFalse(
            any("sem nenhum estágio do estado apontando" in a for a in avisos),
            f"não há mais estágio do master sem cobertura: {avisos}",
        )

    def test_repositorio_real_acusa_a_divida_medida(self) -> None:
        # Números MEDIDOS em E-188, DEPOIS de corrigir o pareamento com EVIDÊNCIA: ZERO estágios do master sem cobertura
        # e UMA cobertura dupla (37, voz e telefonia divididas em dois estágios do estado). Em E-187 eram 4 e 4; a
        # correção veio das evidências: E-105/E-141/E-170..E-177 (runtime omnichannel = estado 34), E-106/E-144..E-156
        # (agentes de atendimento = estado 35), E-109/E-112 (TestBusinessE2E, 21 etapas = estado 38).
        sem_cobertura = mapa.mestres_nao_cobertos()
        self.assertEqual(sem_cobertura, [], "todo estágio do master precisa ter cobertura declarada")
        duplicados = mapa.mestres_cobertos_mais_de_uma_vez()
        self.assertEqual(sorted(duplicados), [37], "a lista de cobertura dupla mudou")
        self.assertGreaterEqual(mapa.situacoes().get(mapa.ESCOPO_DIVERGENTE, 0), 10, "escopo divergente precisa estar declarado")

    def test_documento_gerado_bate_com_a_tabela(self) -> None:
        destino = RAIZ / "docs/mission/HADES_STAGE_MAP.md"
        self.assertTrue(destino.is_file(), "o mapa precisa estar gerado no repositório")
        self.assertEqual(destino.read_text(encoding="utf-8"), mapa.gerar(), "o documento precisa bater com a tabela")

    def test_mapa_importado_como_modulo_nao_quebra_o_verificador(self) -> None:
        # A checagem importa o mapa no mesmo diretório; sem esse caminho ela devolveria AVISO em vez de conferir.
        tmp = Path(tempfile.mkdtemp())
        try:
            estado = tmp / "estado.json"
            indice = tmp / "indice.md"
            shutil.copyfile(RAIZ / "docs/mission/HADES_MASTER_STATE.json", estado)
            indice.write_text("# Índice\n\n| evidence_id | tipo |\n| --- | --- |\n| E-001 | TEST_RESULT |\n", encoding="utf-8")
            problemas, avisos, resumo = verificador.verificar(estado, indice, conferir_gate=False)
            self.assertFalse(
                any("mapa de estágios indisponível" in a for a in avisos),
                "o mapa precisa ser importável pela checagem",
            )
            self.assertIn("estagios_com_escopo_divergente", resumo)
        finally:
            shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    unittest.main()
