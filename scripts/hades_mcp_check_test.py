"""Fault injection do `hades_mcp_check.py` -- `E-1106`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando o cliente MCP deixa de ser
real.*

=> *injeta 3 defeitos numa **copia** do `internal/agent`:*

  1. **apagar o `exec.Command`** (o spawn deixa de existir -- o stdio morre) => (2) falha;
  2. **apagar o `unsafeEgressIP`** (o SSRF deixa de estar guardado) => (3c) falha;
  3. **apagar a `mcp_command_policy.go`** (o comando deixa de ser estruturado) => (4) falha.

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
CHECKER = RAIZ / "scripts" / "hades_mcp_check.py"

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
    # *o servidor tambem (o (6d) le' o bootstrap dele)*
    srv = tmp / "server" / "agent_mcp_bootstrap_test.go"
    srv.parent.mkdir(parents=True, exist_ok=True)
    orig = RAIZ / "server" / "agent_mcp_bootstrap_test.go"
    if orig.is_file():
        shutil.copyfile(orig, srv)
    # *e o estado (o (7) le' o capability do estagio 7)*
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
      (saida.strip().splitlines()[-3] if saida.strip() else "")[:70])

    # ---- (1) o exec.Command ----
    mcp = ag / "mcp.go"
    original = io.open(mcp, encoding="utf-8").read()
    sem = re.sub(r"exec\.Command(Context)?\(", "NAO_E_COMMAND(", original)
    assert sem != original, "a injecao do exec.Command nao mudou nada"
    io.open(mcp, "w", encoding="utf-8", newline="\n").write(sem)
    rc, saida = correr(raiz)
    T("(1) sem o `exec.Command` -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(mcp, "w", encoding="utf-8", newline="\n").write(original)

    # ---- (2) o unsafeEgressIP ----
    rem = ag / "mcp_remote.go"
    orig_rem = io.open(rem, encoding="utf-8").read()
    sem2 = orig_rem.replace("unsafeEgressIP", "NAO_E_EGRESS")
    assert sem2 != orig_rem
    io.open(rem, "w", encoding="utf-8", newline="\n").write(sem2)
    rc, saida = correr(raiz)
    T("(2) sem o `unsafeEgressIP` (o SSRF) -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(rem, "w", encoding="utf-8", newline="\n").write(orig_rem)

    # ---- (3) a politica de comando ----
    pol = ag / "mcp_command_policy.go"
    assert pol.is_file()
    pol.unlink()
    rc, saida = correr(raiz)
    T("(3) sem a `mcp_command_policy.go` -> FALHA", rc != 0,
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
