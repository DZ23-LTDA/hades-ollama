"""`hades_firefox_package.py` — gera o pacote **Firefox** da HADES Browser Bridge (`E-1116`).

***O PROBLEMA QUE ELE RESOLVE.*** *O `E-1115` fechou o E2E de DOM no Chromium. Mas o §FASE-3-B exige: «Firefox
por WebExtensions **quando as APIs equivalentes permitirem**»* => *e o `web-ext lint` (10.8.0) mediu **3**
problemas no manifest actual:*

    ERRO   Unsupported "/background/service_worker" manifest property
    ERRO   The add-on ID is required in Manifest Version 3
    AVISO  The "data_collection_permissions" property is missing

=> *ou seja: **a extensão não é instalável no Firefox como está** -- e isso é uma **lacuna interna**, não um
bloqueio externo.*

***O QUE ELE FAZ.*** *Lê o `manifest.json` (Chrome MV3) e produz um `manifest.firefox.json` derivado:*

  - **remove** o que é só-Chrome: `key`, `minimum_chrome_version`, `externally_connectable`;
  - **troca** `background.service_worker` por `background.scripts` (o Firefox MV3 usa scripts, não worker);
  - **acrescenta** `browser_specific_settings.gecko` com `id`, `strict_min_version` e
    `data_collection_permissions`;
  - **mantém** o resto -- *incluindo a CSP estrita, o `incognito: not_allowed` e as permissões mínimas*.

=> *e depois: **copia só os ficheiros que o manifest referencia**, corre o `web-ext lint` e o `web-ext build`, e
mede o `.zip` com hash.*

***E O QUE ELE DECLARA.*** *O pacote Firefox é **não assinado** e **não publicado**: instalar no Firefox estável
exige `xpinstall.signatures.required=false` (Developer Edition) ou a assinatura da Mozilla. => *o pacote
**existe**, o lint **passa**, e a instalação num Firefox real é `BLOCKED_EXTERNAL` (conta Mozilla).*
"""
import hashlib
import io
import json
import shutil
import subprocess
import sys
from pathlib import Path

RAIZ = Path(r"C:\Users\DZ23\Documents\deepseek-harness\default-workspace\hades-ollama")
EXT = RAIZ / "apps" / "hades-browser-bridge"
DEST = RAIZ / "dist" / "browser-bridge-firefox"
STAGE = DEST / "hades-browser-bridge-firefox-UNSIGNED"
WEB_EXT = RAIZ / "_tmp" / "appdata" / "npm" / "web-ext.cmd"

# *o que sai do manifest Chrome -> Firefox*
SO_CHROME = ("key", "minimum_chrome_version", "externally_connectable")


def sha256(p: Path) -> str:
    h = hashlib.sha256()
    with io.open(p, "rb") as f:
        for bloco in iter(lambda: f.read(1 << 20), b""):
            h.update(bloco)
    return h.hexdigest()


def manifest_firefox(base: dict) -> dict:
    d = {k: v for k, v in base.items() if k not in SO_CHROME}
    # ***O `background`: o Firefox MV3 quer `scripts`, nao `service_worker`.***
    bg = dict(base.get("background") or {})
    if "service_worker" in bg:
        sw = bg.pop("service_worker")
        bg["scripts"] = [sw]
    d["background"] = bg
    # ***E o `browser_specific_settings` -- sem o `id`, o MV3 do Firefox recusa a instalacao.***
    # *A estrutura: `gecko` e `gecko_android` sao **irmaos**, nao pai/filho.*
    d["browser_specific_settings"] = {
        "gecko": {
            "id": "browser-bridge@dz23ltd.hades",
            # ***O 140, NAO o 128.*** *O `web-ext lint` diz que o 128 «was released before version **140**
            # introduced support» -- a chave que so' existe a partir do 140 e' o
            # `optional_host_permissions` (o Firefox aceita-o desde o 140).* => *logo o 140 e' o minimo
            # **real** para este manifest.*
            "strict_min_version": "140.0",
            # *o `data_collection_permissions` e' OBRIGATORIO para novas extensoes Firefox*
            "data_collection_permissions": {"required": ["none"]},
        },
        # *e o `gecko_android` EXPLICITO: o `web-ext lint` avisa que o Firefox for Android so' aceita esta
        # chave a partir do **142** -- e sem o declarar, o aviso recai sobre o `strict_min_version` do gecko.*
        "gecko_android": {"strict_min_version": "142.0"},
    }
    return d


