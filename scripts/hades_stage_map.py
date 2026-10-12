#!/usr/bin/env python3
"""Mapa entre os estágios TRACKADOS no estado e os estágios DEFINIDOS pelo prompt master (E-187).

O PROBLEMA, MEDIDO
------------------
A seção 5 do prompt master define os 38 estágios com título e escopo. O estado da missão, porém, foi construído com
**outra** numeração: medido em E-187, **37 dos 38** títulos divergem, e do estágio **24 em diante o escopo está
DESLOCADO** — o que o estado chama de *"CRM e pipeline de vendas"* é o `AI SALES WORKFORCE` do master (27), enquanto
o master chama de 24 o `COMMERCE ENGINE`, que o estado registra no 25. Sem mapa, "estágio 24 concluído" tem **dois
significados** e nenhuma tabela de cobertura pode ser lida como cobertura do documento da missão.

O QUE ESTE ARQUIVO FAZ — E O QUE NÃO FAZ
----------------------------------------
- **Faz**: declara, estágio por estágio, a situação do pareamento com o master, em uma tabela única; gera
  `docs/mission/HADES_STAGE_MAP.md` a partir dela (com `--check`, para não envelhecer); e torna a **completude
  mecânica** verificável pelo `hades_mission_check.py` (todo estágio do estado tem linha, e o mapa cobre 1..38).
- **Não faz**: reescrever a numeração do estado. Renumerar 38 estágios **muda o significado de todas as evidências
  já acumuladas** — não é decisão reversível de baixo risco, e inventar um pareamento 1:1 onde o escopo diverge
  seria falsificar cobertura. As linhas com escopo divergente ficam **visíveis e contadas**, como dívida de
  governança a resolver por decisão explícita.

A DECISÃO QUE FALTA (procedimento exato)
----------------------------------------
1. Congelar a numeração do estado (`HADES_MASTER_STATE.json`), que carrega o histórico de evidências;
2. declarar o mapa canônico — este arquivo — como a **única** tradução permitida entre os dois;
3. onde a situação for `ESCOPO_DIVERGENTE`, decidir por estágio: (a) manter o escopo do estado e registrar o master
   correspondente como coberto, ou (b) abrir estágio novo para o escopo do master que não existe;
4. só então usar a matriz do gate para afirmar cobertura do master, citando o mapa.

LIMITES DECLARADOS
------------------
- O pareamento é **julgamento de escopo** feito por leitura dos dois documentos; onde não é possível afirmar
  equivalência sem inventar, a linha diz `ESCOPO_DIVERGENTE` ou `SEM_PAR_NO_MESTRE` em vez de forçar um número.
- A tabela é **declarada**: o `--check` garante que o documento gerado corresponda à tabela, **não** que o
  julgamento de escopo esteja certo — quem discordar muda a tabela, e o documento acompanha.
"""

from __future__ import annotations

import argparse
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DESTINO = RAIZ / "docs" / "mission" / "HADES_STAGE_MAP.md"

# Situações possíveis do pareamento.
EQUIVALENTE = "EQUIVALENTE"
TITULO_DIFERENTE = "TITULO_DIFERENTE"
ESCOPO_DIVERGENTE = "ESCOPO_DIVERGENTE"
SEM_PAR_NO_MESTRE = "SEM_PAR_NO_MESTRE"

