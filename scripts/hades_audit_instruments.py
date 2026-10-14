"""Reexecucao periodica dos INSTRUMENTOS de auditoria, com linha de base declarada (E-256).

Os instrumentos de E-253 (oraculo por ORDEM de checagem) e E-254 (escopo por ROTA) nasceram como scripts soltos. Um
instrumento que ninguem roda NAO protege nada: vira documento. Aqui eles passam a rodar com LINHA DE BASE:

- ORACULO por ordem: FALHA se aparecer candidato NOVO (identidade `arquivo::funcao`). Candidato que DESAPARECE e
  MELHORIA, nao falha - foi exatamente o que aconteceu com a decisao de aprovacao (5 -> 4 depois de E-253).
- ESCOPO por rota: FALHA se o numero de handlers sem escopo CRESCER acima do teto declarado. O teto existe porque os
  residuais sao, em sua maioria, LIMITE DO PROPRIO INSTRUMENTO (corpo do handler nao localizado) e rotas
  legitimamente publicas - declarar "35 IDORs" seria falso.

O que este runner NAO faz: provar que o escopo esta CORRETO (ordem e mensagem da recusa). Isso e leitura humana, e
foi por leitura que E-190, E-198, E-252 e E-253 encontraram defeito real.
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"

# LINHA DE BASE MEDIDA: 4 candidatos em 2026-10-16, TODOS falsos positivos de validacao de ENTRADA do proprio
# chamador (o valor comparado e' o pedido do atacante, nao estado de outro tenant). O candidato da decisao de
# aprovacao saiu da lista porque E-253 corrigiu a ordem das checagens.
#
# E-636 (2026-10-14): entrou um QUINTO candidato, `context.go::SkillVersionsForOrganization`, e a investigacao
# que o instrumento exige foi feita: ***nao e' falso positivo de validacao de entrada — o escopo E' aplicado.***
# A funcao le `s.skills[id]` e, quando `organizationID` nao e' vazio, recusa com
# `pluginOwnedByOrganization(atual.OrganizationID, organizationID)`; devolve vazio (e nao erro de escopo) para
# nao revelar a EXISTENCIA de uma skill de outra organizacao. O candidato entra declarado porque o instrumento
# procura a PRESENCA do parametro de organizacao para leitura humana — que e' o que aqui se fez.
BASELINE_ORACULO = (
    "internal/agent/context.go::SkillVersionsForOrganization",
    "internal/agent/mcp.go::register",
    "internal/agent/mcp_remote.go::register",
    "internal/agent/queue.go::EnqueueForOrganization",
    "internal/agent/redis_queue.go::EnqueueForOrganization",
)

# Teto medido: 28 residuais de 181 rotas. Era 35 em E-254; a LEITURA HUMANA dos residuais (E-257) mostrou que os
# quatro `approveCompany*` delegam a `decideCompanyApproval`, que chama `companyForRequest` (organizacao + recusa
# igualada por E-190) e exige papel de aprovador ANTES de decidir. Ensinada a delegacao ao instrumento, 35 -> 28.
# O restante segue declarado: limite do proprio instrumento (corpo do handler nao localizado) e rota publica por
# desenho (blobs, signout, callbacks OAuth/SAML, desktop-only).
TETO_ROTAS = 28

LINHA_CANDIDATO = re.compile(r"^\s*(\S+?)\s*::\s*([A-Za-z0-9_]+)\s*\(")
LINHA_RESIDUAL = re.compile(r"handlers SEM escopo[^\d]*(\d+)")


def normalizar(caminho: str) -> str:
    return caminho.replace("\\", "/")


def candidatos_oraculo(saida: str) -> list[str]:
    """Le a saida do scanner de oraculo e devolve identidades normalizadas `arquivo::funcao`."""
    encontrados = []
    for linha in saida.split("\n"):
        casamento = LINHA_CANDIDATO.match(linha)
        if casamento:
            encontrados.append("%s::%s" % (normalizar(casamento.group(1)), casamento.group(2)))
    return encontrados


def residuais_rota(saida: str) -> int:
    casamento = LINHA_RESIDUAL.search(saida)
    if not casamento:
        raise ValueError("saida do scanner de rota sem a contagem de residuais")
    return int(casamento.group(1))


def avaliar(candidatos: list[str], residuais: int) -> list[str]:
    """Problemas = candidato NOVO no oraculo ou residuais de rota ACIMA do teto. Desaparecer nao e problema."""
    problemas = []
    for candidato in candidatos:
        if candidato not in BASELINE_ORACULO:
            problemas.append("ORACULO NOVO: %s (investigar antes de aceitar como falso positivo)" % candidato)
    if residuais > TETO_ROTAS:
        problemas.append("ROTA: %d handlers sem escopo, acima do teto declarado (%d)" % (residuais, TETO_ROTAS))
    return problemas


def executar(script: str) -> str:
    resultado = subprocess.run(
        [sys.executable, str(SCRIPTS / script)],
        capture_output=True, text=True, cwd=str(RAIZ),
    )
    return resultado.stdout


def coletar() -> tuple[list[str], int]:
    return candidatos_oraculo(executar("hades_tenant_oracle_scan.py")), residuais_rota(executar("hades_route_scope_scan.py"))


def main() -> int:
    parser = argparse.ArgumentParser(description="Reexecuta os instrumentos de auditoria contra a linha de base")
    parser.add_argument("--json", action="store_true", help="saida estruturada")
    args = parser.parse_args()

    candidatos, residuais = coletar()
    problemas = avaliar(candidatos, residuais)
    if args.json:
        print(json.dumps({
            "candidatos_oraculo": candidatos,
            "baseline_oraculo": list(BASELINE_ORACULO),
            "residuais_rota": residuais,
            "teto_rotas": TETO_ROTAS,
            "problemas": problemas,
        }, ensure_ascii=False, indent=2))
    else:
        print("INSTRUMENTOS: oraculo=%d candidatos (baseline %d) · rota=%d residuais (teto %d)"
              % (len(candidatos), len(BASELINE_ORACULO), residuais, TETO_ROTAS))
    if problemas:
        for problema in problemas:
            print("PROBLEMA: %s" % problema)
        print("Rode o instrumento e LEIA o candidato antes de mexer na linha de base.")
        return 1
    print("INSTRUMENTOS CONSISTENTES COM A LINHA DE BASE: nenhum candidato novo, nenhum escopo novo em falta.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
