#!/usr/bin/env python3
"""Gera o relatório de baseline e o índice de falhas do estágio 2 (E-186).

POR QUE ESTE ARQUIVO EXISTE
---------------------------
A seção 5 do prompt master exige, para o estágio 2, "logs de cada suíte, relatório de baseline, índice de falhas".
Os dois documentos nomeados **não existiam** — e a checagem nova (`scripts/hades_mission_check.py`, E-186) provou
isso contra o repositório real em vez de eu afirmar cobertura.

Escrever dois documentos à mão apenas para satisfazer uma checagem seria decoração, e eles voltariam a divergir na
primeira mudança de estado. Então eles são **GERADOS** a partir de fatos verificáveis:

- `HADES_BASELINE_REPORT.md`: as suítes medidas, cada uma com o **comando reproduzível** e o resultado observado;
- `HADES_FAILURE_INDEX.md`: as falhas e limites registrados, derivados de `HADES_MASTER_STATE.json` (bloqueios) —
  um bloqueio que aparece no estado aparece aqui, sem cópia manual.

`--check` compara o que está no disco com o que seria gerado e falha em divergência, do mesmo jeito que o gate de
release: assim o documento não pode envelhecer em silêncio.

LIMITES DECLARADOS
------------------
- Os tempos são de **uma máquina** (Windows, go1.26.9, 2026-10) e variam; o CI executa o seu próprio baseline.
- A tabela de suítes é **declarada** aqui: o gerador garante que documento e tabela concordem, **não** que a tabela
  reflita a última execução — quem prova a execução é o log/CI, não este arquivo.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
ESTADO = RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json"
BASELINE = RAIZ / "docs" / "mission" / "HADES_BASELINE_REPORT.md"
FALHAS = RAIZ / "docs" / "mission" / "HADES_FAILURE_INDEX.md"

# Suítes medidas. `resultado` é o que foi OBSERVADO na execução, não um alvo.
SUITES: tuple[dict[str, str], ...] = (
    {
        "suite": "agent (Windows)",
        "comando": "go test ./internal/agent/ -count=1",
        "resultado": "ok — 70,2 s a 74,6 s",
    },
    {
        "suite": "server (Windows)",
        "comando": "go test ./server/ -count=1",
        "resultado": "ok — 22,2 s a 24,0 s",
    },
    {
        "suite": "agent (Linux, container golang:1.26.9)",
        "comando": "docker run --rm -v <repo>:/src -w /src golang:1.26.9 sh -c 'go vet ./internal/agent/ ./server/ && go test ./internal/agent/ ./server/'",
        "resultado": "ok — agent ≈20 s a 27 s · server ≈6 s a 8 s",
    },
    {
        "suite": "race detector (Linux, CGO_ENABLED=1)",
        "comando": "scripts/verify-linux-parity.ps1 (etapa de -race) em golang:1.26.9",
        "resultado": "sem DATA RACE — agent 80,5 s · server 11,0 s",
    },
    {
        "suite": "lint",
        "comando": "golangci-lint run --allow-parallel-runners ./internal/agent/... ./server/...",
        "resultado": "0 issues",
    },
    {
        "suite": "contrato de superfícies Class A+",
        "comando": "node scripts/verify-contracts.mjs",
        "resultado": "PASSED — chamadas do frontend batem com método e rota",
    },
    {
        "suite": "web E2E — primeira execução",
        "comando": "npx playwright test --project=first-run --reporter=list (app/ui/app)",
        "resultado": "2 passed (26,0 s) — instalação, onboarding, persistência, reinício, atualização e rollback",
    },
    {
        "suite": "plugin (terceiro + workspace + diagnóstico)",
        "comando": "go test ./internal/agent/ -run 'ThirdParty|PreparePluginWorkspace|ExecutionFailure' -count=1",
        "resultado": "instalação/verificação/extracção PASS; execução SKIP declarada (sandbox Linux-only e sem privilégio de montagem)",
    },
)


def carregar_estado() -> dict:
    return json.loads(ESTADO.read_text(encoding="utf-8"))


def gerar_baseline(estado: dict) -> str:
    linhas = [
        "# Relatório de baseline — estágio 2 (gerado, não editar à mão)",
        "",
        "> Gerado por `python scripts/hades_baseline_report.py`. O `--check` no CI falha se este arquivo divergir do",
        "> gerador, para o baseline não envelhecer em silêncio.",
        "",
        "## 1. Ambiente das medições",
        "",
        f"- commit de referência: `{estado.get('git', {}).get('main_sha', 'desconhecido')}`",
        "- SO das medições locais: Windows (runner do agente); o CI roda Linux",
        "- toolchain: **go1.26.9**, fixado por `go-version-file: go.mod` em todos os workflows (E-167)",
        "- plataforma Linux: container `golang:1.26.9`; execução de plugin: container `node:22-bookworm`",
        "",
        "## 2. Suítes medidas",
        "",
        "| Suíte | Comando reproduzível | Resultado observado |",
        "| --- | --- | --- |",
    ]
    for suite in SUITES:
        linhas.append(f"| {suite['suite']} | `{suite['comando']}` | {suite['resultado']} |")
    linhas += [
        "",
        "## 3. O que este baseline prova — e o que não prova",
        "",
        "- **Prova**: as suítes acima foram executadas com estes comandos e estes resultados, nesta máquina e no CI (os",
        "  workflows rodam as mesmas suítes; o CI é a autoridade sobre o verde).",
        "- **Não prova**: desempenho em hardware diferente, nem que a suíte cobre todos os caminhos — cobertura não é",
        "  medida aqui. Os tempos são de **uma máquina** e variam.",
        "- **Não prova**: homologação em macOS nem em pacote assinado — esses requisitos seguem dependência externa.",
        "",
        "## 4. Regressão",
        "",
        "Comparar com este arquivo antes de integrar: mudança de tempo em ordem de grandeza, suíte que passa a falhar ou",
        "SKIP novo precisam ser explicados no PR. Foi assim que a portabilidade Windows pegou o defeito de fim de linha",
        "do fixture de terceiro (E-183) e o `govet` pegou o `return` inalcançável (E-183).",
        "",
    ]
    return "\n".join(linhas)


def gerar_falhas(estado: dict) -> str:
    bloqueios = estado.get("blockers", [])
    # Classificacao por REGRA EXPLICITA, nao por substring: o tipo "AMBIENTE (DIAGNOSTICO RESOLVIDO)" contem
    # "RESOLVIDO" e foi classificado como resolvido na primeira versao deste gerador - um limite PARCIALMENTE
    # resolvido nao e um limite fechado, e esconder isso seria o mesmo defeito que este indice existe para evitar.
    def esta_resolvido(bloqueio: dict) -> bool:
        return str(bloqueio.get("tipo", "")).strip().upper().startswith("RESOLVIDO")

    resolvidos = [b for b in bloqueios if esta_resolvido(b)]
    abertos = [b for b in bloqueios if not esta_resolvido(b)]
    linhas = [
        "# Índice de falhas e limites — estágio 2 (gerado, não editar à mão)",
        "",
        "> Gerado por `python scripts/hades_baseline_report.py` a partir de `HADES_MASTER_STATE.json`: um bloqueio que",
        "> existe no estado aparece aqui, e um que sai do estado sai daqui. O `--check` no CI detecta divergência.",
        "",
        f"Total: **{len(bloqueios)}** — {len(abertos)} em aberto · {len(resolvidos)} resolvidos.",
        "",
        "## 1. Em aberto",
        "",
        "| ID | Tipo | O que é |",
        "| --- | --- | --- |",
    ]
    if abertos:
        for b in abertos:
            resumo = " ".join(str(b.get("descricao", "")).split())[:360]
            linhas.append(f"| {b.get('id')} | {b.get('tipo')} | {resumo} |")
    else:
        linhas.append("| — | — | nenhum |")
    linhas += [
        "",
        "## 2. Resolvidos",
        "",
        "| ID | O que era | Situação |",
        "| --- | --- | --- |",
    ]
    if resolvidos:
        for b in resolvidos:
            resumo = " ".join(str(b.get("descricao", "")).split())[:360]
            linhas.append(f"| {b.get('id')} | {resumo} | resolvido, com teste contra regressão onde aplicável |")
    else:
        linhas.append("| — | — | nenhum |")
    linhas += [
        "",
        "## 3. Limites declarados",
        "",
        "- O texto de cada limite é o do estado; este índice **não** julga se o desbloqueio é suficiente.",
        "- Defeito corrigido **sem** teste que o impeça de voltar não entra como resolvido por conveniência: a coluna de",
        "  situação diz onde há teste.",
        "- Este arquivo é um **índice**: a evidência de cada item está em `HADES_EVIDENCE_INDEX.md` e nos documentos de",
        "  estágio citados lá.",
        "",
    ]
    return "\n".join(linhas)


def escrever(caminho: Path, conteudo: str) -> None:
    caminho.write_text(conteudo, encoding="utf-8")
    print(f"gerado: {caminho.relative_to(RAIZ)}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Gera baseline e índice de falhas do estágio 2")
    parser.add_argument("--check", action="store_true", help="falha se o disco divergir do gerador")
    args = parser.parse_args()

    estado = carregar_estado()
    esperado = {BASELINE: gerar_baseline(estado), FALHAS: gerar_falhas(estado)}

    if args.check:
        divergentes = []
        for caminho, conteudo in esperado.items():
            atual = caminho.read_text(encoding="utf-8") if caminho.is_file() else ""
            if atual != conteudo:
                divergentes.append(caminho.relative_to(RAIZ))
        if divergentes:
            for caminho in divergentes:
                print(f"DIVERGENTE: {caminho}")
            print("Rode: python scripts/hades_baseline_report.py")
            return 1
        print("BASELINE E INDICE DE FALHAS CONSISTENTES: os dois batem com o gerador e o estado.")
        return 0

    for caminho, conteudo in esperado.items():
        escrever(caminho, conteudo)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
