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


def sha256_de(caminho: Path) -> str:
    h = hashlib.sha256()
    with caminho.open("rb") as f:
        for bloco in iter(lambda: f.read(1 << 20), b""):
            h.update(bloco)
    return h.hexdigest()


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
            "bytes": p.stat().st_size,
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


def normalizar(dados: dict) -> dict:
    """Remove o que muda por si, pela licao do `E-987`.

    `generated_at` muda a cada geracao; `source_dirty` muda a cada ficheiro escrito - *e o
    proprio manifesto, ao ser escrito, muda a contagem*. Se o `--check` os comparasse,
    falharia sempre, e um `--check` que grita sempre deixa de ser lido.
    """
    copia = dict(dados)
    copia.pop("generated_at", None)
    copia.pop("source_dirty", None)
    return copia


def main() -> int:
    parser = argparse.ArgumentParser(description="Emite/confere o manifesto do snapshot.")
    parser.add_argument("--check", action="store_true", help="nao escreve: falha se divergir")
    parser.add_argument("--json", action="store_true", help="imprime em vez de escrever")
    args = parser.parse_args()

    dados = construir()
    texto = json.dumps(dados, ensure_ascii=False, indent=2) + "\n"

    if args.json:
        sys.stdout.write(texto)
        return 0

    if args.check:
        if not MANIFESTO.exists():
            print("MANIFESTO AUSENTE: %s" % MANIFESTO.relative_to(RAIZ))
            return 1
        atual = json.loads(MANIFESTO.read_text(encoding="utf-8"))
        if normalizar(atual) != normalizar(dados):
            print("MANIFESTO DIVERGENTE do snapshot: %s" % MANIFESTO.relative_to(RAIZ))
            print("Rode: python scripts/hades_manifest.py")
            return 1
        print("MANIFESTO CONSISTENTE: bate com o snapshot (source_sha, ramo, SBOM, testes, plataformas, limitacoes).")
        return 0

    MANIFESTO.write_text(texto, encoding="utf-8")
    print("gerado:", MANIFESTO.relative_to(RAIZ))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
