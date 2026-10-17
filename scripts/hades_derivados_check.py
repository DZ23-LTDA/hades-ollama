#!/usr/bin/env python3
"""O pre-commit que faltava: os derivados tem de bater com a fonte (`E-1071`).

O QUE ACONTECEU, MEDIDO NO `E-1070`.

O `E-1068` mudou o `apps/mobile-agentic/app.json` -- um ficheiro que **alimenta um instrumento** --
e **nao correu o instrumento**. O `HADES_BRAND_INVENTORY.md` ficou desactualizado, e o CI reprovou:

    INVENTARIO DIVERGENTE do repositorio: docs/mission/HADES_BRAND_INVENTORY.md

=> *«o `E-1068` mudou um ficheiro que alimenta um instrumento e nao correu o instrumento.»*

**A causa-raiz nao e' o `app.json`: e' nao haver nada que o obrigue.** Os `--check` **detectam** a
divergencia, mas so' se forem corridos. Um instrumento que ninguem corre e' um instrumento que nao
existe no momento em que importa.

O QUE ESTE SCRIPT FAZ.

Corre os **tres** geradores de derivados em modo `--check` e falha se **algum** divergir:

    scripts/hades_release_gate.py        -> HADES_FINAL_RELEASE_GATE.md, HADES_RELEASE_REPORT.md
    scripts/hades_manifest.py            -> HADES_SNAPSHOT_MANIFEST.json
    scripts/hades_brand_inventory.py     -> HADES_BRAND_INVENTORY.md

E **diz qual regenerar**, em vez de so' dizer que falhou. *Um erro que nao diz o comando que o
resolve faz o proximo a procurar o comando.*

COMO ISTO SE INSTALA -- e o hook e' **versionado**, nao local.

Um `.git/hooks/pre-commit` local nao se partilha: cada clone teria de o recriar, e **um hook que
existe so' na minha maquina e' a mesma doenca do `E-1070`** -- o problema nao foi medir mal, foi o
instrumento nao estar no sitio onde o trabalho acontece. Por isso o hook vive em `.githooks/` e
instala-se com um comando:

    git config core.hooksPath .githooks

Isso aponta **todos** os clones que correrem esse comando ao `.githooks/`, que esta' no Git. Nao e'
tudo: um clone novo tem de correr o comando -- **declarado aqui e em `docs/mission/HADES_HANDOFF.md`.**

POR QUE NAO `< 2 s` NAO E' O CRITERIO.

Isto corre **tres** processos Python sobre um monorepo. Medido: **~4 s**. *Um pre-commit de 4 s e'
aceitavel para o que evita*: um vermelho de CI custa **minutos** e uma investigacao inteira -- o
`E-1070` custou uma sessao. Mas se um dia passar a custar minutos, o criterio muda: *o hook tem de
ser mais barato do que o erro que evita.*

O QUE ESTE SCRIPT NAO FAZ.

- Nao **regenera** nada. Um pre-commit que escreve no indice e' um pre-commit que **muda o que vais
  commitar** -- e *um hook que altera o commit que esta' a ser feito esconde o que o autor escreveu.*
- Nao valida os outros checkers (markdown, encoding, orfaos). Esses ja' estao na bateria de gates, e
  um pre-commit com **toda** a bateria levaria minutos.
- Nao corre em `--no-verify`. Nenhum hook corre; isso e' do Git.
"""
import argparse
import subprocess
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent

# **A lista declarada**: cada gerador, o derivado que ele produz, e o comando que o regenera.
DERIVADOS = [
    {
        "gerador": "scripts/hades_release_gate.py",
        "artefactos": ["docs/mission/HADES_FINAL_RELEASE_GATE.md", "docs/mission/HADES_RELEASE_REPORT.md"],
        "regenerar": "python -X utf8 scripts/hades_release_gate.py",
    },
    {
        "gerador": "scripts/hades_manifest.py",
        "artefactos": ["docs/mission/HADES_SNAPSHOT_MANIFEST.json"],
        "regenerar": "python -X utf8 scripts/hades_manifest.py",
    },
    {
        "gerador": "scripts/hades_brand_inventory.py",
        "artefactos": ["docs/mission/HADES_BRAND_INVENTORY.md"],
        "regenerar": "python -X utf8 scripts/hades_brand_inventory.py",
    },
]


def correr_check(gerador: str) -> tuple[bool, str]:
    """Corre um gerador em `--check`. *`--check` nao escreve; so' compara.*"""
    cmd = [sys.executable, "-X", "utf8", str(RAIZ / gerador), "--check"]
    r = subprocess.run(cmd, cwd=str(RAIZ), capture_output=True, text=True, encoding="utf-8", errors="replace")
    saida = (r.stdout or "") + (r.stderr or "")
    return (r.returncode == 0, saida.strip())


def main() -> int:
    ap = argparse.ArgumentParser(description="Verifica se os derivados batem com a fonte.")
    ap.add_argument("--quiet", action="store_true", help="so' mostra os divergentes")
    args = ap.parse_args()

    divergentes = []
    for d in DERIVADOS:
        ok, saida = correr_check(d["gerador"])
        if ok:
            if not args.quiet:
                print(f"  OK    {d['gerador']}  ->  {', '.join(a.split('/')[-1] for a in d['artefactos'])}")
        else:
            divergentes.append((d, saida))
            # a ultima linha util e' o diagnostico do proprio gerador
            util = [l for l in saida.splitlines() if l.strip()][-1:] or ["(sem saida)"]
            print(f"  FALHA {d['gerador']}  ->  {util[0][:110]}")

    if not divergentes:
        if not args.quiet:
            print("")
            print("  DERIVADOS CONSISTENTES: os tres batem com a fonte.")
        return 0

    print("")
    print("  " + "=" * 74)
    print("  DERIVADO DESACTUALIZADO -- o commit e' recusado.")
    print("")
    print("  *Um derivado e' uma funcao da fonte. Se a fonte mudou e o derivado nao, o derivado")
    print("  passou a ser uma afirmacao falsa sobre o repositorio -- e o CI vai dizer o mesmo.*")
    print("")
    print("  Corre isto e volta a fazer o commit:")
    print("")
    for d, _ in divergentes:
        print("    " + d["regenerar"])
    print("")
    print("  Depois: git add " + " ".join(a for d, _ in divergentes for a in d["artefactos"]))
    print("  " + "=" * 74)
    return 1


if __name__ == "__main__":
    sys.exit(main())
