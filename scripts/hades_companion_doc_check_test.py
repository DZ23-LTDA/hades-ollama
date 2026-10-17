#!/usr/bin/env python3
"""Fault injection da catraca da documentacao do Companion (`E-1090`).

***O QUE ESTE TESTE PROVA.*** O `hades_companion_doc_check.py` passa no documento real. Mas *um checker
que so' passa nao vale nada* -- a licao do `E-1082`/`E-1078`: **um teste que passa com o defeito nao
discrimina.**

=> *aqui injecto 4 defeitos **no documento real, numa copia descartavel**, e exijo que o checker os
apanhe:*

  1. **apagar um estado** da tabela                          => o estado deixa de estar documentado;
  2. **trocar a chave** das preferencias por outra            => o documento deixa de citar a chave REAL;
  3. **apagar o mapa para os 14**                             => os nomes do prompt desaparecem;
  4. **esconder as limitacoes**                               => o documento promete mais do que ha'.

E o **contraprova**: o documento ORIGINAL tem de continuar a passar.
"""
import io
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
DOC = RAIZ / "docs" / "COMPANION.md"
CHECKER = RAIZ / "scripts" / "hades_companion_doc_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr(doc: Path) -> tuple[int, str]:
    """Corre o checker contra um documento ARBITRARIO.

    *O checker le' o caminho fixo (`docs/COMPANION.md`). Para o testar contra uma copia, o caminho e'
    substituido no FONTE do checker -- numa copia do checker, nao no original.*
    """
    tmp = Path(tempfile.mkdtemp())
    src = tmp / "docs" / "COMPANION.md"
    src.parent.mkdir(parents=True)
    shutil.copyfile(doc, src)
    # copia do checker com o DOC apontado para a copia
    alvo = tmp / "check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'DOC = RAIZ / "docs" / "COMPANION.md"',
        'DOC = __import__("pathlib").Path(r"%s")' % src,
    )
    # e as fontes de codigo continuam a apontar para o repo real
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % RAIZ,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo)],
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not DOC.is_file():
        print("  ERRO: docs/COMPANION.md nao existe")
        return 2

    # ---------- baseline: o documento REAL passa ----------
    rc, saida = correr(DOC)
    T("baseline: o documento real PASSA", rc == 0,
      (saida.splitlines()[-1] if saida.strip() else "")[:60])

    original = io.open(DOC, encoding="utf-8").read()

    # ---------- (1) apagar um estado da tabela ----------
    alvo1 = "| **A planear** |"
    if alvo1 not in original:
        print("  ERRO: nao encontrei a linha 'A planear'")
        return 2
    t = original.replace("| **A planear** |", "| (removido) |", 1).replace("a_planear", "X_planear")
    tmp = Path(tempfile.mkdtemp()) / "d1.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write(t)
    rc, _ = correr(tmp)
    T("(1) estado `a_planear` apagado -> o checker FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---------- (2) trocar a chave das preferencias ----------
    t = original.replace("hades.companion.preferencias.v1", "hades.companion.preferencias.OUTRA")
    tmp = Path(tempfile.mkdtemp()) / "d2.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write(t)
    rc, _ = correr(tmp)
    T("(2) chave das preferencias errada -> o checker FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---------- (3) apagar o mapa para os 14 ----------
    linhas = [l for l in original.split("\n") if not any(
        p in l for p in ("DISABLED", "LISTENING", "WAITING_TOOL", "WAITING_USER", "PAUSED", "OFFLINE"))]
    tmp = Path(tempfile.mkdtemp()) / "d3.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write("\n".join(linhas))
    rc, _ = correr(tmp)
    T("(3) mapa para os 14 apagado -> o checker FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---------- (4) esconder as limitacoes ----------
    t = original.replace("## Limitações declaradas", "## Notas")
    if "jornada completa" in t:
        t = t.replace("jornada completa", "X completa")
    tmp = Path(tempfile.mkdtemp()) / "d4.md"
    io.open(tmp, "w", encoding="utf-8", newline="\n").write(t)
    rc, _ = correr(tmp)
    T("(4) limitacoes escondidas -> o checker FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---------- contraprova: o original continua a passar ----------
    rc, _ = correr(DOC)
    T("contraprova: o documento ORIGINAL continua a PASSAR", rc == 0)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== AS 4 INJECOES + A CONTRAPROVA PASSARAM ====")
    return 0


if __name__ == "__main__":
    sys.exit(main())
