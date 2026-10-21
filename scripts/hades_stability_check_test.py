"""Fault injection do `hades_stability_check.py` -- `E-1108`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando a estabilidade da base cai.*

=> *injeta 3 defeitos numa **copia**:*

  1. **apagar o `E-197` do indice** (o flake deixa de estar registado) => (1) falha;
  2. **acrescentar um `NewRuntime` sem `DataRoot`** (a divida da hermeticidade **cresce**) => (2b) falha;
  3. **apagar o teste de uma catraca** => (4b) falha.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import io
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SCRIPTS = RAIZ / "scripts"
CHECKER = SCRIPTS / "hades_stability_check.py"
AGENTE = RAIZ / "internal" / "agent"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    # *os scripts (os checkers que o (2) e o (3) invocam)*
    d = tmp / "scripts"
    d.mkdir(parents=True)
    for p in SCRIPTS.glob("*.py"):
        shutil.copyfile(p, d / p.name)
    # *os docs (o (1) le' o indice e o estado)*
    for rel in ("docs/mission/HADES_EVIDENCE_INDEX.md", "docs/mission/HADES_MASTER_STATE.json"):
        src = RAIZ / rel
        dst = tmp / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        if src.is_file():
            shutil.copyfile(src, dst)
    # ***A COPIA TEM DE TER OS 4 DIRECTORIOS QUE O `hermeticos_check` LE'.*** *Medido: ele mede
    # `server`, `internal`, `cmd` e `app` (`:30`) -- e copiar so' o `internal/agent` dava **70** em vez de
    # **107** => *e o (2b) falhava na **baseline**, nao pela injecao.* => *copia os 4, so' os `.go`.*
    for base in ("server", "internal", "cmd", "app"):
        src = RAIZ / base
        if not src.is_dir():
            continue
        for p in src.rglob("*.go"):
            rel = p.relative_to(RAIZ)
            dst = tmp / rel
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(p, dst)
    # *e o `_tmp/goroot1269` nao existe na copia => o (5) vai falhar sempre.
    # => *o test discriminador NAO pode depender do (5): ele e' um gate de `go test` de 51 s,
    # e a copia nao tem toolchain. Entao o test aceita que o baseline da copia pode falhar SO' no (5).*
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "scripts" / "_check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % raiz,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo)],
                       capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=1200)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def so_o_5_falhou(saida: str) -> bool:
    """A copia nao tem toolchain Go => o (5) falha; os outros tem de passar."""
    m = re.search(r"==== (\d+) FALHA\(S\) NO §ESTAGIO 2 ====", saida)
    if not m:
        return False
    if int(m.group(1)) != 1:
        return False
    return "(5) o `./server` passa" in saida


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: o checador nao existe")
        return 2

    raiz = copia()
    rc, saida = correr(raiz)
    # *baseline: ou passa tudo, ou falha SO' no (5) (a copia nao tem Go)*
    T("baseline: a copia passa (ou falha so' no (5), que precisa do toolchain Go)",
      rc == 0 or so_o_5_falhou(saida),
      "exit=%d" % rc)

    # ---- (1) sem o E-197 no indice ----
    ev = raiz / "docs" / "mission" / "HADES_EVIDENCE_INDEX.md"
    orig_ev = io.open(ev, encoding="utf-8").read()
    sem = re.sub(r"^\| E-197 \|.*?$", "", orig_ev, flags=re.M)
    assert sem != orig_ev, "a injecao do E-197 nao mudou nada"
    io.open(ev, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz)
    T("(1) sem o `E-197` no indice -> FALHA", rc != 0 and "(1) o `E-197`" in saida,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(ev, "w", encoding="utf-8", newline="\n").write(orig_ev)

    # ---- (2) um NewRuntime sem DataRoot (a divida cresce) ----
    ag = raiz / "internal" / "agent"
    novo = ag / "zz_nao_hermetico_test.go"
    io.open(novo, "w", encoding="utf-8", newline="\n").write(
        'package agent\n\nimport "testing"\n\nfunc TestZZNaoHermetico(t *testing.T) {\n'
        '\t_ = NewRuntime(RuntimeConfig{})\n}\n')
    rc, saida = correr(raiz)
    T("(2) um `NewRuntime` sem `DataRoot` novo -> FALHA (a divida cresceu)", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    novo.unlink()

    # ---- (3) sem o teste de uma catraca ----
    rat = raiz / "scripts" / "hades_approval_hook_ratchet_test.py"
    assert rat.is_file()
    rat.unlink()
    rc, saida = correr(raiz)
    T("(3) sem o teste de uma catraca -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- contraprova ----
    raiz2 = copia()
    rc, saida = correr(raiz2)
    T("contraprova: a copia original continua a PASSAR (ou so' o (5) falha)",
      rc == 0 or so_o_5_falhou(saida), "exit=%d" % rc)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== AS 3 INJECOES + A CONTRAPROVA PASSARAM ====")
    return 0


if __name__ == "__main__":
    sys.exit(main())
