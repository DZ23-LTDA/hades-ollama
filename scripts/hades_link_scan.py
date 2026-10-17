"""MEDE as ligacoes docs -> artefactos em TODO o repositorio (`E-1055`).

Antes de estender o mapa `LIGACOES`, medir. *Nao se declara o que falta sem saber o que existe.*

O que este script responde:

  A. Que ficheiros de **artefacto** existem de facto (CSV, JSON, `.sha256`, `docs/parity/`,
     SBOM, manifests) -- medidos **em disco**, nao deduzidos do nome.
  B. Que documentos canonicos (`docs/mission/*.md`, `docs/*.md`) citam cada um.
  C. Quais **nao sao citados por ninguem** -- esses sao artefactos orfaos de ponteiro.
  D. Quais documentos falam de um artefacto que **nao existe** -- ponteiro quebrado.
"""
import io
import os
import re
import sys

RAIZ = sys.argv[1] if len(sys.argv) > 1 else r'C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama'

# ---------------------------------------------------------------- A. artefactos em disco
PADROES_ART = re.compile(
    r"(HADES_[A-Z0-9_]+\.(?:csv|json|sha256|txt|md)|HADES_T2_[A-Z]+\.md|"
    r"[A-Za-z0-9_.-]+\.sha256|SBOM[A-Za-z0-9_.-]*\.(?:csv|json))"
)

artefactos = {}
for sub in ("", "docs", "docs/mission", "docs/parity", "scripts", "_tmp"):
    d = os.path.join(RAIZ, sub) if sub else RAIZ
    if not os.path.isdir(d):
        continue
    for nome in os.listdir(d):
        p = os.path.join(d, nome)
        if not os.path.isfile(p):
            continue
        if PADROES_ART.fullmatch(nome) and not nome.startswith("HADES_COMPETITIVE"):
            rel = os.path.relpath(p, RAIZ).replace("\\", "/")
            artefactos[rel] = os.path.getsize(p)

# ---------------------------------------------------------------- B. documentos
docs = []
for sub in ("docs/mission", "docs", "docs/parity"):
    d = os.path.join(RAIZ, sub)
    if not os.path.isdir(d):
        continue
    for nome in sorted(os.listdir(d)):
        if nome.endswith(".md"):
            p = os.path.join(d, nome)
            rel = os.path.relpath(p, RAIZ).replace("\\", "/")
            docs.append((rel, p))

citados = {}
for rel_doc, caminho in docs:
    t = io.open(caminho, encoding="utf-8", errors="replace").read()
    for art in artefactos:
        if os.path.basename(art) in t:
            citados.setdefault(art, []).append(rel_doc)

print("  ARTEFACTOS EM DISCO: %d" % len(artefactos))
for art in sorted(artefactos, key=lambda x: -artefactos[x])[:16]:
    quem = citados.get(art, [])
    print("    %-44s %8d bytes  citado por %d doc(s)" % (art, artefactos[art], len(quem)))
if len(artefactos) > 16:
    print("    ... e mais %d" % (len(artefactos) - 16))

orfaos = [a for a in artefactos if a not in citados]
print("")
print("  *** ARTEFACTOS SEM PONTEIRO: %d ***" % len(orfaos))
for a in sorted(orfaos):
    print("    %-44s %8d bytes  ** ninguem cita **" % (a, artefactos[a]))

print("")
print("  DOCUMENTOS: %d em docs/mission, docs e docs/parity" % len(docs))
com_citacao = sum(1 for _, p in docs
                  if any(os.path.basename(a) in io.open(p, encoding="utf-8", errors="replace").read()
                         for a in artefactos))
print("  Documentos que citam pelo menos um artefacto: %d de %d" % (com_citacao, len(docs)))

# ---------------------------------------------------------------- D. ponteiros quebrados
print("")
print("  PONTEIROS QUEBRADOS (o doc nomeia um artefacto que nao existe em disco):")
quebrados = 0
for rel_doc, caminho in docs:
    t = io.open(caminho, encoding="utf-8", errors="replace").read()
    for m in set(re.findall(r"`?([A-Z][A-Z0-9_]{3,}\.(?:csv|json|sha256))`?", t)):
        existe_em_disco = any(os.path.basename(a) == m for a in artefactos)
        if not existe_em_disco and not os.path.exists(os.path.join(RAIZ, m)):
            print("    %-46s -> %s  ** NAO EXISTE **" % (rel_doc, m))
            quebrados += 1
if quebrados == 0:
    print("    (nenhum)")
