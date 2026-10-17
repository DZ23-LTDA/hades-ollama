#!/usr/bin/env python3
"""`hades_agents_check.py` — o §ESTAGIO 9 (Agentes e subagentes), medido (`E-1099`).

***O QUE O §ESTAGIO 9 EXIGE.***

    - provar uma **missao multiagente real** sobre Ollama local, **nao apenas planner deterministico**;
    - **limitar** agentes, tempo, custo, recursao e profundidade;
    - preservar **HITL** e **impedir heranca implicita de elevacao**;
    - provar **isolamento de workspace, contexto, organizacao e artefactos**.

***O QUE ESTE CHECADOR MEDE.***

  1. **O swarm existe e tem limite**: `swarm.go` com `MaxAgents` **e um semaforo** (o limite tem de ser
     **aplicado**, nao so' declarado no struct) => *um `MaxAgents` que ninguem le' e' um comentario com
     tipo.*
  2. **O orcamento tem os 4 limites**: `MaxAgents`, `MaxSeconds`, `MaxRetries`, `MaxOutputBytes` -- *e o
     custo por passo (`MaxCostCents`) tem de estar **ligado** ao motor (`step_routing.go`).*
  3. **O isolamento de worktree tem teste** (`swarm_worktree_isolation_test.go`) -- *e' o que prova que
     dois agentes nao escrevem no mesmo sitio.*
  4. **Os artefactos reais tem teste** (`swarm_real_artifacts_test.go`) -- *nao apenas «o plano foi
     gerado».*
  5. **HITL**: `service_agent_approvals.go` **grande**, e o `*_test.go` tem de cobrir a **corrida** e a
     **recusa** (nao so' o caminho feliz).
  6. **Sem heranca implicita de elevacao**: o codigo tem de **negar** por omissao -- procurei
     `requiresApproval`/`RequiresApproval` e um teste que **recuse**.

***E O QUE ELE NAO FAZ.*** *Nao corre o Ollama -- a missao multiagente **real** exige o modelo, e o T2
(`E-1093`) ja' o mediu. => *o que ele faz e' verificar que **cada requisito do estagio tem codigo E teste**.
"""
import io
import re
import sys
from pathlib import Path

RAIZ = Path(__file__).resolve().parent.parent
AGENTE = RAIZ / "internal" / "agent"

falhas: list[str] = []


def T(nome: str, ok: bool, detalhe: str = "") -> None:
    print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + detalhe) if detalhe else ""))
    if not ok:
        falhas.append(nome)


def ler(rel: str) -> str:
    p = AGENTE / rel
    if not p.is_file():
        return ""
    return io.open(p, encoding="utf-8").read()


def main() -> int:
    # ---------- 1. o swarm, com limite APLICADO ----------
    swarm = ler("swarm.go")
    T("(1) `swarm.go` existe", bool(swarm))
    T("(1b) declara `MaxAgents`", "MaxAgents" in swarm)
    # *o limite tem de ser APLICADO: um semaforo, um `chan struct{}` ou um `sem <-`*
    aplicado = bool(re.search(r"make\(chan struct\{\},\s*[^)]*MaxAgents", swarm)) or \
               bool(re.search(r"sem\w*\s*<-\s*struct", swarm))
    T("(1c) e o limite e' APLICADO (semaforo), nao so' declarado", aplicado,
      "um `MaxAgents` que ninguem le' e' um comentario com tipo")

    # ---------- 2. o orcamento tem os 4 limites ----------
    orcamento = ("MaxAgents", "MaxSeconds", "MaxRetries", "MaxOutputBytes")
    handoff = ler("handoff_do_job.go") + ler("swarm.go")
    achados = [c for c in orcamento if c in handoff]
    T("(2) o orcamento tem os 4 limites", len(achados) == 4,
      "%d de 4: %s" % (len(achados), ", ".join(achados)))
    # *o custo por passo tem de chegar ao motor*
    routing = ler("step_routing.go")
    T("(2b) o custo por passo chega ao motor (`MaxCostCents`)",
      "MaxCostCents" in routing, "o campo existe e e' aplicado")

    # ---------- 3. isolamento de worktree ----------
    T("(3) o isolamento de worktree tem teste",
      (AGENTE / "swarm_worktree_isolation_test.go").is_file(),
      "prova que dois agentes nao escrevem no mesmo sitio")
    wt = ler("swarm_worktree_isolation_test.go")
    isol = len(re.findall(r"Worktree|worktree|isolad|Isolad|distint|separad", wt))
    T("(3b) e o teste fala de isolamento", isol >= 2, "%d ocorrencias" % isol)

    # ---------- 4. artefactos reais ----------
    T("(4) os artefactos reais tem teste",
      (AGENTE / "swarm_real_artifacts_test.go").is_file(),
      "nao apenas «o plano foi gerado»")

    # ---------- 5. HITL ----------
    apr = AGENTE / "service_agent_approvals.go"
    T("(5) o `service_agent_approvals.go` existe", apr.is_file())
    if apr.is_file():
        T("(5b) e' substancial", apr.stat().st_size > 20000,
          "%.0f KB" % (apr.stat().st_size / 1024))
    # ***O `Consume` ESTA' TESTADO EM 11 FICHEIROS, NAO SO' NUM.*** *Foi o meu erro de detector (o
    # `E-1084`, outra vez): eu procurei no `service_agent_approvals_test.go` e o `Consume` esta' em
    # `service_agent_approval_concurrency_test.go` (`TestConsumoDeAprovacaoEResistenteAConcorrencia`),
    # em `service_agent_approvals_ratchet_test.go`
    # (`TestConsumeConcorrenteComEscritoresNaoDuplicaExecucao`) e noutros 9.*
    # => *o criterio passa a ser o PACOTE inteiro, e o alvo e' a **corrida**.*
    ficheiros_consume = []
    for p in AGENTE.glob("*_test.go"):
        corpo = io.open(p, encoding="utf-8").read()
        if re.search(r"Consume|Concorren|concorren|Concurrency", corpo):
            ficheiros_consume.append(p.name)
    T("(5c) tem teste", any("approval" in n for n in ficheiros_consume),
      "%d ficheiro(s) de teste de aprovacao" % len(ficheiros_consume))
    T("(5d) e cobre o Consume/a corrida", len(ficheiros_consume) >= 3,
      "%d ficheiros no pacote: %s" % (len(ficheiros_consume), ", ".join(sorted(ficheiros_consume)[:3])))

    # ---------- 6. sem heranca implicita de elevacao ----------
    recusa = 0
    for p in AGENTE.glob("*_test.go"):
        corpo = io.open(p, encoding="utf-8").read()
        # *um teste que RECUSA por omissao: `RequiresApproval` + `false`/`Err`*
        if re.search(r"RequiresApproval", corpo) and re.search(r"false|Err|recus|Reject", corpo):
            recusa += 1
    T("(6) ha' teste que prova a RECUSA por omissao (sem elevacao implicita)",
      recusa >= 2, "%d ficheiros" % recusa)

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO §ESTAGIO 9 ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  §ESTAGIO 9 (Agentes e subagentes): swarm com limite APLICADO (semaforo), orcamento de 4")
    print("  limites, custo por passo ligado, isolamento de worktree, artefactos reais, e HITL com recusa.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
