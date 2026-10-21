"""Fault injection do `hades_agents_check.py` -- `E-1099`.

***O QUE PROVA.*** *O checador passa com o estado real. Mas tem de **falhar** quando um limite do §ESTAGIO 9
desaparece.*

=> *injeta 3 defeitos numa **copia** do `internal/agent` (os `.go` e os `_test.go`):*

  1. **remover o semaforo** do `swarm.go` (`semaphore := make(chan struct{}, job.Budget.MaxAgents)`) =>
     *`MaxAgents` fica declarado e **nao aplicado*** => (1c) falha;
  2. **apagar o `swarm_worktree_isolation_test.go`** => (3) falha;
  3. **apagar o `Consume`/concorrencia de todos os testes** => (5d) falha.

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
CHECKER = RAIZ / "scripts" / "hades_agents_check.py"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def copia() -> Path:
    tmp = Path(tempfile.mkdtemp())
    dest = tmp / "internal" / "agent"
    dest.mkdir(parents=True)
    # *os `.go` E os `_test.go` -- o `E-119`: uma copia sem o mundo que mede acusa o medidor.*
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

    # ---- (1) remover o semaforo ----
    swarm = dest / "swarm.go"
    original = io.open(swarm, encoding="utf-8").read()
    # ***A INJECCAO TEM DE REMOVER OS DOIS SINAIS.*** *O checador aceita (a) o `make(chan struct{},
    # MaxAgents)` **ou** (b) o `sem <- struct`. => *remover so' o (a) deixava o (b), e o checador
    # continuava a passar -- a injecao nao discriminava (medido).* => *remover os dois: o semaforo
    # vira um `chan` sem buffer e o envio vira um no-op.*
    novo = original
    # (a) o tamanho do canal deixa de vir do MaxAgents
    novo = re.sub(r"make\(chan struct\{\}, job\.Budget\.MaxAgents\)",
                  "make(chan struct{}, 0)", novo)
    # (b) o envio deixa de ser o padrao `semaphore <- struct{}{}`
    # *O regex tem de ter TRES `{}` -- `struct{}`, `{}` e o `:` --, nao dois. Foi esse o erro que fez a
    # injecao nao se aplicar e o teste "passar" sem testar nada.*
    novo = re.sub(r"case semaphore <- struct\{\}\{\}:", "case true:", novo)
    # ***VERIFICAR QUE A INJECCAO MUDOU O TEXTO.*** *Uma injecao que nao se aplica nao testa nada.*
    assert novo != original, "a injecao (1) nao alterou o swarm.go"
    assert "job.Budget.MaxAgents" not in novo, "o canal nao deixou de vir do MaxAgents"
    assert "struct{}{}" not in novo, "o envio no semaforo nao foi substituido"
    io.open(swarm, "w", encoding="utf-8", newline="\n").write(novo)
    rc, saida = correr(raiz)
    T("(1) sem o semaforo -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    io.open(swarm, "w", encoding="utf-8", newline="\n").write(original)

    # ---- (2) apagar o isolamento de worktree ----
    wt = dest / "swarm_worktree_isolation_test.go"
    backup = io.open(wt, encoding="utf-8").read() if wt.is_file() else None
    if wt.is_file():
        wt.unlink()
    rc, saida = correr(raiz)
    T("(2) sem o isolamento de worktree -> FALHA", rc != 0,
      "FAILED" if rc != 0 else "PASSOU (nao discrimina!)")
    if backup is not None:
        io.open(wt, "w", encoding="utf-8", newline="\n").write(backup)

    # ---- (3) remover o Consume/concorrencia de todos os testes ----
    n = 0
    for p in dest.glob("*_test.go"):
        t = io.open(p, encoding="utf-8").read()
        t2 = re.sub(r"Consume|Concorren|concorren|Concurrency", "NAda", t)
        if t2 != t:
            io.open(p, "w", encoding="utf-8", newline="\n").write(t2)
            n += 1
    rc, saida = correr(raiz)
    T("(3) sem o Consume/a corrida -> FALHA", rc != 0 and n > 0,
      "%d ficheiro(s) alterados, exit=%d" % (n, rc))
    # *restaurar do repo real*
    for p in dest.glob("*_test.go"):
        shutil.copyfile(AGENTE / p.name, p)

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
