#!/usr/bin/env python3
"""hades_encoding_check.py — o repositório não pode conter dupla codificação REAL.

POR QUE ISTO EXISTE (E-409): uma varredura manual encontrou 3 arquivos com dupla codificação, e **dois eram
legítimos** — o **regex do próprio detector** (`hades_mission_check.py`) e a **fixture** que prova que ele
dispara (`hades_mission_check_test.py`). Um scan que não distingue a ferramenta do alvo acusa a ferramenta.
E o terceiro achado era **real e repetido**: eu havia transcrito um artefato de console para dentro do
índice de evidências, e a corrupção passou a existir **porque eu a coloquei lá**.

*Descrever um artefato é seguro; transcrevê-lo não é.*

Este checker torna a varredura **reproduzível** em vez de depender de alguém lembrar de rodá-la.

DISPENSAS: são **explícitas, por caminho, com motivo** — e o checker **falha se uma dispensa ficar obsoleta**
(o arquivo dispensado deixou de conter o padrão), porque *dispensa que não casa com nada é dívida disfarçada
de exceção*. Foi exatamente o defeito que corrigi no classificador de escopo (E-383).

O PONTO CEGO QUE ESTE CHECKER TINHA (E-660)
-------------------------------------------
Até o E-658 este checker só procurava **dupla codificação** — e um **BOM UTF-8** passava silenciosamente por
ele. *E passava porque `read_text(encoding="utf-8")` NÃO lança exceção sobre um BOM: o `\ufeff` vira o
primeiro caractere da string.* O padrão de mojibake não casa com `\ufeff`, e o arquivo era declarado limpo.

O custo medido: o `HADES_MASTER_STATE.json` foi gravado **com BOM** no commit `6a391ef5c` (rodada 8), e o
`hades_release_gate.py` — que lê o estado com `encoding="utf-8"` — passou a **estourar** no CI:

    json.decoder.JSONDecodeError: Unexpected UTF-8 BOM (decode using utf-8-sig)

O mesmo commit gravou o ficheiro passando por **Windows-1252** e **dupla-codificou 58 ocorrências**
(travessão `—` virava `â€"`). *O instrumento existia, cobria metade da classe, e essa metade era a que
importava.*

Este checker passa a acusar **as duas formas**, e a varredura é a mesma (`os.walk` podado).
"""

import os
import re
import sys
from pathlib import Path

# BOM UTF-8: o marcador que `read_text(encoding="utf-8")` NÃO denuncia (vira `\ufeff` na string) e que quebra
# leitores estritos — `json.loads` lança JSONDecodeError. Nenhum ficheiro deste repositório o justifica.
BOM_UTF8 = b"\xef\xbb\xbf"

EXTENSOES = {".go", ".md", ".py", ".ts", ".tsx", ".js", ".mjs", ".json", ".yaml", ".yml", ".txt", ".ps1", ".sh"}
IGNORAR_DIR = {"node_modules", ".git", "_tmp", "dist", "build", "vendor", ".next", "coverage", "testdata", "__pycache__"}

# Padrões de dupla codificação: UTF-8 lido como Latin-1. O travessão vira a sequência de dois bytes.
PADROES = ["Ã§", "Ãµ", "Ã£", "Ã©", "Ã¡", "Ã­", "Ã³", "Ãº", "Ã¢", "Ãª", "Ã´", "Ã ", "Ã‡", "Ã‰", "â€", "Âº", "Âª", "Ãƒ"]
REGEX = re.compile("|".join(re.escape(p) for p in PADROES))

# Dispensas: caminho relativo -> motivo. Só estas podem conter os padrões, e só enquanto contiverem.
DISPENSAS = {
    "scripts/hades_mission_check.py":
        "É o PRÓPRIO detector: o comentário dos padrões e o `MOJIBAKE = re.compile(...)`. "
        "A munição do detector parece a doença — *e precisa parecer, senão ele não detecta nada*.",
    "scripts/hades_mission_check_test.py":
        "É a FIXTURE deliberada que prova que o detector dispara. Removê-la seria desarmar o teste do detector.",
    "scripts/hades_encoding_check.py":
        "É o PRÓPRIO checker desta varredura: declara os padrões acima. Pelo mesmo motivo do detector.",
    "scripts/hades_encoding_check_test.py":
        "É a fixture que prova que ESTE checker dispara — e ela precisa conter os padrões para isso.",
}


