"""Fault injection do `hades_security_check.py` -- `E-1104`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando uma protecao cai.*

=> *injeta 3 defeitos numa **copia**:*

  1. **apagar os testes de tenant** => (1) falha;
  2. **tirar o `hades_tenant_scope_check.py` do gates** => (3b) falha;
  3. **meter um segredo falso no codigo** => (5) falha.

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
SERVER = RAIZ / "server"
SCRIPTS = RAIZ / "scripts"
CHECKER = SCRIPTS / "hades_security_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    for base in ("internal/agent", "server", "scripts"):
        src = RAIZ / base
        if not src.is_dir():
            continue
        dest = tmp / base
        dest.mkdir(parents=True, exist_ok=True)
        for p in src.glob("*"):
            if p.is_file():
                shutil.copyfile(p, dest / p.name)
    # *o LIMITACOES.md*
    lim = RAIZ / "docs" / "audit" / "LIMITACOES.md"
    if lim.is_file():
        d = tmp / "docs" / "audit" / "LIMITACOES.md"
        d.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(lim, d)
    return tmp


def correr(raiz: Path) -> tuple[int, str]:
    alvo = raiz / "scripts" / "_check.py"
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
    ag = raiz / "internal" / "agent"
    gates = raiz / "scripts" / "hades_local_gates.ps1"

    rc, saida = correr(raiz)
    T("baseline: a copia passa", rc == 0,
      (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])

    # ---- (1) apagar os testes de tenant ----
    # ***A INJECCAO TEM DE APAGAR OS NOMES, NAO SO' MUDAR O PREFIXO.*** *Renomear `Test...Tenant` para
    # `XTest...Tenant` deixava o `Tenant` no nome, e o critério (que procura a **palavra**) continuava a
    # passar -- medido: a injecao (1) dizia "exit=0".* => *apagar a palavra do nome.*
    # ***A INJECCAO TEM DE COBRIR OS DOIS DIRECTORIOS QUE O CHECADOR LE'.*** *O `testes_que_casam`
    # procura em `internal/agent` **e** em `server` => *injectar so' no `internal/agent` deixava os testes do
    # `server` a casar, e o checador continuava a passar (medido: 12 ficheiros).*
    n = 0
    for p in list(ag.glob("*_test.go")) + list((raiz / "server").glob("*_test.go")):
        t = io.open(p, encoding="utf-8").read()
        novo = re.sub(r"func Test(\w*)Tenant(\w*)\(", r"func Test\1NAdaT\2(", t)
        novo = re.sub(r"func Test(\w*)tenant(\w*)\(", r"func Test\1NAdaT\2(", novo)
        if novo != t:
            io.open(p, "w", encoding="utf-8", newline="\n").write(novo)
            n += 1
    rc, saida = correr(raiz)
    T("(1) sem os testes de tenant -> FALHA", rc != 0 and n > 0,
      "%d ficheiro(s), exit=%d" % (n, rc))

    # ---- (2) tirar o tenant_scope do gates ----
    raiz2 = copia()
    g = raiz2 / "scripts" / "hades_local_gates.ps1"
    gt = io.open(g, encoding="utf-8").read()
    gt2 = gt.replace("hades_tenant_scope_check.py", "OUTRO.py")
    assert gt2 != gt, "a substituicao nao mudou nada"
    io.open(g, "w", encoding="utf-8", newline="\n").write(gt2)
    rc, saida = correr(raiz2)
    T("(2) sem o tenant_scope no gates -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- (3) meter um segredo no codigo (nao num teste) ----
    raiz3 = copia()
    alvo = raiz3 / "internal" / "agent" / "zz_segredo.go"
    io.open(alvo, "w", encoding="utf-8", newline="\n").write(
        'package agent\n\nconst chave = "sk-abcdefghijklmnopqrstuvwx"\n')
    rc, saida = correr(raiz3)
    T("(3) um segredo em texto plano -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")

    # ---- contraprova ----
    raiz4 = copia()
    rc, saida = correr(raiz4)
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
