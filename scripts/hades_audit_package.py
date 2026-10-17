#!/usr/bin/env python3
"""Gera o pacote de auditoria independente (`E-1076`, §FASE-7 item 8).

O QUE O PROMPT PEDE, E POR QUE ESTE SCRIPT EXISTE.

O §FASE-7 item 8: *«O pacote de auditoria independente deve conter no minimo: `README_AUDITOR.md`,
`PARIDADE_EVIDENCIAS.csv`, `GATES.md`, `LIMITACOES.md`, `ARTEFATOS.sha256`, SBOMs, lista de findings
abertos/fechados e instrucoes de reproducao.»*

**Um pacote escrito a mao envelhece no dia seguinte.** E' o `E-1071` outra vez: um documento que
afirma o estado do repositorio tem de ser **funcao** do repositorio, nao uma fotografia.

=> por isso o pacote e' **gerado**, e tem `--check` como todos os outros derivados.

A REGRA QUE GOVERNA O CONTEUDO: **nao inventar**.

O pacote nao declara nada que nao venha de uma fonte medivel:

    README_AUDITOR.md      <- as seccoes, os hashes, e o que o auditor NAO vai encontrar
    PARIDADE_EVIDENCIAS.csv <- cada entrada do indice, com o numero e o estado do registo
    GATES.md               <- os gates DECLARADOS no hades_local_gates.ps1, um por linha
    LIMITACOES.md          <- os blockers do HADES_MASTER_STATE.json + a faixa historica + as
                              limitacoes por plataforma
    ARTEFATOS.sha256       <- SHA-256 dos artefactos que EXISTEM em disco, e so' esses

**O que este script NAO faz**: nao verifica se um gate passa, nao reconstroi evidencia perdida, nao
corre os testes. *Um pacote que se gaba de verificar o que nao correu e' pior do que nenhum.*

O QUE ELE DIZ QUE NAO SABE, E ISSO E' O MAIS IMPORTANTE.

O `README_AUDITOR.md` tem uma seccao **«O que voce NAO vai encontrar»**, e ela e' gerada a partir do
proprio estado: as 133 evidencias sem texto, os artefatos que nao existem, os testes que nunca
correram. *Um pacote honesto comeca por dizer o que lhe falta.*
"""
import argparse
import hashlib
import io
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
PACOTE = RAIZ / "docs" / "audit"
INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"
ESTADO = RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json"
GATES = RAIZ / "scripts" / "hades_local_gates.ps1"

# **Os artefactos que o pacote mede, e SO' esses.** *Uma lista declarada e' auditavel; um `glob`
# apanha o que estiver la' e cala-se sobre o que falta.*
ARTEFATOS_DECLARADOS = [
    "HADES_SBOM_GO.csv",
    "HADES_SBOM_LICENCAS.csv",
    "HADES_SBOM_NPM.csv",
    "THIRD_PARTY_NOTICES.md",
    "LICENSE",
    "docs/mission/HADES_EVIDENCE_INDEX.md",
    "docs/mission/HADES_MASTER_STATE.json",
    "docs/mission/HADES_FINAL_RELEASE_GATE.md",
    "docs/mission/HADES_SNAPSHOT_MANIFEST.json",
    "docs/mission/HADES_BRAND_INVENTORY.md",
    "dist/HadesSetup.exe",
    "dist/windows-ollama-app-amd64.exe",
    "dist/Hades-0.2.0-linux-amd64-unsigned.tar.gz",
    "dist/hades-linux-amd64-unsigned.tar.gz",
]

# **O que este pacote NAO prova** -- escrito a mao porque e' uma decisao, nao uma medicao.
NAO_PROVA = [
    "**nao prova que o CI esta' verde** -- isso e' do SHA, e o SHA muda a cada commit;",
    "**nao prova homologacao em dispositivo** -- nao houve dispositivo;",
    "**nao prova revisao humana independente** -- nao houve;",
    "**nao prova que o produto e' release** -- o veredito e' `NOT_RELEASE_READY`;",
    "**nao prova superioridade competitiva** -- nenhuma capacidade esta' `SUPERIOR_MEASURED`;",
    "**nao executa os gates** -- lista-os; quem os corre e' o auditor;",
]


def sha256(caminho: Path) -> str:
    h = hashlib.sha256()
    with open(caminho, "rb") as f:
        for bloco in iter(lambda: f.read(1 << 20), b""):
            h.update(bloco)
    return h.hexdigest()


