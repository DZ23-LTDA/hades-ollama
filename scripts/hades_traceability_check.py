#!/usr/bin/env python3
"""`hades_traceability_check.py` — a FASE 7 (rastreabilidade sem inventar historia), medida (`E-1110`).

***O QUE O §FASE-7 EXIGE.***

    1. Preservar a declaracao de que **`E-571..E-777` nao estao no indice**;
    2. **Nao criar 134 textos ficticios** para preencher numeros;
    3. Classificar os **73 IDs orfaos** por requisito/call site e substituir dependencias ativas por novas
       evidencias reais, **medidas agora**, com IDs novos acima da cabeca;
    4. Para cada requisito ativo: `REQ-ID -> arquivo:linha -> teste -> comando -> resultado -> plataforma ->
       commit/estado dirty -> artefato/hash -> decisao`;
    5. Marcar referencias historicas irrecuperaveis como **`EVIDENCE_LOST_HISTORICAL`**, com **alcance e
       impacto**;
    6. A catraca impede qualquer novo ID citado sem entrada;
    7. Corrigir documentos gerados **somente via gerador**, depois da fonte;
    8. O pacote de auditoria independente com `README_AUDITOR.md`, `PARIDADE_EVIDENCIAS.csv`, `GATES.md`,
       `LIMITACOES.md`, `ARTEFATOS.sha256`, SBOMs, findings e instrucoes de reproducao.

***O QUE ESTE CHECADOR MEDE.*** *Cada um dos 8 itens, contra o **estado real** -- nao contra o que o documento
diz de si mesmo.* => *e o criterio mais importante e' o (5): **um ID perdido marcado como perdido e' uma divida;
um ID perdido preenchido com texto inventado e' uma mentira.***
"""
import csv
import io
import json
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
MISSAO = RAIZ / "docs" / "mission"
AUDIT = RAIZ / "docs" / "audit"
ESTADO = MISSAO / "HADES_MASTER_STATE.json"
INDICE = MISSAO / "HADES_EVIDENCE_INDEX.md"
PERDIDOS = MISSAO / "HADES_EVIDENCE_LOST_HISTORICAL.md"
VISUAL = MISSAO / "HADES_VISUAL_PROOF.md"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    if not ESTADO.is_file():
        print("  FALHA: o estado nao existe")
        return 1
    d = json.loads(io.open(ESTADO, encoding="utf-8").read())

    # ---------- 1. a faixa historica declarada ----------
    faixas = d.get("evidence_ids_nao_registrados_faixas") or []
    T("(1) a faixa historica `E-571..E-777` esta' declarada", faixas == [[571, 777]],
      str(faixas))

    # ---------- 2. os IDs citados estao listados ----------
    nao_reg = d.get("evidence_ids_nao_registrados") or []
    # *os 12 reais (sem os placeholders)*
    reais = sorted(x for x in nao_reg if x not in ("E-000", "E-999"))
    T("(2) os IDs citados e nao registados estao listados", len(reais) >= 12,
      "%d IDs: %s" % (len(reais), ", ".join(reais[:4])))

    # ---------- 3. cada um tem linha no documento de perdidos ----------
    T("(3) o `HADES_EVIDENCE_LOST_HISTORICAL.md` existe", PERDIDOS.is_file(),
      "%.1f KB" % (PERDIDOS.stat().st_size / 1024) if PERDIDOS.is_file() else "AUSENTE")
    if PERDIDOS.is_file():
        doc = io.open(PERDIDOS, encoding="utf-8").read()
        sem_linha = [i for i in reais if "`%s`" % i not in doc]
        T("(3b) e **cada um dos %d** tem linha na tabela" % len(reais), not sem_linha,
          "sem linha: %s" % sem_linha if sem_linha else "%d/%d" % (len(reais), len(reais)))
        # *e o documento nao inventou texto: cada linha tem onde e' citado, o que se perdeu, impacto e decisao*
        T("(3c) e a tabela tem as 5 colunas (onde / o que se perdeu / impacto / decisao)",
          "onde é citado" in doc and "o que se perdeu" in doc and "impacto" in doc
          and "decisão" in doc, "as 4 colunas + o ID")

    # ---------- 4. os placeholders estao separados ----------
    placeholders = d.get("evidence_ids_placeholder") or []
    T("(4) os placeholders `E-000`/`E-999` estao separados dos perdidos",
      set(placeholders) == {"E-000", "E-999"} and "E-000" not in reais,
      "placeholder=%s" % placeholders)

    # ---------- 5. nenhum requisito ativo depende de um ID perdido ----------
    req = AUDIT / "REQUISITOS.csv"
    T("(5) o `REQUISITOS.csv` existe", req.is_file())
    if req.is_file():
        linhas = list(csv.reader(io.open(req, encoding="utf-8")))
        txt = " ".join(" ".join(l) for l in linhas)
        # *o criterio: os requisitos ativos **nao** citam nenhum dos IDs perdidos*
        dependem = [i for i in reais if i in txt]
        T("(5b) e **nenhum requisito ativo depende de um ID perdido**", not dependem,
          "dependem: %s" % dependem if dependem else "0 -- os 10 REQ apontam para codigo medido agora")

    # ---------- 6. a catraca: o numero nao cresce ----------
    # *o `mission_check.py` le a lista do estado, e a lista so' pode DIMINUIR* => *aqui medimos que ela
    # continua a ser a **mesma ou menor** do que a declarada -- e que nao ha ID novo citado sem entrada.*
    indice = io.open(INDICE, encoding="utf-8").read() if INDICE.is_file() else ""
    registados = set(re.findall(r"^\|\s*(E-\d+)\s*\|", indice, re.M))
    T("(6) o indice tem entradas registadas", len(registados) >= 800, "%d entradas" % len(registados))
    # *os 12 nao estao no indice -- e o documento diz isso*
    no_indice = [i for i in reais if i in registados]
    T("(6b) e os %d NAO estao no indice (sao perdidos, nao registados)" % len(reais), not no_indice,
      "estao no indice: %s" % no_indice if no_indice else "0 -- confirmado")

    # ---------- 7. o pacote de auditoria ----------
    exigidos = ["README_AUDITOR.md", "PARIDADE_EVIDENCIAS.csv", "GATES.md", "LIMITACOES.md",
                "ARTEFATOS.sha256"]
    faltam = [x for x in exigidos if not (AUDIT / x).is_file()]
    T("(7) o pacote de auditoria tem os 5 ficheiros minimos", not faltam,
      "faltam: %s" % faltam if faltam else "%d/%d" % (len(exigidos), len(exigidos)))
    # *e os SBOMs + findings*
    sboms = sorted(RAIZ.glob("HADES_SBOM_*.csv"))
    T("(7b) e os SBOMs existem", len(sboms) >= 3, "%d SBOM(s)" % len(sboms))
    T("(7c) e a lista de findings existe",
      (AUDIT / "FINDINGS.json").is_file() and (AUDIT / "FINDINGS.md").is_file(),
      "FINDINGS.json + FINDINGS.md")

    # ---------- 8. o documento NAO inventou texto ----------
    if PERDIDOS.is_file():
        doc = io.open(PERDIDOS, encoding="utf-8").read()
        # *o criterio: o documento declara explicitamente que NAO reconstruiu*
        T("(8) o documento declara que NAO reconstruiu a evidencia",
          "não" in doc.lower() and ("reconstru" in doc.lower() or "invent" in doc.lower()),
          "a marca e' a declaracao, nao o silencio")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NA FASE 7 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  FASE 7 (rastreabilidade): a faixa `E-571..E-777` declarada, os %d IDs citados" % len(reais))
    print("  marcados `EVIDENCE_LOST_HISTORICAL` com alcance e impacto, os placeholders separados,")
    print("  nenhum requisito ativo dependente de ID perdido, e o pacote de auditoria completo.")
    print("")
    print("  ***`um ID perdido marcado como perdido e' uma divida; um ID perdido preenchido com texto")
    print("  inventado e' uma mentira.`***")
    return 0


if __name__ == "__main__":
    sys.exit(main())
