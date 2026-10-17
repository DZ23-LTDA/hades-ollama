"""Fault injection do `hades_extension_check.py` -- `E-1105`.

***O QUE PROVA.*** *O checador passa com a extensao real. Mas tem de **falhar** quando uma protecao cai.*

=> *injeta 3 defeitos numa **copia** do `apps/hades-browser-bridge`:*

  1. **`incognito: "spanning"`** (o anonimo deixa de estar fora) => (2) falha;
  2. **`<all_urls>` nas `host_permissions`** (o content script passa a poder correr em toda a web) => (3c) falha;
  3. **um `eval` no service worker** => (6) falha.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import io
import json
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
EXT = RAIZ / "apps" / "hades-browser-bridge"
CHECKER = RAIZ / "scripts" / "hades_extension_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    dest = tmp / "apps" / "hades-browser-bridge"
    shutil.copytree(EXT, dest)
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
    if not CHECKER.is_file() or not EXT.is_dir():
        print("  ERRO: o checador ou a extensao nao existem")
        return 2

    raiz = copia()
    mf = raiz / "apps" / "hades-browser-bridge" / "manifest.json"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-3] if saida.strip() else "")[:70])

    # ---- (1) incognito: spanning ----
    original = io.open(mf, encoding="utf-8").read()
    m = json.loads(original)
    m["incognito"] = "spanning"
    io.open(mf, "w", encoding="utf-8", newline="\n").write(json.dumps(m, indent=2, ensure_ascii=False))
    rc, saida = correr(raiz)
    T("(1) `incognito: spanning` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- (2) <all_urls> nas host_permissions ----
    io.open(mf, "w", encoding="utf-8", newline="\n").write(original)
    m = json.loads(original)
    m["host_permissions"] = ["<all_urls>"]
    io.open(mf, "w", encoding="utf-8", newline="\n").write(json.dumps(m, indent=2, ensure_ascii=False))
    rc, saida = correr(raiz)
    T("(2) `<all_urls>` permanente -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- (3) eval no service worker ----
    io.open(mf, "w", encoding="utf-8", newline="\n").write(original)
    sw = raiz / "apps" / "hades-browser-bridge" / "src" / "service_worker.js"
    swt = io.open(sw, encoding="utf-8").read()
    io.open(sw, "w", encoding="utf-8", newline="\n").write(swt + "\nconst perigo = eval('1+1');\n")
    rc, saida = correr(raiz)
    T("(3) um `eval` no service worker -> FALHA", rc != 0,
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