def ler_indice() -> list[tuple[str, str]]:
    """Devolve (ID, primeira celula truncada) para cada entrada do indice."""
    entradas = []
    for linha in io.open(INDICE, encoding="utf-8").read().split("\n"):
        m = re.match(r"^\|\s*(E-\d+)\s*\|\s*(.*?)\s*\|\s*$", linha)
        if m:
            entradas.append((m.group(1), m.group(2)))
    return entradas


def ler_estado() -> dict:
    return json.load(io.open(ESTADO, encoding="utf-8"))


def ler_gates() -> list[str]:
    """Os gates DECLARADOS no `hades_local_gates.ps1`: as linhas `'--- N. ...'`."""
    out = []
    for linha in io.open(GATES, encoding="utf-8").read().split("\n"):
        m = re.match(r"^\s*'--- (\d+)\.\s*(.+?)'\s*$", linha)
        if m:
            out.append((int(m.group(1)), m.group(2)))
    out.sort()
    return out


def gerar() -> dict[str, str]:
    entradas = ler_indice()
    estado = ler_estado()
    gates = ler_gates()
    faixas = estado.get("evidence_ids_nao_registrados_faixas") or []
    blockers = estado.get("blockers") or []
    release = estado.get("release_state") or "?"

    cabeca = max(int(e.split("-")[1]) for e, _ in entradas)
    faixa_txt = " e ".join("`%s..%s`" % (f[0], f[1]) for f in faixas) if faixas else "(nenhuma)"

    # ---------- ARTEFATOS.sha256 ----------
    linhas_art = []
    ausentes = []
    for rel in ARTEFATOS_DECLARADOS:
        p = RAIZ / rel
        if p.is_file():
            linhas_art.append("%s  %s" % (sha256(p), rel))
        else:
            ausentes.append(rel)
    artefatos = "\n".join(linhas_art) + "\n"

    # ---------- PARIDADE_EVIDENCIAS.csv ----------
    csv = ["id,tem_registro,tamanho_primeira_celula"]
    for eid, celula in entradas:
        csv.append("%s,sim,%d" % (eid, len(celula)))
    paridade = "\n".join(csv) + "\n"

    # ---------- GATES.md ----------
    g = [
        "# GATES — os que estão declarados, e como os correr",
        "",
        "**Este ficheiro lista os gates DECLARADOS em `scripts/hades_local_gates.ps1`.** Não afirma",
        "que passam: *quem os corre é o auditor, e o resultado é do estado em que ele os correr.*",
        "",
        "| # | Gate |",
        "| --- | --- |",
    ]
    for n, desc in gates:
        g.append("| %d | %s |" % (n, desc))
    g += [
        "",
        "## Como correr a bateria completa",
        "",
        "```powershell",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_local_gates.ps1",
        "```",
        "",
        "## Como correr os checkers de forma independente",
        "",
        "```powershell",
        "python -X utf8 scripts/hades_release_gate.py --check",
        "python -X utf8 scripts/hades_manifest.py --check",
        "python -X utf8 scripts/hades_brand_inventory.py --check",
        "python -X utf8 scripts/hades_index_shape_check.py --check",
        "python -X utf8 scripts/hades_handoff_window_check.py --check",
        "python -X utf8 scripts/hades_mission_check.py",
        "```",
        "",
        "**Nota**: `hades_release_gate.py`, `hades_manifest.py` e `hades_brand_inventory.py` são",
        "**geradores**; correm em modo de escrita e só depois `--check`. *Um `--check` sem gerar",
        "acusa drift por construção.*",
        "",
    ]
    gates_md = "\n".join(g)

    # ---------- LIMITACOES.md ----------
    lim = [
        "# LIMITAÇÕES — o que este estado NÃO tem",
        "",
        "**Veredito: `%s`.**" % release,
        "",
        "## Bloqueios declarados no estado (`%d`)" % len(blockers),
        "",
    ]
    # **Separar o que esta' resolvido do que nao esta' -- pelo CAMPO, nao pelo texto.**
    #
    # *A primeira versao deste gerador separava por `"RESOLVIDO" in descricao.upper()`* => *e meteu
    # o `B-13` nos resolvidos porque a **descricao** tem a palavra algures, apesar de o `tipo` dele
    # ser... `RESOLVIDO` (por acaso acertou); mas o `B-14`, cuja descricao diz «LIMITE (E-184)», foi
    # para resolvidos por ser `AMBIENTE (DIAGNOSTICO RESOLVIDO)` -- e esse **e' aberto no limite**.*
    # => *E' o `E-83`: **um detector lexical ve termos, nao relacoes.*** => *o `tipo` e' o campo
    # **estruturado**, e a regra passa a ser sobre ele.*
    #
    # Regra: **`tipo` que COMECA por `RESOLVIDO`** => resolvido. Tudo o resto e' aberto, mesmo quando
    # a descricao diz «RESOLVIDO» no meio. *A descricao conta a historia; o `tipo` diz o estado.*
    _abertos, _resolvidos = [], []
    for b in blockers:
        if isinstance(b, dict):
            desc = b.get("descricao") or b.get("desc") or b.get("motivo") or ""
            tipo = (b.get("tipo") or "").upper()
            linha = "- **%s** _(%s)_ — %s" % (b.get("id") or b.get("code") or "?", b.get("tipo") or "?", desc)
        else:
            desc, tipo, linha = str(b), "", "- %s" % b
        alvo = _resolvidos if tipo.startswith("RESOLVIDO") else _abertos
        alvo.append(linha)
    if _abertos:
        lim.append("### Abertos (%d)" % len(_abertos))
        lim.append("")
        lim.extend(_abertos)
        lim.append("")
    if _resolvidos:
        lim.append("### Resolvidos ou corrigidos, mantidos por rastreabilidade (%d)" % len(_resolvidos))
        lim.append("")
        lim.append("*Estao aqui porque a **historia** e' parte do registo -- nao porque ainda bloqueiem.*")
        lim.append("")
        lim.extend(_resolvidos)
        lim.append("")
    lim += [
        "",
        "## Faixa histórica ausente: %s" % faixa_txt,
        "",
        "**%d IDs nunca foram registados neste índice.** A arqueologia do próprio WIP classifica-os" % sum(int(f[1]) - int(f[0]) + 1 for f in faixas),
        "como dívida declarada. **Não foram reconstruídos**: os que estão citados têm o alcance medido",
        "(o `E-1074`), e os que não estão citados continuam **sem texto**.",
        "",
        "## Artefactos declarados que NÃO existem neste checkout (%d)" % len(ausentes),
        "",
    ]
    if ausentes:
        for a in ausentes:
            lim.append("- `%s` — **ausente**" % a)
    else:
        lim.append("- _(nenhum)_")
    lim += [
        "",
        "## Por plataforma",
        "",
        "| Plataforma | SOURCE | BUILD | PACOTE | INSTALL | DEVICE_E2E | ASSINADO |",
        "| --- | --- | --- | --- | --- | --- | --- |",
        "| Windows/amd64 | sim | sim | portátil + instalador | **nao** | **nao** | **nao** |",
        "| Linux/amd64 | sim | sim | tarball instalável | **nao** | **nao** | **nao** |",
        "| macOS | sim | **nao** | **nao** | **nao** | **nao** | **nao** |",
        "| Android | sim | **nao** | **nao** | **nao** | **nao** | **nao** |",
        "| iOS | sim | **nao** | **nao** | **nao** | **nao** | **nao** |",
        "| Web/PWA | sim | sim | build | n/a | 3 motores (Playwright) | n/a |",
        "",
        "**Android**: `BLOCKED` no daemon do Gradle 8.13 (`E-1073`), com **8 hipóteses medidas**; o",
        "desbloqueio é **decisão do dono** (outro Gradle, ou outro host).",
        "",
    ]
    limitacoes = "\n".join(lim)

    # ---------- README_AUDITOR.md ----------
    rd = [
        "# README DO AUDITOR — HADES",
        "",
        "**Este pacote é gerado.** Não edite à mão: corrija a fonte e regenere com",
        "`python -X utf8 scripts/hades_audit_package.py`.",
        "",
        "## O que este repositório afirma, e onde",
        "",
        "| Alegação | Onde vive |",
        "| --- | --- |",
        "| **Veredito**: `%s` | `docs/mission/HADES_MASTER_STATE.json` (`release_state`) |" % release,
        "| **%d entradas** de evidência, de `E-001` a `E-%03d` | `docs/mission/HADES_EVIDENCE_INDEX.md` |" % (len(entradas), cabeca),
        "| **%d gates** declarados | `scripts/hades_local_gates.ps1` |" % len(gates),
        "| **%d bloqueios** | `LIMITACOES.md` (gerado do estado) |" % len(blockers),
        "| **Faixa histórica** %s | declarada no estado |" % faixa_txt,
        "",
        "## Como verificar, em três comandos",
        "",
        "```powershell",
        "cd <este checkout>",
        "powershell -NoProfile -ExecutionPolicy Bypass -File scripts/hades_local_gates.ps1",
        "python -X utf8 scripts/hades_audit_package.py --check",
        "```",
        "",
        "O segundo comando confirma que este pacote **bate com o repositório**. *Se bater, o pacote",
        "foi gerado deste estado; se não bater, alguém mudou o repositório sem regenerar.*",
        "",
        "## O que você **NÃO** vai encontrar aqui",
        "",
    ]
    for n in NAO_PROVA:
        rd.append("- " + n)
    rd += [
        "",
        "## O que este pacote **não** contém, e por que",
        "",
        "- **as %d evidências sem texto**: não foram reconstruídas (§FASE-7 item 2). O" % sum(int(f[1]) - int(f[0]) + 1 for f in faixas),
        "  `PARIDADE_EVIDENCIAS.csv` lista as que **existem**; as da faixa histórica não estão nele",
        "  porque não têm registo;",
        "- **os artefactos ausentes**: `LIMITACOES.md` nomeia-os um a um;",
        "- **o resultado dos gates**: `GATES.md` lista-os, não os executa. *Um pacote que dissesse",
        "  «passam» estaria a afirmar o que não mediu.*",
        "",
        "## Aviso sobre a natureza deste pacote",
        "",
        "**Ele é honesto sobre o que não sabe**, e essa é a única qualidade que um pacote de auditoria",
        "pode ter. *Um pacote que só lista o que existe é uma brochura; um que diz o que falta é um",
        "instrumento.*",
        "",
    ]
    readme = "\n".join(rd)

    return {
        "README_AUDITOR.md": readme,
        "PARIDADE_EVIDENCIAS.csv": paridade,
        "GATES.md": gates_md,
        "LIMITACOES.md": limitacoes,
        "ARTEFATOS.sha256": artefatos,
        # **O `FINDINGS.md` NAO e' gerado aqui**: ele e' gerado pelo `hades_findings_report.py` a
        # partir do `FINDINGS.json` capturado da API. *Se este script o gerasse, o `--check` deste
        # precisaria de rede* -- e um pacote de auditoria que so' se verifica com internet nao se
        # verifica num ambiente isolado. => o `--check` do pacote **exige a presenca** do ficheiro,
        # e o `--check` do findings valida o **conteudo**.
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Gera o pacote de auditoria independente.")
    ap.add_argument("--check", action="store_true", help="compara com o que esta' em disco")
    args = ap.parse_args()

    PACOTE.mkdir(parents=True, exist_ok=True)
    gerado = gerar()

    if not args.check:
        for nome, conteudo in gerado.items():
            (PACOTE / nome).write_text(conteudo, encoding="utf-8", newline="")
            print("  gerado: docs/audit/%s" % nome)
        return 0

    divergentes = []
    for nome, conteudo in gerado.items():
        p = PACOTE / nome
        if not p.is_file() or p.read_text(encoding="utf-8") != conteudo:
            divergentes.append(nome)
    # *A presenca do `FINDINGS.md` e' exigida por este `--check`, mas o CONTEUDO e' do
    # `hades_findings_report.py --check`.* => *dois checkers, duas responsabilidades, e a
    # fronteira e' a rede: este nao a usa.*
    if not (PACOTE / "FINDINGS.md").is_file():
        divergentes.append("FINDINGS.md (ausente; rode scripts/hades_findings_report.py)")
    if not (PACOTE / "FINDINGS.json").is_file():
        divergentes.append("FINDINGS.json (ausente; rode scripts/hades_findings_report.py --capturar)")
    if divergentes:
        print("  PACOTE DE AUDITORIA DIVERGENTE: %s" % ", ".join(divergentes))
        print("  Rode: python -X utf8 scripts/hades_audit_package.py")
        return 1
    print("  PACOTE DE AUDITORIA CONSISTENTE: os 5 artefactos batem com o repositorio.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
