"""Fault injection da catraca dos testes nao-hermeticos -- `E-1094`.

***O QUE PROVA.*** *A catraca passa com a baseline 107. Mas uma baseline que **so' passa** nao vale nada:
tem de **falhar quando o numero sobe**.*

=> *injeta um ficheiro `_test.go` novo, numa **copia do repositorio** (nao no real), com um `NewRuntime`
sem `DataRoot`, e exige que a catraca o apanhe e que o numero passe de 107 para 108.*

***E A CONTRAPROVA.*** *Sem o ficheiro injectado, a catraca continua a passar com 107.*
"""
import io
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
CHECKER = RAIZ / "scripts" / "hades_testes_hermeticos_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def correr(raiz: Path) -> tuple[int, str]:
    """Corre a catraca com o `RAIZ` apontado a uma copia."""
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


def copia_minima() -> Path:
    """Uma copia com a estrutura de pastas e os testes reais (so' os `.go`)."""
    tmp = Path(tempfile.mkdtemp())
    for d in ("server", "internal", "cmd", "app"):
        origem = RAIZ / d
        if not origem.is_dir():
            continue
        for p in origem.rglob("*_test.go"):
            rel = p.relative_to(RAIZ)
            dest = tmp / rel
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(p, dest)
    return tmp


def main() -> int:
    if not CHECKER.is_file():
        print("  ERRO: a catraca nao existe")
        return 2

    raiz = copia_minima()

    rc, saida = correr(raiz)
    T("baseline: a copia passa com a divida declarada", rc == 0,
      (saida.strip().splitlines()[-1] if saida.strip() else "")[:70])
    m = re.search(r"TESTES NAO-HERMETICOS: (\d+)", saida)
    n_antes = int(m.group(1)) if m else -1
    T("e o numero e' 107", n_antes == 107, "n=%d" % n_antes)

    # ---- injecao: um teste NOVO sem DataRoot ----
    novo = raiz / "server" / "zz_injecao_test.go"
    io.open(novo, "w", encoding="utf-8", newline="\n").write(
        'package server\n\n'
        'import "testing"\n\n'
        'func TestInjecaoNaoHermetica(t *testing.T) {\n'
        '\trt, err := agent.NewRuntime(agent.RuntimeConfig{WorkspaceRoot: t.TempDir()})\n'
        '\tif err != nil {\n'
        '\t\tt.Fatal(err)\n'
        '\t}\n'
        '\t_ = rt\n'
        '}\n'
    )
    rc, saida = correr(raiz)
    T("(1) um teste NOVO sem DataRoot -> a catraca FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    m = re.search(r"CRESCIMENTO DE DIVIDA: (\d+)", saida)
    n_depois = int(m.group(1)) if m else -1
    T("(1b) e o numero subiu de 107 para 108", n_depois == 108, "n=%d" % n_depois)

    # ---- contraprova: apagar o injectado e voltar a 107 ----
    novo.unlink()
    rc, saida = correr(raiz)
    T("contraprova: sem o injectado, volta a PASSAR", rc == 0)
    m = re.search(r"TESTES NAO-HERMETICOS: (\d+)", saida)
    n_fim = int(m.group(1)) if m else -1
    T("contraprova: e o numero voltou a 107", n_fim == 107, "n=%d" % n_fim)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  ==== A INJECAO + AS CONTRAPROVAS PASSARAM ====")
    return 0


if __name__ == "__main__":
    sys.exit(main())
