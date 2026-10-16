#!/usr/bin/env python3
"""Testes de DISCRIMINACAO do ID deterministico da extensao (`E-1031`).

Um teste que so' confirma o resultado atual nao prova nada. Estes testes:
  1. recalculam o ID por um caminho INDEPENDENTE do modulo (sha256 + mapeamento a mao);
  2. provam DETERMINISMO (mesma chave -> mesmo ID);
  3. provam SENSIBILIDADE (chave diferente -> ID diferente);
  4. provam a FORMA (^[a-p]{32}$) e que nenhuma letra fora de a-p aparece;
  5. FAULT INJECTION: estragar o `allowed_origins` tem de deixar o `--check` VERMELHO.
     Uma catraca so' prova que serve depois de ter ficado vermelha.

ASCII puro.
"""
import base64
import hashlib
import io
import json
import os
import re
import subprocess
import sys
import tempfile

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BRIDGE = os.path.join(RAIZ, "apps", "hades-browser-bridge")
MOD = os.path.join(RAIZ, "scripts", "hades_native_host.py")
CHAVE = os.path.join(BRIDGE, "native-host", "chave-publica.b64")
MANIFEST = os.path.join(BRIDGE, "manifest.json")
TEMPLATE = os.path.join(BRIDGE, "native-host", "com.dz23ltd.hades.bridge.json")
# Alvo proprio do teste: NAO toca no destino de instalacao real do utilizador.
HOST = os.path.join(tempfile.gettempdir(), "hades_host_test_" + str(os.getpid()) + ".json")
COMPANION = os.path.join(BRIDGE, "native-host", "companion.mjs")

ALFA = "abcdefghijklmnop"
falhas = []


def ok(nome, cond, detalhe=""):
    print(("  OK   " if cond else "  FAIL ") + nome + (("  -- " + detalhe) if detalhe and not cond else ""))
    if not cond:
        falhas.append(nome)


def id_referencia(der: bytes) -> str:
    """Formula recalculada aqui, de forma independente do modulo que se testa."""
    return "".join(ALFA[int(c, 16)] for c in hashlib.sha256(der).hexdigest()[:32])


def carregar_modulo():
    import importlib.util
    spec = importlib.util.spec_from_file_location("hnh", MOD)
    m = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(m)
    return m


print("1) o modulo deriva o MESMO id que o calculo independente")
M = carregar_modulo()
pub_b64 = io.open(CHAVE, encoding="utf-8").read().strip()
der = base64.b64decode(pub_b64)
ref = id_referencia(der)
do_modulo = M.id_da_chave(der)
ok("modulo == referencia", do_modulo == ref, "%s vs %s" % (do_modulo, ref))
ok("id tem 32 chars", len(ref) == 32, str(len(ref)))
ok("id so' usa a-p", bool(re.fullmatch(r"[a-p]{32}", ref)), ref)

print("2) a `key` do manifest da' o MESMO id")
mf = json.load(io.open(MANIFEST, encoding="utf-8"))
ok("manifest tem key", "key" in mf)
ok("key == chave versionada", mf.get("key") == pub_b64)
ok("id do manifest == referencia", M.id_do_manifesto() == ref)

print("3) DETERMINISMO e SENSIBILIDADE")
ok("determinismo", M.id_da_chave(der) == M.id_da_chave(der))
# chave diferente: muda 1 byte no fim do DER
alterado = bytearray(der)
alterado[-1] ^= 0x01
ok("1 byte diferente -> id diferente", M.id_da_chave(bytes(alterado)) != ref)
ok("e o id alterado tambem e' valido", bool(re.fullmatch(r"[a-p]{32}", M.id_da_chave(bytes(alterado)))))
# o mapeamento nao pode produzir 'q'..'z' nunca
todos = set()
for v in range(256):
    todos.add(M.id_da_chave(bytes([v]) * 32)[0])
ok("o 1o char fica em a-p para 256 entradas", todos <= set(ALFA), str(sorted(todos)))

print("3b) o TEMPLATE do repo tem placeholders -- nao serve como instalacao")
tpl = json.load(io.open(TEMPLATE, encoding="utf-8"))
ok("template tem path placeholder", str(tpl.get("path", "")).startswith("REPLACE_WITH_"))
ok("template tem id placeholder", tpl.get("allowed_origins") == ["chrome-extension://REPLACE_WITH_EXTENSION_ID/"])
r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", TEMPLATE],
                   capture_output=True, text=True)
