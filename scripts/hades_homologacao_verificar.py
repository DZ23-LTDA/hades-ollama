#!/usr/bin/env python3
"""Verificador de homologacao: UM comando que produz a tabela de evidencia da Camada A.

PORQUE ISTO EXISTE
------------------
A Camada C (homologacao independente) exige que OUTRA pessoa execute as Camadas A e B a partir de um
clone limpo. O `HOMOLOGACAO_INDEPENDENTE_RUNBOOK.md` listava os comandos em blocos separados; quem
revisa tinha de os copiar um a um e montar a tabela a mao. Isso torna a homologacao CARA em atencao --
e um pedido externo caro e' um pedido que nao se faz.

Este script reduz o trabalho do revisor a: `python -X utf8 scripts/hades_homologacao_verificar.py`.
Ele corre os gates obrigatorios na ordem da missao, recolhe a saida LITERAL de cada um, e escreve um
relatorio em `_tmp/HOMOLOGACAO_<sha>.md` que o revisor pode assinar (nome, data, SHA).

O QUE ELE NAO FAZ, e e' deliberado
----------------------------------
* Nao decide o veredito: so' recolhe factos e diz PASSOU/FALHOU por comando (exit code + cauda).
* Nao toca no repositorio (escreve so' em `_tmp/`, que e' ignorado pelo git).
* Nao substitui a Camada C: um relatorio gerado pelo autor **nao** e' homologacao independente. O
  script existe para que a pessoa que homologa tenha menos trabalho, nao para a substituir.
"""
from __future__ import annotations

import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

RAIZ = Path(__file__).resolve().parents[1]

# Ordem da missao (seccao 8): 10 geradores primeiro, depois os gates de documento.
GERADORES = [
    "hades_sync_state_metadata.py",
    "hades_requisitos.py",
    "hades_release_gate.py",
    "hades_manifest.py",
    "hades_brand_inventory.py",
    "hades_baseline_report.py",
    "hades_stage_map.py",
    "hades_architecture.py",
    "hades_platform_matrix.py",
    "hades_audit_package.py",
]

GATES = [
    ("hades_audit_package.py", "--check"),
    ("hades_index_shape_check.py", "--check"),
    ("hades_mission_check.py", ""),
    ("hades_handoff_window_check.py", "--check"),
    ("hades_release_gate.py", "--check"),
    ("hades_orphan_instruments_check.py", "--check"),
]


def correr(script: str, flag: str = "") -> tuple[int, str]:
    cmd = [sys.executable, "-X", "utf8", str(RAIZ / "scripts" / script)]
    if flag:
        cmd.append(flag)
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, cwd=RAIZ, timeout=900)
    except subprocess.TimeoutExpired:
        return 124, "(tempo esgotado)"
    saida = (proc.stdout or "") + (proc.stderr or "")
    linhas = [ln for ln in saida.strip().splitlines() if ln.strip()]
    cauda = linhas[-1] if linhas else "(sem saida)"
    return proc.returncode, cauda


def sha_actual() -> str:
    try:
        out = subprocess.run(
            ["git", "rev-parse", "--short", "HEAD"], capture_output=True, text=True, cwd=RAIZ
        )
        return (out.stdout or "").strip() or "desconhecido"
    except Exception:
        return "desconhecido"


