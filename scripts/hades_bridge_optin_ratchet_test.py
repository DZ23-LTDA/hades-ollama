#!/usr/bin/env python3
"""Testes da catraca do opt-in (`E-1036`). *Uma catraca so' vale depois de ficar vermelha.*

Trabalha numa **copia descartavel** do repositorio (`--raiz`): *um teste que muta o codigo
de producao e' um risco* -- se for interrompido a meio, deixa a injecao dentro do ficheiro.
Aqui o repositorio real **nunca** e' tocado, e o teste verifica isso mesmo no fim.

Discrimina nos dois sentidos: um chamador novo SEM opt-in fica VERMELHO, e um chamador novo
COM opt-in fica VERDE -- *uma catraca que fica vermelha com tudo nao discrimina nada.*

ASCII puro.
"""
import hashlib
import io
import importlib.util
import os
import shutil
import subprocess
import sys
import tempfile

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MOD = os.path.join(RAIZ, "scripts", "hades_bridge_optin_ratchet.py")
REL = os.path.join("apps", "hades-browser-bridge", "src", "service_worker.js")
REAL = os.path.join(RAIZ, REL)

falhas = []


def ok(nome, cond, detalhe=""):
    print(("  OK   " if cond else "  FAIL ") + nome + (("  -- " + detalhe) if detalhe and not cond else ""))
    if not cond:
        falhas.append(nome)


def carregar():
    spec = importlib.util.spec_from_file_location("ratchet", MOD)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


def correr(raiz=None):
    cmd = [sys.executable, "-X", "utf8", MOD]
    if raiz:
        cmd += ["--raiz", raiz]
    r = subprocess.run(cmd, capture_output=True, text=True)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


def sha(caminho):
    with open(caminho, "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


print("0) o repositorio REAL fica intacto: guardo o hash antes")
hash_antes = sha(REAL)

print("1) o estado atual e' VERDE e encontra os 3 chamadores")
M = carregar()
codigo, saida = correr()
ok("exit 0", codigo == 0, saida[-200:])
ok("encontra 3 chamadores", ": 3" in saida, saida[:120])

print("2) o removedor de comentarios nao ve um chamador comentado")
sem = M.sem_comentarios("// validarUrl(x)\nconst a = 1;\n/* guardarRota(y) */\n")
ok("comentario de linha removido", "validarUrl" not in sem, sem)
ok("comentario de bloco removido", "guardarRota" not in sem, sem)
ok("as LINHAS sao preservadas", sem.count("\n") == 3, str(sem.count("\n")))

# ---------------------------------------------------------------- copia descartavel
tmp = tempfile.mkdtemp(prefix="hades-ratchet-")
try:
    destino = os.path.join(tmp, REL)
    os.makedirs(os.path.dirname(destino), exist_ok=True)
    copia = io.open(REAL, encoding="utf-8").read()
    io.open(destino, "w", encoding="utf-8", newline="\n").write(copia)

    print("3) na COPIA, um chamador novo SEM opt-in tem de ficar VERMELHO")
    io.open(destino, "w", encoding="utf-8", newline="\n").write(copia + """

/* INJECCAO DO TESTE: chamador novo sem o opt-in */
async function chamadorInjetado(url) {
  return validarUrl(url);
}
""")
    codigo, saida = correr(tmp)
    ok("exit != 0 com o chamador novo", codigo != 0, "exit=%d" % codigo)
    ok("diz NAO VERIFICA", "NAO VERIFICA" in saida, saida[-200:])
    ok("aponta o chamador sem opt-in", "chamadores SEM o opt-in" in saida, saida[-200:])
    ok("agora sao 4 chamadores", ": 4" in saida, saida[:160])

    print("4) e um chamador novo COM opt-in fica VERDE -- a catraca DISCRIMINA")
    io.open(destino, "w", encoding="utf-8", newline="\n").write(copia + """

/* INJECCAO DO TESTE: chamador novo COM o opt-in */
async function chamadorInjetadoOk(url) {
  return validarUrl(url, { permitirPrivado: await lerOptIn() });
}
""")
    codigo, saida = correr(tmp)
    ok("exit 0 com o chamador corrigido", codigo == 0, saida[-200:])
    ok("e conta 4 chamadores", ": 4" in saida, saida[:160])

    print("5) um chamador COMENTADO nao conta como chamador")
    io.open(destino, "w", encoding="utf-8", newline="\n").write(copia + """

/* validarUrl(url) num comentario de bloco */
// guardarRota(url, []) num comentario de linha
""")
    codigo, saida = correr(tmp)
    ok("exit 0 (comentario nao e' chamador)", codigo == 0, saida[-200:])
    ok("continua com 3 chamadores", ": 3" in saida, saida[:160])
finally:
    shutil.rmtree(tmp, ignore_errors=True)

print("6) o repositorio REAL nao foi tocado -- o hash tem de ser o mesmo")
ok("hash identico", sha(REAL) == hash_antes, "o ficheiro real mudou!")

print("7) e volta a VERDE no estado real")
codigo, saida = correr()
ok("exit 0 final", codigo == 0, saida[-160:])

print("")
if falhas:
    print("FALHAS: %d -> %s" % (len(falhas), ", ".join(falhas)))
    sys.exit(1)
print("TODOS OS TESTES DA CATRACA PASSARAM.")
sys.exit(0)
