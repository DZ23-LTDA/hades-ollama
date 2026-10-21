"""Fault injection do `hades_plugins_check.py` -- `E-1107`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando uma protecao cai.*

=> *injeta 3 defeitos numa **copia** do `internal/agent`:*

  1. **apagar o `maxPluginBundleBytes`** (o zip-bomb deixa de ter limite) => (4b) falha;
  2. **apagar os `PluginState*`** (instalar deixa de se distinguir de funcionar) => (5) falha;
  3. **apagar o `Stderr` do runtime** (a falha volta a ser silenciosa -- o `E-185`) => (7) falha.

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
CHECKER = RAIZ / "scripts" / "hades_plugins_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    dest = tmp / "internal" / "agent"
    dest.mkdir(parents=True)
    for p in AGENTE.glob("*"):
        if p.is_file():
            shutil.copyfile(p, dest / p.name)
    est = tmp / "docs" / "mission" / "HADES_MASTER_STATE.json"
    est.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(RAIZ / "docs" / "mission" / "HADES_MASTER_STATE.json", est)
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
    if not CHECKER.is_file() or not AGENTE.is_dir():
        print("  ERRO: o checador ou o pacote nao existem")
        return 2

    raiz = copia()
    ag = raiz / "internal" / "agent"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-6] if saida.strip() else "")[:70])

    # ---- (1) sem o maxPluginBundleBytes ----
    bundle = ag / "plugin_bundle.go"
    orig_b = io.open(bundle, encoding="utf-8").read()
    sem = orig_b.replace("maxPluginBundleBytes", "NAO_LIMITADO")
    assert sem != orig_b
    io.open(bundle, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz)
    T("(1) sem o `maxPluginBundleBytes` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(bundle, "w", encoding="utf-8", newline="\n").write(orig_b)

    # ---- (2) sem os PluginState ----
    status = ag / "plugin_status.go"
    orig_s = io.open(status, encoding="utf-8").read()
    sem2 = re.sub(r'PluginState\w+\s*=\s*"[A-Z_]+"', 'XState = "X"', orig_s)
    assert sem2 != orig_s, "a injecao dos estados nao mudou nada"
    io.open(status, "w", encoding="utf-8", newline="\n").write(sem2)
    rc, saida = correr(raiz)
    T("(2) sem os `PluginState*` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(status, "w", encoding="utf-8", newline="\n").write(orig_s)

    # ---- (3) sem o Stderr (a falha silenciosa volta) ----
    rt = ag / "plugin_runtime.go"
    orig_r = io.open(rt, encoding="utf-8").read()
    sem3 = orig_r.replace("Stderr", "NADA").replace("stderr", "nada")
    assert sem3 != orig_r, "a injecao do Stderr nao mudou nada"
    io.open(rt, "w", encoding="utf-8", newline="\n").write(sem3)
    rc, saida = correr(raiz)
    T("(3) sem o `Stderr` (a falha volta a ser silenciosa) -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

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