def main() -> int:
    sha = sha_actual()
    agora = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    linhas = [
        f"# Homologacao -- evidencia recolhida no SHA `{sha}`",
        "",
        f"Gerado por `scripts/hades_homologacao_verificar.py` em {agora} (UTC).",
        "",
        "> Este relatorio foi produzido **pela ferramenta do repositorio**, nao por um revisor humano.",
        "> Por isso **nao** e' homologacao independente: e' o material que o revisor usa para a fazer.",
        "> Quem homologa tem de correr isto a partir de um clone limpo, ler as saidas e assinar.",
        "",
        "## Geradores (tem de correr sem erro)",
        "",
        "| # | Gerador | exit | ultima linha |",
        "| --- | --- | --- | --- |",
    ]
    falhas = 0
    for i, gerador in enumerate(GERADORES, start=1):
        rc, cauda = correr(gerador)
        if rc != 0:
            falhas += 1
        linhas.append(f"| {i} | `{gerador}` | {rc} | {cauda[:120]} |")

    linhas += [
        "",
        "## Gates de documento (todos tem de dar 0)",
        "",
        "| Gate | exit | ultima linha |",
        "| --- | --- | --- |",
    ]
    for script, flag in GATES:
        rc, cauda = correr(script, flag)
        if rc != 0:
            falhas += 1
        rotulo = f"{script} {flag}".strip()
        linhas.append(f"| `{rotulo}` | {rc} | {cauda[:120]} |")

    linhas += [
        "",
        "## Resultado",
        "",
        f"- Comandos com exit diferente de 0: **{falhas}**",
        f"- Veredito deste script: **{'AUSENCIA DE FALHAS' if falhas == 0 else 'HA FALHAS'}**",
        "",
        "## O que este relatorio NAO cobre",
        "",
        "- Camada B item a item (instalacao limpa, assinatura, desinstalacao, rollback) -- ver",
        "  `docs/mission/HADES_USER_READY_EVIDENCE.md` e os pedidos externos em `PEDIDOS_EXTERNOS.md`.",
        "- A bateria local completa (`scripts/hades_local_gates.ps1`), que depende de host Windows com",
        "  toolchain Go disponivel.",
        "- A verificacao adversarial (Camada C, requisito C2), que so' um revisor humano pode fazer.",
        "",
        "## SHAs citados nos documentos da missao (informativo, nao falha o script)",
        "",
        "Um identificador citado e' insubstituivel: se um SHA for inventado, tudo o que esta' correto ao",
        "lado perde credito. Esta verificacao existe porque o autor deste repositorio escreveu uma vez um",
        "SHA que nao existia num relatorio, e a leitura nao o apanharia -- so' o `git cat-file`.",
        "",
    ]

    # Caça aos SHAs fantasma: varre docs/mission/*.md por `abc123456` e confirma cada um.
    # Informativo de proposito: documentos antigos podem citar commits que ja' nao existem
    # localmente (histórico reescrito), e transformar isso em falha criaria ruido permanente.
    try:
        import re

        citados: set[str] = set()
        for doc in (RAIZ / "docs" / "mission").glob("*.md"):
            texto = doc.read_text(encoding="utf-8", errors="replace")
            citados.update(re.findall(r"`([0-9a-f]{9})`", texto))
        fantasma = []
        for candidato in sorted(citados):
            # `git cat-file -e <sha>` SEM `^{commit}`: nos documentos aparecem tambem SHAs de
            # BLOB (ex.: «restaurei-o byte a byte do blob `b5f7aefeb`»). Exigir tipo commit dava
            # falso positivo -- foi o que aconteceu na primeira versao deste check.
            verificado = subprocess.run(
                ["git", "cat-file", "-e", candidato],
                capture_output=True,
                cwd=RAIZ,
            )
            if verificado.returncode != 0:
                fantasma.append(candidato)
        linhas.append(f"- SHAs distintos citados: **{len(citados)}**")
        linhas.append(f"- SHAs inexistentes neste clone: **{len(fantasma)}**")
        if fantasma:
            linhas.append("")
            linhas.append("| SHA citado | existe? |")
            linhas.append("| --- | --- |")
            for f in fantasma:
                linhas.append(f"| `{f}` | **nao** |")
    except Exception as erro:  # pragma: no cover - informativo
        linhas.append(f"- (verificacao de SHAs indisponivel: {type(erro).__name__})")

    destino_dir = RAIZ / "_tmp"
    destino_dir.mkdir(exist_ok=True)
    destino = destino_dir / f"HOMOLOGACAO_{sha}.md"
    destino.write_text("\n".join(linhas) + "\n", encoding="utf-8")

    print("\n".join(linhas))
    print()
    print(f"RELATORIO: {destino}")
    print(f"HOMOLOGACAO_VERIFICAR={'PASS' if falhas == 0 else 'FAIL'}")
    return 0 if falhas == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
