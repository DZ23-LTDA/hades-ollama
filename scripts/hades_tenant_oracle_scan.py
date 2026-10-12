"""Varredura da classe 'oraculo por ORDEM de checagem' em superficies multi-tenant.

Classe de defeito (E-198 na fila, E-252 no OAuth): a funcao compara algo que o ATACANTE controla (id, provider,
status) e devolve erros DIFERENTES antes de comparar a organizacao. Resultado: um intruso distingue
"existe, mas e de outro tenant" de "nao existe" e enumera recursos alheios.

O scan procura, por funcao que recebe organizationID:
  1. a linha da comparacao de tenant (`... != organizationID` ou `OrganizationID != organizationID`);
  2. returns com erro ANTES dessa linha, quando o erro menciona algo do recurso (provider/status/state/expired).

Saida: candidatos para LEITURA HUMANA. Nao e veredito: cada candidato precisa ser lido (a ordem pode ser legitima
quando o dado comparado NAO e do atacante).
"""

import re
from pathlib import Path

RAIZ = Path(".")
PASTAS = ("internal/agent", "server")

# Sinais de que o erro revela algo do RECURSO (e nao da organizacao).
SINAIS_RECURSO = re.compile(
    r"provider mismatch|status|state|expired|revoked|already|conflict|not published|mismatch|invalid",
    re.IGNORECASE,
)
COMPARA_TENANT = re.compile(r"(OrganizationID|organizationID|organization_id)\s*!=\s*|!=\s*(organizationID|OrganizationID)")


def funcoes(texto: str):
    """Devolve (assinatura, corpo) por funcao de topo, de forma simples e conservadora."""
    linhas = texto.split("\n")
    i = 0
    while i < len(linhas):
        m = re.match(r"^func (\([^)]*\) )?([A-Za-z0-9_]+)\(", linhas[i])
        if m:
            assinatura = linhas[i]
            corpo = []
            profundidade = 0
            j = i
            while j < len(linhas):
                corpo.append(linhas[j])
                profundidade += linhas[j].count("{") - linhas[j].count("}")
                if j > i and profundidade <= 0:
                    break
                j += 1
            yield assinatura, "\n".join(corpo)
            i = j + 1
        else:
            i += 1


def analisar(assinatura: str, corpo: str, arquivo: str) -> dict | None:
    if "organizationID" not in assinatura and "organization_id" not in assinatura:
        return None
    linhas = corpo.split("\n")
    linha_tenant = None
    for i, linha in enumerate(linhas):
        if COMPARA_TENANT.search(linha):
            linha_tenant = i
            break
    if linha_tenant is None:
        return None
    anteriores = []
    for i in range(linha_tenant):
        linha = linhas[i]
        if "return" in linha and SINAIS_RECURSO.search(linha) and "errors.New" in linha + "\n".join(linhas[i : i + 3]):
            anteriores.append((i + 1, linha.strip()))
    if not anteriores:
        return None
    nome = re.search(r"\) ([A-Za-z0-9_]+)\(", assinatura)
    return {
        "arquivo": arquivo,
        "funcao": nome.group(1) if nome else assinatura.strip(),
        "linha_tenant": linha_tenant + 1,
        "erros_antes": anteriores,
    }


candidatos = []
for pasta in PASTAS:
    for caminho in sorted((RAIZ / pasta).rglob("*.go")):
        if caminho.name.endswith("_test.go"):
            continue
        texto = caminho.read_text(encoding="utf-8", errors="replace")
        for assinatura, corpo in funcoes(texto):
            achado = analisar(assinatura, corpo, str(caminho))
            if achado:
                candidatos.append(achado)

print("candidatos (erro de RECURSO devolvido ANTES da comparacao de tenant):", len(candidatos))
for c in candidatos:
    print("\n  %s :: %s (comparacao de tenant na linha %d)" % (c["arquivo"], c["funcao"], c["linha_tenant"]))
    for numero, erro in c["erros_antes"]:
        print("      L%s  %s" % (numero, erro[:120]))
