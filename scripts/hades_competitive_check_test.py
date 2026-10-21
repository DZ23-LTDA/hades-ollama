"""Fault injection do `hades_competitive_check.py` -- `E-1109`.

***O QUE PROVA.*** *O checador passa com a matriz real. Mas tem de **falhar** quando a matriz mente ou omite.*

=> *injeta 3 defeitos numa **copia**:*

  1. **remover uma referencia do prompt** (`lemonade`) do CSV => (3) falha;
  2. **marcar uma linha `SUPERIOR_MEASURED` sem benchmark** => (4) falha;
  3. **tirar a seccao das correccoes factuais** do `.md` => (6) falha.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import csv
import io
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
MISSAO = RAIZ / "docs" / "mission"
CHECKER = RAIZ / "scripts" / "hades_competitive_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    d = tmp / "docs" / "mission"
    d.mkdir(parents=True)
    for nome in ("HADES_COMPETITIVE_CAPABILITY_MATRIX.md", "HADES_COMPETITIVE_CAPABILITY_MATRIX.csv"):
        shutil.copyfile(MISSAO / nome, d / nome)
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "scripts" / "_check.py"
    alvo.parent.mkdir(parents=True, exist_ok=True)
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
    md = raiz / "docs" / "mission" / "HADES_COMPETITIVE_CAPABILITY_MATRIX.md"
    cs = raiz / "docs" / "mission" / "HADES_COMPETITIVE_CAPABILITY_MATRIX.csv"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-6] if saida.strip() else "")[:70])

    orig_csv = io.open(cs, encoding="utf-8").read()
    orig_md = io.open(md, encoding="utf-8").read()

    # ---- (1) remover uma referencia do prompt ----
    linhas = list(csv.reader(io.StringIO(orig_csv)))
    sem = [l for l in linhas if not (len(l) > 3 and "lemonade" in l[3].lower())]
    assert len(sem) < len(linhas), "a remocao nao mudou nada"
    buf = io.StringIO()
    csv.writer(buf).writerows(sem)
    io.open(cs, "w", encoding="utf-8", newline="\n").write(buf.getvalue())
    rc, saida = correr(raiz)
    T("(1) sem a referencia `lemonade` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(cs, "w", encoding="utf-8", newline="\n").write(orig_csv)

    # ---- (2) SUPERIOR_MEASURED sem benchmark ----
    linhas = list(csv.reader(io.StringIO(orig_csv)))
    cab = linhas[0]
    i_est = cab.index("estado_hades")
    i_prov = cab.index("prova_hades")
    for l in linhas[1:]:
        if len(l) > max(i_est, i_prov):
            l[i_est] = "SUPERIOR_MEASURED"
            l[i_prov] = "nos somos melhores"
            break
    buf = io.StringIO()
    csv.writer(buf).writerows(linhas)
    io.open(cs, "w", encoding="utf-8", newline="\n").write(buf.getvalue())
    rc, saida = correr(raiz)
    T("(2) `SUPERIOR_MEASURED` sem benchmark -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(cs, "w", encoding="utf-8", newline="\n").write(orig_csv)

    # ---- (3) sem a seccao das correccoes factuais ----
    import re
    md2 = md
    sem_md = re.sub(r"## 16\..*?(?=## 17\.)", "", orig_md, flags=re.S)
    assert sem_md != orig_md, "a remocao da seccao 16 nao mudou nada"
    io.open(md, "w", encoding="utf-8", newline="\n").write(sem_md)
    rc, saida = correr(raiz)
    T("(3) sem a seccao das correccoes factuais -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(md, "w", encoding="utf-8", newline="\n").write(orig_md)

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
