#!/usr/bin/env python3
"""HADES — ESCOPO DE TENANT NO ARMAZENAMENTO (E-338).

POR QUE ESTE INSTRUMENTO EXISTE
-------------------------------
E-337 mediu a causa-raiz: `consentScope("")` devolve `LocalOrganizationID` e `sanitizeIDComponent("")` devolve `geral`.
Consequência: um armazenamento que monta caminho por organização **sem validar o valor CRU** grava o dado "sem tenant"
num arquivo **COMPARTILHADO** — *um dado sem tenant vira dado de todos*.

Depois da correção (E-337), a medição desta rodada deu: **4 arquivos com guarda e 18 sem**. Dívida declarada em
documento envelhece e some; aqui ela vira **catraca**:

  - um arquivo de armazenamento precisa de `requireOrganization(` **ou** de uma **dispensa declarada com motivo**;
  - dispensa **sem motivo** é violação;
  - arquivo **novo** sem guarda e sem dispensa **reprova** — *a lista de pendências pode encolher, nunca crescer em
    silêncio*;
  - o relatório lista as pendências como **`NAO_MEDIDO`**, nunca como "ok": ausência de auditoria não é aprovação.

LIMITE DECLARADO: o instrumento audita a **presença da guarda no arquivo**, não a **correção de cada caminho de
escrita**. Um arquivo com guarda em um ponto e outro caminho sem guarda passaria — por isso a varredura de
comportamento (E-337) continua sendo a prova, e esta é a **catraca**.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DIRETORIO = RAIZ / "internal/agent"

# Dispensas DECLARADAS: arquivo -> motivo. Toda dispensa aqui é visível e datada; o objetivo é encolher a lista.
DISPENSAS = {
    "benchmark_suite.go": "o componente vem de NOME DE TAREFA, não de organização: não há escopo de tenant a validar.",
    "consent_registry.go": "CONTRATO TESTADO: a organizacao vazia e' normalizada para a organizacao local por decisao do repositorio (testes TestConsentRegistryFailsClosedForUnknownContact, TestWhatsAppGatewaySendIsBlockedWithoutConsent e TestWhatsAppOutboundRequiresAllowlistAndApproval exigem a normalizacao). Guarda de escopo cru seria contradizer contrato - E-340.",
    "scheduling_crossprocess.go": "monta o caminho do LOCK; a organização já foi validada pelo chamador (requireOrganization em scheduling.go).",
    "marketplace_sync.go": "NAO_MEDIDO: monta caminho de armazenamento por organizacao e ainda NAO foi varrido por comportamento (falta teste que ataque a API com organizacao vazia). Pendencia declarada em E-338 — nao e' aprovacao.",
}


def arquivos_de_armazenamento(diretorio: Path = DIRETORIO) -> dict[str, list[str]]:
    """Devolve, por arquivo não-teste, as linhas que montam caminho com sanitizeIDComponent."""
    encontrados: dict[str, list[str]] = {}
    if not diretorio.is_dir():
        return encontrados
    for caminho in sorted(diretorio.glob("*.go")):
        if caminho.name.endswith("_test.go"):
            continue
        linhas = caminho.read_text(encoding="utf-8", errors="replace").split("\n")
        for numero, linha in enumerate(linhas, 1):
            if "sanitizeIDComponent(" not in linha:
                continue
            if not any(pista in linha for pista in ("filepath.Join", "caminho", "path", "lock", "Root")):
                continue
            encontrados.setdefault(caminho.name, []).append(f"{caminho.name}:{numero}")
    return encontrados


# Checagem CRUA de organizacao vazia (segundo caminho reconhecido como guarda correta).
CHECAGEM_CRUA = re.compile(
    r'(organizationID|companyID|org|organizacao)\s*==\s*""',
)


def tem_guarda_reconhecida(texto: str) -> bool:
    """Reconhece o helper OU a checagem crua de vazio (antes de sanitizar).

    POR QUE DOIS CAMINHOS (E-342): `marketplace_sync.Sync` e `company.Create` validam `organizationID == ""` no valor
    cru — correto — e a catraca os acusava porque so' conhecia `requireOrganization(`. ***Instrumento que reconhece um
    unico padrao transforma codigo certo em divida.***
    """
    if "requireOrganization(" in texto:
        return True
    return bool(CHECAGEM_CRUA.search(texto))


def auditar(diretorio: Path = DIRETORIO) -> tuple[list[str], list[str]]:
    """Devolve (violações, pendências não medidas)."""
    violacoes: list[str] = []
    pendentes: list[str] = []
    for nome, locais in arquivos_de_armazenamento(diretorio).items():
        caminho = diretorio / nome
        texto = caminho.read_text(encoding="utf-8", errors="replace")
        tem_guarda = tem_guarda_reconhecida(texto)
        dispensa = DISPENSAS.get(nome)
        if dispensa is not None and not dispensa.strip():
            violacoes.append(f"{nome}: dispensa SEM MOTIVO (dispensa sem justificativa é aceitação silenciosa)")
            continue
        if not tem_guarda and dispensa is None:
            violacoes.append(
                f"{nome}: monta caminho por organização SEM `requireOrganization` e SEM dispensa declarada "
                f"({len(locais)} local(is): {', '.join(locais)})"
            )
            continue
        if not tem_guarda:
            pendentes.append(f"{nome} — dispensa: {dispensa}")
    return violacoes, pendentes


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Catraca de escopo de tenant no armazenamento (E-338).")
    parser.add_argument("--check", action="store_true", help="falha (exit 1) se houver violação")
    args = parser.parse_args(argv)

    violacoes, pendentes = auditar()
    if pendentes:
        print(f"PENDÊNCIAS DECLARADAS (NAO_MEDIDO, não 'ok'): {len(pendentes)}")
        for item in pendentes:
            print(f"  - {item}")
    if violacoes:
        for violacao in violacoes:
            print(f"VIOLACAO: {violacao}")
        print(f"\n{len(violacoes)} violação(ões): armazenamento sem guarda e sem dispensa motivada.")
        return 1
    print("ESCOPO DE TENANT: todo armazenamento tem guarda crua ou dispensa DECLARADA com motivo.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
