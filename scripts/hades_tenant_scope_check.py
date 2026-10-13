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
    "scheduling_hours.go": "IDENTIFICADOR de tempo/campos (dia/horário), não da organização; isolamento por organização é por arquivo (medido em E-351).",
    "scheduling_breaks.go": "IDENTIFICADOR de tempo/campos (semana/dia), não da organização (medido em E-351).",
    "supervisor_incidents.go": "IDENTIFICADOR de ASSUNTO do incidente: risco de colisão, NÃO de vazamento entre tenants (medido em E-351).",
    "scheduling_crossprocess.go": "monta o caminho do LOCK; a organização já foi validada pelo chamador (requireOrganization em scheduling.go).",
    "marketplace_sync.go": "NAO_MEDIDO: monta caminho de armazenamento por organizacao e ainda NAO foi varrido por comportamento (falta teste que ataque a API com organizacao vazia). Pendencia declarada em E-338 — nao e' aprovacao.",
}


def _chave_relativa(raiz: Path, caminho: Path) -> str:
    """Chave = caminho RELATIVO (E-382 / ADV-56/4).

    **MEDIDO (3ª auditoria independente)**: a chave era o **basename** ✗, e isso produzia dois defeitos: dois `util.go`
    em pastas distintas viravam **uma chave só** (contagem inflada com a mesma localização) e — pior — a leitura
    seguinte usava `(diretorio / basename)`, que **não existe** se o arquivo estiver em subpasta ⇒ `FileNotFoundError`
    ⇒ *abortava a varredura e a violação REAL de outro arquivo deixava de ser reportada*. ***Instrumento que morre no
    primeiro arquivo estranho aprova tudo o que não chegou a ver.***
    """
    try:
        return caminho.relative_to(raiz).as_posix()
    except ValueError:
        return caminho.name


def arquivos_de_armazenamento(diretorio: Path = DIRETORIO) -> dict[str, list[str]]:
    """Devolve, por arquivo não-teste, as linhas que montam caminho com sanitizeIDComponent."""
    encontrados: dict[str, list[str]] = {}
    if not diretorio.is_dir():
        return encontrados
    for caminho in sorted(diretorio.rglob("*.go")):
        if caminho.name.endswith("_test.go"):
            continue
        try:
            linhas = caminho.read_text(encoding="utf-8", errors="replace").split("\n")
        except OSError as erro:  # E-382 / ADV-56/4: um arquivo ilegível NÃO pode abortar a varredura
            encontrados.setdefault(_chave_relativa(diretorio, caminho), []).append(
                "%s: arquivo ilegível (%s)" % (_chave_relativa(diretorio, caminho), erro))
            continue
        for numero, linha in enumerate(linhas, 1):
            if "sanitizeIDComponent(" not in linha:
                continue
            if not any(pista in linha for pista in ("filepath.Join", "caminho", "path", "lock", "Root")):
                continue
            chave = _chave_relativa(diretorio, caminho)
            encontrados.setdefault(chave, []).append(f"{chave}:{numero}")
    return encontrados


# Checagem CRUA de organizacao vazia (segundo caminho reconhecido como guarda correta).
CHECAGEM_CRUA = re.compile(
    r'(organizationID|companyID|org|organizacao)\s*==\s*""',
)


