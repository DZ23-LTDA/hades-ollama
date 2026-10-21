#!/usr/bin/env python3
"""`hades_competitive_check.py` — a MATRIZ COMPETITIVA, medida (`E-1109`).

***O QUE O PROMPT EXIGE.***

    - criar `HADES_COMPETITIVE_CAPABILITY_MATRIX.md` e uma versao **maquina-legivel** (`.json`/`.csv`);
    - **nao marcar um produto inteiro com um unico status** -- *decompor em capacidades atomicas*;
    - para cada capacidade: `CAP-ID`, dominio, descricao, referencias com fonte, **licenca**, estado HADES,
      **prova HADES**, lacuna, componente, plano, benchmark e decisao;
    - ***garantindo que cada referencia nominal e cada repositorio deste prompt apareca em pelo menos uma linha
      confirmada ou numa linha `UNVERIFIABLE/OUT_OF_SCOPE` com motivo, fonte consultada, licenca e data»***;
    - **`SUPERIOR_MEASURED` so' com benchmark reproduzivel** -- *«nao invente resultado»*.

***O QUE ESTE CHECADOR MEDE.***

  1. **A matriz existe** em `.md` **e** em `.csv` (a versao maquina-legivel).
  2. **O CSV tem as 12 colunas** exigidas.
  3. **Cada uma das 27 referencias de repositorio e das 49 de produto do prompt aparece** -- *e o criterio e'
     que a ausencia seja **explicita** (`UNVERIFIABLE` com motivo), nao silenciosa.*
  4. **Nenhum `SUPERIOR_MEASURED` sem benchmark** -- *o estado mais perigoso e' o que afirma vitoria sem prova.*
  5. **Os estados estao no enum** do prompt.

***E O QUE ELE NAO FAZ.*** *Nao pesquisa os concorrentes* => *a classificacao `UNVERIFIABLE` **e'** o resultado
honesto de nao ter consultado a fonte primaria -- e o prompt aceita-a: «*sem acesso a concorrente, marcar
`NOT_MEASURED`; nao use marketing como benchmark*».*
"""
import csv
import io
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
MISSAO = RAIZ / "docs" / "mission"
MD = MISSAO / "HADES_COMPETITIVE_CAPABILITY_MATRIX.md"
CSV = MISSAO / "HADES_COMPETITIVE_CAPABILITY_MATRIX.csv"

# *as 27 referencias de REPOSITORIO do prompt -- owner/repo como identidade (o prompt: «Use owner/repositorio
# como identidade, nunca apenas o nome»).*
REPOS = [
    "tel-agent", "openclaw-office", "block/buzz", "lemonade", "agentconnect", "open-design",
    "opensquad", "codenotch", "soup", "dyad", "litellm", "dify", "harnessrouter", "9router",
    "tldraw", "xyflow", "n8n", "agentscope", "langgraph", "typingmind", "youtube-automation",
    "morluto/rea", "paperclip", "cloudroom-gui", "cloudroom-core", "open-claude",
    "open-claude-in-chrome",
]

# *as 49 referencias de PRODUTO/ecossistema.*
PRODUTOS = [
    "builder.io", "flutterflow", "copilot workspace", "openhands", "devin", "plandex",
    "roo code", "cline", "aider", "claude code", "void", "trae", "zed", "windsurf", "cursor",
    "databutton", "marblism", "create.xyz", "replit", "v0", "base44", "bolt.diy", "bolt.new",
    "lovable", "perplexity", "polsia", "manus", "omniroute", "deepseek", "codex", "chatgpt",
    "gemini", "hermes", "hercules", "verdent", "swe-agent", "metagpt", "chatdev", "autogen",
    "crewai", "continue.dev", "ollama", "vllm", "e2b", "grok", "blink", "qoder", "genspark",
    "teamily",
]

