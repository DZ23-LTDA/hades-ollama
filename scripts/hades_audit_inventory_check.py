#!/usr/bin/env python3
"""`hades_audit_inventory_check.py` — o §ESTAGIO 1 (Auditoria total e inventario real) (`E-1100`).

***O QUE O §ESTAGIO 1 EXIGE.***

    - **reconciliar** documentos canonicos e historicos;
    - **remover instrucoes destrutivas de retomada**, como `reset --hard`, **ou marca-las
      inequivocamente como historicas e proibidas com WIP**;
    - **atualizar inventario de codigo, plataformas, workflows, pacotes e lacunas**;
    - **registar divergencias factuais sem reescrever historia**.

***O QUE ESTE CHECADOR MEDE.***

  1. **A matriz de plataformas** existe, cobre as **6 plataformas** (Windows, Linux, macOS, Android,
     iOS, Web/PWA) e os **7 eixos** (SOURCE, BUILD, PACKAGE, INSTALL, DEVICE_E2E, assinatura, hash).
  2. **ZERO instrucoes destrutivas sem aviso**: *todo documento com `reset --hard`/`git clean`/`force-push`
     tem de ter o aviso de **HISTORICO e PROIBIDO com WIP**.* => *e' a diferenca entre «o documento diz o
     que dizia» e «o documento manda apagar o trabalho de alguem».*
  3. **O inventario de workflows**: os **15** tem de estar **citados** na matriz -- *um workflow que existe
     e nao e' inventariado e' um workflow que ninguem sabe que corre.*
  4. **As divergencias factuais** estao registadas (o `HADES_STAGE_MAP.md` com a seccao do mapa).

***E O QUE ELE NAO FAZ.*** *Nao corre os workflows (o CI ja' o faz) nem reconstroi historia. => *o que ele faz
e' verificar que o **inventario** cobre o que existe e que **nenhuma instrucao destrutiva fica sem aviso**.*
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOCS = RAIZ / "docs"
MATRIZ = RAIZ / "docs" / "mission" / "HADES_PLATFORM_MATRIX.md"
WORKFLOWS = RAIZ / ".github" / "workflows"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    # ---------- 1. a matriz de plataformas ----------
    T("(1) o `HADES_PLATFORM_MATRIX.md` existe", MATRIZ.is_file())
    if MATRIZ.is_file():
        m = io.open(MATRIZ, encoding="utf-8").read()
        plataformas = ("Windows", "Linux", "macOS", "Android", "iOS", "Web/PWA")
        faltam = [p for p in plataformas if p not in m]
        T("(1b) cobre as 6 plataformas", not faltam,
          "faltam: %s" % ", ".join(faltam) if faltam else "as 6")
        eixos = ("SOURCE", "BUILD", "PACKAGE", "INSTALL", "DEVICE_E2E", "Assinatura", "Hash")
        faltam_e = [e for e in eixos if e not in m]
        T("(1c) cobre os 7 eixos", not faltam_e,
          "faltam: %s" % ", ".join(faltam_e) if faltam_e else "os 7")
        # *e a regra que distingue: cross-compile nao e' teste nativo*
        T("(1d) declara a regra «cross-compile nao e' teste nativo»",
          "cross-compile" in m.lower() or "cross-build" in m.lower(),
          "a distincao fica escrita")

    # ---------- 2. instrucoes destrutivas -- ZERO sem aviso ----------
    sem_aviso = []
    total_docs = 0
    for p in DOCS.rglob("*.md"):
        try:
            t = io.open(p, encoding="utf-8").read()
        except UnicodeDecodeError:
            continue
        if not re.search(r"reset --hard|git clean -[fdx]|force-push", t):
            continue
        total_docs += 1
        if not re.search(r"HIST[OÓ]RICO|proibid|PROIBID|nunca use|NUNCA USE|nao execute|NAO EXECUTE|E-1100", t):
            sem_aviso.append(str(p.relative_to(RAIZ)))
    T("(2) todos os documentos com instrucoes destrutivas tem AVISO",
      not sem_aviso, "%d sem aviso: %s" % (len(sem_aviso), ", ".join(sem_aviso[:3])) if sem_aviso
      else "%d documento(s), todos com aviso" % total_docs)

    # ---------- 3. os workflows estao inventariados ----------
    wfs = sorted(w.name for w in WORKFLOWS.glob("*.y*ml"))
    T("(3) ha' workflows", len(wfs) >= 10, "%d workflows" % len(wfs))
    if MATRIZ.is_file():
        m = io.open(MATRIZ, encoding="utf-8").read()
        fora = [w for w in wfs if w not in m]
        T("(3b) todos os workflows estao CITADOS na matriz", not fora,
          "fora: %s" % ", ".join(fora[:3]) if fora else "os %d" % len(wfs))

    # ---------- 4. as divergencias factuais estao registadas ----------
    mapa = RAIZ / "docs" / "mission" / "HADES_STAGE_MAP.md"
    T("(4) o `HADES_STAGE_MAP.md` (o mapa de divergencias) existe", mapa.is_file())
    if mapa.is_file():
        mp = io.open(mapa, encoding="utf-8").read()
        T("(4b) e declara as lacunas do escopo", "lacuna" in mp.lower() or "SEM dono" in mp,
          "as divergencias ficam escritas, sem reescrever historia")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 1 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 1 (Auditoria total): a matriz com 6 plataformas e 7 eixos, %d workflows citados," % len(wfs))
    print("  e ZERO instrucoes destrutivas sem o aviso de HISTORICO e PROIBIDO com WIP.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
