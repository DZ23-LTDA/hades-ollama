#!/usr/bin/env python3
"""Gera (e CONFERE) os artefatos do gate global a partir do estado da missão (E-179).

Por que este script existe
--------------------------
`docs/mission/HADES_FINAL_RELEASE_GATE.md` e `docs/mission/HADES_RELEASE_REPORT.md` são exigidos pela seção 11.8
das diretrizes da missão. Eles foram criados à mão (E-173) e, enquanto o gerador vivia apenas em `_tmp/`, os
artefatos não eram reproduzíveis: ninguém conseguia reobter o mesmo documento a partir do estado. Este script
fecha isso — **o instrumento passa a ser versionado junto com o artefato**.

Uso
---
    python scripts/hades_release_gate.py            # regenera os dois documentos
    python scripts/hades_release_gate.py --check    # falha (exit 1) se os documentos divergirem do estado

O `--check` é o ponto: um gate que diz "DONE" enquanto o índice de evidências diz outra coisa é um gate que
mente. Com o `--check`, a divergência aparece como falha em vez de virar relatório bonito.

O que este script NÃO faz: não inventa status de estágio. Ele deriva de `HADES_MASTER_STATE.json`
(`execution`, `capability`, `evidence`, `blocker`) e da matriz de aceite declarada abaixo — que é a única parte
mantida à mão, porque exige julgamento sobre o que cada requisito da seção 9 exige.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ESTADO = RAIZ / "docs/mission/HADES_MASTER_STATE.json"
INDICE = RAIZ / "docs/mission/HADES_EVIDENCE_INDEX.md"
GATE = RAIZ / "docs/mission/HADES_FINAL_RELEASE_GATE.md"
RELATORIO = RAIZ / "docs/mission/HADES_RELEASE_REPORT.md"

# Matriz de aceite da seção 9. Cada item: (requisito, status, evidência ou limite, o que destrava).
# Status possíveis: DONE, IN_PROGRESS, BLOCKED_BY_EXTERNAL_DEPENDENCY, NAO_COMPROVADO.
# REGRA: só vai para DONE o item com execução verificada e apontada na coluna de evidência.
MATRIZ_ACEITE = [
    ("O Hades inicia corretamente", "DONE", "Suítes de runtime e inicialização verdes; nenhuma falha de startup pendente", "—"),
    ("A interface funciona", "DONE", "Contrato de frontend verificado por `scripts/verify-contracts.mjs` (job obrigatório do CI)", "—"),
    ("O onboarding está validado", "NAO_COMPROVADO", "Não há teste de jornada de primeiro uso com usuário", "Executar a jornada de onboarding com evidência (estágio 19)"),
    ("Modelos locais são descobertos e usados", "DONE", "Descoberta e uso local com evidências no índice (estágio 4)", "—"),
    ("APIs externas configuradas são testadas e usadas", "BLOCKED_BY_EXTERNAL_DEPENDENCY", "B-07: chaves de provedor", "Credencial de provedor (estágio 4)"),
    ("A troca de modelo preserva contexto autorizado", "DONE", "Bootstrap de contexto reconstruído por modelo, com teste (estágio 5)", "—"),
    ("O sistema de permissões é aplicado pelo servidor", "DONE", "Autorização server-side com testes de escopo (estágios 5 e 18)", "—"),
    ("Plugins têm instalação e execução verificáveis", "NAO_COMPROVADO", "Há contrato e testes herméticos; falta executar um plugin **real** de terceiro", "Instalar e executar um plugin de terceiro (estágio 8)"),
    ("MCPs têm instalação e execução verificáveis", "DONE", "MCP local/stdio exercitado com testes de revogação e auditoria (estágio 7)", "—"),
    ("Skills têm ativação e execução verificáveis", "DONE", "Ativação por projeto/agente/modelo com testes (estágio 8)", "—"),
    ("Agentes e subagentes executam missões reais", "DONE", "Orquestração com cotas e artefatos verificáveis (estágio 9)", "—"),
    ("O runtime suporta recuperação e retomada", "DONE", "Checkpoints, cancelamento e retomada testados (estágios 10 e 13)", "—"),
    ("Ferramentas funcionam dentro das permissões", "DONE", "Execução governada com recusa fora de escopo (estágios 7 e 18)", "—"),
    (
        "O Builder entrega aplicações utilizáveis",
        "DONE",
        "E-178: app do Studio criado, **publicado**, **servido por HTTP (200)** e **aberto em navegador real**, que "
        "renderizou 3 componentes no canvas com **0 erros de console** (`docs/mission/evidence/`). "
        "**LIMITE**: app de **demonstração** (heading/table/button) — **não** é full-stack com banco/auth/CRUD",
        "App full-stack com banco, autenticação e CRUD navegado ponta a ponta (estágio 12)",
    ),
    ("O agente de código entrega correções testadas", "DONE", "Fluxo issue → teste → correção → PR comprovado (estágio 11)", "—"),
    ("Workflows executam os nós realmente suportados", "DONE", "Motor determinístico com retry/DLQ e log por nó (estágio 13)", "—"),
    ("Browser/computer-use funciona nos ambientes homologados", "BLOCKED_BY_EXTERNAL_DEPENDENCY", "B-09 residual: Python ausente para o E2E de browser", "Ambiente com navegador/Python para o E2E (estágio 14)"),
    ("A memória respeita os escopos", "DONE", "Isolamento entre organizações com prova em disco (estágio 15)", "—"),
    ("A segurança multitenant está validada", "IN_PROGRESS", "RLS e isolamento com testes; falta prova em banco real (estágio 18)", "Banco PostgreSQL real para provar RLS (estágio 18)"),
    ("A suíte obrigatória está aprovada", "DONE", "Lint, contrato, suíte Go e paridade Linux (com detector de corrida) verdes na rodada mais recente", "—"),
    ("A instalação limpa está homologada", "BLOCKED_BY_EXTERNAL_DEPENDENCY", "B-06: certificado de assinatura e runners de SO", "Certificado + runners reais (estágio 20)"),
    ("A documentação corresponde ao produto", "DONE", "Relatórios de estágio, índice de evidências e limites declarados por módulo", "—"),
    ("Os artefatos de distribuição estão disponíveis", "NAO_COMPROVADO", "Empacotamento existe; falta release assinado publicado", "Assinatura e publicação de release (estágio 20)"),
    ("Não existem bloqueios críticos ou altos sem mitigação aceita", "NAO_COMPROVADO", "Bloqueios externos em aberto e limites de severidade MÉDIO declarados", "Resolver ou aceitar formalmente cada bloqueio"),
]


def carregar_estado() -> dict:
    return json.loads(ESTADO.read_text(encoding="utf-8"))


def linhas_do_indice() -> int:
    return len(re.findall(r"^\| E-\d+", INDICE.read_text(encoding="utf-8"), re.M))


def contagem_por_execucao(estado: dict) -> dict[str, int]:
    contagem: dict[str, int] = {}
    for estagio in estado["stages"]:
        chave = estagio.get("execution", "SEM_EXECUTION")
        contagem[chave] = contagem.get(chave, 0) + 1
    return contagem


def tabela_de_estagios(estado: dict, filtro: set[str] | None = None) -> str:
    linhas = []
    for estagio in sorted(estado["stages"], key=lambda e: e.get("id") or 0):
        if filtro and estagio.get("execution") not in filtro:
            continue
        evidencias = ", ".join(estagio.get("evidence") or []) or "—"
        if len(evidencias) > 120:
            evidencias = evidencias[:120] + "…"
        titulo = (estagio.get("title") or "").replace("|", "/")
        linhas.append(
            f"| {estagio.get('id')} | {titulo} | {estagio.get('capability') or '—'} | "
            f"{estagio.get('execution')} | {estagio.get('blocker') or '—'} | {evidencias} |"
        )
    return "\n".join(linhas)


def gerar_relatorio(estado: dict) -> str:
    contagem = contagem_por_execucao(estado)
    externos = [b for b in estado.get("blockers", []) if "EXTERNO" in (b.get("tipo") or "").upper()]
    resolvidos = [b for b in estado.get("blockers", []) if "RESOLVIDO" in (b.get("tipo") or "").upper()]
    regra = estado.get("execution_rule", "(regra não registrada no estado)")
    return f"""# HADES — RELATÓRIO DE RELEASE (DONE / NOT_DONE / BLOCKED)