def _sem_comentarios(texto: str) -> str:
    """Remove comentários (`//` e `/* */`) respeitando STRINGS (E-381 / ADV-56/3).

    **MEDIDO com violação injetada (3ª auditoria independente)**: a versão anterior cortava a linha no primeiro `//`,
    o que errava nas DUAS direções ✗:
      - um **comentário de bloco** `/* requireOrganization(org) … */` **aprovava** um arquivo sem guarda ✗;
      - uma **string** com a menção (`const doc = "chame requireOrganization(org)"`) também **aprovava** ✗;
      - e o inverso: `//` **dentro de uma string** (`if url != "https://x" && requireOrganization(org) != nil {`)
        **apagava uma guarda real** ⇒ **falso positivo** ✗.

    ***Guarda que se satisfaz com uma menção — em comentário de qualquer sintaxe ou em string — é pior que guarda
    ausente: ela dá a impressão de que alguém verificou.*** Agora as strings são respeitadas (só as aspas ficam: o
    CONTEÚDO de uma string não é código) e os dois estilos de comentário são removidos.
    """
    saida: list[str] = []
    em_bloco = False
    for linha in texto.split("\n"):
        resultado: list[str] = []
        i = 0
        em_string = None
        while i < len(linha):
            if em_bloco:
                fim = linha.find("*/", i)
                if fim < 0:
                    i = len(linha)
                    continue
                em_bloco = False
                i = fim + 2
                continue
            caractere = linha[i]
            if em_string is not None:
                # STRING não é código: o CONTEÚDO não pode satisfazer a guarda (MEDIDO: `const doc = "chame
                # requireOrganization(org)"` aprovava um arquivo SEM guarda ✗). Mantemos só as aspas, para não
                # colar tokens; e o escape é respeitado para não terminar a string cedo.
                if caractere == "\\" and i + 1 < len(linha):
                    i += 2
                    continue
                if caractere == em_string:
                    resultado.append(caractere)
                    em_string = None
                i += 1
                continue
            if caractere in ('"', "'", "`"):
                em_string = caractere
                resultado.append(caractere)
                i += 1
                continue
            if linha.startswith("//", i):
                break
            if linha.startswith("/*", i):
                em_bloco = True
                i += 2
                continue
            resultado.append(caractere)
            i += 1
        saida.append("".join(resultado))
    return "\n".join(saida)


def tem_guarda_reconhecida(texto: str) -> bool:
    """Reconhece o helper OU a checagem crua de vazio (antes de sanitizar).

    POR QUE DOIS CAMINHOS (E-342): `marketplace_sync.Sync` e `company.Create` validam `organizationID == ""` no valor
    cru — correto — e a catraca os acusava porque so' conhecia `requireOrganization(`. ***Instrumento que reconhece um
    unico padrao transforma codigo certo em divida.***

    E-376: a procura agora é feita **fora dos comentários** — medido que uma menção em comentário aprovava um arquivo
    sem guarda nenhuma ✗.
    """
    codigo = _sem_comentarios(texto)
    if "requireOrganization(" in codigo:
        return True
    return bool(CHECAGEM_CRUA.search(codigo))


def auditar(diretorio: Path = DIRETORIO) -> tuple[list[str], list[str]]:
    """Devolve (violações, pendências não medidas)."""
    violacoes: list[str] = []
    pendentes: list[str] = []
    # E-352: IDENTIFICADOR também conta. Medido em E-351: 27 arquivos usam `sanitizeIDComponent` e a rede só olhava
    # caminho; identificador derivado de organização compartilhada é a MESMA classe de defeito.
    try:
        import importlib.util as _ilu
        _spec = _ilu.spec_from_file_location("classificador_de_escopo", Path(__file__).resolve().parent / "hades_scope_classifier.py")
        _mod = _ilu.module_from_spec(_spec)
        _spec.loader.exec_module(_mod)
        # MEDIDO (E-352): sem passar o diretório, o classificador varria o REPOSITÓRIO REAL mesmo em teste com
        # fixture — 6 testes da catraca quebraram por isso. *Instrumento que ignora o escopo da própria auditoria.*
        for arquivo, dados in _mod.classificar_usos(diretorio).items():
            if not dados["identificador"]:
                continue
            # E-382 / ADV-56/4: try POR ARQUIVO — antes, um caminho inexistente abortava o laço inteiro e as violações
            # seguintes NÃO eram reportadas ✗. *Instrumento que morre no primeiro arquivo estranho aprova tudo o que
            # não chegou a ver.*
            try:
                texto_arquivo = (diretorio / arquivo).read_text(encoding="utf-8", errors="replace")
            except OSError as erro:
                violacoes.append("%s: não foi possível LER para verificar a guarda (%s)" % (arquivo, erro))
                continue
            if tem_guarda_reconhecida(texto_arquivo) or DISPENSAS.get(arquivo):
                continue
            violacoes.append("%s: uso de IDENTIFICADOR derivado de organização sem guarda nem dispensa (%s)"
                             % (arquivo, ", ".join(dados["identificador"])))
    except Exception as erro:  # o classificador é um instrumento: falhar em silêncio seria pior
        violacoes.append("classificador de escopo indisponível: %s" % erro)
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
