#!/usr/bin/env python3
"""Testes da catraca do encaixe (`E-1044`). *Uma catraca so' vale depois de ficar vermelha.*

TRES injecoes, numa COPIA descartavel -- o repositorio real nunca e' tocado:

  1. **uma rota HTTP que chama `FindApproved`** (o caso do `server/`) -> tem de acordar;
  2. **um handler de approvals sem nenhuma das formas de sessao** -> tem de acordar;
  3. **o encaixe `MOTIVO_SEM_ROTA` apagado do Companion** -> tem de acordar.

E a contraprova: **um handler COM sessao nao pode acordar a catraca** -- *uma catraca que fica
vermelha com tudo nao discrimina nada* (`E-1036`).

ASCII puro.
"""
import hashlib
import io
import os
import shutil
import subprocess
import sys
import tempfile

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MOD = os.path.join(RAIZ, "scripts", "hades_approval_hook_ratchet.py")
REL_SERVIDOR = os.path.join("server", "agent_routes.go")
REL_COMPANION = os.path.join("apps", "hades-browser-bridge", "native-host", "companion.mjs")
# *A catraca verifica que `FindApproved` existe.* A primeira versao do teste copiava so' o
# `agent_routes.go` e o `companion.mjs` -- e acordava com "FindApproved DESAPARECEU", que era
# um falso vermelho DO TESTE, nao da catraca. *Um teste que nao monta o mundo que o checker
# mede acusa o checker de um defeito que e' seu.*
REL_AGENTE = os.path.join("internal", "agent", "service_agent_approvals.go")

falhas = []


def ok(nome, cond, detalhe=""):
    print(("  OK   " if cond else "  FAIL ") + nome + (("  -- " + detalhe) if detalhe and not cond else ""))
    if not cond:
        falhas.append(nome)


def sha(caminho):
    with open(caminho, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def correr(raiz=None):
    cmd = [sys.executable, "-X", "utf8", MOD]
    if raiz:
        cmd += ["--raiz", raiz]
    r = subprocess.run(cmd, capture_output=True, text=True)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


print("0) o repositorio REAL fica intacto: guardo os hashes antes")
real_servidor = os.path.join(RAIZ, REL_SERVIDOR)
real_companion = os.path.join(RAIZ, REL_COMPANION)
h_servidor, h_companion = sha(real_servidor), sha(real_companion)

print("1) o estado atual e' VERDE")
codigo, saida = correr()
ok("exit 0", codigo == 0, saida[-260:])
ok("diz que o encaixe continua por ligar", "CONTINUA POR LIGAR" in saida, saida[-200:])
ok("mostra os chamadores de FindApproved", "chamadores de FindApproved" in saida, saida[:300])
ok("nomeia a forma de sessao dos handlers", "via " in saida, saida[:400])

tmp = tempfile.mkdtemp(prefix="hades-hook-")
try:
    for rel in (REL_SERVIDOR, REL_COMPANION, REL_AGENTE):
        destino = os.path.join(tmp, rel)
        os.makedirs(os.path.dirname(destino), exist_ok=True)
        io.open(destino, "w", encoding="utf-8", newline="\n").write(
            io.open(os.path.join(RAIZ, rel), encoding="utf-8").read()
        )
    base_servidor = io.open(os.path.join(tmp, REL_SERVIDOR), encoding="utf-8").read()
    base_companion = io.open(os.path.join(tmp, REL_COMPANION), encoding="utf-8").read()

    print("2) INJECAO 1: uma rota HTTP que chama `FindApproved`")
    io.open(os.path.join(tmp, REL_SERVIDOR), "w", encoding="utf-8", newline="\n").write(
        base_servidor + '\nfunc (a *agentAPI) rotaNovaDeAprovacao(c *gin.Context) {\n'
        '\tstore := a.runtime.ApprovalStore()\n'
        '\t_, _, _ = store.FindApproved("org", "fp", time.Now())\n}\n'
    )
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("aponta a rota HTTP", "ROTA HTTP CHAMA" in saida, saida[-300:])
    ok("diz para ligar o encaixe", "LIGUE-O" in saida, saida[-300:])

    print("3) INJECAO 2: um handler de approvals SEM sessao")
    io.open(os.path.join(tmp, REL_SERVIDOR), "w", encoding="utf-8", newline="\n").write(
        base_servidor + '\nfunc (a *agentAPI) decideApprovalSemSessao(c *gin.Context) {\n'
        '\tc.JSON(200, gin.H{"ok": true})\n}\n'
    )
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("aponta o handler sem sessao", "HANDLER DE APPROVALS SEM SESSAO" in saida, saida[-300:])

    print("4) CONTRA-PROVA: um handler de approvals COM sessao NAO acorda")
    io.open(os.path.join(tmp, REL_SERVIDOR), "w", encoding="utf-8", newline="\n").write(
        base_servidor + '\nfunc (a *agentAPI) decideApprovalComSessao(c *gin.Context) {\n'
        '\torganizationID, ok := a.requireContextOrganization(c)\n'
        '\tif !ok { return }\n'
        '\t_ = organizationID\n}\n'
    )
    codigo, saida = correr(tmp)
    ok("fica VERDE com handler de sessao", codigo == 0, saida[-300:])
    ok("e conta-o no relatorio", "decideApprovalComSessao" in saida, saida[:600])

    print("5) INJECAO 3: o encaixe apagado do Companion")
    io.open(os.path.join(tmp, REL_SERVIDOR), "w", encoding="utf-8", newline="\n").write(base_servidor)
    io.open(os.path.join(tmp, REL_COMPANION), "w", encoding="utf-8", newline="\n").write(
        base_companion.replace('export const MOTIVO_SEM_ROTA = "sem_rota_autenticavel_no_runtime";',
                               'const MOTIVO_SEM_ROTA = "x";')
    )
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("aponta o encaixe desaparecido", "ENCAIXE" in saida and "DESAPARECEU" in saida, saida[-300:])

    print("6) INJECAO 4: `FindApproved` apagado")
    io.open(os.path.join(tmp, REL_COMPANION), "w", encoding="utf-8", newline="\n").write(base_companion)
    base_agente = io.open(os.path.join(tmp, REL_AGENTE), encoding="utf-8").read()
    io.open(os.path.join(tmp, REL_AGENTE), "w", encoding="utf-8", newline="\n").write(
        base_agente.replace("func (s *ServiceAgentApprovalStore) FindApproved(",
                            "func (s *ServiceAgentApprovalStore) FindApprovedXX(")
    )
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("aponta a funcao desaparecida", "DESAPARECEU" in saida, saida[-300:])
finally:
    shutil.rmtree(tmp, ignore_errors=True)

print("7) o repositorio REAL nao foi tocado -- os dois hashes tem de bater")
ok("server/agent_routes.go intacto", sha(real_servidor) == h_servidor, "mudou!")
ok("companion.mjs intacto", sha(real_companion) == h_companion, "mudou!")

print("8) e volta a VERDE no estado real")
codigo, saida = correr()
ok("exit 0 final", codigo == 0, saida[-200:])

print("")
if falhas:
    print("FALHAS: %d -> %s" % (len(falhas), ", ".join(falhas)))
    sys.exit(1)
print("TODOS OS TESTES DA CATRACA DO ENCAIXE PASSARAM.")
sys.exit(0)