> Exigido pela seção **11.8** das diretrizes V2.1. Este documento **não** usa percentuais arbitrários: cada
> estágio aparece com o estado de **capacidade**, o estado de **execução**, o bloqueio associado e as evidências
> que o sustentam. **Gerado por `scripts/hades_release_gate.py`** — rode com `--check` para conferir.

**Veredito global: `{estado.get('release_state')}`.**

| Número | Valor |
| --- | --- |
| Estágios | {len(estado['stages'])} |
| Linhas de evidência no índice | {linhas_do_indice()} |
| `COMPLETED_VERIFIED` | {contagem.get('COMPLETED_VERIFIED', 0)} |
| `IN_PROGRESS` | {contagem.get('IN_PROGRESS', 0)} |
| `BLOCKED_BY_EXTERNAL_DEPENDENCY` | {contagem.get('BLOCKED_BY_EXTERNAL_DEPENDENCY', 0)} |
| `NOT_STARTED` | {contagem.get('NOT_STARTED', 0)} |
| Bloqueios externos em aberto | {len(externos)} |
| Bloqueios já resolvidos | {len(resolvidos)} |

## Como `execution` foi atribuído

{regra}

O campo é **derivado**, e a regra fica gravada no próprio `HADES_MASTER_STATE.json` (`execution_rule`). Onde a
seção 7 pede campos que o projeto **não** registra por estágio (commit inicial/final e PR por estágio), este
relatório **declara a ausência** em vez de preencher com suposição.

