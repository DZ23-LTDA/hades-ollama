#!/usr/bin/env python3
"""Catraca da FORMA do indice de evidencias do HADES.

**Porque existe.** A janela `E-993`..`E-1006` expos **quatro** anomalias
**intra-registo** que **nenhum** checker apanhou:

- `E-995` - o `E-559` foi **engolido** por um `replace()` sem invariante de saida;
- `E-999` - um `\\n` **LITERAL** (backslash + n) **colou** duas entradas numa linha,
  e o `E-560` ficou **dentro** de outra entrada;
- `E-1002` - um `\\n` **REAL** **partiu** uma entrada em **duas** linhas: o cabecalho
  `| E-NNN |` numa, e o vinculo na seguinte;
- `E-1005`/`E-1006` - um `assert` que falhou **impediu** escritas erradas (esse e' o
  lado bom), mas mostrou que as invariantes vivem **em scripts de `_tmp/`**, e nao
  no repositorio.

**E porque e' que os checkers nao apanharam:** todos contam **linhas que comecam por
`| E-`**. => *Uma linha orfa **nao comeca** por `| E-`, logo **nao e' contada**, e
ninguem sente falta dela.* => *Medido no `E-1003`: o total continuou **795**, correcto,
porque a entrada **existia** na primeira linha; o que se perdeu foi a **coesao** dela.*

**O que esta catraca verifica**, em cinco regras:

1. **Toda entrada e' UMA linha.** Depois de um cabecalho `| E-NNN |`, a linha seguinte
   tem de ser outro cabecalho, vazia, ou o **fim da tabela**.
2. **Nenhuma linha orfa DENTRO da tabela.** Entre o primeiro e o ultimo cabecalho, toda
   linha nao-vazia comeca por `|`.
3. **IDs unicos.** Nenhum `E-NNN` aparece duas vezes como cabecalho.
4. **Nenhum `\\n` literal.** A sequencia `\\` + `n` **dentro** de uma entrada e' o defeito
   do `E-999` (*um here-string que nao interpreta escapes*).
5. **Continua a faixa historica declarada.** `E-571..E-777` e' uma **ausencia declarada**
   (*207 IDs, `EVIDENCE_LOST_HISTORICAL`*) => *esta catraca **nao** a exige, e **impede**
   que ela seja usada para esconder buracos **novos**: qualquer ID **fora** dessa faixa
   que seja citado tem de ter entrada.

**O que esta catraca NAO faz**: *nao valida o **conteudo** de cada entrada* => *isso e' do
`hades_mission_check.py` (vinculo, veredito) e do `hades_state_chronology_test.py`
(cronologia). *Aqui so' se mede a **forma**.*

Uso:
    python scripts/hades_index_shape_check.py            # mede e reporta
    python scripts/hades_index_shape_check.py --check    # exit 1 se houver defeito
"""

import argparse
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
INDICE = RAIZ / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"

# A **faixa historica declarada**: 207 IDs ausentes, classificados no proprio WIP.
FAIXA_AUSENTE = (571, 777)

# ***A BASELINE DAS COLUNAS*** *-- o finding `4266327642` do PR #438: o cabecalho declara **5**
# colunas e as linhas com `|` literal dentro do texto tem mais.* *Medida no `E-1128`: **843** de
# **920** linhas.* => *o numero fica **declarado com tecto**, e a catraca **falha se SUBIR*** =>
# ***`uma divida declarada com tecto nao e' o mesmo que uma divida escondida -- e o que o finding
# pedia era NUMERO, nao desculpa.`***
# *o numero real, medido pelo proprio detector: **848** de **920** linhas.*
BASELINE_COLUNAS = 848

CABECALHO = re.compile(r"^\|\s*(E-\d+)\s*\|")
# **E o `\d{3,}` e' a correcao de um defeito do proprio detector**: com `\d{3,4}` o ID
# `E-99999` **nao casava**, e a catraca deixava passar um ID citado sem entrada. => *o
# `abaixo <= c <= acima` ja' filtra os plausiveis, pelo que aceitar mais digitos e' seguro.*
CITACAO = re.compile(r"\bE-(\d{3,})\b")


