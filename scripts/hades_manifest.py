#!/usr/bin/env python3
"""Emite o MANIFESTO do snapshot candidato (ESTAGIO 22, item 3).

Por que este script existe
--------------------------
O §ESTAGIO 22 item 3 pede: *«emitir **manifestos** de testes, plataformas, artefatos,
hashes, SBOM e limitacoes»*. Medido antes desta cura: **nao existia nenhum manifesto do
HADES**. O unico `manifest.json` do repositorio e' o do PWA (`app/ui/app/dist/`,
**551 B**), que declara icones e `start_url` -- *nada a ver com o snapshot de engenharia*.

**O que este manifesto NAO faz, e e' o ponto**: ele **nao avalia**. Nao diz se o projeto
esta' bom, nem repete o `NOT_RELEASE_READY` como se fosse uma conclusao sua. Ele
**registra** o que foi medido, de onde, e com que resultado -- *e deixa as limitacoes
declaradas por quem as conhece, numa lista que **nao** e' inventada aqui*.

Campos, um a um, e a razao de cada um
-------------------------------------
- `source_sha` / `source_dirty` / `source_branch`: o §FASE-6 exige que **todo** artefato
  registre o commit de origem e o estado sujo. *Sem isto, o manifesto nao se pode
  reproduzir -- que e' o defeito que o `E-987` curou no gate.*
- `generated_at`: quando foi lido. *Datado, porque o resto dos campos envelhece.*
- `sbom`: os ficheiros de SBOM que existem, com **linhas** e **sha256**. *Se um nao
  existir, aparece como `AUSENTE` -- em vez de desaparecer da lista.*
- `tests`: cada comando com o **exit code** observado. *Um teste que nao corre e' um
  teste que nao prova nada; o exit fica escrito.*
- `platforms`: o que se pode afirmar por plataforma. *E a distincao que o prompt faz
  questao: **SOURCE** nao e' **BUILD**, `BUILD` nao e' `PACKAGE`, `PACKAGE` nao e'
  `INSTALL`, e nenhum deles e' `DEVICE_E2E`.*
- `limitations`: as limitacoes **declaradas**, com a fonte. *Uma limitacao sem fonte e'
  uma opiniao; por isso cada uma carrega `onde`.*

Uso
---
    python scripts/hades_manifest.py            # escreve docs/mission/HADES_SNAPSHOT_MANIFEST.json
    python scripts/hades_manifest.py --check    # falha se o manifesto divergir do snapshot
    python scripts/hades_manifest.py --json     # imprime em vez de escrever

O `--check` compara **o que e' estavel** (`source_sha`, `source_branch`, `sbom`, `tests`,
`platforms`, `limitations`) e ignora o que muda por si (`generated_at`, `source_dirty`),
pela licao do `E-987`: *um ficheiro gerado com uma linha volatil deixa o `--check`
inutil, e um `--check` que grita sempre nao e' lido.*
"""

from __future__ import annotations

import argparse
import hashlib
import json
import platform
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
MANIFESTO = RAIZ / "docs/mission/HADES_SNAPSHOT_MANIFEST.json"

# Os SBOM que o §FASE-5 item 7 pede. **Declarados aqui, um a um** -- se um faltar, ele
# aparece como AUSENTE no manifesto em vez de sumir da lista: *uma ausencia declarada e'
# uma divida; uma ausencia omitida e' um buraco.*
SBOMS_DECLARADOS = (
    "HADES_SBOM_GO.csv",
    "HADES_SBOM_NPM.csv",
    "HADES_SBOM_LICENCAS.csv",
    "THIRD_PARTY_NOTICES.md",
)

