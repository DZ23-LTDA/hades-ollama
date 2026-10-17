"""Fault injection do `hades_findings_revalidation.py` -- `E-1111`.

***O QUE PROVA.*** *O revalidador passa com o codigo real. Mas tem de **falhar** quando uma correcao recua.*

=> *injeta 3 defeitos numa **copia**:*

  1. **tirar o `consumeLocked`** do `service_agent_approvals.go` => o P0 (`4266197772`) falha;
  2. **tirar o `finditer`** do `hades_test_clock_check.py` => os 2 findings de `.Add(` (`4266043779`/`787`)
     falham;
  3. **voltar a por a palavra `nao consumido`** no `step_routing.go` => `4266327646` falha.

***E A CONTRAPROVA.*** *Sem o injectado, o revalidador continua a passar.*
"""
import io
import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CHECKER = RAIZ / "scripts" / "hades_findings_revalidation.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    # *os 4 directorios que o revalidador le'*
    for base in ("internal", "scripts", "docs"):
        src = RAIZ / base
        if not src.is_dir():
            continue
        for p in src.rglob("*"):
            if not p.is_file():
                continue
            rel = p.relative_to(RAIZ)
            dst = tmp / rel
            dst.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(p, dst)
    return tmp


def correr(raiz: Path, modo: str = "") -> tuple[int, str]:
    alvo = raiz / "scripts" / "_check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % raiz,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    cmd = [sys.executable, "-X", "utf8", str(alvo)]
    if modo:
        cmd.append(modo)
    r = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: o revalidador nao existe")
        return 2

    raiz = copia()
    ap = raiz / "internal" / "agent" / "service_agent_approvals.go"
    clock = raiz / "scripts" / "hades_test_clock_check.py"
    routing = raiz / "internal" / "agent" / "step_routing.go"

    # *o baseline corre em modo --check (nao escreve)*
    rc, saida = correr(raiz, "--check")
    T("baseline: a copia passa (em --check)", rc == 0,
      (saida.strip().splitlines()[-6] if saida.strip() else "")[:70])

    orig_ap = io.open(ap, encoding="utf-8").read()
    orig_clock = io.open(clock, encoding="utf-8").read()
    orig_routing = io.open(routing, encoding="utf-8").read()

    # ---- (1) tirar o consumeLocked ----
    sem = orig_ap.replace("consumeLocked", "consumeSemLock")
    assert sem != orig_ap, "a injecao do consumeLocked nao mudou nada"
    io.open(ap, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz, "--check")
    T("(1) sem o `consumeLocked` -> FALHA no P0", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(ap, "w", encoding="utf-8", newline="\n").write(orig_ap)

    # ---- (2) tirar o finditer ----
    sem2 = orig_clock.replace("finditer", "re.search")
    assert sem2 != orig_clock, "a injecao do finditer nao mudou nada"
    io.open(clock, "w", encoding="utf-8", newline="\n").write(sem2)
    rc, saida = correr(raiz, "--check")
    T("(2) sem o `finditer` -> FALHA nos findings do `.Add(`", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(clock, "w", encoding="utf-8", newline="\n").write(orig_clock)

    # ---- (3) voltar a por a palavra ----
    novo = orig_routing + "\n// o roteamento por passo nao consumido nesta versao\n"
    io.open(routing, "w", encoding="utf-8", newline="\n").write(novo)
    rc, saida = correr(raiz, "--check")
    T("(3) a palavra `nao consumido` de volta -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(routing, "w", encoding="utf-8", newline="\n").write(orig_routing)

    # ---- contraprova ----
    raiz2 = copia()
    rc, saida = correr(raiz2, "--check")
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
