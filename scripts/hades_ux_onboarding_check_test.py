"""Fault injection do checador do §ESTAGIO 19 -- `E-1095`.

***O QUE PROVA.*** *O `hades_ux_onboarding_check.py` passa com o estado real. Mas um checador que **so'
passa** nao vale nada: tem de **falhar** quando o criterio falta.*

=> *injeta 3 defeitos no codigo real, **numa copia** do `src`, e exige que o checador os apanhe:*

  1. apagar o `/goal` do `ChatForm.tsx`            => o composer deixa de ter o comando;
  2. apagar o ficheiro `Onboarding.tsx`            => o onboarding deixa de existir;
  3. apagar o `aria-` do `ChatForm.tsx`           => a acessibilidade desaparece.

***E A CONTRAPROVA.*** *Sem o injectado, o checador continua a passar.*
"""
import io
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
SRC = RAIZ / "app" / "ui" / "app" / "src"
CHECKER = RAIZ / "scripts" / "hades_ux_onboarding_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia_src() -> Path:
    """Copia so' o `src` (os `.ts`/`.tsx`) para um tmp."""
    tmp = Path(tempfile.mkdtemp())
    for p in SRC.rglob("*"):
        if p.suffix not in (".ts", ".tsx"):
            continue
        if "node_modules" in str(p):
            continue
        rel = p.relative_to(SRC)
        dest = tmp / "app" / "ui" / "app" / "src" / rel
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(p, dest)
    # *o inventario de marca tem de vir tambem*
    inv = RAIZ / "docs" / "mission" / "HADES_BRAND_INVENTORY.md"
    if inv.is_file():
        dest = tmp / "docs" / "mission" / "HADES_BRAND_INVENTORY.md"
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(inv, dest)
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "_check.py"
    t = io.open(CHECKER, encoding="utf-8").read()
    t = t.replace(
        'RAIZ = Path(__file__).resolve().parent.parent',
        'RAIZ = __import__("pathlib").Path(r"%s")' % raiz,
    )
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(t)
    r = subprocess.run([sys.executable, "-X", "utf8", str(alvo), "--check"],
                       capture_output=True, text=True, encoding="utf-8", errors="replace")
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: o checador nao existe")
        return 2

    raiz = copia_src()
    src = raiz / "app" / "ui" / "app" / "src"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0, (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])

    # ---- (1) sem /goal no ChatForm ----
    chat = src / "components" / "ChatForm.tsx"
    original = io.open(chat, encoding="utf-8").read()
    io.open(chat, "w", encoding="utf-8", newline="\n").write(original.replace("/goal", "gol"))
    rc, saida = correr(raiz)
    T("(1) `/goal` apagado do ChatForm -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(chat, "w", encoding="utf-8", newline="\n").write(original)

    # ---- (2) sem Onboarding.tsx ----
    onb = src / "components" / "Onboarding.tsx"
    backup = io.open(onb, encoding="utf-8").read()
    onb.unlink()
    rc, saida = correr(raiz)
    T("(2) Onboarding.tsx apagado -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(onb, "w", encoding="utf-8", newline="\n").write(backup)

    # ---- (3) sem aria- no ChatForm ----
    io.open(chat, "w", encoding="utf-8", newline="\n").write(
        original.replace("aria-", "ariaX"))
    rc, saida = correr(raiz)
    T("(3) aria- apagado do ChatForm -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(chat, "w", encoding="utf-8", newline="\n").write(original)

    # ---- contraprova ----
    rc, saida = correr(raiz)
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
