"""Fault injection do `hades_architecture_check.py` -- `E-1102`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando a arquitectura deixa de ser
canonica.*

=> *injeta 3 defeitos numa **copia** do `internal/agent` (e dos docs):*

  1. **acrescentar um SEGUNDO `type MissionControl struct`** => *o segundo motor que o prompt proibe* =>
     (1d) falha;
  2. **apagar os testes de contrato** => (2) falha;
  3. **apagar a seccao `## Ownership`** da arquitectura => (3b) falha.

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
AGENTE = RAIZ / "internal" / "agent"
ARQ = RAIZ / "docs" / "mission" / "HADES_ARCHITECTURE.md"
API = RAIZ / "docs" / "agentic" / "API.md"
CHECKER = RAIZ / "scripts" / "hades_architecture_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    dest = tmp / "internal" / "agent"
    dest.mkdir(parents=True)
    for p in AGENTE.glob("*.go"):
        shutil.copyfile(p, dest / p.name)
    for rel in ("docs/mission/HADES_ARCHITECTURE.md", "docs/agentic/API.md"):
        src = RAIZ / rel
        if src.is_file():
            d = tmp / rel
            d.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(src, d)
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
    dest = raiz / "internal" / "agent"
    arq = raiz / "docs" / "mission" / "HADES_ARCHITECTURE.md"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])

    # ---- (1) um SEGUNDO MissionControl (o motor paralelo) ----
    novo = dest / "zz_segundo_motor.go"
    io.open(novo, "w", encoding="utf-8", newline="\n").write(
        "package agent\n\n// o SEGUNDO motor -- o que o prompt proibe\ntype MissionControl struct {\n\tEstado string\n}\n")
    rc, saida = correr(raiz)
    T("(1) um segundo `MissionControl` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    novo.unlink()

    # ---- (2) apagar os testes de contrato ----
    apagados = 0
    for p in list(dest.glob("*contract*_test.go")):
        p.unlink()
        apagados += 1
    rc, saida = correr(raiz)
    T("(2) sem os testes de contrato -> FALHA", rc != 0 and apagados > 0,
      "%d ficheiro(s) removidos, exit=%d" % (apagados, rc))
    # *restaurar*
    for p in AGENTE.glob("*contract*_test.go"):
        shutil.copyfile(p, dest / p.name)

    # ---- (3) apagar a seccao Ownership ----
    original = io.open(arq, encoding="utf-8").read()
    sem = re.sub(r"## Ownership.*?(?=## )", "", original, flags=re.S)
    io.open(arq, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz)
    T("(3) sem a seccao Ownership -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(arq, "w", encoding="utf-8", newline="\n").write(original)

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
