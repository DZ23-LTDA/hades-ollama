"""Fault injection do `hades_harness_check.py` -- `E-1098`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas um checador que so' passa nao vale nada: tem de
**falhar** quando uma das 7 capacidades perde o teste.*

=> *injeta 3 defeitos numa **copia** do `internal/agent` (so' os nomes, nao os corpos):*

  1. apagar todos os testes de **cancelamento**        => (1) falha;
  2. apagar `cancel_cross_instance_test.go`             => (2) falha;
  3. apagar a **sonda opt-in** do CLI                   => (4b2) falha.

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
CHECKER = RAIZ / "scripts" / "hades_harness_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    dest = tmp / "internal" / "agent"
    dest.mkdir(parents=True)
    # *copiar TAMBEM os `.go` -- o checador verifica o `planner.go` e as referencias ao Ollama, e uma
    # copia so' com `_test.go` acusaria falhas que sao da copia, nao do codigo.*
    for p in AGENTE.glob("*.go"):
        shutil.copyfile(p, dest / p.name)
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

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])

    # ---- (1) apagar os testes de cancelamento ----
    removidos = 0
    for p in list(dest.glob("*_test.go")):
        t = io.open(p, encoding="utf-8").read()
        # *renomear as funcoes `Test*Cancel*` para nao contarem*
        novo = re.sub(r"func (Test\w*Cancel\w*)", r"func X\1", t)
        if novo != t:
            io.open(p, "w", encoding="utf-8", newline="\n").write(novo)
            removidos += 1
    rc, saida = correr(raiz)
    T("(1) sem testes de cancelamento -> FALHA", rc != 0 and removidos > 0,
      "%d ficheiro(s) alterados, exit=%d" % (removidos, rc))
    # *restaurar*
    for p in list(dest.glob("*_test.go")):
        t = io.open(p, encoding="utf-8").read()
        io.open(p, "w", encoding="utf-8", newline="\n").write(re.sub(r"func X(Test\w+)", r"func \1", t))

    # ---- (2) apagar o cancel_cross_instance_test.go ----
    alvo = dest / "cancel_cross_instance_test.go"
    if alvo.is_file():
        alvo.unlink()
    rc, saida = correr(raiz)
    T("(2) sem cancelamento entre instancias -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    # *restaurar do repo real*
    shutil.copyfile(AGENTE / "cancel_cross_instance_test.go", alvo)

    # ---- (3) apagar a sonda OptIn do CLI ----
    cli = list(dest.glob("cli_harness*_test.go"))
    for p in cli:
        t = io.open(p, encoding="utf-8").read()
        io.open(p, "w", encoding="utf-8", newline="\n").write(
            re.sub(r"func (Test\w*OptIn\w*)", r"func X\1", t))
    rc, saida = correr(raiz)
    T("(3) sem a sonda OptIn -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- contraprova: a copia ORIGINAL ----
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