# Título exato do prompt master, por estágio (usado como referência do mapa).
TITULOS_MESTRES: dict[int, str] = {
    1: "AUDITORIA TOTAL DO REPOSITÓRIO",
    2: "ESTABILIZAÇÃO E CONSOLIDAÇÃO",
    3: "ARQUITETURA UNIVERSAL DO HADES",
    4: "MODEL GATEWAY UNIVERSAL",
    5: "CONTEXTO AUTOMÁTICO PARA QUALQUER IA",
    6: "HARNESS RUNTIME UNIVERSAL",
    7: "MCP MANAGER COMPLETO",
    8: "PLUGINS E SKILLS",
    9: "AGENTES E SUBAGENTES ESPECIALIZADOS",
    10: "PARIDADE OPERACIONAL COM MANUS",
    11: "PARIDADE COM CODEX, CLAUDE CODE E DEVIN",
    12: "PARIDADE COM LOVABLE, BOLT, V0, BASE44 E FLUTTERFLOW",
    13: "MOTOR DE AUTOMAÇÃO ESTILO N8N",
    14: "BROWSER, DESKTOP E COMPUTER-USE",
    15: "MEMÓRIA UNIVERSAL E CONTEXTO PERSISTENTE",
    16: "CANAIS, VOZ E TEL-AGENT",
    17: "FRIGG-LIKE AGENT WORKSPACE E COLABORAÇÃO",
    18: "SEGURANÇA ENTERPRISE E MULTITENANCY",
    19: "EXPERIÊNCIA PROFISSIONAL PARA USUÁRIOS LEIGOS",
    20: "DEVOPS, INSTALADORES E RELEASE",
    21: "BENCHMARK COMPETITIVO E HOMOLOGAÇÃO FINAL",
    22: "RELEASE READINESS INTERMEDIÁRIO (GATE DE ENGENHARIA)",
    23: "COMPANY FACTORY",
    24: "COMMERCE ENGINE",
    25: "MARKETPLACE HUB",
    26: "SOCIAL MEDIA OS",
    27: "AI SALES WORKFORCE",
    28: "VOICE & CALL CENTER",
    29: "OMNICHANNEL INBOX",
    30: "FINANCE & OPERATIONS",
    31: "AUTONOMOUS BUSINESS SUPERVISOR",
    32: "BUSINESS E2E CERTIFICATION",
    33: "CONSOLIDAR WHATSAPP E TELEGRAM EXISTENTES",
    34: "HADES OMNICHANNEL RUNTIME",
    35: "AGENTES DE ATENDIMENTO E VENDAS",
    36: "SOCIAL MEDIA E MARKETPLACE OPERACIONAL",
    37: "VOZ E TELEFONIA COM TEL-AGENT",
    38: "HOMOLOGAÇÃO DE UMA EMPRESA AUTÔNOMA",
}