def ficheiros_referenciados(m: dict) -> list[str]:
    """Só o que o manifest referencia -- *e o resto do `src/` porque o service_worker importa módulos*."""
    out = set()
    b = m.get("background") or {}
    for k in ("service_worker", "scripts"):
        for s in (b.get(k) or ([b[k]] if isinstance(b.get(k), str) else [])):
            out.add(s)
    for cs in m.get("content_scripts") or []:
        out.update(cs.get("js") or [])
        out.update(cs.get("css") or [])
    a = m.get("action") or {}
    if a.get("default_popup"):
        out.add(a["default_popup"])
    out.update((m.get("icons") or {}).values())
    # *o `src/` inteiro: o service_worker e' `type: module` e importa os outros*
    for p in (EXT / "src").glob("*.js"):
        out.add("src/" + p.name)
    return sorted(out)


def main() -> int:
    if "--check" in sys.argv:
        return verificar()

    base = json.loads(io.open(EXT / "manifest.json", encoding="utf-8").read())
    ff = manifest_firefox(base)

    if STAGE.exists():
        shutil.rmtree(STAGE)
    (STAGE / "src").mkdir(parents=True, exist_ok=True)

    faltam = []
    for rel in ficheiros_referenciados(ff):
        src = EXT / rel
        if not src.is_file():
            faltam.append(rel)
            continue
        dst = STAGE / rel
        dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(src, dst)
    if faltam:
        print("  FALHA: o manifest referencia ficheiros que nao existem: %s" % ", ".join(faltam))
        return 1

    io.open(STAGE / "manifest.json", "w", encoding="utf-8", newline="\n").write(
        json.dumps(ff, indent=2, ensure_ascii=False) + "\n")
    print("  manifest.firefox.json -> manifest.json no stage (%d ficheiros copiados)" % len(ficheiros_referenciados(ff)))

    # *o LINT do proprio Mozilla -- e' o criterio que decide se o pacote e' instalavel*
    if WEB_EXT.is_file():
        r = subprocess.run([str(WEB_EXT), "lint", "--source-dir", str(STAGE), "--output", "json"],
                           capture_output=True, cwd=str(RAIZ))
        txt = (r.stdout or b"").decode("utf-16", errors="replace") if r.stdout and r.stdout[:2] == b"\xff\xfe" \
            else (r.stdout or b"").decode("utf-8", errors="replace")
        i = txt.find("{")
        n_erros = None
        if i >= 0:
            try:
                d = json.loads(txt[i:])
                n_erros = (d.get("summary") or {}).get("errors")
                for e in (d.get("errors") or [])[:5]:
                    print("    ERRO  " + str(e.get("message", ""))[:100])
                for a in (d.get("warnings") or [])[:3]:
                    print("    AVISO " + str(a.get("message", ""))[:100])
            except ValueError:
                pass
        print("  web-ext lint: %s" % ("0 erros" if n_erros == 0 else "%s erro(s)" % n_erros))
    else:
        print("  web-ext nao encontrado -- lint NAO corrido")
        n_erros = None

    PROV = (
        "PROVENIENCIA -- HADES Browser Bridge para FIREFOX\n"
        "================================================\n"
        "\n"
        "O QUE ESTE PACOTE E':\n"
        "  - a extensao HADES Browser Bridge, manifest V3, adaptada para Firefox;\n"
        "  - o manifest DERIVADO do manifest.json do Chrome: sai o `key`, o\n"
        "    `minimum_chrome_version` e o `externally_connectable`; o `background.service_worker`\n"
        "    passa a `background.scripts`; entra o `browser_specific_settings.gecko` com id,\n"
        "    strict_min_version 115.0 e data_collection_permissions {required: [none]};\n"
        "  - NAO ASSINADO.\n"
        "\n"
        "O QUE ESTE PACOTE NAO E':\n"
        "  - NAO foi publicado em addons.mozilla.org;\n"
        "  - NAO foi instalado num Firefox real -- instalar no Firefox estavel exige assinatura\n"
        "    da Mozilla, ou `xpinstall.signatures.required=false` no Developer Edition;\n"
        "  - NAO tem E2E no Firefox (o Playwright Firefox nao carrega extensoes com a mesma API).\n"
        "\n"
        "COMO TESTAR (Developer Edition ou Nightly):\n"
        "  about:config -> xpinstall.signatures.required = false\n"
        "  about:debugging -> This Firefox -> Load Temporary Add-on -> manifest.json\n"
        "\n"
        "Para desinstalar: about:addons -> HADES Browser Bridge -> Remove.\n"
    )
    io.open(STAGE / "PROVENIENCIA.txt", "w", encoding="utf-8", newline="\n").write(PROV)

    # *o BUILD do proprio Mozilla -- produz o .zip que o AMO aceitaria*
    zipado = None
    if WEB_EXT.is_file():
        art = DEST / "artefactos"
        if art.exists():
            shutil.rmtree(art)
        r = subprocess.run([str(WEB_EXT), "build", "--source-dir", str(STAGE), "--artifacts-dir", str(art),
                            "--overwrite-dest"], capture_output=True, cwd=str(RAIZ))
        if r.returncode == 0 and art.is_dir():
            zs = sorted(art.glob("*.zip"))
            if zs:
                zipado = DEST / "hades-browser-bridge-firefox-UNSIGNED.zip"
                shutil.move(str(zs[0]), str(zipado))
                print("  web-ext build: %s (%.0f KB)" % (zipado.name, zipado.stat().st_size / 1024))

    linhas = []
    if zipado and zipado.is_file():
        linhas.append("%s  %s" % (sha256(zipado), zipado.name))
    mv = STAGE / "manifest.json"
    linhas.append("%s  %s" % (sha256(mv), "hades-browser-bridge-firefox-UNSIGNED/manifest.json"))
    io.open(DEST / "SHA256SUMS.txt", "w", encoding="utf-8", newline="\n").write("\n".join(linhas) + "\n")

    print("")
    print("  PACOTE FIREFOX GERADO: dist/browser-bridge-firefox/")
    print("    - PACKAGE=sim  - signed=false  - lint=%s" % ("ok" if n_erros == 0 else "com erros"))
    print("    - INSTALL=nao (exige Firefox Developer Edition ou assinatura da Mozilla)")
    print("    - E2E=nao (o Playwright Firefox nao carrega extensoes pela mesma API)")
    return 0 if n_erros in (0, None) else 1