# As plataformas-alvo do §IDENTIDADE item 8, com o que **este host** consegue afirmar.
# Nao se marca PACKAGE onde nao ha pacote, nem INSTALL onde nao houve instalacao.
PLATAFORMAS = (
    # (plataforma, SOURCE, BUILD, PACKAGE, INSTALL, DEVICE_E2E, assinatura)
    ("Windows", True, None, None, None, None, "nao_assinado"),
    ("Linux", True, None, None, None, None, "nao_assinado"),
    ("macOS", True, None, None, None, None, "nao_assinado"),
    ("Android", True, None, None, None, None, "nao_assinado"),
    ("iOS", True, None, None, None, None, "nao_assinado"),
    ("Web/PWA", True, None, None, None, None, "nao_aplicavel"),
)

# Limitacoes **declaradas**, cada uma com a fonte onde foi medida. *Nao se inventa aqui
# nenhuma: as que estao sao as que o proprio repositorio registra.*
LIMITACOES = (
    {
        "id": "LIM-01",
        "texto": "O gate global declarou `gate_snapshot = DIVERGENTE`: o `measured_commit` do "
                 "`MASTER_STATE` e' ancestral do HEAD, com dezenas de commits de distancia.",
        "onde": "docs/mission/HADES_RELEASE_REPORT.md (`gate_snapshot`)",
        "impacto": "O texto do relatorio foi gerado noutro snapshot; as contagens podem ter envelhecido.",
    },
    {
        "id": "LIM-02",
        "texto": "Nenhum artefato de distribuicao (instalador, APK, `.app`) foi produzido do candidato.",
        "onde": "docs/mission/HADES_RELEASE_REPORT.md (bloqueios de distribuicao)",
        "impacto": "PACKAGE/INSTALL/DEVICE_E2E ficam nulos em todas as plataformas.",
    },
    {
        "id": "LIM-03",
        "texto": "`go test -race` nao corre neste host: `CGO_ENABLED=0` por ausencia de mingw-w64.",
        "onde": "ambos os blocos de `test.yaml` declaram `CGO_ENABLED: '1'`",
        "impacto": "A prova de corrida fica no CI (`race (ubuntu|macos)`), nao aqui.",
    },
    {
        "id": "LIM-04",
        "texto": "O `golangci-lint` local corre com `CGO_ENABLED=0` e `GOOS` escolhido a mao, "
                 "enquanto o CI corre com `CGO_ENABLED=1`.",
        "onde": "test.yaml:418 e test.yaml:516",
        "impacto": "O lint local mede um produto diferente; o CI e' o juiz.",
    },
    {
        "id": "LIM-05",
        "texto": "O indice tem a faixa `E-571..E-777` (207 IDs) ausente, declarada como "
                 "`EVIDENCE_LOST_HISTORICAL`.",
        "onde": "docs/mission/HADES_EVIDENCE_INDEX.md",
        "impacto": "Requisitos ativos nao podem depender desses IDs.",
    },
)


def _git(*args: str) -> str:
    try:
        saida = subprocess.run(
            ["git", *args], cwd=RAIZ, capture_output=True, text=True, timeout=30, check=False
        )
        return saida.stdout.strip() if saida.returncode == 0 else ""
    except (OSError, subprocess.SubprocessError):
        return ""


def bytes_do_conteudo(caminho: Path) -> bytes:
    """Os bytes do **conteudo**, e nao os do ficheiro de trabalho.

    *Um derivado tem de medir o mesmo conteudo em qualquer checkout.* Com
    `core.autocrlf=true` (o default do Git no Windows) o ficheiro de trabalho tem `CRLF` e
    o **blob commitado tem `LF`**; um `sha256` calculado sobre o ficheiro de trabalho fica
    **consistente no Windows e divergente no CI Linux**. Foi exactamente assim que o job
    `class-a-plus-integrity` ficou vermelho: o manifesto gravava o hash dos `CRLF` locais e
    o CI, ao recalcular sobre o checkout `LF`, chamava-lhe divergencia.

    O `E-183` protege o caso **oposto** (uma fixture de terceiro com bytes publicados, marcada
    `-text` no `.gitattributes`, cujo `sha256` tem de bater *no blob*); aqui nao ha bytes
    publicados -- o SBOM e' gerado -- portanto **normaliza-se**. Ficheiros binarios (`dist/*`)
    nao sao tocados: o `\\x00` e' o mesmo criterio que o proprio Git usa para os distinguir.
    """
    dados = caminho.read_bytes()
    if b"\x00" in dados:
        return dados
    return dados.replace(b"\r\n", b"\n")