def arquivos(raiz: Path):
    """Percorre podando os diretórios ignorados DURANTE a caminhada.

    MEDIDO (E-410): a primeira versão usava `raiz.rglob("*")` e filtrava depois — e **passou de 10 minutos**
    ✗, porque `rglob` **entra** em `dist/`, `build/` e `node_modules/` antes de o filtro descartá-los.
    *Checker que não termina não é checker; e o custo estava no **caminho**, não no trabalho.*
    """
    for pasta, subpastas, nomes in os.walk(raiz):
        subpastas[:] = [d for d in subpastas if d not in IGNORAR_DIR]
        for nome in sorted(nomes):
            caminho = Path(pasta) / nome
            if caminho.suffix.lower() in EXTENSOES:
                yield caminho


def medir(raiz: Path):
    """Devolve (sujos, ilegiveis, dispensas_obsoletas, com_bom)."""
    sujos, ilegiveis, com_bom, usadas = [], [], [], set()
    for caminho in arquivos(raiz):
        relativo = caminho.relative_to(raiz).as_posix()
        try:
            dados = caminho.read_bytes()
        except OSError as erro:
            ilegiveis.append("%s (%s)" % (relativo, erro))
            continue
        # O BOM tem de ser medido nos BYTES: decodificado como UTF-8 ele vira `\ufeff`, que não é erro e não
        # casa com nenhum padrão de mojibake — era exatamente por isso que passava em silêncio.
        if dados.startswith(BOM_UTF8):
            com_bom.append(relativo)
        try:
            texto = dados.decode("utf-8")
        except UnicodeDecodeError as erro:
            ilegiveis.append("%s (%s)" % (relativo, erro))
            continue
        if not REGEX.search(texto):
            continue
        if relativo in DISPENSAS:
            usadas.add(relativo)
            continue
        sujos.append(relativo)
    obsoletas = sorted(set(DISPENSAS) - usadas)
    return sujos, ilegiveis, obsoletas, com_bom


def main(argv):
    raiz = Path(argv[1]) if len(argv) > 1 else Path(".")
    medidos = sum(1 for _ in arquivos(raiz))
    sujos, ilegiveis, obsoletas, com_bom = medir(raiz)
    print("CODIFICACAO: %d arquivo(s) medido(s), %d dispensa(s) declarada(s)" % (medidos, len(DISPENSAS)))
    if medidos == 0:
        print("NADA MEDIDO: nenhum arquivo de texto encontrado em %s — isto NÃO é aprovação." % raiz)
        return 1
    falhou = False
    for relativo in sujos:
        print("VIOLACAO: dupla codificacao REAL em %s — descreva o artefato, nao o transcreva" % relativo)
        falhou = True
    for relativo in com_bom:
        print("VIOLACAO: BOM UTF-8 no inicio de %s — leitor estrito quebra "
              "(json.loads: \"Unexpected UTF-8 BOM\"). Grave como UTF-8 SEM BOM: `encoding=\"utf-8\"` "
              "(nao `utf-8-sig`) e, no PowerShell 5.1, nao use `Set-Content -Encoding UTF8`." % relativo)
        falhou = True
    for item in ilegiveis:
        print("VIOLACAO: arquivo nao decodifica como UTF-8: %s" % item)
        falhou = True
    for relativo in obsoletas:
        print("VIOLACAO: dispensa OBSOLETA (o arquivo nao contem mais o padrao): %s — remova a dispensa ou explique"
              % relativo)
        falhou = True
    if not falhou:
        print("  (sem BOM e sem dupla codificacao; as %d dispensas sao o detector, a fixture dele, "
              "este checker e a fixture dele)" % len(DISPENSAS))
    return 1 if falhou else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv))
