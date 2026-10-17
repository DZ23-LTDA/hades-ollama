#!/usr/bin/env python3
"""Testes da ligacao docs -> artefactos (`E-1054`). *Uma catraca so' vale depois de ficar vermelha.*

QUATRO injecoes, numa COPIA descartavel -- o repositorio real nunca e' tocado:

  1. **o artefacto apagado** -> tem de acordar com `ARTEFACTO AUSENTE`;
  2. **o artefacto esvaziado** (0 bytes) -> `SUSPEITO` -- *um ficheiro vazio nao e' um artefacto*;
  3. **o ponteiro removido do documento** -> `SEM PONTEIRO`;
  4. **o documento apagado** -> `DOCUMENTO AUSENTE`.

E a contraprova: **o estado real fica verde** -- *uma catraca que fica vermelha com tudo nao
discrimina nada* (`E-1036`).

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
MOD = os.path.join(RAIZ, "scripts", "hades_stage_artifacts_check.py")

DOC = os.path.join("docs", "mission", "HADES_STAGE_21_BENCHMARK.md")
ART = os.path.join("docs", "parity", "HADES_T2_MATRIX.md")
ART2 = os.path.join("docs", "parity", "HADES_T2_REPETICAO.md")
# *O teste tem de montar o mundo INTEIRO que o checker mede.* A primeira versao copiava so' o
# `DOC` e acusava `DOCUMENTO AUSENTE` nos outros dois -- um falso vermelho DO TESTE, nao da
# catraca. E' a mesma licao do `E-1044`: *um teste que nao monta o mundo que o checker mede
# acusa o checker de um defeito que e' seu.*
DOCS = (
    os.path.join("docs", "mission", "HADES_STAGE_21_BENCHMARK.md"),
    os.path.join("docs", "mission", "HADES_BENCHMARK_METHOD.md"),
    os.path.join("docs", "mission", "HADES_BENCHMARK_21.md"),
)

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
        # o checker deriva RAIZ do proprio caminho; o teste copia-o para a copia
        cmd = [sys.executable, "-X", "utf8", os.path.join(raiz, "scripts", "hades_stage_artifacts_check.py")]
    r = subprocess.run(cmd, capture_output=True, text=True)
    return r.returncode, (r.stdout or "") + (r.stderr or "")


print("0) o repositorio REAL fica intacto: guardo os hashes antes")
TODOS = DOCS + (ART, ART2)
reais = {p: sha(os.path.join(RAIZ, p)) for p in TODOS}

print("1) o estado atual e' VERDE")
codigo, saida = correr()
ok("exit 0", codigo == 0, saida[-300:])
ok("diz que todas as ligacoes existem", "TODAS AS LIGACOES" in saida, saida[-200:])
ok("lista os tres documentos", saida.count("HADES_STAGE_21_BENCHMARK.md") >= 1, saida[:400])
ok("mostra os bytes dos artefactos", "1548 bytes" in saida, saida[:400])

tmp = tempfile.mkdtemp(prefix="hades-lig-")
try:
    # copia o checker E os ficheiros que ele mede
    os.makedirs(os.path.join(tmp, "scripts"))
    os.makedirs(os.path.join(tmp, "docs", "mission"))
    os.makedirs(os.path.join(tmp, "docs", "parity"))
    shutil.copy2(MOD, os.path.join(tmp, "scripts", "hades_stage_artifacts_check.py"))
    for p in TODOS:
        shutil.copy2(os.path.join(RAIZ, p), os.path.join(tmp, p))

    def restaurar():
        for p in TODOS:
            shutil.copy2(os.path.join(RAIZ, p), os.path.join(tmp, p))

    print("2) INJECAO 1: o artefacto APAGADO")
    os.remove(os.path.join(tmp, ART))
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("diz ARTEFACTO AUSENTE", "ARTEFACTO AUSENTE" in saida, saida[-300:])
    restaurar()

    print("3) INJECAO 2: o artefacto ESVAZIADO (0 bytes)")
    io.open(os.path.join(tmp, ART), "w", encoding="utf-8", newline="\n").write("")
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("diz SUSPEITO", "SUSPEITO" in saida, saida[-300:])
    restaurar()

    print("4) INJECAO 3: o PONTEIRO removido do documento")
    texto = io.open(os.path.join(tmp, DOC), encoding="utf-8").read()
    texto = texto.replace("HADES_T2_MATRIX.md", "XX_MATRIX_XX.md")
    io.open(os.path.join(tmp, DOC), "w", encoding="utf-8", newline="\n").write(texto)
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("diz SEM PONTEIRO", "SEM PONTEIRO" in saida, saida[-300:])
    restaurar()

    print("5) CONTRA-PROVA: o estado restaurado volta a VERDE")
    codigo, saida = correr(tmp)
    ok("fica verde depois de restaurar", codigo == 0, saida[-260:])

    print("6) INJECAO 4: o DOCUMENTO apagado")
    os.remove(os.path.join(tmp, DOC))
    codigo, saida = correr(tmp)
    ok("acorda (exit != 0)", codigo != 0, "exit=%d" % codigo)
    ok("diz DOCUMENTO AUSENTE", "DOCUMENTO AUSENTE" in saida, saida[-300:])
finally:
    shutil.rmtree(tmp, ignore_errors=True)

print("7) o repositorio REAL nao foi tocado -- os hashes tem de bater")
for p, h in reais.items():
    ok("%s intacto" % os.path.basename(p), sha(os.path.join(RAIZ, p)) == h, "mudou!")

print("8) e o `--check` silencioso funciona")
r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check"], capture_output=True, text=True)
ok("--check exit 0 no estado real", r.returncode == 0, (r.stdout + r.stderr)[-200:])

print("")
if falhas:
    print("FALHAS: %d -> %s" % (len(falhas), ", ".join(falhas)))
    sys.exit(1)
print("TODOS OS TESTES DA LIGACAO docs->artefactos PASSARAM.")
sys.exit(0)
