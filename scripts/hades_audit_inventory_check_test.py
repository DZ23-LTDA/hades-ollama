"""Fault injection do `hades_audit_inventory_check.py` -- `E-1100`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando o inventario perde uma
plataforma, um workflow ou o aviso.*

=> *injeta 3 defeitos numa **copia** do repositorio (so' o que o checador le'):

  1. **apagar a linha do Android** da matriz => (1b) falha;
  2. **apagar o aviso de um documento** (o `E-1100` e o aviso, no `HADES_NEXT_GOAL_READINESS.md`) =>
     (2) falha;
  3. **acrescentar um workflow novo nao citado** => (3b) falha.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import io
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOCS = RAIZ / "docs"
WORKFLOWS = RAIZ / ".github" / "workflows"
CHECKER = RAIZ / "scripts" / "hades_audit_inventory_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    """Copia o `docs/` (os `.md`), o `.github/workflows` e a matriz."""
    tmp = Path(tempfile.mkdtemp())
    for p in DOCS.rglob("*.md"):
        rel = p.relative_to(RAIZ)
        dest = tmp / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(p, dest)
    for p in WORKFLOWS.glob("*.y*ml"):
        dest = tmp / ".github" / "workflows" / p.name
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(p, dest)
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "_check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % raiz,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo)],
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: o checador nao existe")
        return 2

    raiz = copia()
    matriz = raiz / "docs" / "mission" / "HADES_PLATFORM_MATRIX.md"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])

    # ---- (1) apagar o Android da matriz ----
    original = io.open(matriz, encoding="utf-8").read()
    linhas = [l for l in original.split("\n") if "Android" not in l]
    io.open(matriz, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, saida = correr(raiz)
    T("(1) sem a plataforma Android -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(matriz, "w", encoding="utf-8", newline="\n").write(original)

    # ---- (2) apagar o aviso do documento destrutivo ----
    alvo = raiz / "docs" / "mission" / "HADES_NEXT_GOAL_READINESS.md"
    bak = io.open(alvo, encoding="utf-8").read()
    # *tirar o bloco do aviso: as linhas com E-1100/HISTORICO/PROIBIDO e a citacao adjacente*
    linhas = []
    pular = False
    for l in bak.split("\n"):
        if "AVISO (E-1100" in l or "E-1100" in l:
            pular = True
            continue
        if pular and l.startswith(">"):
            continue
        pular = False
        linhas.append(l)
    io.open(alvo, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, saida = correr(raiz)
    T("(2) sem o aviso de historico -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(bak)

    # ---- (3) acrescentar um workflow nao citado ----
    novo_wf = raiz / ".github" / "workflows" / "zz_injecao.yaml"
    io.open(novo_wf, "w", encoding="utf-8", newline="\n").write("name: injecao\non: push\n")
    rc, saida = correr(raiz)
    T("(3) um workflow nao citado -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    novo_wf.unlink()

    # ---- contraprova ----
    raiz2 = copia()
    rc, saida = correr(raiz2)
    T("contraprova: a copia original continua a PASSAR", rc == 0)

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