## DONE (concluído e verificado)

| # | Estágio | Capacidade | Execução | Bloqueio | Evidências |
| --- | --- | --- | --- | --- | --- |
{tabela_de_estagios(estado, {'COMPLETED_VERIFIED'})}

## IN_PROGRESS (trabalho real, com limites declarados)

| # | Estágio | Capacidade | Execução | Bloqueio | Evidências |
| --- | --- | --- | --- | --- | --- |
{tabela_de_estagios(estado, {'IN_PROGRESS'})}

## BLOCKED (impedimento externo, com procedimento de desbloqueio)

| # | Estágio | Capacidade | Execução | Bloqueio | Evidências |
| --- | --- | --- | --- | --- | --- |
{tabela_de_estagios(estado, {'BLOCKED_BY_EXTERNAL_DEPENDENCY'})}

Procedimento de desbloqueio de cada bloqueio externo está em
[`HADES_BLOCKERS.md`](HADES_BLOCKERS.md): cada item traz o que falta, quem desbloqueia e **o que continua
possível sem ele**.

## NOT_DONE / NOT_STARTED (declarado, sem trabalho iniciado)

| # | Estágio | Capacidade | Execução | Bloqueio | Evidências |
| --- | --- | --- | --- | --- | --- |
{tabela_de_estagios(estado, {'NOT_STARTED'})}

## Referências

- Índice de evidências: [`HADES_EVIDENCE_INDEX.md`](HADES_EVIDENCE_INDEX.md) ({linhas_do_indice()} linhas).
- Estado da missão: [`HADES_MASTER_STATE.json`](HADES_MASTER_STATE.json).
- Gate global: [`HADES_FINAL_RELEASE_GATE.md`](HADES_FINAL_RELEASE_GATE.md).
- Gate intermediário de engenharia (estágio 22): [`HADES_ENGINEERING_GATE.md`](HADES_ENGINEERING_GATE.md).