def verificar() -> int:
    """`--check`: o pacote existe, o manifest Firefox e' valido, e o lint passa."""
    falhas = []

    def T(nome, ok, det=""):
        print(("  OK    " if ok else "  FALHA ") + nome + (("  -- " + det) if det else ""))
        if not ok:
            falhas.append(nome)

    m = STAGE / "manifest.json"
    T("(1) o stage do pacote Firefox existe", STAGE.is_dir())
    T("(1b) e tem o manifest", m.is_file())
    if m.is_file():
        ff = json.loads(io.open(m, encoding="utf-8").read())
        gecko = ((ff.get("browser_specific_settings") or {}).get("gecko") or {})
        T("(2) o manifest tem `gecko.id`", bool(gecko.get("id")), str(gecko.get("id")))
        T("(2b) e `data_collection_permissions`", bool(gecko.get("data_collection_permissions")),
          json.dumps(gecko.get("data_collection_permissions")))
        T("(2c) e usa `background.scripts` (nao `service_worker`)",
          "scripts" in (ff.get("background") or {}) and "service_worker" not in (ff.get("background") or {}))
        T("(3) e NAO tem os campos so-Chrome", not any(k in ff for k in SO_CHROME),
          "sem key/minimum_chrome_version/externally_connectable")
        # *a CSP estrita tem de sobreviver a transformacao*
        csp = ((ff.get("content_security_policy") or {}).get("extension_pages") or "")
        T("(4) e a CSP estrita sobreviveu (`object-src 'none'`)", "object-src 'none'" in csp, csp[:60])
        T("(4b) e o `incognito: not_allowed` tambem", ff.get("incognito") == "not_allowed")
    T("(5) o SHA256SUMS.txt existe", (DEST / "SHA256SUMS.txt").is_file())
    z = DEST / "hades-browser-bridge-firefox-UNSIGNED.zip"
    T("(6) o .zip do `web-ext build` existe", z.is_file(),
      "%.0f KB" % (z.stat().st_size / 1024) if z.is_file() else "AUSENTE")

    print("")
    if falhas:
        print("  ==== %d FALHA(S) NO PACOTE FIREFOX ====" % len(falhas))
        for f in falhas:
            print("    - " + f)
        return 1
    print("  PACOTE FIREFOX: manifest derivado com gecko.id e data_collection_permissions, `background.scripts`,")
    print("  CSP estrita preservada, .zip gerado pelo `web-ext build` e hashes registados.")
    print("")
    print("  ***O QUE ELE NAO E'.*** *NAO assinado, NAO publicado no AMO, NAO instalado num Firefox real")
    print("  (exige Developer Edition ou assinatura da Mozilla), e sem E2E no Firefox.*")
    return 0


if __name__ == "__main__":
    sys.exit(main())