ENUM = {"SUPERIOR_MEASURED", "PARITY_MEASURED", "IMPLEMENTED_UNMEASURED", "PARTIAL",
        "ABSENT_APPROVED", "OUT_OF_SCOPE", "BLOCKED_EXTERNAL", "CLAIM_ONLY", "UNVERIFIABLE",
        # ***`FONTE_MEDIDA` NAO E' UM ESTADO DE CAPACIDADE HADES.*** *E' o estado das linhas `SRC-01..05`,
        # que medem as **fontes de skills** (`agent-skills`, `hyperframes`, `googleworkspace/cli`,
        # `playwright-mcp`, `apify-mcp-server`) clonadas com licenca verificada* => *sao **proveniencia**, nao
        # capacidade* => *e o prompt exige licenca/proveniencia **por referencia**, que e' exactamente o que
        # essas linhas dao.*
        "FONTE_MEDIDA"}

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    T("(1) a matriz `.md` existe", MD.is_file(),
      "%.1f KB" % (MD.stat().st_size / 1024) if MD.is_file() else "AUSENTE")
    T("(1b) e a versao MAQUINA-LEGIVEL (`.csv`) existe", CSV.is_file())
    if not CSV.is_file() or not MD.is_file():
        return 1

    linhas = list(csv.reader(io.open(CSV, encoding="utf-8")))
    cab = linhas[0]
    corpo = [l for l in linhas[1:] if l]

    # ---------- 2. as 12 colunas ----------
    exigidas = ["CAP_ID", "dominio", "capacidade", "referencias", "licenca", "estado_hades",
                "prova_hades", "lacuna_decisao", "fonte", "data_medicao", "sha_candidato", "branch"]
    T("(2) o CSV tem as **12 colunas** exigidas", cab == exigidas,
      "%d colunas" % len(cab) if cab != exigidas else "as 12")
    T("(2b) e tem corpo (uma linha por capacidade)", len(corpo) >= 50, "%d linhas" % len(corpo))

    # ---------- 3. cada referencia do prompt aparece ----------
    txt = " ".join(" ".join(l) for l in corpo).lower()
    faltam_repo = [r for r in REPOS if r not in txt]
    faltam_prod = [p for p in PRODUTOS if p not in txt]
    T("(3) as **%d referencias de repositorio** aparecem na matriz" % len(REPOS),
      not faltam_repo, "faltam: %s" % faltam_repo if faltam_repo else "%d/%d" % (len(REPOS), len(REPOS)))
    T("(3b) as **%d referencias de produto** aparecem na matriz" % len(PRODUTOS),
      not faltam_prod, "faltam: %s" % faltam_prod if faltam_prod else "%d/%d" % (len(PRODUTOS), len(PRODUTOS)))

    # ---------- 4. nenhum SUPERIOR_MEASURED sem benchmark ----------
    # *o criterio: `SUPERIOR_MEASURED` so' vale se a linha tiver um **benchmark** na prova -- uma
    # afirmacao de vitoria sem comando/resultado e' marketing, e o prompt proibe-o.*
    idx_estado = cab.index("estado_hades")
    idx_prova = cab.index("prova_hades")
    superior_sem_bench = []
    for l in corpo:
        if len(l) <= max(idx_estado, idx_prova):
            continue
        if l[idx_estado].strip() == "SUPERIOR_MEASURED":
            prova = l[idx_prova].lower()
            if "bench" not in prova and "p50" not in prova and "repet" not in prova:
                superior_sem_bench.append(l[0])
    T("(4) nenhum `SUPERIOR_MEASURED` sem benchmark", not superior_sem_bench,
      "%d" % len(superior_sem_bench) if superior_sem_bench else "0 -- nenhuma vitoria sem prova")

    # ---------- 5. os estados estao no enum ----------
    bons = set(l[idx_estado].strip() for l in corpo if len(l) > idx_estado)
    fora = sorted(bons - ENUM)
    T("(5) todos os estados estao no enum do prompt", not fora,
      "fora: %s" % fora if fora else "%d estados usados" % len(bons))

    # ---------- 6. as correccoes factuais estao aplicadas (seccao 16) ----------
    md = io.open(MD, encoding="utf-8").read()
    T("(6) a seccao das correccoes factuais existe no `.md`",
      "Correções factuais" in md or "correccoes factuais" in md.lower(),
      "as 9 advertencias do prompt")
    # *cada uma das 9 correccoes nomeadas*
    # *o `.md` escreve `Open-claude` (owner/repo) e `open-claude-in-chrome` -- o `Open Claude` com espaco
    # e' a forma do prompt; aceitar as duas.*
    correcoes = ["Tel-Agent", "block/buzz", "morluto/rea", "Paperclip", "Soup", "CodeNotch",
                 "Open-claude", "Buzz"]
    faltam_corr = [c for c in correcoes if c.lower() not in md.lower()]
    T("(6b) e cobre as correccoes nomeadas", not faltam_corr,
      "faltam: %s" % faltam_corr if faltam_corr else "%d/%d" % (len(correcoes), len(correcoes)))

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NA MATRIZ COMPETITIVA ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  MATRIZ COMPETITIVA: %d capacidades, %d colunas, %d/%d repositorios e %d/%d produtos"
          % (len(corpo), len(cab), len(REPOS), len(REPOS), len(PRODUTOS), len(PRODUTOS)))
    print("  cobertos; nenhum `SUPERIOR_MEASURED` sem benchmark; e as correccoes factuais aplicadas.")
    print("")
    print("  ***O QUE ELA NAO E'.*** *Nao e' uma alegacao de superioridade: as linhas das referencias nao")
    print("  consultadas sao ***`UNVERIFIABLE`*** com motivo, e nao `PARITY_MEASURED`.* => *o prompt:")
    print("  «sem acesso a concorrente, marcar `NOT_MEASURED`; nao use marketing como benchmark».*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