def sha256_de(caminho: Path) -> str:
    return hashlib.sha256(bytes_do_conteudo(caminho)).hexdigest()


def bloco_sbom() -> list[dict]:
    itens = []
    for nome in SBOMS_DECLARADOS:
        p = RAIZ / nome
        if not p.exists():
            itens.append({"arquivo": nome, "estado": "AUSENTE"})
            continue
        linhas = sum(1 for _ in p.open(encoding="utf-8", errors="replace"))
        itens.append({
            "arquivo": nome,
            "estado": "PRESENTE",
            "bytes": len(bytes_do_conteudo(p)),
            "linhas": linhas,
            "sha256": sha256_de(p),
        })
    return itens


def bloco_testes() -> list[dict]:
    """Os comandos que **este** host consegue correr, com o exit observado.

    *Nao se declara aqui um teste que nao se correu.* Um `exit` que nao foi medido aparece
    como `null`, e nao como `0` -- *um zero inventado e' a pior forma de mentir num
    manifesto.*
    """
    return [
        {
            "comando": "python scripts/hades_release_gate.py --check",
            "prova": "o gate bate com o estado da missao",
            "exit": None,
            "nota": "medido pelo proprio operador do manifesto; nao auto-invocado",
        },
        {
            "comando": "python -X utf8 -m unittest scripts.hades_test_clock_check_test",
            "prova": "12 testes do checker de relogio",
            "exit": None,
            "nota": "medido pelo proprio operador do manifesto; nao auto-invocado",
        },
        {
            "comando": "go test ./internal/agent ./server",
            "prova": "os dois pacotes Go tocados nesta janela",
            "exit": None,
            "nota": "exige _tmp/goroot1269 e isolamento de APPDATA/LOCALAPPDATA/TMP",
        },
    ]


def bloco_plataformas() -> list[dict]:
    campos = ("plataforma", "source", "build", "package", "install", "device_e2e", "assinatura")
    return [dict(zip(campos, linha)) for linha in PLATAFORMAS]


def construir() -> dict:
    sujo = [l for l in _git("status", "--porcelain").splitlines() if l.strip()]
    return {
        "schema": "hades.snapshot.manifest/v1",
        "gerado_por": "scripts/hades_manifest.py",
        "generated_at": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "source_sha": _git("rev-parse", "HEAD") or "NAO_MEDIDO",
        "source_branch": _git("rev-parse", "--abbrev-ref", "HEAD") or "NAO_MEDIDO",
        "source_dirty": len(sujo),
        "host": {
            "sistema": platform.system(),
            "release": platform.release(),
            "maquina": platform.machine(),
            "python": platform.python_version(),
        },
        "sbom": bloco_sbom(),
        "tests": bloco_testes(),
        "platforms": bloco_plataformas(),
        "limitations": list(LIMITACOES),
        "nao_avalia": (
            "Este manifesto REGISTRA medicoes; nao emite veredito. O veredito de release e' "
            "do `HADES_RELEASE_REPORT.md`, e continua `NOT_RELEASE_READY`."
        ),
    }


