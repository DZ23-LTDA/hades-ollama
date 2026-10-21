#!/usr/bin/env python3
"""ID DETERMINISTICO DA EXTENSAO + MANIFESTO DO NATIVE HOST (`E-1031`).

O PROBLEMA MEDIDO: o `manifest.json` da extensao **nao tem `key`**, e um manifesto de
native messaging precisa de `allowed_origins` com o ID da extensao. Sem `key`, o ID so'
existe **depois** de a extensao ser carregada no navegador -- o que torna o `allowed_origins`
um placeholder (`REPLACE_WITH_EXTENSION_ID`) e a instalacao do host impossivel de
verificar antes disso.

A SOLUCAO E' A DO PROPRIO CHROME: o ID de uma extensao e' derivado da chave publica.
    id = sha256(chave_publica_DER)[:16]  -> hex  ->  cada digito 0-f vira a-p
Com a chave publica versionada, o ID e' **calculavel sem navegador** e o manifesto
do host nasce com o ID real.

O QUE ESTE SCRIPT NAO FAZ: nao instala nada, nao escreve no registo, nao gera a chave
privada de distribuicao. Ele deriva o ID e escreve/verifica o manifesto.
"""
import argparse
import base64
import hashlib
import io
import json
import os
import re
import sys

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BRIDGE = os.path.join(RAIZ, "apps", "hades-browser-bridge")
MANIFEST = os.path.join(BRIDGE, "manifest.json")
CHAVE_PUB = os.path.join(BRIDGE, "native-host", "chave-publica.b64")
NOME_DO_HOST = "com.dz23ltd.hades.bridge"
# O manifesto de native messaging TEM de ter caminho absoluto, logo e' especifico da
# maquina e NAO se versiona. O `--check` procura onde ele realmente vive.
HOST_JSON = os.path.join(BRIDGE, "native-host", NOME_DO_HOST + ".json")


def destino_de_instalacao():
    """Onde o manifesto REAL vive. Nunca no repo: tem o caminho absoluto da maquina."""
    local = os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")
    return os.path.join(local, "HADES", "NativeHost", NOME_DO_HOST + ".json")


def _caminhos_do_manifesto():
    """Os sitios onde procurar o manifesto INSTALADO, por ordem."""
    return [destino_de_instalacao()]


def manifesto_existente():
    """O manifesto INSTALADO. O template do repo nao conta: ele tem placeholders."""
    for c in _caminhos_do_manifesto():
        if not os.path.isfile(c):
            continue
        with io.open(c, encoding="utf-8") as f:
            h = json.load(f)
        if str(h.get("path", "")).startswith("REPLACE_WITH_"):
            continue
        return c
    return None

# Mapeamento hex -> 'a'..'p' do Chrome. Cada digito hexadecimal vira uma letra.
_ALFA = "abcdefghijklmnop"


def id_da_chave(chave_publica_der: bytes) -> str:
    """Deriva o ID da extensao. Esta e' a formula que o Chrome usa."""
    digest = hashlib.sha256(chave_publica_der).hexdigest()[:32]
    return "".join(_ALFA[int(c, 16)] for c in digest)


def id_do_manifesto() -> str | None:
    """O ID derivado da chave VERSIONADA no manifesto, ou None se nao houver."""
    if not os.path.isfile(MANIFEST):
        return None
    with io.open(MANIFEST, encoding="utf-8") as f:
        m = json.load(f)
    if "key" not in m:
        return None
    # A `key` no manifesto e' a chave publica em base64 (DER SubjectPublicKeyInfo).
    try:
        der = base64.b64decode(m["key"])
    except Exception:
        return None
    return id_da_chave(der)


def id_da_chave_versionada() -> str | None:
    """O ID derivado do ficheiro `chave-publica.b64` (a fonte que o `--check` usa)."""
    if not os.path.isfile(CHAVE_PUB):
        return None
    with io.open(CHAVE_PUB, encoding="utf-8") as f:
        texto = f.read().strip()
    if not texto or texto.startswith("REPLACE_WITH_"):
        return None
    try:
        return id_da_chave(base64.b64decode(texto))
    except Exception:
        return None


# ------------------------------------------------------------------ manifesto


