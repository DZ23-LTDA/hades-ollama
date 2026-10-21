"""`hades_ci_timeout_check.py` -- nenhum `go test` do CI que inclua o `internal/agent` fica sem `-timeout` (`E-1120`).

***O DEFEITO QUE ISTO IMPEDE.*** *O `go test` tem um default de **10 minutos**. O package `internal/agent` leva
**353,4 s** neste host (32 threads) com **1722** testes -- e o maior teste **so'zinho** leva **66,06 s**, porque
enche **5 000** produtos (`DefaultOnboardingLimits().MaxRows`) pelo caminho direto.*

=> *num runner `windows-latest` (**2 cores**) o package leva **2-3x** => **12-18 min** => o CI deu
`panic: test timed out after 10m0s` no `f813386bc`* => *e o `pr-gate`, que exige que os outros gates concluam com
`success`, falhou **em consequencia** (uma causa, dois vermelhos).*

***E POR QUE ISTO NAO E' "AUMENTAR O TIMEOUT PARA ESCONDER UM DEADLOCK".*** *O prompt proibe essa pratica, e com
razao. A distincao e' demonstravel:*
  - *um **deadlock** **nao termina** -- 25 min nao o salvam;*
  - *este e' um **teste de volume** que **termina em 353 s medidos**, e cujo proposito **e'** medir o custo do
    caminho direto (ver `commerce_budget_test.go`);*
  - *e o `-timeout 25m` fica **dentro** do `timeout-minutes: 30` do job -- nao e' um numero inventado, e' o tempo
    medido com margem.*

=> *este checker impede que um `go test` novo (ou uma copia do comando) volte a ficar sem `-timeout`.*
"""
import io
import re
import sys
from pathlib import Path

# A raiz vem do FICHEIRO, nao de um caminho absoluto da maquina de quem o escreveu.
# Antes: um `Path(r"C:\Users\<utilizador>\...\hades-ollama")` escrito a mao -- funcionava aqui e dava
# runner Linux, que foi como este instrumento falhou no CI (E-1517).
RAIZ = Path(__file__).resolve().parents[1]
WORKFLOWS = RAIZ / ".github" / "workflows"

# *o minimo aceitavel: o package leva 353 s e o runner e' 2-3x mais lento => >= 20 min*
MINIMO = 20

falhas: list[str] = []


def T(nome: str, ok: bool, det: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
    if not ok:
        falhas.append(nome)


def main() -> int:
    arquivos = sorted(WORKFLOWS.glob("*.y*ml"))
    T("(0) ha' workflows para verificar", len(arquivos) > 0, "%d ficheiros" % len(arquivos))

    # ---------- (1) cada `go test` que toca no internal/agent (ou `./...`) tem `-timeout` ----------
    sem_timeout = []
    com_timeout = []
    for p in arquivos:
        for n, linha in enumerate(io.open(p, encoding="utf-8", errors="replace").read().split("\n"), 1):
            s = linha.strip()
            if not s or s.startswith("#"):
                continue
            if "go test" not in s:
                continue
            # *so' os que correm o package inteiro: os `-run <teste>` sao alvo unico e rapidos*
            if "-run" in s:
                continue
            # ***SO' O `./...` DA RAIZ INCLUI O AGENT.*** *Um `./app/...` ou `./server/...` NAO o inclui*
            # => *o `"/..." in s` era largo de mais: dava falso positivo no `./app/...`* =>
            # ***`um criterio que acusa o que nao devia e' um criterio que se desliga; um criterio que
            # acusa o que devia e' o que serve.`***
            toca = ("internal/agent" in s) or bool(re.search(r"go test[^|]*\s\./\.\.\.(\s|$)", s))
            if not toca:
                continue
            m = re.search(r"-timeout[= ](\d+)([smh])", s)
            if m:
                valor, unidade = int(m.group(1)), m.group(2)
                minutos = valor * {"s": 1 / 60, "m": 1, "h": 60}[unidade]
                com_timeout.append((p.name, n, minutos))
                T("(1) `%s:%d` tem `-timeout` suficiente (%.0f min >= %d)" % (p.name, n, minutos, MINIMO),
                  minutos >= MINIMO, s[:70])
            else:
                sem_timeout.append((p.name, n, s))

    T("(1a) e nenhum `go test` do agent/`./...` ficou SEM `-timeout`", not sem_timeout,
      "%d sem timeout" % len(sem_timeout))
    for nome, n, s in sem_timeout:
        print("       SEM TIMEOUT: %s:%d %s" % (nome, n, s[:80]))

    # ---------- (2) o `-timeout` cabe no `timeout-minutes` do job ----------
    # *um `-timeout 25m` dentro de um job com `timeout-minutes: 10` seria pior do que nao ter nada: o job morria
    # antes de o teste poder terminar.*
    if com_timeout:
        p = next(x for x in arquivos if x.name == com_timeout[0][0])
        txt = io.open(p, encoding="utf-8", errors="replace").read()
        tm = [int(x) for x in re.findall(r"timeout-minutes:\s*(\d+)", txt)]
        T("(2) o workflow declara `timeout-minutes` (o `-timeout` tem de caber dentro)",
          len(tm) > 0, "min=%d max=%d" % (min(tm), max(tm)) if tm else "AUSENTE")
        if tm:
            T("(2b) e o maior `-timeout` cabe no maior `timeout-minutes`",
              max(c[2] for c in com_timeout) < max(tm),
              "%.0f min < %d min" % (max(c[2] for c in com_timeout), max(tm)))

    # ---------- (3) o teste de volume existe e declara o teto real ----------
    # *se alguem "optimizasse" o teste baixando o `MaxRows`, o `-timeout` deixaria de ser necessario -- e o
    # proposito do teste (medir o custo real) perdia-se.*
    budget = RAIZ / "internal" / "agent" / "commerce_budget_test.go"
    T("(3) o teste de volume existe", budget.is_file())
    if budget.is_file():
        t = io.open(budget, encoding="utf-8", errors="replace").read()
        T("(3b) e usa o **teto REAL** (`DefaultOnboardingLimits().MaxRows`), nao um numero reduzido",
          "DefaultOnboardingLimits().MaxRows" in t,
          "baixar o teto para o teste correr depressa perderia o proposito")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO TIMEOUT DO CI ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  CI TIMEOUT: todo `go test` que corre o `internal/agent` (ou `./...`) declara um `-timeout`")
    print("  de pelo menos %d min -- e cabe no `timeout-minutes` do job." % MINIMO)
    print("")
    print("  ***E NAO E' ESCONDER UM DEADLOCK***: *o default de 10 min nao cabe num package de 353 s medidos")
    print("  num runner 2-3x mais lento. Um deadlock nao terminaria em 25 min; um teste de volume termina.*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
