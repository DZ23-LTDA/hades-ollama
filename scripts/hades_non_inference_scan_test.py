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

    def test_o_conjunto_de_candidatos_e_exatamente_o_esperado(self) -> None:
        # A lista precisa ser CURTA E EXPLICADA. Cada candidato abaixo foi olhado; os dois são falsos positivos
        # conhecidos do heurístico, e o motivo fica escrito aqui — não escondido:
        #
        # - `mcp_remote.go::call`: o caminho de TENANT devolve a mesma recusa (E-194). O scanner o aponta porque o
        #   caminho GLOBAL mantém, de propósito, a mensagem "não registrado" — ali não há tenant de quem inferir.
        # - `auth.go::refreshOAuthCredential`: o caminho de TENANT devolve a MESMA mensagem de "fora da organização"
        #   (E-195). O scanner compara o texto dos dois ramos e não enxerga a condicional `organizationID != ""`; sem
        #   tenant não existe o ramo de organização, então não há assimetria para um tenant.
        #
        # Exigir o conjunto EXATO (e não apenas "contém") é o que transforma a lista num instrumento: um candidato
        # novo aparece aqui como falha de calibração, e um candidato que sumir avisa que o scanner ficou cego.
        esperado = {
            ("internal/agent/auth.go", "refreshOAuthCredential"),
            ("internal/agent/mcp_remote.go", "call"),
        }
        obtido = {(achado["arquivo"], achado["metodo"]) for achado in scanner.varrer()}
        self.assertEqual(esperado, obtido, f"conjunto de candidatos mudou: {sorted(obtido)}")

if __name__ == "__main__":
    unittest.main()
