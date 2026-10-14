"""hades_limites_check.py - o criterio FIXO para contar limites nao medidos.

***POR QUE ESTE FICHEIRO EXISTE*** (E-511/E-512):

Na rodada 245 eu contei os limites nao medidos **duas vezes, com listas de marcadores diferentes** - 30 na primeira,
18 na segunda - e quase tratei as duas como a mesma medicao. *O erro nao estava em medir: estava em **comparar
medicoes que nao sao comparaveis**.*

⇒ o criterio passa a estar **num so' lugar, escrito e versionado**. Quem contar, conta com este.

***E a distincao que importa***: um limite **fechado** continua com a palavra «nao medido» na coluna LIMITES - porque a
coluna e' **historica**. O que muda e' a marca **`REMEDIDO EM`**, que cita a evidencia que o fechou.

⇒ o que este checker mede, entao, sao **tres** numeros diferentes:

  1. LIMITES CITADOS    - quantas evidencias tem clausula de limite (todas tem algo)
  2. SEM MEDICAO        - os que dizem «nao medido» (o texto historico)
  3. AINDA ABERTOS      - os que dizem «nao medido» **e nao tem** `REMEDIDO EM`  <- **este e' o numero que conta**

*Confundir (2) com (3) foi exactamente o que eu fiz na rodada 245.*
"""

import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"

# ***O CRITERIO - fixo, aqui, e so' aqui.***
#
# E-513 (MEDIDO): a lista anterior era LARGA de mais, e o caso concreto apareceu no `E-285` - cuja clausula diz
# «**LIMITES DECLARADOS** ... `mattn/go-sqlite3` exige CGO ... `jackc/pgx/v5` exige PostgreSQL rodando ⇒ a persistencia
# em arquivo com escrita atomica e' DECISAO JUSTIFICADA». ***Isso NAO e' um limite por medir: sao os limites DO
# ARTEFACTO*** - *o que ele nao assume, e por que.*
#
# ⇒ a mesma palavra, **dois sentidos**: *«o que eu nao medi»* x *«o que o artefacto nao depende»*.
# O extractor so' pode medir o primeiro, e para isso tem de exigir a **primeira pessoa**: nao medi / nao executei /
# nao verifiquei. *Nao basta a palavra «limite» nem «sem teste» no meio de uma frase sobre o produto.*
SEM_MEDICAO = (
    "nao medi", "não medi", "nao medid", "não medid",
    "nao executei", "não executei", "nao executados", "não executados",
    "nao verifiquei", "não verifiquei",
    "nao foi medido", "não foi medido",
    "nao testado", "não testado", "nao testad",
)

linhas = INDICE.read_text(encoding="utf-8").split("\n")
citados = sem_medicao = abertos = 0
lista_abertos = []
for linha in linhas:
    if not linha.startswith("| E-"):
        continue
    colunas = linha.split("|")
    if len(colunas) < 5:
        continue
    # ***E-515 (MEDIDO): o marcador tem de ser procurado NA CLAUSULA, nao na linha toda.***
    # *O `E-325`, o `E-326` e o `E-345` apareciam como «abertos» porque o `nao medid` estava **fora** da clausula de
    # limites - e a linha inteira era varrida.* ⇒ ***o instrumento procurava no sitio errado da linha.***
    # A clausula de limites vive na **coluna 4** (o corpo da evidencia).
    corpo = colunas[4]
    if "LIMITES" not in corpo:
        continue
    citados += 1
    clausula = corpo[corpo.index("LIMITES"):]
    if any(marcador in clausula for marcador in SEM_MEDICAO):
        sem_medicao += 1
        if "REMEDIDO EM" not in corpo:
            abertos += 1
            lista_abertos.append(colunas[1].strip())

print("  LIMITES CITADOS  : %d  (*evidencias que tem clausula de limite*)"
      % citados)
print("  SEM MEDICAO      : %d  (*o texto historico ainda diz «nao medido»*)" % sem_medicao)
print("  **AINDA ABERTOS**: %d  <- ***este e' o numero que conta***" % abertos)
print()
for identificador in lista_abertos:
    print("     %s" % identificador)
print()
print("  *Confundir SEM MEDICAO com AINDA ABERTOS foi o erro da rodada 245.*")
print("  *O criterio vive aqui, e nao numa lista escolhida por corrida.*")

sys.exit(0 if abertos == 0 else 1)