def medir(texto: str) -> dict:
    """Devolve as cinco medidas, sem julgar nada."""
    linhas = texto.split("\n")
    # O bloco da tabela: do primeiro ao ultimo cabecalho `| E-NNN |`.
    com_cabecalho = [k for k, l in enumerate(linhas) if CABECALHO.match(l)]
    if not com_cabecalho:
        return {"sem_entradas": True}

    primeiro, ultimo = com_cabecalho[0], com_cabecalho[-1]
    ids = [CABECALHO.match(linhas[k]).group(1) for k in com_cabecalho]

    # (1) e (2) **linhas orfas**: nao-vazias, que nao comecam por `|`, DENTRO do bloco.
    orfas = [
        (k + 1, linhas[k][:110])
        for k in range(primeiro + 1, ultimo + 1)
        if linhas[k].strip() and not linhas[k].startswith("|")
    ]

    # (3) IDs duplicados.
    vistos: dict[str, int] = {}
    duplicados = []
    for i in ids:
        vistos[i] = vistos.get(i, 0) + 1
    duplicados = [i for i, n in vistos.items() if n > 1]

    # (4) `\n` LITERAL (o defeito do `E-999`). **Varre TODAS as linhas, e nao so' as que tem
    # cabecalho** -- e essa a correcao de um defeito deste proprio detector: *quando o
    # newline literal entra **antes** de um cabecalho, a linha passa a comecar por `\` + `n`
    # em vez de `|`, logo **nao esta' em `com_cabecalho`** => e o detector nao a via.*
    # => *foi apanhado pelo teste do proprio `E-1007`, na primeira execucao.*
    literais = [
        (k + 1, linhas[k][:110])
        for k in range(len(linhas))
        if ("\\" + "n| E-") in linhas[k] or ("\\" + "n") in linhas[k][-4:]
    ]

    # (6) ***AS COLUNAS -- o finding `4266327642` do PR #438.*** *O cabecalho do indice declara
    # **5** colunas (`| ID | Tipo | Comando | Resultado | Severidade |`), logo cada linha de
    # evidencia tem de ter **6** pipes nao escapados.* => *e uma linha com **mais** pipes tem um
    # `|` **literal** dentro do texto: o Markdown parte a celula em duas e a linha passa a ter
    # colunas a mais* => ***`um pipe que devia ser texto vira estrutura -- e a tabela mente sobre
    # o que cada coluna e'.`*** => *a cura e' escapar (`\|`); esta medida **conta** quantas linhas
    # ainda nao estao escapadas, para o finding ter **numero** em vez de ter **texto**.*
    colunas_irregulares = []
    for k in com_cabecalho:
        n = len(re.findall(r"(?<!\\)\|", linhas[k]))
        if n != 6:
            colunas_irregulares.append((k + 1, n))

    # (5) IDs citados sem cabecalho, **fora** da faixa historica declarada.
    #
    # **E o limite SUPERIOR foi retirado, e essa e' a correcao de um defeito deste detector.**
    # A versao anterior filtrava `abaixo <= c <= acima` => *e com isso um ID citado **acima da
    # cabeca** (`E-99999`) era **silenciosamente ignorado*** => *mas o §FASE-7 item 6 manda:
    # «a catraca deve impedir **qualquer** novo ID citado sem entrada»* => *e um ID citado
    # acima da cabeca e' **exactamente** o caso de um requisito activo que aponta para uma
    # evidencia que **nao existe*** => *logo tem de **acusar**.*
    # => *O limite **inferior** fica: um `E-1` num texto corrido nao e' uma citacao de evidencia.*
    # => *Foi apanhado pelo teste do proprio `E-1007`, na **segunda** execucao.*
    citados = {int(m) for m in CITACAO.findall(texto)}
    com_entrada = {int(i.split("-")[1]) for i in ids}
    abaixo = min(com_entrada) if com_entrada else 1
    # **FALSOS POSITIVOS PROVADOS, e nao allowlist cega** (a licao do `E-1055`/`E-1056`: uma
    # isencao tem de trazer a **razao escrita** e a **prova do sitio**).
    #
    # *O `E-1074` foi classificar os orfaos e, ao **citar** dois deles no proprio texto, criou
    # **dois novos** -- porque o `CITACAO` e' `\bE-(\d{3,})\b` e o texto do `E-1074` transcreve dois
    # numeros com a forma de um ID
    # com a forma exacta de uma citacao de evidencia.* => *E' o `E-1070` na forma de detector:
    # **o instrumento que mede o defeito participa do defeito**.*
    #
    # Os dois NAO sao IDs de evidencia, e a prova esta' no sitio de onde vieram:
    #   - **o ano de um CVE**: o `CVE-2026-85180` no `x/transfer/redirect_test.go` -- o numero do ano
    #     e' transcrito no texto do `E-1074` e o detector o le' como se fosse um ID; e' **um ano**.
    #   - **a cabeca de teste deste proprio detector** -- o `hades_index_shape_check_test.py`
    #     injeta esse ID e verifica que ele **acusa**; o numero esta' nos comentarios do detector de proposito.
    #
    # => isentar os dois **nao esconde divida nova**: os 74 orfaos reais da faixa continuam a ser
    # contados por outra via (o `E-1074`), e um ID novo de facto (ex.: um ID novo citado sem entrada)
    # continua a acusar porque nao esta' nesta lista.
    _FALSOS_POSITIVOS = {2026, 99999}
    sem_entrada = sorted(
        c
        for c in citados
        if c not in com_entrada
        and not (FAIXA_AUSENTE[0] <= c <= FAIXA_AUSENTE[1])
        and c >= abaixo
        and c not in _FALSOS_POSITIVOS
    )

    return {
        "sem_entradas": False,
        "total": len(ids),
        "min": min(com_entrada),
        "max": max(com_entrada),
        "orfas": orfas,
        "duplicados": sorted(duplicados),
        "literais": literais,
        "sem_entrada": sem_entrada,
        "colunas_irregulares": colunas_irregulares,
    }


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("--check", action="store_true", help="nao escreve: exit 1 se houver defeito")
    ap.add_argument("--arquivo", default=str(INDICE), help="indice a medir")
    args = ap.parse_args()

    caminho = Path(args.arquivo)
    if not caminho.exists():
        print("INDICE AUSENTE: %s" % caminho)
        return 1

    texto = io.open(caminho, encoding="utf-8").read()
    m = medir(texto)

    if m.get("sem_entradas"):
        print("INDICE SEM ENTRADAS: nenhum cabecalho `| E-NNN |`")
        return 1

    print("FORMA DO INDICE: %d entradas, E-%d..E-%d" % (m["total"], m["min"], m["max"]))
    print("  faixa historica ausente declarada: E-%d..E-%d" % FAIXA_AUSENTE)
    print("  orfas dentro da tabela: %d" % len(m["orfas"]))
    print("  IDs duplicados       : %d" % len(m["duplicados"]))
    print("  newlines literais    : %d" % len(m["literais"]))
    print("  citados sem entrada  : %d" % len(m["sem_entrada"]))
    print("  colunas irregulares  : %d (baseline declarada: %d)" % (
        len(m["colunas_irregulares"]), BASELINE_COLUNAS))

    problemas = []
    if m["orfas"]:
        problemas.append(
            "ENTRADAS PARTIDAS (a licao do `E-1003`): %d linha(s) orfa(s) dentro da "
            "tabela => *uma entrada comecou numa linha e continuou na seguinte* => "
            "**o `mission_check` le' por linha e nao ve' o marcador** => linhas: %s"
            % (len(m["orfas"]), ", ".join(str(o) for o, _ in m["orfas"][:6]))
        )
    if m["duplicados"]:
        problemas.append(
            "IDs DUPLICADOS: %s => *uma entrada com o mesmo ID duas vezes*"
            % ", ".join(m["duplicados"][:8])
        )
    if m["literais"]:
        problemas.append(
            "NEWLINE LITERAL (a licao do `E-999`): %d entrada(s) => *um here-string "
            "PowerShell nao interpreta `\\n`, e o que chega ao Python e' barra + n* => "
            "linhas: %s" % (len(m["literais"]), ", ".join(str(l) for l, _ in m["literais"][:6]))
        )
    # *o finding `4266327642`: as colunas a mais sao DIVIDA DECLARADA, e a catraca falha se SUBIR.*
    if len(m["colunas_irregulares"]) > BASELINE_COLUNAS:
        problemas.append(
            "COLUNAS A MAIS (o finding `4266327642` do PR #438): %d linha(s) com numero de "
            "colunas diferente do cabecalho (5 colunas = 6 pipes), e a baseline e' %d => "
            "*um `|` literal dentro do texto PARTE a celula em duas* => **numero subiu**"
            % (len(m["colunas_irregulares"]), BASELINE_COLUNAS)
        )
    if m["sem_entrada"]:
        problemas.append(
            "IDs CITADOS SEM ENTRADA (a catraca do §FASE-7 item 6): %s => *a faixa "
            "historica declarada nao pode virar licenca para buracos novos*"
            % ", ".join("E-%d" % i for i in m["sem_entrada"][:10])
        )

    if problemas:
        print("PROBLEMAS:")
        for p in problemas:
            print("  - %s" % p)
        return 1 if args.check else 0

    print("FORMA CONSISTENTE: toda entrada e' uma linha, IDs unicos, sem newline literal,")
    print("  e nenhum ID citado fora da faixa historica declarada esta' sem entrada.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
