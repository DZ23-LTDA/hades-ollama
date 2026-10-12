"""Calibração do scanner de não inferência (E-191).

UM SCANNER SEM TESTE É UMA OPINIÃO. O scanner nasceu errado nos DOIS sentidos, medidos:
- com o padrão de "id" casando com `organizationID`, apontou **6 falsos positivos**;
- depois de corrigir, passou a **não encontrar nada** — inclusive a forma que eu tinha acabado de corrigir à mão.

Estes testes usam a forma REAL do defeito B-15 (pré e pós-correção) como positivo e negativo conhecidos. Sem eles, não
há como saber se uma lista vazia significa "código limpo" ou "instrumento cego".
"""

from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import hades_non_inference_scan as scanner  # noqa: E402

RAIZ = Path(__file__).resolve().parent.parent

# FORMA PRÉ-CORREÇÃO: exatamente o defeito B-15, como estava no repositório antes de E-190.
PRE_CORRECAO = '''
func (s *CompanyStore) TelAgentHistoryForOrganization(id, organizationID string, limit int) ([]TelAgentExchange, error) {
	company, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(company.OrganizationID) != strings.TrimSpace(organizationID) {
		return nil, ErrTelAgentOrganizationMismatch
	}
	return nil, nil
}
'''

# FORMA PÓS-CORREÇÃO: as duas recusas são a MESMA (a busca que falha devolve a negação unificada).
POS_CORRECAO = '''
func (s *CompanyStore) TelAgentHistoryForOrganization(id, organizationID string, limit int) ([]TelAgentExchange, error) {
	company, err := s.Get(id)
	if err != nil {
		return nil, ErrTelAgentOrganizationMismatch
	}
	if strings.TrimSpace(company.OrganizationID) != strings.TrimSpace(organizationID) {
		return nil, ErrTelAgentOrganizationMismatch
	}
	return nil, nil
}
'''

# SEM ORGANIZAÇÃO no par: não é o padrão que interessa.
SEM_ORGANIZACAO = '''
func (s *CompanyStore) Get(id string) (Company, error) {
	company, err := s.load(id)
	if err != nil {
		return Company{}, err
	}
	if company.Deleted {
		return Company{}, errCompanyRemoved
	}
	return company, nil
}
'''

# SEM BUSCA que falhe: não há o primeiro portão.
SEM_BUSCA = '''
func (s *Store) Resolve(organizationID, companyID string) (Company, error) {
	company := s.cache[companyID]
	if company.OrganizationID != organizationID {
		return Company{}, errAgentForbidden
	}
	return company, nil
}
'''


class CalibracaoDoScanner(unittest.TestCase):
    def test_encontra_a_forma_pre_correcao(self) -> None:
        achados = scanner.metodos_suspeitos(PRE_CORRECAO, "fixture.go")
        self.assertEqual(len(achados), 1, f"o positivo conhecido precisa ser encontrado: {achados}")
        self.assertEqual(achados[0]["metodo"], "TelAgentHistoryForOrganization")
        self.assertIn("organização", achados[0]["motivo"])

    def test_nao_acusa_a_forma_pos_correcao(self) -> None:
        achados = scanner.metodos_suspeitos(POS_CORRECAO, "fixture.go")
        self.assertEqual(achados, [], f"a versão corrigida não pode ser acusada: {achados}")

    def test_nao_acusa_metodo_sem_organizacao(self) -> None:
        achados = scanner.metodos_suspeitos(SEM_ORGANIZACAO, "fixture.go")
        self.assertEqual(achados, [], f"método sem parâmetro de organização não é o padrão: {achados}")

    def test_nao_acusa_metodo_sem_busca(self) -> None:
        achados = scanner.metodos_suspeitos(SEM_BUSCA, "fixture.go")
        self.assertEqual(achados, [], f"método sem busca que falhe não tem o primeiro portão: {achados}")

    def test_varredura_do_repositorio_real_roda(self) -> None:
        # Não afirma "zero suspeitos" (isso seria prova, não é papel do scanner): afirma que a varredura completa
        # funciona e devolve estrutura consistente.
        achados = scanner.varrer()
        for achado in achados:
            self.assertIn("arquivo", achado)
            self.assertIn("metodo", achado)
            self.assertIn("motivo", achado)

    def test_os_metodos_ja_corrigidos_nao_aparecem(self) -> None:
        # E-190 corrigiu os dois primeiros; E-192 corrigiu os quatro que o scanner apontou depois. Se algum voltar a
        # aparecer, é REGRESSÃO da correção (ou falso positivo do scanner) — as duas hipóteses exigem olhar.
        suspeitos = {achado["metodo"] for achado in scanner.varrer()}
        for metodo in (
            "TelAgentHistoryForOrganization",
            "CompanyFactoryReportFor",
            "GetForOrganization",
            "TelAgentSessionForOrganization",
        ):
            self.assertNotIn(metodo, suspeitos, f"método já corrigido voltou a ser suspeito: {metodo}")

    def test_o_candidato_restante_esta_declarado(self) -> None:
        # O scanner deixa UM candidato (o `call` do MCP remoto). Ele não foi verificado, e a metodologia declarada diz
        # que apontar não é provar: o teste exige que ele CONTINUE VISÍVEL em vez de ser silenciado para a lista ficar
        # bonita. Se ele sumir, é porque alguém o corrigiu (então atualize este teste) ou o scanner ficou cego.
        suspeitos = {(achado["arquivo"], achado["metodo"]) for achado in scanner.varrer()}
        self.assertIn(
            ("internal/agent/mcp_remote.go", "call"),
            suspeitos,
            "o candidato não verificado do MCP remoto precisa continuar visível (B-17)",
        )


if __name__ == "__main__":
    unittest.main()
