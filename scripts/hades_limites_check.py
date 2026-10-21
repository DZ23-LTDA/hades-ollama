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
    # ***E-522 (MEDIDO) — O E-515 ESTAVA ERRADO, E A RAZAO E' ESTRUTURAL.***
    #
    # No E-515 eu passei a ler **so' a coluna 4**, com o argumento «a clausula e' a coluna, nao a linha».
    # *MEDIDO*: as linhas do indice tem entre **7 e 25** colunas (7->54, 8->120, 9->79, 10->178, 11->30, 12->16,
    # 13->16, 14->13, 15->7, 16->5, 18->1, 25->1 linhas) - porque **o corpo da evidencia contem tabelas Markdown**, e os
    # `|` dessas tabelas **partem** a linha.
    #
    # ⇒ ***«a coluna 4» NAO EXISTE como conceito estavel.*** O meu checker nao estava a ler «a coluna errada»:
    #   ***a coluna nao existe.*** E por isso ele media **7** onde havia **~192** - *uma reducao de que eu me gabei.*
    #
    # ⇒ a leitura volta a ser **da LINHA**, como antes do E-515, mantendo o criterio de **primeira pessoa** do E-513 -
    #   *que era, e e', a unica das tres correccoes que estava certa.*
    if "LIMITES" not in linha:
        continue
    citados += 1
    clausula = linha[linha.index("LIMITES"):]
    if any(marcador in clausula for marcador in SEM_MEDICAO):
        sem_medicao += 1
        if "REMEDIDO EM" not in linha:
            abertos += 1
            lista_abertos.append(linha.split("|")[1].strip())

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
