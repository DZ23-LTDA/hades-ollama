"""Fault injection do ambiente T2 -- `E-1093`.

***O QUE PROVA.*** *O `hades_t2_contexto.py --check` passa no documento real. Mas um `--check` que so'
passa nao vale nada: tem de **falhar** quando um campo falta.*

=> *injeta 3 defeitos no documento real, numa copia descartavel, e exige que o `--check` os apanhe:*

  1. apagar a seccao de **hardware** (CPU/RAM/GPU)      => o ambiente deixa de ter a maquina;
  2. apagar a seccao do **Ollama** (versao)              => o ambiente deixa de ter a versao;
  3. apagar as **lacunas** (warm/cold, separacao, erro)  => o documento passa a prometer mais do que ha'.

E a **contraprova**: o documento original continua a passar.
"""
import io
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOC = RAIZ / "docs" / "parity" / "HADES_T2_AMBIENTE.md"
CHECKER = RAIZ / "scripts" / "hades_t2_contexto.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr(doc: Path) -> tuple[int, str]:
    tmp = Path(tempfile.mkdtemp())
    src = tmp / "docs" / "parity" / "HADES_T2_AMBIENTE.md"
    src.parent.mkdir(parents=True)
    shutil.copyfile(doc, src)
    alvo = tmp / "check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'SAIDA = RAIZ / "docs" / "parity" / "HADES_T2_AMBIENTE.md"',
        'SAIDA = __import__("pathlib").Path(r"%s")' % src,
    )
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % RAIZ,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo), "--check"],
                       cwd=str(RAIZ), capture_output=True, text=True, encoding="utf-8",
                       errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not DOC.is_file():
        print("  ERRO: HADES_T2_AMBIENTE.md nao existe")
        return 2
    original = io.open(DOC, encoding="utf-8").read()

    rc, saida = correr(DOC)
    T("baseline: o ambiente real PASSA", rc == 0,
      (saida.splitlines()[-1] if saida.strip() else "")[:70])

    # (1) sem hardware
    linhas = [l for l in original.split("\n") if not any(
        k in l for k in ("| CPU |", "| RAM |", "| GPU |", "## Hardware"))]
    tmp = Path(tempfile.mkdtemp()) / "d1.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, _ = correr(tmp)
    T("(1) hardware apagado -> o checker FALHA", rc != 0, "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # (2) sem Ollama
    linhas = [l for l in original.split("\n") if not any(
        k in l for k in ("| Versão |", "## Ollama", "| Endpoint |"))]
    tmp = Path(tempfile.mkdtemp()) / "d2.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, _ = correr(tmp)
    T("(2) Ollama apagado -> o checker FALHA", rc != 0, "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # (3) sem as lacunas
    linhas = [l for l in original.split("\n") if not any(
        k in l for k in ("warm/cold", "separação de tempos", "| erro |", "comparação externa"))]
    tmp = Path(tempfile.mkdtemp()) / "d3.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, _ = correr(tmp)
    T("(3) lacunas apagadas -> o checker FALHA", rc != 0, "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # contraprova
    rc, _ = correr(DOC)
    T("contraprova: o ambiente ORIGINAL continua a PASSAR", rc == 0)

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