def escrever_manifesto(ext_id: str, caminho_companion: str, destino: str = None) -> str:
    """Escreve o manifesto do native host com o ID REAL. Devolve o JSON.

    O `path` NAO pode apontar para o `.mjs`: medido no `E-1032`, o Windows **nao executa**
    `.mjs` directamente -- o Chrome arranca o processo, nao corre o ficheiro, e a resposta
    e' «Error when communicating with the native messaging host». O `path` aponta para o
    **lancador** (`.cmd`), que chama o `node`.
    """
    caminho_companion = os.path.abspath(caminho_companion)
    lancador = escrever_lancador(caminho_companion, os.path.dirname(destino or HOST_JSON))
    m = {
        "name": NOME_DO_HOST,
        "description": (
            "HADES Local Companion -- host de native messaging do HADES Browser Bridge "
            "(NAO PUBLICADO, NAO ASSINADO)."
        ),
        "path": lancador,
        "type": "stdio",
        # A REGRA: SO' a extensao HADES. Nada de curinga, nada de <all_urls>, nada de http.
        "allowed_origins": ["chrome-extension://%s/" % ext_id],
    }
    texto = json.dumps(m, indent=2, ensure_ascii=False) + "\n"
    alvo = destino or HOST_JSON
    with io.open(alvo, "w", encoding="utf-8", newline="\n") as f:
        f.write(texto)
    return texto


def escrever_lancador(caminho_companion: str, pasta: str) -> str:
    """Gera o lancador `.cmd`. Devolve o caminho absoluto do lancador.

    `.cmd` (e nao `.bat`) porque e' a extensao que o `CreateProcess` do `cmd.exe` aceita
    sem ambiguidade. `@echo off` e `%~dp0` mantem o caminho relativo ao proprio lancador,
    para o pacote poder ser movido.
    """
    if pasta and not os.path.isdir(pasta):
        os.makedirs(pasta)
    lancador = os.path.join(pasta, NOME_DO_HOST + ".cmd")
    # Caminho ABSOLUTO embutido: o manifesto ja' e' especifico da maquina, e um caminho
    # relativo a partir de `%LOCALAPPDATA%` nao chegaria ao repo.
    # ASCII puro: o `cmd.exe` deste host le' em OEM/ANSI e um acento aqui vira lixo.
    corpo = (
        "@echo off\r\n"
        "rem HADES Local Companion -- lancador do host de native messaging.\r\n"
        "rem O Windows NAO executa `.mjs` directamente (medido no E-1032): o Chrome\r\n"
        "rem arranca este processo e o ficheiro nao corre. Este lancador chama o node.\r\n"
        "rem NAO PUBLICADO, NAO ASSINADO.\r\n"
        'node "%s" %%*\r\n' % caminho_companion
    )
    with io.open(lancador, "w", encoding="ascii", newline="") as f:
        f.write(corpo)
    return lancador


def verificar(alvo: str = None) -> int:
    """Verifica que o ID do manifesto e do host batem com a chave versionada."""
    problemas = []
    ext_id = id_da_chave_versionada()
    if ext_id is None:
        problemas.append(
            "SEM CHAVE: `native-host/chave-publica.b64` ausente ou placeholder. "
            "Sem chave publica nao ha' ID deterministico e o `allowed_origins` volta a ser adivinhacao."
        )
    else:
        do_manifesto = id_do_manifesto()
        if do_manifesto != ext_id:
            problemas.append(
                "MANIFESTO DIVERGE: a `key` do `manifest.json` da' id=%s e a chave versionada da' id=%s"
                % (do_manifesto or "(sem key)", ext_id)
            )
        instalado = alvo or manifesto_existente()
        if instalado is None:
            so_template = os.path.isfile(HOST_JSON)
            problemas.append(
                ("HOST NAO INSTALADO: existe apenas o TEMPLATE do repo (%s), que tem "
                 "placeholders e nao serve." % HOST_JSON) if so_template else
                "HOST NAO INSTALADO: nenhum manifesto em %s." % destino_de_instalacao()
            )
            problemas.append(
                "COMO INSTALAR: `powershell -File scripts\\hades_native_host_install.ps1` "
                "(e `-Desinstalar` para remover)."
            )
        else:
            caminho_manifesto = instalado
            with io.open(caminho_manifesto, encoding="utf-8") as f:
                h = json.load(f)
            origens = h.get("allowed_origins") or []
            if origens != ["chrome-extension://%s/" % ext_id]:
                problemas.append(
                    "ALLOWED_ORIGINS DIVERGE: %r (esperado exactamente a extensao HADES id=%s)"
                    % (origens, ext_id)
                )
            if h.get("type") != "stdio":
                problemas.append("TYPE NAO E' stdio: %r" % h.get("type"))
            caminho = h.get("path") or ""
            if caminho.startswith("REPLACE_WITH_") or not caminho:
                problemas.append("PATH PLACEHOLDER: o Companion nao esta' apontado")
            elif not os.path.isfile(caminho):
                problemas.append("PATH NAO EXISTE: %s" % caminho)
            elif caminho.lower().endswith(".mjs"):
                # Medido no `E-1032`: o Windows NAO executa `.mjs`, e o Chrome responde
                # «Error when communicating with the native messaging host».
                problemas.append(
                    "PATH APONTA PARA .mjs: o Windows nao executa .mjs directamente. "
                    "O `path` tem de apontar para o lancador (.cmd)."
                )
    if problemas:
        print("  NAO VERIFICA:")
        for p in problemas:
            print("    - " + p)
        return 1
    print("  VERIFICA: id=%s (derivado da chave versionada)" % ext_id)
    print("           allowed_origins: so' a extensao HADES")
    print("           type: stdio | path: resolve")
    return 0


