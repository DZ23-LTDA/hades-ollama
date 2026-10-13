#!/usr/bin/env python3
"""Testes da catraca de escopo de tenant (E-338).

O instrumento é exercitado nas DUAS direções com fixtures: sem isso, uma catraca que nunca fecha vira enfeite.
"""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import hades_tenant_scope_check as catraca  # noqa: E402


class CatracaDeEscopoTest(unittest.TestCase):
    def _dir(self, **arquivos: str) -> Path:
        temp = Path(tempfile.mkdtemp())
        for nome, conteudo in arquivos.items():
            (temp / nome).write_text(conteudo, encoding="utf-8")
        return temp

    def test_arquivo_com_guarda_passa(self) -> None:
        raiz = self._dir(**{"com_guarda.go": (
            "func (s *Store) caminho(org string) string {\n"
            "\tif err := requireOrganization(org); err != nil { return \"\" }\n"
            "\treturn filepath.Join(s.root, sanitizeIDComponent(org)+\".json\")\n"
            "}\n")})
        violacoes, pendentes = catraca.auditar(raiz)
        self.assertEqual([], violacoes, violacoes)
        self.assertEqual([], pendentes, pendentes)

    def test_arquivo_sem_guarda_e_sem_dispensa_reprova(self) -> None:
        raiz = self._dir(**{"sem_guarda.go": (
            "func (s *Store) caminho(org string) string {\n"
            "\treturn filepath.Join(s.root, sanitizeIDComponent(org)+\".json\")\n"
            "}\n")})
        violacoes, _ = catraca.auditar(raiz)
        self.assertTrue(any("SEM `requireOrganization`" in v for v in violacoes), violacoes)

    def test_dispensa_sem_motivo_reprova(self) -> None:
        raiz = self._dir(**{"dispensado.go": (
            "func (s *Store) caminho(org string) string {\n"
            "\treturn filepath.Join(s.root, sanitizeIDComponent(org)+\".json\")\n"
            "}\n")})
        original = dict(catraca.DISPENSAS)
        try:
            catraca.DISPENSAS["dispensado.go"] = ""
            violacoes, _ = catraca.auditar(raiz)
            self.assertTrue(any("dispensa SEM MOTIVO" in v for v in violacoes), violacoes)
        finally:
            catraca.DISPENSAS.clear()
            catraca.DISPENSAS.update(original)

    def test_dispensa_com_motivo_vira_pendencia_e_nao_aprovacao(self) -> None:
        raiz = self._dir(**{"declarado.go": (
            "func (s *Store) caminho(org string) string {\n"
            "\treturn filepath.Join(s.root, sanitizeIDComponent(org)+\".json\")\n"
            "}\n")})
        original = dict(catraca.DISPENSAS)
        try:
            catraca.DISPENSAS["declarado.go"] = "NAO_MEDIDO: pendente de varredura de comportamento."
            violacoes, pendentes = catraca.auditar(raiz)
            self.assertEqual([], violacoes, violacoes)
            # ARMADILHA: dispensa não pode virar "aprovado" — tem de aparecer como PENDÊNCIA.
            self.assertEqual(1, len(pendentes))
            self.assertIn("NAO_MEDIDO", pendentes[0])
        finally:
            catraca.DISPENSAS.clear()
            catraca.DISPENSAS.update(original)

    def test_uso_que_nao_e_de_armazenamento_e_ignorado(self) -> None:
        raiz = self._dir(**{"outro_uso.go": (
            "func nomeArquivo(valor string) string {\n"
            "\treturn sanitizeIDComponent(valor)\n"
            "}\n")})
        violacoes, pendentes = catraca.auditar(raiz)
        self.assertEqual([], violacoes, violacoes)
        self.assertEqual([], pendentes, pendentes)

    def test_checagem_crua_de_vazio_tambem_e_guarda_reconhecida(self) -> None:
        """E-342: um arquivo pode estar CERTO sem usar o helper (checando o valor cru)."""
        raiz = self._dir(**{"cria_cru.go": (
            "func (s *Store) Sync(req Request) error {\n"
            "\torganizationID := strings.TrimSpace(req.OrganizationID)\n"
            "\tif organizationID == \"\" {\n"
            "\t\treturn ErrInvalid\n"
            "\t}\n"
            "\tpath := filepath.Join(s.root, sanitizeIDComponent(organizationID)+\".json\")\n"
            "\treturn nil\n"
            "}\n")})
        violacoes, pendentes = catraca.auditar(raiz)
        self.assertEqual([], violacoes, violacoes)
        self.assertEqual([], pendentes, pendentes)

    def test_arquivo_sem_guarda_alguma_continua_reprovando(self) -> None:
        """A direcao negativa: sem helper E sem checagem crua, reprova."""
        raiz = self._dir(**{"sem_nada.go": (
            "func (s *Store) caminho(org string) string {\n"
            "\treturn filepath.Join(s.root, sanitizeIDComponent(org)+\".json\")\n"
            "}\n")})
        violacoes, _ = catraca.auditar(raiz)
        self.assertTrue(any("SEM `requireOrganization`" in v for v in violacoes), violacoes)

    def test_arquivo_de_teste_e_ignorado(self) -> None:
        raiz = self._dir(**{"coisa_test.go": (
            "func TestX(t *testing.T) {\n"
            "\tp := filepath.Join(root, sanitizeIDComponent(\"org-a\")+\".json\")\n"
            "}\n")})
        violacoes, pendentes = catraca.auditar(raiz)
        self.assertEqual([], violacoes, violacoes)
        self.assertEqual([], pendentes, pendentes)

    def test_diretorio_inexistente_nao_quebra(self) -> None:
        violacoes, pendentes = catraca.auditar(Path(tempfile.mkdtemp()) / "nao-existe")
        self.assertEqual([], violacoes)
        self.assertEqual([], pendentes)

    def test_repositorio_real_passa_e_declara_as_pendencias(self) -> None:
        violacoes, pendentes = catraca.auditar()
        self.assertEqual([], violacoes, "armazenamento sem guarda e sem dispensa:\n" + "\n".join(violacoes))
        # As pendências declaradas existem e são VISÍVEIS (o número pode encolher; crescer exige declarar).
        self.assertTrue(pendentes, "as pendências declaradas precisam aparecer no relatório")
        # Cada pendência tem de ter MOTIVO explícito e se declarar em uma de DUAS categorias honestas:
        #   (a) NAO_MEDIDO — falta varredura de comportamento;  (b) validação por DELEGAÇÃO (o chamador validou).
        # O que não pode existir é dispensa muda — essa já é reprovada por `test_dispensa_sem_motivo_reprova`.
        for pendencia in pendentes:
            self.assertIn(" — dispensa: ", pendencia, pendencia)
            _, motivo = pendencia.split(" — dispensa: ", 1)
            self.assertTrue(motivo.strip(), f"pendência sem motivo: {pendencia}")
            # Três categorias honestas: (a) NAO_MEDIDO (falta varredura); (b) validação por DELEGAÇÃO;
            # (c) CONTRATO TESTADO do repositório que decide o contrário (o contrato ganha da minha preferência).
            categorizada = ("NAO_MEDIDO" in motivo or "NÃO MEDIDO" in motivo or "já foi validada" in motivo
                            or "não há escopo" in motivo or "CONTRATO TESTADO" in motivo)
            self.assertTrue(categorizada, f"pendência sem categoria honesta (NAO_MEDIDO ou delegação): {pendencia}")


if __name__ == "__main__":
    unittest.main(verbosity=2)
