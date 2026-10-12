"""Varredura de escopo por organizacao na camada de ROTA (instrumento calibrado).

PRIMEIRA VERSAO (errada como instrumento): contava handlers que nao mencionavam escopo TEXTUALMENTE e acusava 103
de 182 rotas. Ao LER, descobriu-se que o padrao do projeto delega o escopo a helpers `*ForRequest`
(ex.: `builderForRequest` -> `GetForOrganization(id, companyOrganizationID(c))`) e que `companyForRequest` ja
iguala "nao existe" e "outro tenant" (E-190). O numero era um LIMITE SUPERIOR inutil: acusava o padrao correto.

ESTA VERSAO resolve o padrao:
  1. handler da rota;
  2. se ele chama `a.<algo>ForRequest(c)` (ou `...ForOrganization`), segue UM salto e olha o corpo do helper;
  3. so acusa quem, depois do salto, nao compara organizacao de NENHUMA forma.

O que ela NAO prova: que a comparacao esta correta (ordem, mensagem, mesclagem de recusa). Isso e' leitura humana -
e foi exatamente onde E-190, E-198, E-252 e E-253 encontraram defeito real.
"""

import re
from pathlib import Path

RAIZ = Path("server")
# TRES padroes de escopo existem neste projeto (descobertos LENDO, nao supondo):
#   1. helpers *ForRequest(c)              -> ex.: builderForRequest, companyForRequest
#   2. comparacao explicita de organizacao -> companyOrganizationID(c), OrganizationID
#   3. runtime ja vinculado ao tenant      -> a.scopedRuntime(c)
# A primeira versao so conhecia (1) e (2) e acusava 38 de 181; os 38 eram, em boa parte, padrao (3).
SINAIS_ESCOPO = re.compile(
    r"ForOrganization|OrganizationID|organizationID|organization_id|requireOrg|companyOrganizationID|"
    r"tenantID|LocalOrganizationID|ForRequest|scopedRuntime|RuntimeForOrganization|scopedCompany",
)
CHAMADA_HELPER = re.compile(r"\.([A-Za-z0-9_]+ForRequest)\(")


def funcoes(texto: str):
    for m in re.finditer(r"(?m)^func (?:\([^)]*\) )?([A-Za-z0-9_]+)\([^\n]*\{\n", texto):
        nome = m.group(1)
        inicio = m.end()
        profundidade = 1
        i = inicio
        while i < len(texto) and profundidade > 0:
            if texto[i] == "{":
                profundidade += 1
            elif texto[i] == "}":
                profundidade -= 1
            i += 1
        yield nome, texto[inicio:i]


corpos = {}
for caminho in sorted(RAIZ.rglob("*.go")):
    if caminho.name.endswith("_test.go"):
        continue
    for nome, corpo in funcoes(caminho.read_text(encoding="utf-8", errors="replace")):
        corpos.setdefault(nome, (str(caminho), corpo))

# Ancora na CHAMADA do metodo HTTP: .POST("/rota", handler). A primeira versao pegava o proprio "POST"
# como se fosse o handler e acusava 184 de 184 - instrumento pior que o anterior.
ROTA = re.compile(
    r'\.(?:POST|GET|PUT|DELETE|PATCH|HEAD|OPTIONS|Handle|HandleFunc)\(\s*'
    r'"(/[A-Za-z0-9_./:*-]*:[A-Za-z0-9_]+[A-Za-z0-9_./:*-]*)"\s*,\s*([A-Za-z0-9_.]+)'
)
rotas = set()
for caminho in sorted(RAIZ.rglob("*.go")):
    if caminho.name.endswith("_test.go"):
        continue
    texto = caminho.read_text(encoding="utf-8", errors="replace")
    for m in ROTA.finditer(texto):
        rotas.add((m.group(1), m.group(2).split(".")[-1]))


def escopado(nome: str, visitados: set) -> bool:
    if nome in visitados:
        return False
    visitados.add(nome)
    arquivo, corpo = corpos.get(nome, ("", ""))
    if SINAIS_ESCOPO.search(corpo):
        return True
    for m in CHAMADA_HELPER.finditer(corpo):
        if escopado(m.group(1), visitados):
            return True
    return False


sem_escopo = []
for padrao, handler in sorted(rotas):
    if handler == "?":
        continue
    if not escopado(handler, set()):
        sem_escopo.append((padrao, handler, corpos.get(handler, ("?", ""))[0]))

print("rotas com parametro de caminho:", len(rotas))
print("handlers SEM escopo apos resolver o padrao *ForRequest:", len(sem_escopo))
print()
for padrao, handler, arquivo in sem_escopo:
    print("  %-56s %-32s %s" % (padrao[:56], handler[:32], arquivo))