def verificar_repo() -> int:
    """O que e' verificavel SEM instalacao: o ID derivado, a `key`, o template.

    *Um gate que so' ficasse verde com o host instalado estaria a medir a MAQUINA, nao o
    repositorio* -- e num CI limpo nunca passaria. Este modo mede o que o Git transporta;
    o `--check` estrito mede a instalacao.
    """
    problemas = []
    ext_id = id_da_chave_versionada()
    if ext_id is None:
        problemas.append(
            "SEM CHAVE: `native-host/chave-publica.b64` ausente ou placeholder. "
            "Sem chave publica nao ha' ID deterministico e o `allowed_origins` volta a ser adivinhacao."
        )
    else:
        do_manifesto = id_do_manifesto()
        if do_manifesto != ext_id:
            problemas.append(
                "MANIFESTO DIVERGE: a `key` do `manifest.json` da' id=%s e a chave versionada da' id=%s"
                % (do_manifesto or "(sem key)", ext_id)
            )
        if not os.path.isfile(HOST_JSON):
            problemas.append("TEMPLATE AUSENTE: %s" % HOST_JSON)
        else:
            with io.open(HOST_JSON, encoding="utf-8") as f:
                tpl = json.load(f)
            if not str(tpl.get("path", "")).startswith("REPLACE_WITH_"):
                problemas.append(
                    "TEMPLATE COM CAMINHO REAL: o repo nao pode versionar o `path` desta maquina "
                    "(o manifesto real vive no destino de instalacao)"
                )
            if tpl.get("type") != "stdio":
                problemas.append("TEMPLATE SEM type=stdio: %r" % tpl.get("type"))
            if tpl.get("allowed_origins") != ["chrome-extension://REPLACE_WITH_EXTENSION_ID/"]:
                problemas.append(
                    "TEMPLATE COM ID REAL: %r (o template tem de ter placeholder)"
                    % tpl.get("allowed_origins")
                )
    if problemas:
        print("  NAO VERIFICA (repo):")
        for p in problemas:
            print("    - " + p)
        return 1
    print("  VERIFICA (repo): id=%s (derivado da chave versionada)" % ext_id)
    print("           manifest.json tem `key` que da' o MESMO id")
    print("           template com placeholders (o manifesto real nao se versiona)")
    print("           instalacao: %s" % ("presente" if manifesto_existente() else "ausente (normal num repo limpo)"))
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description="ID deterministico da extensao + manifesto do native host")
    ap.add_argument("--check", action="store_true", help="verifica sem escrever")
    ap.add_argument("--check-repo", action="store_true",
                    help="verifica o que e' verificavel SEM instalacao (o que o CI mede)")
    ap.add_argument("--manifesto", metavar="CAMINHO",
                    help="manifesto a verificar (por omissao: o instalado)")
    ap.add_argument("--derivar", metavar="B64", help="deriva o ID de uma chave publica em base64")
    ap.add_argument("--escrever", metavar="COMPANION",
                    help="escreve o manifesto com o ID real no destino de instalacao")
    ap.add_argument("--para", metavar="CAMINHO",
                    help="alvo alternativo do --escrever (usado pelo teste)")
    a = ap.parse_args()

    if a.derivar:
        print(id_da_chave(base64.b64decode(a.derivar)))
        return 0
    if a.escrever:
        ext_id = id_da_chave_versionada()
        if ext_id is None:
            print("  ERRO: sem chave publica versionada; nada a escrever", file=sys.stderr)
            return 1
        alvo = a.para or destino_de_instalacao()
        pai = os.path.dirname(alvo)
        if pai and not os.path.isdir(pai):
            os.makedirs(pai)
        escrever_manifesto(ext_id, a.escrever, alvo)
        print("  manifesto escrito em %s com id=%s" % (alvo, ext_id))
        return 0
    if a.check_repo:
        return verificar_repo()
    if a.check or len(sys.argv) == 1:
        return verificar(a.manifesto)
    return 0


if __name__ == "__main__":
    sys.exit(main())