# ---------------------------------------------------------------------------
# **E-996 — CLASSIFICACAO OBRIGATORIA DE CADA CAMPO.** A inversao para «volatil por omissao»
# resolve as tres falhas do CI, *mas cria um risco que eu tenho de declarar*: um campo que
# **devesse** ser comparado e nao fosse classificado ficaria **silenciosamente** de fora => e
# isso e' um **verde falso**, que o §ANTILOOP item 8 proibe.
#
# *A cura e' exigir a classificacao*: todo campo que `construir()` produza tem de estar numa
# das duas listas, ou o `--check` **falha**. Assim:
#   - um campo novo **nao pode** entrar em silencio (falha ate' ser classificado);
#   - e a decisao de o comparar ou nao fica **escrita**, com motivo, e nao inferida.
#
# ESTAVEIS: descrevem **o que o manifesto prova** — conteudo derivado de ficheiros commitados.
# VOLATEIS: descrevem **onde e quando** ele foi produzido.
# ---------------------------------------------------------------------------
CAMPOS_ESTAVEIS = (
    # Ordem = ordem em que o `--check` os compara; muda-la muda o dict, mas nao o resultado.
    "schema", "gerado_por", "nao_avalia",
    "sbom", "tests", "platforms", "limitations",
)
CAMPOS_VOLATEIS = (
    # **E-993/E-994/E-996**: todos descrevem o AMBIENTE, e o CI nao e' o ambiente onde o
    # manifesto foi gerado. `host` foi o terceiro a falhar (run 38926745194): o manifesto e'
    # gerado em Windows e o CI corre em Linux.
    "generated_at", "source_sha", "source_branch", "source_dirty", "host",
)


def normalizar(dados: dict) -> dict:
    """Devolve **apenas** os campos que o manifesto PROVA - o resto e' declarado, nao comparado.

    **E-996, e esta funcao foi INVERTIDA por causa de tres falhas seguidas no CI.**

    A versao original **removia** as volateis uma a uma (`generated_at`, `source_dirty`,
    `source_sha`, `source_branch`). *Parecia uma allowlist de excepcoes; era uma allowlist
    pelo avesso* -- *e cada campo novo nascia **comparado** por omissao*. O resultado, medido
    em tres runs do `class-a-plus-integrity`:

    - **38922978710** (`dd43d85fc`) -> passo 6 (`hades_release_gate.py --check`) falhou:
      `gate_head`/`gate_branch` volateis no CI;
    - **38924032504** (`bddeb7ebd`) -> passo 7 falhou: `source_sha`/`source_branch`;
    - **38926745194** (`a4ddaa4e0`) -> passo 7 falhou **outra vez**, agora pelo `host`:
      `{"sistema": "Windows", "release": "11", "maquina": "AMD64", "python": "3.12.10"}`.
      *O manifesto e' gerado localmente, em Windows; o CI corre em Linux.* => *`sistema`,
      `release`, `maquina` e `python` sao **todos** diferentes.*

    *Tres instrumentos, quatro classes, e em todos eu curei o que o CI apontou e deixei o
    resto.* => **curar o que o CI aponta e' perseguir sintomas.**

    **A inversao**: em vez de listar o que **nao** se compara, lista-se o que **se** compara.
    *Um campo novo passa a ser **volatil por omissao***. **E `classificar_campos()` exige que
    todo campo esteja numa das listas** -- *para que «volatil por omissao» nao vire «esquecido
    com sorte»*.

    **O que ele PROVA** (e por isso e' comparado): o **SBOM** com `sha256` de cada ficheiro
    (*prova que os artefactos nao mudaram*), as **plataformas** (*o que se pode afirmar por
    SO*), as **limitacoes** declaradas e os **comandos de teste**. => *tudo isso e' **conteudo**,
    derivado de ficheiros commitados.*

    **O que ele apenas DECLARA** (e fica no documento, sem ser criterio): `generated_at`,
    `source_sha`, `source_branch`, `source_dirty` e `host` => *todos descrevem **onde e quando**
    o manifesto foi produzido, e nenhum descreve **o que ele prova***. *E o `source_sha`
    continua gravado porque o §FASE-6 exige a proveniencia em todo artefacto: **declarado, nao
    comparado** -- que e' a distincao que o `E-993` estabeleceu para o gate.*
    """
    return {chave: dados[chave] for chave in CAMPOS_ESTAVEIS if chave in dados}


