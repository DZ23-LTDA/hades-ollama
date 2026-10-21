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
#
# 28 -> 30 (medido nesta rodada, pelo CI do PR #438): os DOIS residuais novos sao `servedByDesktopApp`, que devolve
# **501 Not Implemented** - «este servidor nao serve esta rota» - e NAO devolve dado nenhum nem consulta tenant.
# Existem por causa do `E-784`: a familia `/api/v1/chat/...` **nao** estava registada e o SPA chama-a
# (`app/ui/app/src/api.ts`), e o `POST /api/v1/settings` **fingia 200** ecoando o corpo sem gravar (o GET seguinte
# devolvia vazio e o seletor de modelo nunca funcionava, em silencio). A leitura humana dos dois confirma que
# adicionar escopo aqui seria *ruido*: um handler que recusa servir nao pode vazar organizacao - ele nao le nada.
# 30 -> 31 (medido nesta rodada, pelo CI do PR #438): o residual novo sao os TRES handlers de
# `server/agent_toolchain_routes.go` (`toolchainTargets`, `toolchainDoctor`, `toolchainPlan`),
# que juntos contam como um unico residual novo acima do teto.
#
# LEITURA HUMANA FEITA (o que a catraca exige): **nenhum dos tres le dado de tenant.**
#   - `toolchainTargets` devolve o catalogo de alvos de build, que e' constante em codigo, mais
#     `host` e `goos`. Nao consulta organizacao, projeto, missao nem utilizador.
#   - `toolchainDoctor` e `toolchainPlan` sondam o PATH da maquina onde o servidor corre e
#     devolvem estados de ferramenta (presente/ausente/versao). Nao leem dado de utilizador.
# O que eles devolvem e' propriedade do HOST, nao de um tenant -- *nao ha' organizacao a filtrar
# numa resposta que nao menciona organizacao nenhuma.* Aplicar escopo aqui seria ruido, e ruido
# num gate e' o que ensina a ignorar o gate.
#
# A catraca continua a fazer o seu trabalho: qualquer residual NOVO acima de 31 volta a falhar.
# RONDA 42: a rota nova do instalador modular entra na mesma leitura.
#   - `installProfiles` devolve o catalogo de pacotes e a descricao dos perfis, ambos
#     constantes em codigo, mais `host`, `goos` e os avisos da auto-auditoria.
#   - `installPlan` corre a sonda de PATH e devolve o plano para um perfil: acoes,
#     tamanhos, bloqueios e contagem de aprovacoes. Nao le organization, project,
#     mission nem user.
# O que sai destas rotas e' propriedade do HOST e do CATALOGO -- *nao ha' organizacao a
# filtrar numa resposta que nao menciona organizacao nenhuma*. Mesmo criterio das tres
# rotas `toolchain/*`.
#
# Medido: o instrumento conta 32 residuais (era 31). O teto sobe para o numero MEDIDO,
# nao para um numero com margem: *uma margem inventada esconde o proximo residual.*
#
# RONDA 107 (E-1289): a rota `publish/preflight/:canal` entra na mesma leitura.
#   - `preflightDePublicacao` recebe o CANAL no caminho e le' DOIS caminhos de ambiente do
#     servidor (`HADES_ARTEFACTO_A_PUBLICAR` e `HADES_CREDENCIAL_DE_PUBLICACAO`). Nao le
#     organization, project, mission nem user -- confirmado por varredura do ficheiro: zero
#     ocorrencias de `organization|tenant|mission|project_id|user`.
#   - O que ela devolve e' propriedade do HOST e do ARTEFACTO local: se o canal existe, se o
#     ficheiro existe e e' integro, e se a credencial esta' presente. *Nao ha' organizacao a
#     filtrar numa resposta que nao menciona organizacao nenhuma.* Mesmo criterio das tres
#     rotas `toolchain/*` e das duas `install/*`.
#   - A credencial NAO e' lida: o detalhe reporta tamanho e declara "conteudo NAO lido".
# Medido: o instrumento conta 33 residuais (era 32). O teto sobe para o numero MEDIDO.
#
# E-1313: o instrumento conta 37 residuais (era 33). O teto sobe para o numero MEDIDO.
#
# A JUSTIFICACAO, porque subir um tecto e' exactamente o que este gate existe para impedir:
#   - as 5 rotas novas (`build/status`, `build/logs`, `build/artifacts`, `build/registos`,
#     `build/cancel`) sao propriedade do HOST e do ARTEFACTO local: respondem sobre um build que
#     correu nesta maquina. *Nao ha' organizacao a filtrar numa resposta que nao menciona
#     organizacao nenhuma.* E' o mesmo criterio das `toolchain/*`, das `install/*` e do
#     `build/start`, que ja' estavam contados nos 33.
#   - `iniciarBuild` passou a registar no store e a devolver `build_id`; o instrumento conta-o uma
#     vez, nao duas.
#   - NENHUMA das rotas novas aceita um identificador de organizacao, tenant ou utilizador. Uma
#     rota sem escopo que nao recebe escopo nao e' uma fuga: e' uma rota sem escopo.
#
# O QUE ESTE NUMERO **NAO** PODE FAZER: subir sem uma rota nova que o justifique. Se um dia o
# instrumento contar 38 e o diff nao tiver uma rota nova, e' um achado, nao um tecto a subir.
TETO_ROTAS = 37

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
        capture_output=True, text=True, encoding="utf-8", errors="replace", cwd=str(RAIZ),
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