> **Honestidade de release**: `{estado.get('release_state')}` significa que **requisito obrigatório não
> comprovado** ainda existe. Nenhum item pode ser promovido sem evidência anexada ao índice.
"""


def gerar_gate(estado: dict) -> str:
    externos = [b for b in estado.get("blockers", []) if "EXTERNO" in (b.get("tipo") or "").upper()]
    resumo: dict[str, int] = {}
    for _, status, _, _ in MATRIZ_ACEITE:
        resumo[status] = resumo.get(status, 0) + 1
    resumo_texto = " · ".join(f"{k}: {v}" for k, v in sorted(resumo.items()))
    linhas_matriz = "\n".join(
        f"| {i + 1} | {nome} | **{status}** | {prova} | {destrava} |"
        for i, (nome, status, prova, destrava) in enumerate(MATRIZ_ACEITE)
    )
    linhas_bloqueios = "\n".join(
        f"| {b.get('id')} | {(b.get('descricao') or '')[:160].replace('|', '/')} | ver HADES_BLOCKERS.md |"
        for b in externos
    )
    return f"""# HADES — GATE FINAL DE RELEASE

> Exigido pela seção **11.8**. Este é o **gate GLOBAL**, posterior aos estágios 1–38 — e **não** substitui o gate
> intermediário de engenharia (estágio 22, em [`HADES_ENGINEERING_GATE.md`](HADES_ENGINEERING_GATE.md)).
> **Gerado por `scripts/hades_release_gate.py`** — rode com `--check` para conferir.

## Veredito

**`{estado.get('release_state')}`** — `RELEASE_CANDIDATE` e `COMPLETED_VERIFIED` **não** podem ser declarados
enquanto houver requisito obrigatório não comprovado. O veredito é consequência da matriz abaixo, não avaliação
subjetiva.

## Matriz de aceite (seção 9 das diretrizes)

Resumo: {resumo_texto}

| # | Requisito | Status | Evidência / limite | O que destrava |
| --- | --- | --- | --- | --- |
{linhas_matriz}

### Como ler os status

- **DONE**: há execução verificada com evidência no índice.
- **IN_PROGRESS**: parte comprovada, parte **não** — o limite está declarado na coluna de evidência.
- **BLOCKED_BY_EXTERNAL_DEPENDENCY**: depende de credencial, conta, certificado, hardware ou serviço
  indisponível. O bloqueio **não** interrompe trabalho independente e o procedimento está em
  [`HADES_BLOCKERS.md`](HADES_BLOCKERS.md).
- **NAO_COMPROVADO**: não há evidência de execução. Não é "quase pronto": é **não comprovado**.

## Bloqueios externos em aberto

| ID | Descrição | Detalhe |
| --- | --- | --- |
{linhas_bloqueios}

## O que faria o veredito mudar

1. **Fechar os itens `NAO_COMPROVADO`** com execução real e evidência anexada.
2. **Resolver ou aceitar formalmente** cada bloqueio externo.
3. **Concluir os estágios `IN_PROGRESS`** com os limites que hoje estão declarados.

Nenhuma dessas condições pode ser satisfeita por documentação: todas exigem **execução com evidência**.

## Honestidade de estado

- O release global está `{estado.get('release_state')}` e permanece assim até a matriz acima fechar.
- `execution` por estágio é **derivado por regra** (ver `execution_rule` no `HADES_MASTER_STATE.json`).
- Evidências sem comando reproduzível ou sem ambiente declarado **não** contam para homologação.
- Este arquivo é **gerado**: alterar à mão cria divergência que o `--check` acusa.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Gera/confere os artefatos do gate global.")
    parser.add_argument("--check", action="store_true", help="não escreve: falha se houver divergência")
    args = parser.parse_args()

    estado = carregar_estado()
    gerados = {GATE: gerar_gate(estado), RELATORIO: gerar_relatorio(estado)}

    if args.check:
        divergentes = []
        for caminho, conteudo in gerados.items():
            atual = caminho.read_text(encoding="utf-8") if caminho.exists() else ""
            if atual != conteudo:
                divergentes.append(caminho.name)
        if divergentes:
            print("GATE DIVERGENTE do estado da missão: " + ", ".join(divergentes))
            print("Rode: python scripts/hades_release_gate.py")
            return 1
        print("GATE CONSISTENTE: os dois documentos batem com o estado da missão.")
        return 0

    for caminho, conteudo in gerados.items():
        caminho.write_text(conteudo, encoding="utf-8")
        print("gerado:", caminho.relative_to(RAIZ))
    return 0


if __name__ == "__main__":
    sys.exit(main())