ok("check do template -> RED", r.returncode != 0)

print("3c) `--escrever --para` gera o manifesto REAL")
r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--escrever", COMPANION, "--para", HOST],
                   capture_output=True, text=True)
ok("escrever exit 0", r.returncode == 0, (r.stdout + r.stderr)[-200:])
ok("manifesto existe", os.path.isfile(HOST))

print("4) o manifesto do host carrega o id real e SO' a extensao HADES")
h = json.load(io.open(HOST, encoding="utf-8"))
esperado = ["chrome-extension://%s/" % ref]
ok("allowed_origins exacto", h.get("allowed_origins") == esperado, repr(h.get("allowed_origins")))
ok("type == stdio", h.get("type") == "stdio")
ok("sem curinga", not any("*" in o or "<all_urls>" in o for o in h.get("allowed_origins", [])))
ok("sem http(s)", not any(o.startswith("http") for o in h.get("allowed_origins", [])))
ok("path absoluto e exists", os.path.isabs(h.get("path", "")) and os.path.isfile(h["path"]))
ok("nome do host", h.get("name") == "com.dz23ltd.hades.bridge")

print("5) o check VERDE no estado atual")
r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", HOST], capture_output=True, text=True)
ok("check exit 0", r.returncode == 0, (r.stdout + r.stderr)[-200:])

print("6) FAULT INJECTION -- uma catraca so' vale depois de ficar vermelha")
# 6a) estragar o allowed_origins -> o check TEM de ficar vermelho
copia = io.open(HOST, encoding="utf-8").read()
try:
    mau = json.loads(copia)
    mau["allowed_origins"] = ["chrome-extension://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/"]
    io.open(HOST, "w", encoding="utf-8", newline="\n").write(json.dumps(mau, indent=2) + "\n")
    r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", HOST], capture_output=True, text=True)
    ok("id errado no host -> RED", r.returncode != 0)
    ok("e diz ALLOWED_ORIGINS", "ALLOWED_ORIGINS" in (r.stdout + r.stderr))
finally:
    io.open(HOST, "w", encoding="utf-8", newline="\n").write(copia)

# 6b) estragar o `path` -> RED
try:
    mau = json.loads(copia)
    mau["path"] = "REPLACE_WITH_ABSOLUTE_PATH_TO_HADES_COMPANION"
    io.open(HOST, "w", encoding="utf-8", newline="\n").write(json.dumps(mau, indent=2) + "\n")
    r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", HOST], capture_output=True, text=True)
    ok("path placeholder -> RED", r.returncode != 0)
    ok("e diz PATH PLACEHOLDER", "PATH PLACEHOLDER" in (r.stdout + r.stderr))
finally:
    io.open(HOST, "w", encoding="utf-8", newline="\n").write(copia)

# 6c) trocar a `key` do manifest -> RED (o id deixa de bater)
mf_copia = io.open(MANIFEST, encoding="utf-8").read()
try:
    mau = json.loads(mf_copia)
    outra = bytearray(der)
    outra[-1] ^= 0x02
    mau["key"] = base64.b64encode(bytes(outra)).decode()
    io.open(MANIFEST, "w", encoding="utf-8", newline="\n").write(json.dumps(mau, indent=2, ensure_ascii=False) + "\n")
    r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", HOST], capture_output=True, text=True)
    ok("key trocada -> RED", r.returncode != 0)
    ok("e diz MANIFESTO DIVERGE", "MANIFESTO DIVERGE" in (r.stdout + r.stderr))
finally:
    io.open(MANIFEST, "w", encoding="utf-8", newline="\n").write(mf_copia)

print("7) volta a VERDE depois de restaurar")
r = subprocess.run([sys.executable, "-X", "utf8", MOD, "--check", "--manifesto", HOST], capture_output=True, text=True)
ok("check exit 0 apos restaurar", r.returncode == 0, (r.stdout + r.stderr)[-200:])

print("8) limpeza do alvo de teste")
if os.path.isfile(HOST):
    os.remove(HOST)
ok("alvo de teste removido", not os.path.isfile(HOST))

print("")
if falhas:
    print("FALHAS: %d -> %s" % (len(falhas), ", ".join(falhas)))
    sys.exit(1)
print("TODOS OS TESTES DO ID PASSARAM.")
sys.exit(0)