# (id_do_estado, título_no_estado, IDs do master correspondentes, situacao, observacao)
#
# O terceiro campo é uma TUPLA porque um estágio do estado pode cobrir MAIS DE UM estágio do prompt master: o
# master separa 32 (BUSINESS E2E CERTIFICATION) de 38 (HOMOLOGAÇÃO DE UMA EMPRESA AUTÔNOMA) e o estado tem um só
# estágio final que carrega os dois. Forçar um-para-um ali produziria uma lacuna FALSA - o tipo tem que caber no
# que os documentos dizem.
MAPA: tuple[tuple[int, str, tuple[int, ...], str, str], ...] = (
    (1, "Auditoria total e inventario real", (1,), TITULO_DIFERENTE, "mesmo escopo: auditoria do repositório"),
    (2, "Estabilizacao e consolidacao da base", (2,), TITULO_DIFERENTE, "mesmo escopo: baseline e falhas"),
    (3, "Arquitetura canonica dos componentes", (3,), TITULO_DIFERENTE, "mesmo escopo: 16 componentes"),
    (4, "Model Gateway (roteamento, rotacao, fallback)", (4,), TITULO_DIFERENTE, "mesmo escopo"),
    (5, "Context Bootstrap / contexto de projeto", (5,), TITULO_DIFERENTE, "mesmo escopo"),
    (6, "Harness Runtime (execucao governada)", (6,), TITULO_DIFERENTE, "mesmo escopo"),
    (7, "MCP Manager (servidores, ferramentas, escopo)", (7,), TITULO_DIFERENTE, "mesmo escopo"),
    (8, "Plugins e Skills", (8,), EQUIVALENTE, "título e escopo coincidem"),
    (9, "Agentes e subagentes (delegacao)", (9,), TITULO_DIFERENTE, "mesmo escopo"),
    (10, "Mission Control multiagente (missoes, eventos)", (10,), ESCOPO_DIVERGENTE,
     "DECISAO_B (LACUNA DECLARADA): o master 10 pede PARIDADE OPERACIONAL ponta a ponta e este estagio entrega o COMPONENTE Mission Control; nao afirmo cobertura operacional sem evidencia de missao autonoma completa. Escopo do master 10 sem dono declarado. Historico: o estado descreve o COMPONENTE Mission Control; o master pede PARIDADE OPERACIONAL COM MANUS (missão autônoma ponta a ponta)"),
    (11, "Paridade Codex/Claude (harnesses de codigo)", (11,), TITULO_DIFERENTE, "mesmo escopo"),
    (12, "Studio / Builder visual", (12,), TITULO_DIFERENTE, "mesmo escopo"),
    (13, "Workflow Engine (fluxos, schedules, condicoes)", (13,), TITULO_DIFERENTE, "mesmo escopo"),
    (14, "Browser Operator / computer-use", (14,), TITULO_DIFERENTE, "mesmo escopo"),
    (15, "Memoria e RAG local", (15,), TITULO_DIFERENTE, "mesmo escopo"),
    (16, "Canais de IM (WhatsApp/Telegram/inbox unificado)", (16,), ESCOPO_DIVERGENTE,
     "DECISAO_A (COBERTURA POR UNIAO): o master 16 junta IM + voz/Tel-Agent; o estado cobre IM aqui e voz/telefonia no estagio 29 - a uniao 16+29 cobre o escopo. Historico: o master inclui VOZ e TEL-AGENT neste estágio; no estado, voz/telefonia aparecem adiante"),
    (17, "Workspace do agente (threads, artefatos, presenca)", (17,), TITULO_DIFERENTE, "mesmo escopo (frigg-like)"),
    (18, "Seguranca, permissoes e multitenancy", (18,), TITULO_DIFERENTE, "mesmo escopo"),
    (19, "UX/onboarding e jornada de primeira execucao", (19,), TITULO_DIFERENTE, "mesmo escopo"),
    (20, "Distribuicao e instaladores (Windows/macOS/Linux)", (20,), TITULO_DIFERENTE, "mesmo escopo"),
    (21, "Benchmarks e comparativo competitivo", (21,), TITULO_DIFERENTE, "mesmo escopo"),
    (22, "Gate de engenharia intermediario", (22,), TITULO_DIFERENTE, "mesmo escopo"),
    (23, "Company Factory (empresa orquestrada)", (23,), TITULO_DIFERENTE, "mesmo escopo"),
    (24, "CRM e pipeline de vendas", (27,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DESLOCAMENTO): CRM/pipeline corresponde ao master 27 (AI SALES WORKFORCE) e a evidencia do estado sustenta esse escopo. Historico: DESLOCAMENTO: o master chama de 24 o COMMERCE ENGINE; CRM/pipeline corresponde ao master 27 (AI SALES WORKFORCE)"),
    (25, "Commerce / loja e pedidos", (24,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DESLOCAMENTO): e o COMMERCE ENGINE do master 24, com evidencia propria. Historico: DESLOCAMENTO: é o COMMERCE ENGINE do master (24)"),
    (26, "Marketplace multi-vendedor", (25,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DESLOCAMENTO): e o MARKETPLACE HUB do master 25. Historico: DESLOCAMENTO: é o MARKETPLACE HUB do master (25)"),
    (27, "Pagamentos e faturamento", (30,), ESCOPO_DIVERGENTE,
     "DECISAO_B (LACUNA DECLARADA): o FINANCE & OPERATIONS (master 30) pede contas a pagar/receber, fluxo de caixa e conciliacao, e NAO existe estagio do estado com esse escopo - a evidencia aqui e CRM/vendas. Escopo do master 30 sem dono declarado. Historico: o master trata pagamentos dentro de COMMERCE ENGINE (24) e FINANCE & OPERATIONS (30); no estado estão isolados"),
    (28, "Atendimento e suporte (helpdesk)", (29,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DESLOCAMENTO): e o OMNICHANNEL INBOX do master 29. Historico: é o OMNICHANNEL INBOX do master (29); o master 28 é VOICE & CALL CENTER, que o estado trata no 29"),
    (29, "Telefonia e call center", (28,), ESCOPO_DIVERGENTE, "DECISAO_B (LACUNA DECLARADA): a evidencia ENTREGUE aqui e a CAIXA DE ENTRADA UNIFICADA (master 29), nao o VOICE & CALL CENTER do master 28 - o conteudo nao corresponde ao par declarado, e telefonia real depende de provedor (B-04). Escopo do master 28 sem dono declarado. Historico: é o VOICE & CALL CENTER do master (28)"),
    (30, "Marketing e campanhas", (26,), TITULO_DIFERENTE,
     "é o SOCIAL MEDIA OS do master (26): campanhas, calendário e conteúdo sob aprovação"),
    (31, "Social e publicacao", (36,), ESCOPO_DIVERGENTE,
     "DECISAO_B (LACUNA DECLARADA): a evidencia ENTREGUE aqui e o SUPERVISOR AUTONOMO DE NEGOCIO (master 31), nao a operacao de redes do master 36; a parte de marketplace do master 36 e coberta pelo estado 26, mas a operacao de redes com publicacao autorizada e metricas fica SEM dono. Historico: operação de redes com publicação autorizada e métricas corresponde ao master 36 (SOCIAL MEDIA E MARKETPLACE OPERACIONAL); a parte de MARKETPLACE do master 36 é coberta pelo estado 26"),
    (32, "BI / business status e metricas", (31,), ESCOPO_DIVERGENTE,
     "DECISAO_A (COBERTURA PARCIAL DECLARADA): dashboards e indicadores atendem ao monitoramento do master 31. Historico: dashboards/indicadores correspondem ao AUTONOMOUS BUSINESS SUPERVISOR do master (31)"),
    (33, "Canal Telegram", (33,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DIVISAO DECLARADA): o master 33 junta WhatsApp e Telegram; o estado divide em 33 e 34 e a uniao cobre. Historico: o master junta WhatsApp E Telegram em UM estágio (33); o estado separa em 33 e 34"),
    (34, "Canal WhatsApp", (34,), ESCOPO_DIVERGENTE,
     "DECISAO_A (CORRIGIDA COM EVIDENCIA E-188): este estagio carrega o RUNTIME OMNICHANNEL do master 34. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega o RUNTIME OMNICHANNEL do master 34 - E-105 aponta `docs/mission/HADES_STAGE_34_OMNICHANNEL.md` e E-141/E-170..E-177 medem dead letter, rate limit, conectores, cache de eventos, reserva em voo e -race. O canal do master 33 é coberto pelo estado 33"),
    (35, "Inbox unificado omnicanal", (35,), ESCOPO_DIVERGENTE,
     "DECISAO_A (COBERTURA COM EVIDENCIA E-212): o escopo AGENTES DE ATENDIMENTO E VENDAS (master 35) esta implementado em internal/agent/service_agent*.go - ServiceAgentCatalog, NewServiceAgentRuntime, RunTurn, aprovacoes, trilha de auditoria, conhecimento e ferramentas de comercio/empresa, com 85 testes PASS. A lacuna era de ATRIBUICAO no estado, nao de implementacao. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega os AGENTES DE ATENDIMENTO E VENDAS do master 35 - E-106 aponta `docs/mission/HADES_STAGE_35_SERVICE_AGENT...` e E-144..E-156 medem `TestServiceAgent` (política, auditoria, aprovação). O OMNICHANNEL INBOX do master 29 é coberto pelo estado 28"),
    (36, "Voz e transcricao", (37,), ESCOPO_DIVERGENTE,
     "DECISAO_A (DIVISAO DECLARADA): e o master 37, atendido por DOIS estagios do estado (36 e 37); telefonia real segue dependente de provedor (B-04). Historico: VOZ E TELEFONIA COM TEL-AGENT do master (37)"),
    (37, "Chamadas e follow-up", (37,), ESCOPO_DIVERGENTE, "DECISAO_A (DIVISAO DECLARADA): SEGUNDO estagio do estado para o MESMO master 37 - divisao declarada, nao colisao. Historico: segundo estágio do estado para o mesmo do master (37)"),
    (38, "Gate final de release verificado", (32, 38), ESCOPO_DIVERGENTE,
     "DECISAO_A (COBERTURA POR UNIAO): o gate final cobre o master 32 e o master 38, como a propria linha declara. Historico: CORRIGIDO COM EVIDENCIA (E-188): este estágio carrega a HOMOLOGAÇÃO DE EMPRESA do master 32 - E-109 e E-112 medem `TestBusinessE2E` (2/2, 21 etapas, com reinício real sobre o mesmo diretório) e `TestBusinessCertification`. O GATE de release global NÃO é um estágio do master: é o requisito transversal da seção 11.8, e por isso não aparece como par"),
)


def situacoes() -> dict[str, int]:
    contagem: dict[str, int] = {}
    for _, _, _, situacao, _ in MAPA:
        contagem[situacao] = contagem.get(situacao, 0) + 1
    return contagem


def estagios_mestres_cobertos() -> list[int]:
    return sorted({mestre for _, _, mestres, _, _ in MAPA for mestre in mestres})


def mestres_nao_cobertos() -> list[int]:
    cobertos = set(estagios_mestres_cobertos())
    return [i for i in sorted(TITULOS_MESTRES) if i not in cobertos]


def mestres_cobertos_mais_de_uma_vez() -> dict[int, int]:
    contagem: dict[int, int] = {}
    for _, _, mestres, _, _ in MAPA:
        for mestre in mestres:
            contagem[mestre] = contagem.get(mestre, 0) + 1
    return {m: c for m, c in sorted(contagem.items()) if c > 1}


def gerar() -> str:
    contagem = situacoes()
    duplicados = mestres_cobertos_mais_de_uma_vez()
    nao_cobertos = mestres_nao_cobertos()
    linhas = [
        "# Mapa de estágios: estado da missão × prompt master (gerado, não editar à mão)",
        "",
        "> Gerado por `python scripts/hades_stage_map.py`. O `--check` no CI falha se este arquivo divergir da tabela.",
        "",
        "## 1. Por que este mapa existe (E-187)",
        "",
        "A seção 5 do prompt master define os 38 estágios. O estado da missão foi construído com **outra numeração**: "
        "**37 dos 38** títulos divergem e, **do 24 em diante, o escopo está deslocado**. Sem esta tradução, *\"estágio "
        "24 concluído\"* tem **dois significados** — `CRM e pipeline de vendas` no estado e `COMMERCE ENGINE` no "
        "master — e nenhuma tabela de cobertura pode ser lida como cobertura do documento da missão.",
        "",
        "## 2. Situação do pareamento",
        "",
        "| Situação | Quantos |",
        "| --- | --- |",
    ]
    for situacao in (EQUIVALENTE, TITULO_DIFERENTE, ESCOPO_DIVERGENTE, SEM_PAR_NO_MESTRE):
        linhas.append(f"| {situacao} | {contagem.get(situacao, 0)} |")
    linhas += [
        "",
        f"**Dívida de governança declarada:** {contagem.get(ESCOPO_DIVERGENTE, 0)} estágios com escopo divergente e "
        f"{contagem.get(SEM_PAR_NO_MESTRE, 0)} sem par no master.",
        "",
        "## 3. O mapa",
        "",
        "| Estado | Título no estado | Mestre | Título no master | Situação | Observação |",
        "| --- | --- | --- | --- | --- | --- |",
    ]
    for ident, titulo, mestres, situacao, observacao in MAPA:
        alvo = ", ".join(str(m) for m in mestres) if mestres else "—"
        titulo_mestre = " + ".join(TITULOS_MESTRES.get(m, "—") for m in mestres) if mestres else "—"
        linhas.append(f"| {ident} | {titulo} | {alvo} | {titulo_mestre} | **{situacao}** | {observacao} |")
    linhas += [
        "",
        "## 4. Cobertura dos estágios do master",
        "",
        f"- estágios do master **cobertos** pelo estado: {len(estagios_mestres_cobertos())} de {len(TITULOS_MESTRES)}",
        f"- estágios do master **sem nenhum** estágio do estado apontando: {nao_cobertos if nao_cobertos else 'nenhum'}",
        f"- estágios do master apontados por **mais de um** estágio do estado: {duplicados if duplicados else 'nenhum'}",
        "",
        "## 5. O que este mapa NÃO faz",
        "",
        "- **Não** renumera o estado: renumerar 38 estágios muda o significado de todas as evidências acumuladas, o que "
        "não é decisão reversível de baixo risco.",
        "- **Não** afirma que os pareamentos estão certos: cada linha é **julgamento de escopo** por leitura dos dois "
        "documentos, e onde equivalência não se sustenta sem inventar, a linha diz `ESCOPO_DIVERGENTE` ou "
        "`SEM_PAR_NO_MESTRE` em vez de forçar um número.",
        "- **Não** substitui a matriz do gate: para afirmar cobertura do master é preciso citar esta tradução.",
        "",
        "## 6. Procedimento de reconciliação",
        "",
        "1. congelar a numeração do estado (que carrega o histórico de evidências);",
        "2. tratar este mapa como a **única** tradução permitida entre os dois;",
        "3. por estágio `ESCOPO_DIVERGENTE`, decidir: (a) manter o escopo do estado e registrar o master como coberto, "
        "ou (b) abrir estágio para o escopo do master que não existe;",
        "4. só então usar a matriz do gate para afirmar cobertura do master, citando o mapa.",
        "",
    ]
    return "\n".join(linhas)


def main() -> int:
    parser = argparse.ArgumentParser(description="Gera o mapa estado × prompt master")
    parser.add_argument("--check", action="store_true", help="falha se o disco divergir da tabela")
    args = parser.parse_args()
    conteudo = gerar()
    if args.check:
        atual = DESTINO.read_text(encoding="utf-8") if DESTINO.is_file() else ""
        if atual != conteudo:
            print(f"DIVERGENTE: {DESTINO.relative_to(RAIZ)}")
            print("Rode: python scripts/hades_stage_map.py")
            return 1
        print("MAPA DE ESTAGIOS CONSISTENTE: o documento bate com a tabela declarada.")
        return 0
    DESTINO.write_text(conteudo, encoding="utf-8")
    print(f"gerado: {DESTINO.relative_to(RAIZ)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