def classificar_campos(dados: dict) -> list[str]:
    """Devolve os campos que **nao** foram classificados nem como estaveis nem como volateis.

    *E' este o contrapeso da inversao*: sem ele, um campo novo entrar no manifesto e nunca ser
    comparado seria **indistinguivel** de um campo novo ser corretamente volatil. => *com ele,
    um campo novo **falha**, e obriga a uma decisao escrita.*
    """
    conhecidos = set(CAMPOS_ESTAVEIS) | set(CAMPOS_VOLATEIS)
    return sorted(set(dados) - conhecidos)


def main() -> int:
    parser = argparse.ArgumentParser(description="Emite/confere o manifesto do snapshot.")
    parser.add_argument("--check", action="store_true", help="nao escreve: falha se divergir")
    parser.add_argument("--json", action="store_true", help="imprime em vez de escrever")
    args = parser.parse_args()

    dados = construir()
    texto = json.dumps(dados, ensure_ascii=False, indent=2) + "\n"

    # **E-996**: a classificacao obrigatoria. *Um campo que nao seja estavel nem volatil declarado
    # torna este manifesto **incomparavel** -- e falhar alto e' melhor que comparar metade em
    # silencio.* => *e' o contrapeso da inversao para «volatil por omissao».*
    nao_classificados = classificar_campos(dados)
    if nao_classificados:
        print("MANIFESTO: campo(s) de topo NAO CLASSIFICADO(S): %s" % ", ".join(nao_classificados))
        print("Cada campo tem de entrar em CAMPOS_ESTAVEIS (o manifesto PROVA) ou em")
        print("CAMPOS_VOLATEIS (declara o ambiente) neste script. *Um campo novo nao pode")
        print("entrar em silencio* -- e' o contrapeso de «volatil por omissao» (E-996).")
        return 1

    if args.json:
        sys.stdout.write(texto)
        return 0

    if args.check:
        if not MANIFESTO.exists():
            print("MANIFESTO AUSENTE: %s" % MANIFESTO.relative_to(RAIZ))
            return 1
        atual = json.loads(MANIFESTO.read_text(encoding="utf-8"))
        # **E-996, e o teste apanhou a metade que faltava**: classificar so' o `dados`
        # (*o que `construir()` produz*) deixava passar um campo novo **escrito a mao no JSON**
        # -- *porque o `--check` nao reconstroi o ficheiro, so' o compara*. => *logo classificam-se
        # **os dois lados***: o construido e o lido. *Um campo novo, venha de onde vier, falha.*
        for rotulo, origem in (("construido", dados), ("no ficheiro", atual)):
            por_classificar = classificar_campos(origem)
            if por_classificar:
                print("MANIFESTO: campo(s) de topo NAO CLASSIFICADO(S) %s: %s"
                      % (rotulo, ", ".join(por_classificar)))
                print("Cada campo tem de entrar em CAMPOS_ESTAVEIS (o manifesto PROVA) ou em")
                print("CAMPOS_VOLATEIS (declara o ambiente) neste script. *Um campo novo nao pode")
                print("entrar em silencio* -- e' o contrapeso de «volatil por omissao» (E-996).")
                return 1
        if normalizar(atual) != normalizar(dados):
            print("MANIFESTO DIVERGENTE do snapshot: %s" % MANIFESTO.relative_to(RAIZ))
            print("Rode: python scripts/hades_manifest.py")
            return 1
        # **E-994**: a mensagem tem de dizer o que REALMENTE e' comparado. *Ela dizia
        # «source_sha, ramo, SBOM, ...» depois de eu ter tirado o `source_sha` e o ramo da
        # comparacao* -- e uma mensagem que mente sobre o que faz e' o `E-986` na forma de
        # saida: *documentar o defeito pode reintroduzi-lo, agora na representacao.*
        print("MANIFESTO CONSISTENTE: bate com o snapshot (SBOM com sha256, testes, "
              "plataformas, limitacoes).")
        print("  NAO comparados por serem volateis no CI (`pull_request` faz checkout do "
              "merge commit): source_sha, source_branch, source_dirty, generated_at.")
        return 0

    MANIFESTO.write_text(texto, encoding="utf-8")
    print("gerado:", MANIFESTO.relative_to(RAIZ))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
